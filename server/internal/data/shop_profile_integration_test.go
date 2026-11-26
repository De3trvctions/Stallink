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

func TestShopProfileReadWriteAndSlugUntouched(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "shop_profile")
	store := NewShopStore(pool)
	var rawID string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name, slug, is_enabled) VALUES ('晚风', 'wanfeng', false) RETURNING id::text`).Scan(&rawID); err != nil {
		t.Fatal(err)
	}
	shopID := integrationID(t, rawID)

	got, err := store.ShopProfile(ctx, shopID)
	if err != nil {
		t.Fatalf("read disabled shop profile: %v", err)
	}
	if got.Slug != "wanfeng" || got.Enabled || got.Contact != "" || got.BoothNumber != "" || got.DefaultCurrency != nil || got.FeaturedEvent != nil || got.OnSaleCount != 0 || got.UpdatedAt.IsZero() {
		t.Fatalf("unexpected initial profile: %+v", got)
	}

	sgd := "SGD"
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{
		Name: "晚风工作室", Notice: "现场自取", Enabled: true, Contact: "IG @wanfeng.studio", BoothNumber: "A-27", DefaultCurrency: &sgd,
	}); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	got, err = store.ShopProfile(ctx, shopID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "晚风工作室" || got.Notice != "现场自取" || !got.Enabled || got.Contact != "IG @wanfeng.studio" || got.BoothNumber != "A-27" || got.DefaultCurrency == nil || *got.DefaultCurrency != "SGD" {
		t.Fatalf("unexpected updated profile: %+v", got)
	}
	if got.Slug != "wanfeng" {
		t.Fatalf("slug must stay platform-assigned, got %q", got.Slug)
	}
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{Name: "晚风工作室", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if got, _ = store.ShopProfile(ctx, shopID); got.DefaultCurrency != nil {
		t.Fatalf("null defaultCurrency must clear the value, got %v", *got.DefaultCurrency)
	}
	missing := integrationID(t, "999999999")
	if _, err := store.ShopProfile(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing shop: want ErrNotFound, got %v", err)
	}
	if err := store.UpdateShopProfile(ctx, missing, ShopProfileInput{Name: "x", Enabled: true}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing shop: want ErrNotFound, got %v", err)
	}
}

func TestShopProfileDisableRevokesSessionsAtomically(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "shop_profile_disable")
	store := NewShopStore(pool)
	shopID := insertProfileShop(t, ctx, pool, "停用店", "disable-shop")
	var user string
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('disable-owner','hash','shop_admin',$1) RETURNING id::text`, shopID).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(user_id, token_hash, expires_at) VALUES ($1, '\x01', now() + interval '1 hour')`, user); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{Name: "停用店", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, user).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("enabled save must keep sessions: %d %v", sessions, err)
	}
	// A CHECK violation in the same call must roll back the session revocation too.
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{Name: "停用店", Enabled: false, BoothNumber: " bad "}); err == nil {
		t.Fatal("padded booth number must be rejected by the database check")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, user).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("failed save must not revoke sessions: %d %v", sessions, err)
	}
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{Name: "停用店", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, user).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("disable must revoke sessions in the same call: %d %v", sessions, err)
	}
}

func TestShopProfileFeaturedEventAndOnSaleCount(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "shop_profile_onsale")
	store := NewShopStore(pool)
	shopID := insertProfileShop(t, ctx, pool, "预览店", "preview-shop")
	setStatus := func(eventID ID, status, booth string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE events SET status = $2, booth_number = $3 WHERE id = $1`, eventID, status, booth); err != nil {
			t.Fatal(err)
		}
	}
	ended := insertProfileEvent(t, ctx, pool, shopID, "已结束", "2026-01-01", "MYR")
	setStatus(ended, "ended", "Z-1")
	scheduled := insertProfileEvent(t, ctx, pool, shopID, "未开始", "2026-02-01", "SGD")
	setStatus(scheduled, "scheduled", "S-1")
	if got, err := store.ShopProfile(ctx, shopID); err != nil || got.FeaturedEvent == nil || got.FeaturedEvent.ID != scheduled {
		t.Fatalf("without live events the first scheduled event is shown: %+v %v", got.FeaturedEvent, err)
	}
	live := insertProfileEvent(t, ctx, pool, shopID, "进行中", "2026-03-01", "MYR")
	setStatus(live, "live", "L-1")
	got, err := store.ShopProfile(ctx, shopID)
	if err != nil || got.FeaturedEvent == nil || got.FeaturedEvent.ID != live || got.FeaturedEvent.BoothNumber != "L-1" || got.FeaturedEvent.Currency != "MYR" || got.FeaturedEvent.Status != "live" || got.FeaturedEvent.Name != "进行中" {
		t.Fatalf("a live event beats a scheduled one: %+v %v", got.FeaturedEvent, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE shops SET featured_event_id = $2 WHERE id = $1`, shopID, scheduled); err != nil {
		t.Fatal(err)
	}
	if got, err = store.ShopProfile(ctx, shopID); err != nil || got.FeaturedEvent == nil || got.FeaturedEvent.ID != scheduled {
		t.Fatalf("the featured event wins: %+v %v", got.FeaturedEvent, err)
	}

	addListing := func(eventID ID, name string, stock int, review string, listed bool, reserved int) {
		t.Helper()
		var good string
		if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id, name, total_stock, review_status) VALUES ($1, $2, $3, $4) RETURNING id::text`, shopID, name, stock, review).Scan(&good); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO event_listings(shop_id, event_id, good_id, listed_qty, unit_price_minor, is_listed, reserved_qty) VALUES ($1, $2, $3, $4, 1000, $5, $6)`,
			shopID, eventID, good, stock, listed, reserved); err != nil {
			t.Fatal(err)
		}
	}
	addListing(scheduled, "可售", 5, "approved", true, 1)
	addListing(scheduled, "预占卖完", 2, "approved", true, 2)
	addListing(scheduled, "待审核", 5, "pending", true, 0)
	addListing(scheduled, "已下架", 5, "approved", false, 0)
	addListing(scheduled, "零库存", 0, "approved", true, 0)
	addListing(live, "别的活动", 5, "approved", true, 0)
	if got, err = store.ShopProfile(ctx, shopID); err != nil || got.OnSaleCount != 1 {
		t.Fatalf("onSaleCount = %d %v, want 1 (listed, approved, sellable after reservations, current event only)", got.OnSaleCount, err)
	}
}
