package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type DB struct {
	pool *pgxpool.Pool
	url  string
}

func OpenDB(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db := &DB{pool: pool, url: url}
	if err := db.ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() {
	if db != nil && db.pool != nil {
		db.pool.Close()
	}
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (db *DB) migrate(ctx context.Context) error {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()
	// Serialize migrations across API nodes; CREATE TABLE IF NOT EXISTS is
	// still racy in PostgreSQL when two sessions execute it concurrently.
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(724173001)`); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_unlock(724173001)`); err != nil {
			log.Error().Err(err).Msg("unlock migrations failed")
		}
	}()
	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		var applied bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`,
			entry.Name(),
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return fmt.Errorf("apply migration %s: %w (rollback: %w)", entry.Name(), err, rollbackErr)
			}
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, entry.Name(),
		); err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				return fmt.Errorf("record migration %s: %w", entry.Name(), errors.Join(err, rollbackErr))
			}
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}
