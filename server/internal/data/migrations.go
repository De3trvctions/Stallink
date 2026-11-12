package data

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationLockID int64 = 0x5354414c4c494e4b

type migration struct {
	version uint64
	name    string
	sql     []byte
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Serialize bootstrap too: two fresh instances must not race to create
	// the base tables or the migration ledger.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin schema bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("lock schema bootstrap: %w", err)
	}
	for _, stmt := range schema {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("bootstrap schema: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version bigint PRIMARY KEY,
		name text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit schema bootstrap: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, m := range migrations {
		if err := applyMigration(ctx, pool, m); err != nil {
			return err
		}
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)
	loaded := make([]migration, 0, len(entries))
	for _, entry := range entries {
		base := path.Base(entry)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ".sql") {
			return nil, fmt.Errorf("invalid migration filename %q", base)
		}
		version, err := strconv.ParseUint(parts[0], 10, 63)
		if err != nil || version == 0 {
			return nil, fmt.Errorf("invalid migration version in %q", base)
		}
		contents, err := migrationFiles.ReadFile(entry)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", base, err)
		}
		loaded = append(loaded, migration{version: version, name: strings.TrimSuffix(parts[1], ".sql"), sql: contents})
	}
	for i := 1; i < len(loaded); i++ {
		if loaded[i-1].version == loaded[i].version {
			return nil, fmt.Errorf("duplicate migration version %d", loaded[i].version)
		}
	}
	return loaded, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %03d_%s: %w", m.version, m.name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("lock migration %03d_%s: %w", m.version, m.name, err)
	}
	var applied bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, m.version).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %03d_%s: %w", m.version, m.name, err)
	}
	if applied {
		return tx.Commit(ctx)
	}
	if _, err := tx.Conn().PgConn().Exec(ctx, string(m.sql)).ReadAll(); err != nil {
		return fmt.Errorf("run migration %03d_%s: %w", m.version, m.name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version, name) VALUES($1, $2)`, m.version, m.name); err != nil {
		return fmt.Errorf("record migration %03d_%s: %w", m.version, m.name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %03d_%s: %w", m.version, m.name, err)
	}
	return nil
}
