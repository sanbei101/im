package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"uuid"

	"github.com/cockroachdb/pebble"
	"github.com/phuslu/lru"
)

const (
	// Decoded archive files kept in memory: repeated cold paging re-filters
	// the same rows instead of re-decoding the whole parquet file. Entries
	// larger than decodedArchiveMaxBytes are never admitted, bounding the
	// cache at maxFiles * maxBytes.
	decodedArchiveMaxFiles = 8
	decodedArchiveMaxBytes = 4 << 20
)

// decodedArchive is a decoded parquet file. bytes is the rough decoded size
// used for the admission cap.
type decodedArchive struct {
	rows  []ParquetMessage
	bytes int64
}

func newDecodedArchive(rows []ParquetMessage) *decodedArchive {
	var bytes int64
	for i := range rows {
		bytes += int64(len(rows[i].Payload) + len(rows[i].Ext) + 64)
	}
	return &decodedArchive{rows: rows, bytes: bytes}
}

// remoteReader serves archived parquet files through a bounded on-disk cache,
// so repeated cold reads do not re-fetch the same object.
type remoteReader struct {
	objects  ObjectStore
	dir      string
	maxBytes int64
	decoded  *lru.LRUCache[string, *decodedArchive]

	mu    sync.Mutex
	bytes int64
}

// AttachRemote enables the S3 tier: archived messages become readable through
// the object store behind a local disk cache of cacheBytes. Without it the
// store serves only the cached tail and Pebble, which keeps the rollout of
// upload and deletion independent of the read path.
func (s *Store) AttachRemote(objects ObjectStore, cacheDir string, cacheBytes int64) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create archive cache dir: %w", err)
	}
	reader := &remoteReader{
		objects: objects, dir: cacheDir, maxBytes: cacheBytes,
		decoded: lru.NewLRUCache[string, *decodedArchive](decodedArchiveMaxFiles),
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			os.Remove(filepath.Join(cacheDir, entry.Name())) // clean crashed uploads
			continue
		}
		if info, err := entry.Info(); err == nil {
			reader.bytes += info.Size()
		}
	}
	s.remote = reader
	return nil
}

// readArchivedRange returns the archived messages with seq in [lo, hi) in
// ascending order, loading every covering manifest file through the disk cache.
// Files are sorted by start seq and never overlap, but the file containing lo
// starts before lo, so the manifest preceding the seek position is checked too.
func (s *Store) readArchivedRange(ctx context.Context, room uuid.UUID, lo, hi uint64) ([]Message, error) {
	if s.remote == nil || hi <= lo {
		return nil, nil
	}
	var prefixBuf [32]byte
	prefix := appendArchiveIndexPrefix(prefixBuf[:0], room)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: prefixUpperBound(prefix),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var messages []Message
	loKey := appendArchiveIndexKey(nil, room, lo)
	if valid := iter.SeekGE(loKey); !valid {
		// No manifest starts at or above lo: only the newest one, whose start
		// lies below lo, can still cover the window.
		if valid := iter.Last(); valid {
			if err := s.appendArchiveFile(ctx, iter, lo, hi, &messages); err != nil {
				return nil, err
			}
		}
	} else {
		if iter.Prev() {
			// The predecessor may reach into the window from below.
			if err := s.appendArchiveFile(ctx, iter, lo, hi, &messages); err != nil {
				return nil, err
			}
		}
		for seek := iter.SeekGE(loKey); seek; seek = iter.Next() {
			if err := contextErr(ctx); err != nil {
				return nil, err
			}
			start, ok := archiveIndexStart(iter.Key())
			if !ok {
				continue
			}
			if start >= hi {
				break
			}
			if err := s.appendArchiveFile(ctx, iter, lo, hi, &messages); err != nil {
				return nil, err
			}
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return messages, nil
}

// appendArchiveFile loads the manifest's parquet file through the disk cache
// and appends its rows that fall inside [lo, hi). iter is positioned on the
// manifest record.
func (s *Store) appendArchiveFile(
	ctx context.Context,
	iter *pebble.Iterator,
	lo, hi uint64,
	messages *[]Message,
) error {
	manifest, err := decodeManifest(iter.Value())
	if err != nil {
		return err
	}
	if start, ok := archiveIndexStart(iter.Key()); ok && start >= hi {
		return nil
	}
	if manifest.EndSeq < lo {
		return nil
	}
	decoded, ok := s.remote.decoded.Get(manifest.S3Key)
	if !ok {
		data, err := s.remote.load(ctx, manifest.S3Key)
		if err != nil {
			return fmt.Errorf("load archive %s: %w", manifest.S3Key, err)
		}
		rows, err := decodeParquet(data)
		if err != nil {
			return err
		}
		decoded = newDecodedArchive(rows)
		if decoded.bytes <= decodedArchiveMaxBytes {
			s.remote.decoded.Set(manifest.S3Key, decoded)
		}
	}
	for i := range decoded.rows {
		if decoded.rows[i].RoomSeq >= lo && decoded.rows[i].RoomSeq < hi {
			*messages = append(*messages, decoded.rows[i].message())
		}
	}
	return nil
}

// remoteMessage fetches a single archived message by seq, used by MessageByID
// once the local body is gone.
func (s *Store) remoteMessage(ctx context.Context, roomID uuid.UUID, seq uint64) (Message, error) {
	if s.remote == nil {
		return Message{}, ErrNotFound
	}
	watermark, err := s.archiveWatermark(roomID)
	if err != nil {
		return Message{}, err
	}
	if seq > watermark {
		return Message{}, ErrNotFound
	}
	messages, err := s.readArchivedRange(ctx, roomID, seq, seq+1)
	if err != nil {
		return Message{}, err
	}
	if len(messages) == 0 {
		return Message{}, ErrNotFound
	}
	return messages[0], nil
}

func archiveIndexStart(key []byte) (uint64, bool) {
	if len(key) != 25 {
		return 0, false
	}
	return binary.BigEndian.Uint64(key[17:]), true
}

func (r *remoteReader) load(ctx context.Context, key string) ([]byte, error) {
	path := r.path(key)
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	data, err := r.objects.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	r.storeFile(path, data) // a failed cache write must not fail the read
	return data, nil
}

func (r *remoteReader) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(r.dir, hex.EncodeToString(sum[:]))
}

func (r *remoteReader) storeFile(path string, data []byte) {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return
	}
	r.mu.Lock()
	r.bytes += int64(len(data))
	for r.bytes > r.maxBytes {
		if !r.evictOne() {
			break
		}
	}
	r.mu.Unlock()
}

// evictOne removes the least recently written cached file. Caller holds r.mu.
func (r *remoteReader) evictOne() bool {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return false
	}
	oldestName, oldestInfo := "", fs.FileInfo(nil)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if oldestInfo == nil || info.ModTime().Before(oldestInfo.ModTime()) {
			oldestName, oldestInfo = entry.Name(), info
		}
	}
	if oldestInfo == nil {
		r.bytes = 0
		return false
	}
	if err := os.Remove(filepath.Join(r.dir, oldestName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false
	}
	r.bytes -= oldestInfo.Size()
	if r.bytes < 0 {
		r.bytes = 0
	}
	return true
}
