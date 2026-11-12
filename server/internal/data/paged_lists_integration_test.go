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

func TestPagedListsSearchAndShopIsolation(t *testing.T) {
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
	schema := fmt.Sprintf("paged_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
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
	if err = Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var shop, other, actor, event string
	if err = pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Shop','shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Other','other') RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'活动A',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for i := 0; i < 45; i++ {
		number := fmt.Sprintf("AAAA%c%c", alphabet[i/len(alphabet)], alphabet[i%len(alphabet)])
		if _, err = pool.Exec(ctx, `INSERT INTO orders(shop_id,event_id,number,status,booth_name,total_minor,currency,opened_by,opened_by_role,created_at) VALUES ($1,$2,$3,'pending',$4,100,'MYR',$5,'shop_admin', '2026-01-01')`, shop, event, number, fmt.Sprintf("顾客%d", i), actor); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 105; i++ {
		if _, err = pool.Exec(ctx, `INSERT INTO audit_logs(shop_id,actor_id,action,subject_type,subject_id,created_at) VALUES($1,$2,'order.create','event',$3,'2026-01-01')`, shop, actor, event); err != nil {
			t.Fatal(err)
		}
	}
	store := NewShopStore(pool)
	p1, more, err := store.ListOrderPage(ctx, shop, "", "", 20, 0)
	if err != nil || len(p1) != 20 || !more {
		t.Fatalf("page 1: %d %v %v", len(p1), more, err)
	}
	p2, more, err := store.ListOrderPage(ctx, shop, "", "", 20, 20)
	if err != nil || len(p2) != 20 || !more {
		t.Fatalf("page 2: %d %v %v", len(p2), more, err)
	}
	seen := map[string]bool{}
	for _, o := range p1 {
		seen[o.ID] = true
	}
	for _, o := range p2 {
		if seen[o.ID] {
			t.Fatalf("duplicate order %s", o.ID)
		}
	}
	p3, more, err := store.ListOrderPage(ctx, shop, "", "", 20, 40)
	if err != nil || len(p3) != 5 || more {
		t.Fatalf("page 3: %d %v %v", len(p3), more, err)
	}
	result, _, err := store.ListOrderPage(ctx, shop, event, "顾客12", 20, 0)
	if err != nil || len(result) != 1 {
		t.Fatalf("search: %d %v", len(result), err)
	}
	result, _, err = store.ListOrderPage(ctx, other, "", "", 20, 0)
	if err != nil || len(result) != 0 {
		t.Fatalf("tenant isolation: %d %v", len(result), err)
	}
	result, _, err = store.ListOrderPage(ctx, shop, "", "%", 20, 0)
	if err != nil || len(result) != 0 {
		t.Fatalf("literal search: %d %v", len(result), err)
	}
	logs, more, err := store.ListAuditPage(ctx, shop, "创建订单", 20, 100)
	if err != nil || len(logs) != 5 || more {
		t.Fatalf("audit page: %d %v %v", len(logs), more, err)
	}
	logs, _, err = store.ListAuditPage(ctx, other, "", 20, 0)
	if err != nil || len(logs) != 0 {
		t.Fatalf("audit tenant isolation: %d %v", len(logs), err)
	}
}
