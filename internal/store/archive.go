package store

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
	"github.com/phuslu/log"
)

// ArchiveManifest describes one uploaded parquet file. The manifest key embeds
// StartSeq and manifests are written in seq order, so the newest manifest of a
// room is its archive watermark: seqs <= EndSeq live in the object store,
// seqs above it live in Pebble. Manifests and message deletes commit in the
// same batch, which keeps the two tiers disjoint at all times.
type ArchiveManifest struct {
	EndSeq   uint64
	MsgCount uint32
	MinTime  int64
	MaxTime  int64
	S3Key    string
}

func encodeManifest(manifest ArchiveManifest) []byte {
	data := make([]byte, 0, 8+4+8+8+4+len(manifest.S3Key))
	var number [8]byte
	putU64(number[:], manifest.EndSeq)
	data = append(data, number[:]...)
	binary.BigEndian.PutUint32(number[:4], manifest.MsgCount)
	data = append(data, number[:4]...)
	putI64(number[:], manifest.MinTime)
	data = append(data, number[:]...)
	putI64(number[:], manifest.MaxTime)
	data = append(data, number[:]...)
	return appendString(data, manifest.S3Key)
}

func decodeManifest(data []byte) (ArchiveManifest, error) {
	d := decoder{data: data}
	manifest := ArchiveManifest{}
	var err error
	if manifest.EndSeq, err = d.u64(); err != nil {
		return ArchiveManifest{}, err
	}
	if d.pos+4 > len(d.data) {
		return ArchiveManifest{}, errors.New("invalid manifest record")
	}
	manifest.MsgCount = binary.BigEndian.Uint32(d.data[d.pos:])
	d.pos += 4
	if manifest.MinTime, err = d.i64(); err != nil {
		return ArchiveManifest{}, err
	}
	if manifest.MaxTime, err = d.i64(); err != nil {
		return ArchiveManifest{}, err
	}
	if manifest.S3Key, err = d.string(); err != nil {
		return ArchiveManifest{}, err
	}
	if !d.done() {
		return ArchiveManifest{}, errors.New("invalid manifest record")
	}
	return manifest, nil
}

// ArchiveParams bounds and eligibility rules of the archiver. Injected by the
// caller so tests can shrink them; defaults keep the last 7 days on local disk.
type ArchiveParams struct {
	// Owns filters rooms down to the slots of this API node; nil keeps all.
	Owns        func(uuid.UUID) bool
	MaxAge      time.Duration // messages older than this become eligible
	MaxMessages int           // max messages archived per room per cycle
	MaxBytes    int           // soft bound of one parquet file
	Prefix      string        // object key prefix, e.g. "archives"
}

func DefaultArchiveParams() ArchiveParams {
	return ArchiveParams{
		MaxAge:      7 * 24 * time.Hour,
		MaxMessages: 2000,
		MaxBytes:    32 << 20,
		Prefix:      "archives",
	}
}

// ArchiveOnce archives the eligible message prefix of every owned room: it
// uploads a parquet file and then atomically records the manifest. Failed
// uploads are reported and retried on the next cycle; a crashed cycle leaves
// at worst an orphan object that the deterministic seq-range key lets the
// next run overwrite.
func (s *Store) ArchiveOnce(ctx context.Context, objects ObjectStore, params ArchiveParams) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if params.MaxMessages <= 0 || params.MaxBytes <= 0 {
		return errors.New("invalid archive params")
	}

	// Room scan is key-first: the room id comes straight from the key so the
	// ownership filter runs before any decode, and only a lightweight header
	// (no strings, no embedded message) is parsed for the surviving rooms.
	var prefixBuf [1]byte
	prefixBuf[0] = 'r'
	prefix := appendRoomPrefix(prefixBuf[:0])
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
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
		room, err := decodeRoomHead(iter.Value())
		if err != nil {
			return fmt.Errorf("decode room %s: %w", roomID, err)
		}
		if err := s.archiveRoom(ctx, objects, params, &room); err != nil {
			log.Warn().Err(err).Str("room", roomID.String()).Msg("archive room failed")
		}
	}
	return iter.Error()
}

func (s *Store) archiveRoom(ctx context.Context, objects ObjectStore, params ArchiveParams, room *Room) error {
	watermark, err := s.archiveWatermark(room.RoomID)
	if err != nil {
		return err
	}
	if room.LastSeq <= watermark {
		return nil
	}
	messages, err := s.collectArchivable(room.RoomID, watermark+1, room.LastSeq, params)
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

	// Deterministic seq-range key: re-uploads after a crash overwrite the
	// orphan instead of accumulating duplicates.
	key := fmt.Sprintf("%s/rooms/%s/%012d_%012d.parquet", params.Prefix, room.RoomID, start, end)
	if err := objects.Put(ctx, key, bytes.NewReader(data), int64(len(data))); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	manifest := ArchiveManifest{
		EndSeq: end, MsgCount: uint32(len(rows)),
		MinTime: minTime, MaxTime: maxTime, S3Key: key,
	}
	if err := s.setBytes(batch, appendArchiveIndexKey(nil, room.RoomID, start), encodeManifest(manifest)); err != nil {
		return err
	}
	// Range tombstone over the archived message bodies plus per-key dedup
	// records, atomically with the manifest: after this commit the range lives
	// only in the object store. The 'i' index is kept so MessageByID can still
	// resolve archived messages by id.
	if err := batch.DeleteRange(
		appendMessageKey(nil, room.RoomID, start),
		appendMessageKey(nil, room.RoomID, end+1),
		nil,
	); err != nil {
		return err
	}
	var dedupBuf [49]byte
	for i := range messages {
		if err := batch.Delete(
			appendDedupKey(dedupBuf[:0], room.RoomID, messages[i].SenderID, messages[i].ClientMsgID),
			nil,
		); err != nil {
			return err
		}
	}
	if err := commit(batch); err != nil {
		return fmt.Errorf("commit archive manifest: %w", err)
	}
	return nil
}

// collectArchivable scans the unarchived message prefix and returns the run of
// messages to archive: the oldest-first run of messages past the age cutoff,
// or the first MaxMessages unarchived messages once that many piled up
// (whichever applies), always bounded by MaxBytes.
func (s *Store) collectArchivable(room uuid.UUID, start, lastSeq uint64, params ArchiveParams) ([]Message, error) {
	cutoff := time.Now().Add(-params.MaxAge).UnixMicro()
	countTriggered := lastSeq-start+1 >= uint64(params.MaxMessages)

	var lowerBuf, upperBuf [25]byte
	iter, err := s.db.NewIter(&pebble.IterOptions{
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
// has never been archived.
func (s *Store) archiveWatermark(room uuid.UUID) (uint64, error) {
	var keyBuf [32]byte
	prefix := appendArchiveIndexPrefix(keyBuf[:0], room)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
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
	return manifest.EndSeq, nil
}
