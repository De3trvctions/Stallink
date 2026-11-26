//go:build integration

package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func execSQLFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if _, err := pool.Exec(ctx, string(raw)); err != nil {
		t.Fatalf("exec %s: %v", path, err)
	}
}

func insertProfileShop(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, slug string) ID {
	t.Helper()
	var raw string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name, slug) VALUES ($1, $2) RETURNING id::text`, name, slug).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	return integrationID(t, raw)
}

func insertProfileEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, shopID ID, name, startsOn, currency string) ID {
	t.Helper()
	var raw string
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id, name, starts_on, ends_on, currency, status)
		VALUES ($1, $2, $3::date, $3::date, $4, 'scheduled') RETURNING id::text`, shopID, name, startsOn, currency).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	return integrationID(t, raw)
}

func shopCurrency(t *testing.T, ctx context.Context, pool *pgxpool.Pool, shopID ID) (*string, time.Time) {
	t.Helper()
	var currency *string
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT default_currency, updated_at FROM shops WHERE id = $1`, shopID).Scan(&currency, &updatedAt); err != nil {
		t.Fatal(err)
	}
	return currency, updatedAt
}

func wantCurrency(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if want == "" {
		if got != nil {
			t.Fatalf("%s: want NULL default_currency, got %q", label, *got)
		}
		return
	}
	if got == nil || *got != want {
		t.Fatalf("%s: want default_currency %q, got %v", label, want, got)
	}
}

// TestMigration024RollbackReapplyAndNoBackfill rolls 024 back with the down
// script, seeds shops as they exist before 024, re-runs 024 and checks that it
// only changes structure (K-Q16 = A: no backfill in the automatic migration).
func TestMigration024RollbackReapplyAndNoBackfill(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "migration_024")

	execSQLFile(t, ctx, pool, migration024DownFile)
	var columns, indexes, applied int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema()
			AND ((table_name = 'shops' AND column_name IN ('contact', 'booth_number', 'default_currency'))
			  OR (table_name = 'platform_appearance' AND column_name IN ('updated_at', 'updated_by')))),
		(SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'audit_logs_shop_created_id_idx'),
		(SELECT count(*) FROM schema_migrations WHERE version = 24)`).Scan(&columns, &indexes, &applied); err != nil {
		t.Fatal(err)
	}
	if columns != 0 || indexes != 0 || applied != 0 {
		t.Fatalf("after rollback: columns=%d index=%d ledger=%d, want all 0", columns, indexes, applied)
	}

	featured := insertProfileShop(t, ctx, pool, "主推店", "featured-shop")
	eventID := insertProfileEvent(t, ctx, pool, featured, "新加坡场", "2026-01-10", "SGD")
	if _, err := pool.Exec(ctx, `UPDATE shops SET featured_event_id = $2 WHERE id = $1`, featured, eventID); err != nil {
		t.Fatal(err)
	}
	empty := insertProfileShop(t, ctx, pool, "空店", "empty-shop")
	var featuredBefore time.Time
	if err := pool.QueryRow(ctx, `SELECT updated_at FROM shops WHERE id = $1`, featured).Scan(&featuredBefore); err != nil {
		t.Fatal(err)
	}

	execSQLFile(t, ctx, pool, migration024File)
	// Running it a second time must be a no-op (IF NOT EXISTS guards).
	execSQLFile(t, ctx, pool, migration024File)

	for _, shopID := range []ID{featured, empty} {
		var contact, booth string
		var currency *string
		if err := pool.QueryRow(ctx, `SELECT contact, booth_number, default_currency FROM shops WHERE id = $1`, shopID).Scan(&contact, &booth, &currency); err != nil {
			t.Fatal(err)
		}
		if contact != "" || booth != "" || currency != nil {
			t.Fatalf("024 must not backfill shop %s: contact=%q booth=%q currency=%v", shopID, contact, booth, currency)
		}
	}
	if _, after := shopCurrency(t, ctx, pool, featured); !after.Equal(featuredBefore) {
		t.Fatalf("024 must not touch shops.updated_at: %v -> %v", featuredBefore, after)
	}
	var checks int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid = 'shops'::regclass
		AND conname IN ('shops_contact_check', 'shops_booth_number_check', 'shops_default_currency_check')`).Scan(&checks); err != nil {
		t.Fatal(err)
	}
	if checks != 3 {
		t.Fatalf("want 3 named shop checks after reapply, got %d", checks)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'audit_logs_shop_created_id_idx'`).Scan(&indexes); err != nil || indexes != 1 {
		t.Fatalf("audit index after reapply: %d %v", indexes, err)
	}
}

func TestMigration024ShopConstraints(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "migration_024_checks")
	shopID := insertProfileShop(t, ctx, pool, "约束店", "check-shop")
	for name, stmt := range map[string]struct {
		sql string
		arg any
	}{
		"booth padded":     {`UPDATE shops SET booth_number = $2 WHERE id = $1`, " A-27"},
		"booth too long":   {`UPDATE shops SET booth_number = $2 WHERE id = $1`, strings.Repeat("A", 41)},
		"booth control":    {`UPDATE shops SET booth_number = $2 WHERE id = $1`, "A\t27"},
		"contact too long": {`UPDATE shops SET contact = $2 WHERE id = $1`, strings.Repeat("客", 121)},
		"contact control":  {`UPDATE shops SET contact = $2 WHERE id = $1`, "IG\n@x"},
		"bad currency":     {`UPDATE shops SET default_currency = $2 WHERE id = $1`, "USD"},
	} {
		_, err := pool.Exec(ctx, stmt.sql, shopID, stmt.arg)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Fatalf("%s: want CHECK violation 23514, got %v", name, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE shops SET booth_number = $2, contact = $3, default_currency = 'CNY' WHERE id = $1`,
		shopID, strings.Repeat("A", 40), strings.Repeat("客", 120)); err != nil {
		t.Fatalf("boundary values must pass: %v", err)
	}
}

// TestManual024aBackfill: the manual script takes the currency of the most
// recent event (starts_on DESC, id DESC) and ignores the featured event.
func TestManual024aBackfillAndRollback(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "manual_024a")

	featuredOld := insertProfileShop(t, ctx, pool, "主推旧场", "featured-old")
	oldEvent := insertProfileEvent(t, ctx, pool, featuredOld, "旧场", "2026-01-01", "SGD")
	insertProfileEvent(t, ctx, pool, featuredOld, "新场", "2026-03-01", "MYR")
	if _, err := pool.Exec(ctx, `UPDATE shops SET featured_event_id = $2 WHERE id = $1`, featuredOld, oldEvent); err != nil {
		t.Fatal(err)
	}
	eventsOnly := insertProfileShop(t, ctx, pool, "只有活动", "events-only")
	insertProfileEvent(t, ctx, pool, eventsOnly, "更早", "2026-02-01", "MYR")
	insertProfileEvent(t, ctx, pool, eventsOnly, "最近", "2026-04-01", "CNY")
	noEvents := insertProfileShop(t, ctx, pool, "没有活动", "no-events")
	shops := []ID{featuredOld, eventsOnly, noEvents}
	want := []string{"MYR", "CNY", ""}
	before := map[ID]time.Time{}
	for _, id := range shops {
		_, before[id] = shopCurrency(t, ctx, pool, id)
	}

	for run := 1; run <= 2; run++ {
		execSQLFile(t, ctx, pool, manual024aBackfillFile)
		for i, id := range shops {
			got, updatedAt := shopCurrency(t, ctx, pool, id)
			wantCurrency(t, "backfill run", got, want[i])
			if !updatedAt.Equal(before[id]) {
				t.Fatalf("run %d: backfill must not change updated_at of shop %s", run, id)
			}
		}
	}

	execSQLFile(t, ctx, pool, manual024aRollbackFile)
	for _, id := range shops {
		got, _ := shopCurrency(t, ctx, pool, id)
		wantCurrency(t, "after rollback", got, "")
	}
	var backup *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass('manual_024a_shop_currency_backup')::text`).Scan(&backup); err != nil {
		t.Fatal(err)
	}
	if backup != nil {
		t.Fatalf("rollback must drop the backup table, still have %s", *backup)
	}

	// An owner edit after the backfill survives the rollback.
	execSQLFile(t, ctx, pool, manual024aBackfillFile)
	if _, err := pool.Exec(ctx, `UPDATE shops SET default_currency = 'SGD', updated_at = clock_timestamp() WHERE id = $1`, eventsOnly); err != nil {
		t.Fatal(err)
	}
	execSQLFile(t, ctx, pool, manual024aRollbackFile)
	got, _ := shopCurrency(t, ctx, pool, eventsOnly)
	wantCurrency(t, "owner-edited shop after rollback", got, "SGD")
	got, _ = shopCurrency(t, ctx, pool, featuredOld)
	wantCurrency(t, "untouched shop after rollback", got, "")
}
