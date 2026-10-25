package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"uuid"

	"github.com/cockroachdb/pebble"
	"github.com/phuslu/lru"
)

// Slot migration, the offline twin of the static shard topology: when the
// node count changes, every record of a room whose owner changed must land on
// the new owner's engine before the cluster restarts under the new topology.
// Migration runs with all API processes stopped — Pebble's directory lock
// makes a concurrent run fail loudly instead of racing.

// One migration batch is bounded by both entry count and bytes so a single
// room with a huge history streams through in chunks instead of one batch.
const (
	migrateBatchEntries = 4096
	migrateBatchBytes   = 4 << 20
)

// MigrateParams bounds one Migrate run.
type MigrateParams struct {
	// Owner maps a room to its destination index in the dsts slice passed to
	// Migrate, or -1 to keep the room on this engine.
	Owner func(uuid.UUID) int
	// DryRun reports what would move without touching either engine.
	DryRun bool
}

// MigrateStats counts what one Migrate pass moved.
type MigrateStats struct {
	Keys  int
	Bytes int64
	Files int
}

// migrateBuffer accumulates one destination's pending move: records staged in
// both engines' batches plus the source archive files backing the staged
// manifests. Source deletes commit only after the destination batch is
// durable, so an interrupted pass is a safe no-op that a re-run finishes.
type migrateBuffer struct {
	dst     *Engine
	set     *pebble.Batch
	del     *pebble.Batch
	entries int
	bytes   int64
	files   []string
}

// Migrate moves every record of every room that Owner reassigns away from
// this engine into the destination engines, archive files included. Records
// are copied verbatim, so seqs, dedup credentials and cursors survive the
// move unchanged. It is idempotent: a pass interrupted between the
// destination and source commits converges when re-run.
func (e *Engine) Migrate(ctx context.Context, dsts []*Engine, params MigrateParams) (MigrateStats, error) {
	if params.Owner == nil {
		return MigrateStats{}, errors.New("engine: migrate owner function is required")
	}
	var stats MigrateStats
	buffers := make([]*migrateBuffer, len(dsts))
	defer func() {
		for _, buffer := range buffers {
			if buffer != nil {
				buffer.set.Close()
				buffer.del.Close()
			}
		}
	}()

	flush := func(buffer *migrateBuffer) error {
		if buffer.entries == 0 {
			return nil
		}
		if !params.DryRun {
			if err := commit(buffer.set); err != nil {
				return fmt.Errorf("commit destination batch: %w", err)
			}
			if err := commit(buffer.del); err != nil {
				return fmt.Errorf("commit source deletes: %w", err)
			}
			for _, file := range buffer.files {
				path := filepath.Join(e.archiveDir, file)
				if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("remove migrated archive file: %w", err)
				}
				// Drop the room's archive directory once its last file is
				// gone; a non-empty directory refuses the removal harmlessly.
				os.Remove(filepath.Dir(path))
			}
		}
		buffer.set.Reset()
		buffer.del.Reset()
		stats.Keys += buffer.entries
		stats.Bytes += buffer.bytes
		stats.Files += len(buffer.files)
		buffer.entries = 0
		buffer.bytes = 0
		buffer.files = buffer.files[:0]
		return nil
	}

	iter, err := e.db.NewIter(nil)
	if err != nil {
		return stats, fmt.Errorf("scan source: %w", err)
	}
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		if err := contextErr(ctx); err != nil {
			return stats, err
		}
		key, value := iter.Key(), iter.Value()
		room, ok := keyRoom(key)
		if !ok {
			continue
		}
		index := params.Owner(room)
		if index < 0 {
			continue
		}
		if index >= len(buffers) {
			return stats, fmt.Errorf("owner returned destination %d, have %d", index, len(buffers))
		}
		buffer := buffers[index]
		if buffer == nil {
			if dsts[index] == nil {
				return stats, fmt.Errorf("destination %d is nil", index)
			}
			buffer = &migrateBuffer{
				dst: dsts[index],
				set: dsts[index].db.NewBatch(),
				del: e.db.NewBatch(),
			}
			buffers[index] = buffer
		}
		// A manifest also carries its parquet file: copy the file before the
		// manifest becomes visible on the destination, and remove it from the
		// source only after both commits.
		if len(key) == archiveIndexKeyLen && key[0] == prefixArchive {
			manifest, err := decodeManifest(value)
			if err != nil {
				return stats, fmt.Errorf("decode archive manifest: %w", err)
			}
			if !params.DryRun {
				if buffer.dst.archiveDir == "" {
					return stats, fmt.Errorf("destination has no archive directory for %s", manifest.File)
				}
				if err := e.copyArchiveFile(buffer.dst, manifest.File); err != nil {
					return stats, err
				}
			}
			buffer.files = append(buffer.files, manifest.File)
		}
		if err := buffer.set.Set(key, value, nil); err != nil {
			return stats, fmt.Errorf("stage destination record: %w", err)
		}
		if err := buffer.del.Delete(key, nil); err != nil {
			return stats, fmt.Errorf("stage source delete: %w", err)
		}
		buffer.entries++
		buffer.bytes += int64(len(key) + len(value))
		if buffer.entries >= migrateBatchEntries || buffer.bytes >= migrateBatchBytes {
			if err := flush(buffer); err != nil {
				return stats, err
			}
		}
	}
	if err := iter.Error(); err != nil {
		return stats, fmt.Errorf("scan source: %w", err)
	}
	for _, buffer := range buffers {
		if buffer == nil {
			continue
		}
		if err := flush(buffer); err != nil {
			return stats, err
		}
	}
	if !params.DryRun {
		// Migration rewrote both stores behind every in-memory view's back;
		// the offline contract means no reader races this wholesale reset.
		e.dropCaches()
		for _, dst := range dsts {
			dst.dropCaches()
		}
	}
	return stats, nil
}

// dropCaches forgets every in-memory view so subsequent reads repopulate
// from the migrated Pebble state.
func (e *Engine) dropCaches() {
	if e == nil {
		return
	}
	e.cache = newMessageCache()
	e.dedupCache = lru.NewLRUCache[dedupLookupKey, dedup](dedupCacheSize)
	e.archiveMarks = lru.NewLRUCache[uuid.UUID, uint64](
		archiveMarkRooms,
		lru.WithShards[uuid.UUID, uint64](64),
	)
	e.decodedCache = lru.NewLRUCache[string, *decodedArchive](decodedArchiveMaxFiles)
	for i := range e.roomShards {
		e.roomShards[i].rooms = lru.NewLRUCache[uuid.UUID, uint64](roomCachePerShard)
	}
}

// copyArchiveFile lands one parquet file of this engine's archive directory
// under the same relative path in dst's.
func (e *Engine) copyArchiveFile(dst *Engine, file string) error {
	data, err := os.ReadFile(filepath.Join(e.archiveDir, file))
	if err != nil {
		return fmt.Errorf("read archive file: %w", err)
	}
	return writeFile(filepath.Join(dst.archiveDir, file), data)
}

// keyRoom extracts the room id from any engine key. Every family embeds the
// room id directly after its prefix byte; only the dedup family carries the
// two-byte day bucket first.
func keyRoom(key []byte) (uuid.UUID, bool) {
	var room uuid.UUID
	if len(key) < 2 {
		return room, false
	}
	offset := 1
	switch key[0] {
	case prefixRoom, prefixMessage, prefixArchive, prefixReaction, prefixPin, prefixRead:
	case prefixDedup:
		offset += dayLen
	default:
		return room, false
	}
	if len(key) < offset+uuidLen {
		return room, false
	}
	copy(room[:], key[offset:])
	return room, true
}
