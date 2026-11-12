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

func TestUpdateBundleCreatesNewComponentAtomically(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("isolated test database required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("bundle_update_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var shop, actor, event, one, two string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Shop','shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'event',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	for _, id := range []*string{&one, &two} {
		if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'item',5) RETURNING id::text`, shop).Scan(id); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor) VALUES ($1,$2,$3,5,100)`, shop, event, *id); err != nil {
			t.Fatal(err)
		}
	}
	store := NewShopStore(pool)
	input := BundleInput{Name: "bundle", EventID: event, UnitPriceMinor: 150, IsListed: true, Components: []BundleComponentInput{{GoodID: one, Quantity: 1}, {GoodID: two, Quantity: 1}}}
	id, _, err := store.CreateBundle(ctx, shop, actor, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Components = append(input.Components, BundleComponentInput{DraftKey: "new", Quantity: 1, Draft: &BundleGoodDraft{Name: "new item", TotalStock: 3, ListedQty: 3, UnitPriceMinor: 100, IsListed: true}})
	if _, err := store.UpdateBundle(ctx, shop, actor, id, 1, input, false); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bundle_components WHERE bundle_id=$1`, id).Scan(&count); err != nil || count != 3 {
		t.Fatalf("components = %d; err = %v", count, err)
	}
	if _, err := store.UpdateBundle(ctx, shop, actor, id, 1, input, false); err == nil {
		t.Fatal("stale revision must fail")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM goods WHERE name='new item'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("draft count = %d; err = %v", count, err)
	}
}
