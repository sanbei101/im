package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

// The archive tier: messages past the retention window are exported to
// parquet files under the archive directory and deleted from Pebble in the
// same batch that records their manifest, so the two tiers stay disjoint at
// all times. The engine only owns the files on local disk; shipping them to
// remote storage is the caller's business.

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

// ArchiveManifest describes one exported parquet file. The manifest key embeds
// StartSeq and manifests are written in seq order, so the newest manifest of
// a room is its archive watermark: seqs <= EndSeq live in the archive
// directory, seqs above it live in Pebble.
type ArchiveManifest struct {
	EndSeq   uint64
	MsgCount uint32
	MinTime  int64
	MaxTime  int64
	File     string // parquet path relative to the archive directory
}

func encodeManifest(manifest ArchiveManifest) []byte {
	data := make([]byte, 0, 8+4+8+8+4+len(manifest.File))
	var number [8]byte
	putU64(number[:], manifest.EndSeq)
	data = append(data, number[:]...)
	binary.BigEndian.PutUint32(number[:4], manifest.MsgCount)
	data = append(data, number[:4]...)
	putI64(number[:], manifest.MinTime)
	data = append(data, number[:]...)
	putI64(number[:], manifest.MaxTime)
	data = append(data, number[:]...)
	return appendString(data, manifest.File)
}

func decodeManifest(data []byte) (ArchiveManifest, error) {
	d := decoder{data: data}
	manifest := ArchiveManifest{
		EndSeq:   d.u64(),
		MsgCount: d.u32(),
		MinTime:  d.i64(),
		MaxTime:  d.i64(),
		File:     d.string(),
	}
	if err := d.check("manifest"); err != nil {
		return ArchiveManifest{}, err
	}
	return manifest, nil
}

// ArchiveParams bounds and eligibility rules of the archiver. Injected by the
// caller so tests can shrink them; defaults keep the last 7 days in Pebble.
type ArchiveParams struct {
	// Owns filters rooms down to the slots of this API node; nil keeps all.
	Owns        func(uuid.UUID) bool
	MaxAge      time.Duration // messages older than this become eligible
	MaxMessages int           // max messages archived per room per cycle
	MaxBytes    int           // soft bound of one parquet file
}

func DefaultArchiveParams() ArchiveParams {
	return ArchiveParams{
		MaxAge:      7 * 24 * time.Hour,
		MaxMessages: 2000,
		MaxBytes:    32 << 20,
	}
}

// ArchiveOnce is the engine's periodic maintenance cycle: it expires dedup
// buckets past the idempotency window, then exports the eligible message
// prefix of every owned room as parquet files with atomic manifests. Failed
// exports are reported and retried on the next cycle; a crashed cycle leaves
// at worst a .tmp file that the next run overwrites.
func (e *Engine) ArchiveOnce(ctx context.Context, params ArchiveParams) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := e.expireDedup(); err != nil {
		return fmt.Errorf("expire dedup: %w", err)
	}
	if e.archiveDir == "" {
		return errors.New("engine: no archive directory configured")
	}
	if params.MaxMessages <= 0 || params.MaxBytes <= 0 {
		return errors.New("invalid archive params")
	}

	// Room scan is key-first: the room id comes straight from the key so the
	// ownership filter runs before any decode, and only the last sequence is
	// parsed for the surviving rooms.
	prefix := appendRoomPrefix(nil)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return fmt.Errorf("scan rooms: %w", err)
	}
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		if err := contextErr(ctx); err != nil {
			return err
		}
		key := iter.Key()
		if len(key) != len(prefix)+16 {
			continue
		}
		var roomID uuid.UUID
		copy(roomID[:], key[len(prefix):])
		if params.Owns != nil && !params.Owns(roomID) {
			continue
		}
		lastSeq, err := decodeLastSeq(iter.Value())
		if err != nil {
			return fmt.Errorf("decode room %s: %w", roomID, err)
		}
		if err := e.archiveRoom(params, roomID, lastSeq); err != nil {
			e.logger.Warn().Err(err).Str("room", roomID.String()).Msg("archive room failed")
		}
	}
	return iter.Error()
}

func (e *Engine) archiveRoom(params ArchiveParams, roomID uuid.UUID, lastSeq uint64) error {
	watermark, err := e.archiveWatermark(roomID)
	if err != nil {
		return err
	}
	if lastSeq <= watermark {
		return nil
	}
	messages, err := e.collectArchivable(roomID, watermark+1, lastSeq, params)
	if err != nil || len(messages) == 0 {
		return err
	}

	start, end := messages[0].RoomSeq, messages[len(messages)-1].RoomSeq
	rows := make([]ParquetMessage, len(messages))
	minTime, maxTime := messages[0].ServerTime, messages[0].ServerTime
	for i := range messages {
		rows[i] = newParquetMessage(&messages[i])
		minTime = min(minTime, messages[i].ServerTime)
		maxTime = max(maxTime, messages[i].ServerTime)
	}
	data, err := encodeParquet(rows)
	if err != nil {
		return err
	}

	// Deterministic seq-range name: re-exports after a crash overwrite the
	// leftover file instead of accumulating duplicates.
	file := fmt.Sprintf("rooms/%s/%012d_%012d.parquet", roomID, start, end)
	if err := writeFile(filepath.Join(e.archiveDir, file), data); err != nil {
		return err
	}

	batch := e.db.NewBatch()
	defer batch.Close()
	manifest := ArchiveManifest{
		EndSeq: end, MsgCount: uint32(len(rows)),
		MinTime: minTime, MaxTime: maxTime, File: file,
	}
	if err := e.setBytes(batch, appendArchiveIndexKey(nil, roomID, start), encodeManifest(manifest)); err != nil {
		return err
	}
	// Range tombstone over the archived message bodies, atomically with the
	// manifest: after this commit the range lives only in the archive
	// directory.
	if err := batch.DeleteRange(
		appendMessageKey(nil, roomID, start),
		appendMessageKey(nil, roomID, end+1),
		nil,
	); err != nil {
		return err
	}
	if err := commit(batch); err != nil {
		return fmt.Errorf("commit archive manifest: %w", err)
	}
	e.archiveMarks.Set(roomID, manifest.EndSeq)
	return nil
}

// expireDedup range-deletes dedup buckets older than the two-day replay
// window, so idempotency storage is bounded by traffic, not history.
func (e *Engine) expireDedup() error {
	today := dedupDay(time.Now())
	batch := e.db.NewBatch()
	defer batch.Close()
	var loBuf, hiBuf [dedupDayPrefixLen]byte
	lo := appendDedupDayPrefix(loBuf[:0], today-2)
	hi := appendDedupDayPrefix(hiBuf[:0], today-1)
	if err := batch.DeleteRange(lo, hi, nil); err != nil {
		return err
	}
	return commit(batch)
}

// writeFile lands data at path atomically: a crash mid-write leaves the .tmp
// behind and the real file absent, so a reader never sees a torn parquet.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write archive file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("land archive file: %w", err)
	}
	return nil
}

// collectArchivable scans the unarchived message prefix and returns the run of
// messages to archive: the oldest-first run of messages past the age cutoff,
// or the first MaxMessages unarchived messages once that many piled up
// (whichever applies), always bounded by MaxBytes.
func (e *Engine) collectArchivable(room uuid.UUID, start, lastSeq uint64, params ArchiveParams) ([]Message, error) {
	cutoff := time.Now().Add(-params.MaxAge).UnixMicro()
	countTriggered := lastSeq-start+1 >= uint64(params.MaxMessages)

	var lowerBuf, upperBuf [messageKeyLen]byte
	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: appendMessageKey(lowerBuf[:0], room, start),
		UpperBound: appendMessageKey(upperBuf[:0], room, lastSeq+1),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var collected []Message
	var totalBytes int
	for iter.First(); iter.Valid(); iter.Next() {
		value := append([]byte(nil), iter.Value()...)
		message, err := decodeMessage(value)
		if err != nil {
			return nil, err
		}
		old := message.ServerTime < cutoff
		if len(collected) == 0 {
			if !old && !countTriggered {
				return nil, nil // nothing eligible yet
			}
		} else {
			if (!old && !countTriggered) || len(collected) >= params.MaxMessages || totalBytes > params.MaxBytes {
				break
			}
		}
		collected = append(collected, message)
		totalBytes += len(message.Payload) + len(message.Ext) + 64
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return collected, nil
}

// archiveWatermark returns the room's archived-through seq, 0 when the room
// has never been archived. Only this node's archiver moves a watermark, and
// it refreshes the cache on every archive, so a cached mark never goes stale.
func (e *Engine) archiveWatermark(room uuid.UUID) (uint64, error) {
	if mark, ok := e.archiveMarks.Get(room); ok {
		return mark, nil
	}
	var keyBuf [archiveIndexPrefixLen]byte
	prefix := appendArchiveIndexPrefix(keyBuf[:0], room)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	if !iter.Last() {
		return 0, iter.Error()
	}
	manifest, err := decodeManifest(iter.Value())
	if err != nil {
		return 0, err
	}
	e.archiveMarks.Set(room, manifest.EndSeq)
	return manifest.EndSeq, nil
}

// readArchivedRange returns the archived messages with seq in [lo, hi) in
// ascending order, loading every covering file through the decoded cache.
// Files are sorted by start seq and never overlap, but the file containing lo
// starts before lo, so the manifest preceding the seek position is checked
// too. Covering files are resolved first so the result is allocated once.
func (e *Engine) readArchivedRange(ctx context.Context, room uuid.UUID, lo, hi uint64) ([]Message, error) {
	if e.archiveDir == "" || hi <= lo {
		return nil, nil
	}
	var prefixBuf [archiveIndexPrefixLen]byte
	prefix := appendArchiveIndexPrefix(prefixBuf[:0], room)
	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: prefixUpperBound(prefix),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var files []*decodedArchive
	rows := 0
	// resolve loads the decoded file of the manifest iter is positioned on
	// when it covers any part of [lo, hi).
	resolve := func() error {
		if start, ok := archiveIndexStart(iter.Key()); ok && start >= hi {
			return nil
		}
		manifest, err := decodeManifest(iter.Value())
		if err != nil {
			return err
		}
		if manifest.EndSeq < lo {
			return nil
		}
		decoded, err := e.archiveFile(manifest.File)
		if err != nil {
			return err
		}
		files = append(files, decoded)
		rows += len(decoded.rows)
		return nil
	}

	loKey := appendArchiveIndexKey(nil, room, lo)
	if valid := iter.SeekGE(loKey); !valid {
		// No manifest starts at or above lo: only the newest one, whose start
		// lies below lo, can still cover the window.
		if valid := iter.Last(); valid {
			if err := resolve(); err != nil {
				return nil, err
			}
		}
	} else {
		if iter.Prev() {
			// The predecessor may reach into the window from below.
			if err := resolve(); err != nil {
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
			if err := resolve(); err != nil {
				return nil, err
			}
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, min(rows, int(hi-lo)))
	for _, decoded := range files {
		for i := range decoded.rows {
			if seq := decoded.rows[i].RoomSeq; seq >= lo && seq < hi {
				messages = append(messages, decoded.rows[i].message())
			}
		}
	}
	return messages, nil
}

// archiveFile loads and decodes one parquet file, caching the decoded rows.
func (e *Engine) archiveFile(file string) (*decodedArchive, error) {
	if decoded, ok := e.decodedCache.Get(file); ok {
		return decoded, nil
	}
	data, err := os.ReadFile(filepath.Join(e.archiveDir, file))
	if err != nil {
		return nil, fmt.Errorf("read archive %s: %w", file, err)
	}
	rows, err := decodeParquet(data)
	if err != nil {
		return nil, err
	}
	decoded := newDecodedArchive(rows)
	if decoded.bytes <= decodedArchiveMaxBytes {
		e.decodedCache.Set(file, decoded)
	}
	return decoded, nil
}

// archivedMessage fetches a single archived message by seq, used by
// MessageByID once the local body is gone.
func (e *Engine) archivedMessage(ctx context.Context, roomID uuid.UUID, seq uint64) (Message, error) {
	if e.archiveDir == "" {
		return Message{}, ErrNotFound
	}
	watermark, err := e.archiveWatermark(roomID)
	if err != nil {
		return Message{}, err
	}
	if seq > watermark {
		return Message{}, ErrNotFound
	}
	messages, err := e.readArchivedRange(ctx, roomID, seq, seq+1)
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
