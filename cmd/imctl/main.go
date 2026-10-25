// imctl is the offline operations tool for the message plane. Its migrate
// command moves room data between API node stores when the shard topology
// changes: it must run while every API process is stopped, because Pebble's
// directory lock refuses a concurrent open and rooms must not move under a
// live topology.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/engine"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "migrate":
		err = runMigrate(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "imctl: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "imctl:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: imctl <command> [flags]

commands:
  migrate   move room data between API node stores after a topology change

migrate reassigns every room to its owner under the target topology and moves
the records the current stores hold for rooms they no longer own — messages,
room heads, dedup credentials, read cursors, pins, reactions and archive
files — into the new owner's store. The procedure:

  1. stop every gateway and API process
  2. imctl migrate -slots <shard.slots> -old-dirs <dirs> -new-dirs <dirs> [--dry-run]
  3. update node_index/node_count and gateway api_addrs to the new topology
  4. start the cluster

Dir lists are comma-separated and ordered by node_index. A node that keeps
its index keeps its directory; fresh nodes get fresh empty directories. An
interrupted pass is safe to re-run.
`)
}

func runMigrate(args []string) error {
	flags := flag.NewFlagSet("imctl migrate", flag.ContinueOnError)
	slots := flags.Int("slots", 1024, "shard slot count, equal to shard.slots of the cluster config")
	oldDirs := flags.String("old-dirs", "", "comma-separated store paths of the current nodes, ordered by node_index")
	newDirs := flags.String(
		"new-dirs",
		"",
		"comma-separated store paths under the target topology, ordered by node_index",
	)
	dryRun := flags.Bool("dry-run", false, "report what would move without writing")
	if err := flags.Parse(args); err != nil {
		return err
	}
	olds, err := splitDirs("old-dirs", *oldDirs)
	if err != nil {
		return err
	}
	news, err := splitDirs("new-dirs", *newDirs)
	if err != nil {
		return err
	}
	if err := validateTopology(*slots, olds, news); err != nil {
		return err
	}

	mode := ""
	if *dryRun {
		mode = " (dry run)"
	}
	fmt.Printf("migrating %d store(s) onto a %d-node topology, slots=%d%s\n",
		len(olds), len(news), *slots, mode)
	for _, oldDir := range olds {
		stats, err := migrateSource(oldDir, news, *slots, *dryRun)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", oldDir, err)
		}
		fmt.Printf("  %s: moved %d keys (%s), %d archive file(s)\n",
			oldDir, stats.Keys, humanBytes(stats.Bytes), stats.Files)
	}
	fmt.Println("migration complete")
	return nil
}

// migrateSource moves out of one old store every room the new topology
// assigns elsewhere. Destinations whose path equals the source are skipped:
// that node keeps its data where it is. A dry run accumulates into one
// throwaway engine instead of the real destinations, so it leaves the new
// stores' directories untouched.
func migrateSource(oldDir string, news []string, slots int, dryRun bool) (engine.MigrateStats, error) {
	var stats engine.MigrateStats
	src, err := engine.Open(oldDir, engine.ArchiveDir(filepath.Join(oldDir, "archive")))
	if err != nil {
		return stats, fmt.Errorf("open source: %w", err)
	}
	var targets []*engine.Engine
	var scratchDir string
	defer func() {
		closeErr := src.Close()
		for _, target := range targets {
			closeErr = errors.Join(closeErr, target.Close())
		}
		if closeErr != nil {
			fmt.Fprintln(os.Stderr, "imctl: close engine:", closeErr)
		}
		if scratchDir != "" {
			os.RemoveAll(scratchDir)
		}
	}()
	nodeToTarget := make([]int, len(news))
	for node := range nodeToTarget {
		nodeToTarget[node] = -1
	}
	for node, dir := range news {
		if dir == oldDir {
			continue
		}
		if dryRun {
			if len(targets) == 0 {
				scratchDir, err = os.MkdirTemp("", "imctl-migrate-")
				if err != nil {
					return stats, fmt.Errorf("create scratch store: %w", err)
				}
				dummy := filepath.Join(scratchDir, "dst")
				target, err := engine.Open(dummy, engine.ArchiveDir(filepath.Join(dummy, "archive")))
				if err != nil {
					return stats, fmt.Errorf("open scratch store: %w", err)
				}
				targets = append(targets, target)
			}
			nodeToTarget[node] = 0
			continue
		}
		target, err := engine.Open(dir, engine.ArchiveDir(filepath.Join(dir, "archive")))
		if err != nil {
			return stats, fmt.Errorf("open destination: %w", err)
		}
		targets = append(targets, target)
		nodeToTarget[node] = len(targets) - 1
	}
	owner := func(room uuid.UUID) int {
		slot, err := pkg.RoomSlot(room, slots)
		if err != nil {
			return -1
		}
		node, err := pkg.NodeIndex(slot, slots, len(news))
		if err != nil {
			return -1
		}
		return nodeToTarget[node]
	}
	return src.Migrate(context.Background(), targets, engine.MigrateParams{
		Owner: owner, DryRun: dryRun,
	})
}

func splitDirs(flagName, raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%s is required", flagName)
	}
	parts := strings.Split(raw, ",")
	dirs := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		dir := filepath.Clean(strings.TrimSpace(part))
		if dir == "" {
			return nil, fmt.Errorf("%s contains an empty path", flagName)
		}
		if seen[dir] {
			return nil, fmt.Errorf("%s lists %q twice", flagName, dir)
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func validateTopology(slots int, olds, news []string) error {
	if slots < 1 || slots > pkg.MaxSlots {
		return fmt.Errorf("slots must be within [1, %d]", pkg.MaxSlots)
	}
	if len(news) == 0 {
		return errors.New("new-dirs is required")
	}
	for _, dir := range olds {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("old store %s: %w", dir, err)
		}
		if len(entries) == 0 {
			return fmt.Errorf("old store %s is empty", dir)
		}
	}
	kept := make(map[string]bool, len(olds))
	for _, dir := range olds {
		kept[dir] = true
	}
	// A new path that is not a kept old store must be fresh, or a store that
	// a previously interrupted pass already filled — which holds only rooms
	// owned by that node. Anything else would leave rooms behind that the new
	// topology assigns elsewhere.
	for node, dir := range news {
		if kept[dir] {
			continue
		}
		if entries, err := os.ReadDir(dir); err != nil || len(entries) == 0 {
			continue // fresh: absent or empty
		}
		if err := ensureOwned(dir, node, len(news), slots); err != nil {
			return err
		}
	}
	return nil
}

// ensureOwned fails when a store holds records of rooms owned by other nodes
// under the target topology. It reuses Migrate as the detector: a dry-run
// that would move anything out means the store is not clean for this node.
func ensureOwned(dir string, node, nodeCount, slots int) error {
	e, err := engine.Open(dir, engine.ArchiveDir(filepath.Join(dir, "archive")))
	if err != nil {
		return fmt.Errorf("new store %s is not a usable store: %w", dir, err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "imctl: close engine:", err)
		}
	}()
	scratchDir, err := os.MkdirTemp("", "imctl-check-")
	if err != nil {
		return fmt.Errorf("create scratch store: %w", err)
	}
	defer os.RemoveAll(scratchDir)
	scratch := filepath.Join(scratchDir, "dst")
	dst, err := engine.Open(scratch, engine.ArchiveDir(filepath.Join(scratch, "archive")))
	if err != nil {
		return fmt.Errorf("open scratch store: %w", err)
	}
	defer dst.Close()

	stats, err := e.Migrate(context.Background(), []*engine.Engine{dst}, engine.MigrateParams{
		Owner: func(room uuid.UUID) int {
			slot, err := pkg.RoomSlot(room, slots)
			if err != nil {
				return -1
			}
			owner, err := pkg.NodeIndex(slot, slots, nodeCount)
			if err != nil {
				return -1
			}
			if owner == node {
				return -1
			}
			return 0
		},
		DryRun: true,
	})
	if err != nil {
		return fmt.Errorf("inspect new store %s: %w", dir, err)
	}
	if stats.Keys != 0 {
		return fmt.Errorf("new store %s holds %d keys of rooms owned by other nodes", dir, stats.Keys)
	}
	return nil
}

func humanBytes(bytes int64) string {
	switch value := float64(bytes); {
	case value >= 1<<30:
		return fmt.Sprintf("%.1f GiB", value/(1<<30))
	case value >= 1<<20:
		return fmt.Sprintf("%.1f MiB", value/(1<<20))
	case value >= 1<<10:
		return fmt.Sprintf("%.1f KiB", value/(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
