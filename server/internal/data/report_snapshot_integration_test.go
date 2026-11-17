//go:build integration

package data

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDailyReportSnapshotIsRepeatableAndReadOnly(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	schema := fmt.Sprintf("daily_report_snapshot_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
	}()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `CREATE TABLE snapshot_probe (value integer NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO snapshot_probe(value) VALUES (1)`); err != nil {
		t.Fatal(err)
	}

	tx, err := beginDailyReportSnapshot(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var first int
	if err := tx.QueryRow(ctx, `SELECT value FROM snapshot_probe`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `UPDATE `+schema+`.snapshot_probe SET value = 2`); err != nil {
		t.Fatal(err)
	}
	var second int
	if err := tx.QueryRow(ctx, `SELECT value FROM snapshot_probe`).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if first != second || second != 1 {
		t.Fatalf("repeatable snapshot values = %d, %d; want 1, 1", first, second)
	}
	if _, err := tx.Exec(ctx, `UPDATE snapshot_probe SET value = 3`); err == nil {
		t.Fatal("report snapshot transaction accepted a write")
	}
}
