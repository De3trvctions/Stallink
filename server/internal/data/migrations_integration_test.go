//go:build integration

package data

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationsFreshConcurrentAndIdempotent(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer adminPool.Close()

	schema := fmt.Sprintf("stallink_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
	})

	testConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer testPool.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < cap(errs); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Migrate(ctx, testPool)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	var versions int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version = 1`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if versions != 1 {
		t.Fatalf("migration version 1 recorded %d times", versions)
	}

	var shopID, userID, eventID, goodID, listingID, orderID string
	if err := testPool.QueryRow(ctx, `INSERT INTO shops(name, slug) VALUES ('Test Shop', 'test-shop') RETURNING id::text`).Scan(&shopID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO users(login_name, password_hash, role, shop_id) VALUES ('test-owner', 'test-hash', 'shop_admin', $1) RETURNING id::text`, shopID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO events(shop_id, name, starts_on, ends_on, currency, status) VALUES ($1, 'Test Event', CURRENT_DATE, CURRENT_DATE, 'MYR', 'scheduled') RETURNING id::text`, shopID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO goods(shop_id, name, total_stock) VALUES ($1, 'Test Good', 1) RETURNING id::text`, shopID).Scan(&goodID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO event_listings(shop_id, event_id, good_id, listed_qty, unit_price_minor) VALUES ($1, $2, $3, 1, 1) RETURNING id::text`, shopID, eventID, goodID).Scan(&listingID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO orders(shop_id, event_id, number, status, total_minor, currency, opened_by, opened_by_role) VALUES ($1, $2, '234567', 'pending', 0, 'MYR', $3, 'shop_admin') RETURNING id::text`, shopID, eventID, userID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, `INSERT INTO order_lines(order_id, good_id, listing_id, shop_id, event_id, good_name, unit_price_minor, qty, line_total_minor, snapshot_version) VALUES ($1, $2, $3, $4, $5, 'Free promotion', 0, 1, 0, 1)`, orderID, goodID, listingID, shopID, eventID); err != nil {
		t.Fatal(err)
	}

	if err := Migrate(ctx, testPool); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, testPool); err != nil {
		t.Fatal(err)
	}
	var zeroLine int64
	if err := testPool.QueryRow(ctx, `SELECT line_total_minor FROM order_lines WHERE order_id = $1`, orderID).Scan(&zeroLine); err != nil {
		t.Fatal(err)
	}
	if zeroLine != 0 {
		t.Fatalf("repeat migration rewrote a legitimate zero-value line to %d", zeroLine)
	}
}

func TestMigrationBackfillsLegacyLinesBySnapshotVersion(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer adminPool.Close()
	schemaName := fmt.Sprintf("stallink_legacy_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA "+schemaName+" CASCADE")
	})
	testConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	testConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer testPool.Close()
	for _, statement := range schema {
		if _, err := testPool.Exec(ctx, statement); err != nil {
			t.Fatalf("create legacy base schema: %v", err)
		}
	}
	var shopID, userID, eventID, goodID, listingID, orderID string
	if err := testPool.QueryRow(ctx, `INSERT INTO shops(name, slug) VALUES ('Legacy Shop', 'legacy-shop') RETURNING id::text`).Scan(&shopID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO users(login_name, password_hash, role, shop_id) VALUES ('legacy-owner', 'test-hash', 'shop_admin', $1) RETURNING id::text`, shopID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO events(shop_id, name, starts_on, ends_on, currency, status) VALUES ($1, 'Legacy Event', CURRENT_DATE, CURRENT_DATE, 'MYR', 'scheduled') RETURNING id::text`, shopID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO goods(shop_id, name, total_stock) VALUES ($1, 'Legacy Good', 2) RETURNING id::text`, shopID).Scan(&goodID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO event_listings(event_id, good_id, listed_qty, unit_price_minor) VALUES ($1, $2, 2, 20) RETURNING id::text`, eventID, goodID).Scan(&listingID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO orders(shop_id, event_id, number, status, total_minor, currency, opened_by, opened_by_role) VALUES ($1, $2, '234567', 'pending', 40, 'MYR', $3, 'shop_admin') RETURNING id::text`, shopID, eventID, userID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, `INSERT INTO order_lines(order_id, good_id, listing_id, good_name, unit_price_minor, qty, line_total_minor) VALUES ($1, $2, $3, 'Legacy Good', 20, 2, 0)`, orderID, goodID, listingID); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, testPool); err != nil {
		t.Fatal(err)
	}
	var migratedTotal int64
	if err := testPool.QueryRow(ctx, `SELECT line_total_minor FROM order_lines WHERE order_id = $1`, orderID).Scan(&migratedTotal); err != nil {
		t.Fatal(err)
	}
	if migratedTotal != 40 {
		t.Fatalf("legacy line total = %d, want 40", migratedTotal)
	}
	if _, err := testPool.Exec(ctx, `INSERT INTO order_lines(order_id, good_id, listing_id, shop_id, event_id, good_name, unit_price_minor, qty, line_total_minor, snapshot_version) VALUES ($1, $2, $3, $4, $5, 'Free promotion', 20, 1, 0, 1)`, orderID, goodID, listingID, shopID, eventID); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, testPool); err != nil {
		t.Fatal(err)
	}
	var legalZero int64
	if err := testPool.QueryRow(ctx, `SELECT line_total_minor FROM order_lines WHERE good_name = 'Free promotion'`).Scan(&legalZero); err != nil {
		t.Fatal(err)
	}
	if legalZero != 0 {
		t.Fatalf("repeat migration rewrote a valid zero-value snapshot to %d", legalZero)
	}
}
