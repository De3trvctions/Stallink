//go:build integration

package data

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	shopID := integrationID(t, shop)
	otherID := integrationID(t, other)
	eventID := integrationID(t, event)
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
	p1, more, err := store.ListOrderPage(ctx, shopID, "", "", "", "", "", 20, 0)
	if err != nil || len(p1) != 20 || !more {
		t.Fatalf("page 1: %d %v %v", len(p1), more, err)
	}
	p2, more, err := store.ListOrderPage(ctx, shopID, "", "", "", "", "", 20, 20)
	if err != nil || len(p2) != 20 || !more {
		t.Fatalf("page 2: %d %v %v", len(p2), more, err)
	}
	seen := map[ID]bool{}
	for _, o := range p1 {
		seen[o.ID] = true
	}
	for _, o := range p2 {
		if seen[o.ID] {
			t.Fatalf("duplicate order %s", o.ID)
		}
	}
	p3, more, err := store.ListOrderPage(ctx, shopID, "", "", "", "", "", 20, 40)
	if err != nil || len(p3) != 5 || more {
		t.Fatalf("page 3: %d %v %v", len(p3), more, err)
	}
	result, _, err := store.ListOrderPage(ctx, shopID, eventID.String(), "", "顾客12", "", "", 20, 0)
	if err != nil || len(result) != 1 {
		t.Fatalf("search: %d %v", len(result), err)
	}
	result, _, err = store.ListOrderPage(ctx, otherID, "", "", "", "", "", 20, 0)
	if err != nil || len(result) != 0 {
		t.Fatalf("tenant isolation: %d %v", len(result), err)
	}
	result, _, err = store.ListOrderPage(ctx, shopID, "", "", "%", "", "", 20, 0)
	if err != nil || len(result) != 0 {
		t.Fatalf("literal search: %d %v", len(result), err)
	}
	logs, more, err := store.ListAuditPage(ctx, shopID, "创建订单", 20, 100)
	if err != nil || len(logs) != 5 || more {
		t.Fatalf("audit page: %d %v %v", len(logs), more, err)
	}
	logs, _, err = store.ListAuditPage(ctx, otherID, "", 20, 0)
	if err != nil || len(logs) != 0 {
		t.Fatalf("audit tenant isolation: %d %v", len(logs), err)
	}

	// S4: createdAt is a real timestamp that carries an offset.
	logs, _, err = store.ListAuditPage(ctx, shopID, "", 1, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("latest audit: %d %v", len(logs), err)
	}
	if logs[0].CreatedAt.IsZero() {
		t.Fatal("audit createdAt must be a real timestamp")
	}
	if got := logs[0].CreatedAt.Format(time.RFC3339); !strings.ContainsAny(got[len(got)-6:], "+-Z") {
		t.Fatalf("audit createdAt must carry an offset, got %q", got)
	}

	// S3: subjectLabel resolves live names per subject type within the shop.
	var orderID, orderNumber, good, listing, foreignOrder string
	if err = pool.QueryRow(ctx, `SELECT id::text, number FROM orders WHERE shop_id = $1 ORDER BY id LIMIT 1`, shop).Scan(&orderID, &orderNumber); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'丝印海报',5) RETURNING id::text`, shop).Scan(&good); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor) VALUES ($1,$2,$3,5,1000) RETURNING id::text`, shop, event, good).Scan(&listing); err != nil {
		t.Fatal(err)
	}
	var otherActor, otherEvent string
	if err = pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('other-owner','hash','shop_admin',$1) RETURNING id::text`, other).Scan(&otherActor); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'别家活动',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, other).Scan(&otherEvent); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role) VALUES ($1,$2,'ZZZZZZ','pending',100,'MYR',$3,'shop_admin') RETURNING id::text`, other, otherEvent, otherActor).Scan(&foreignOrder); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ action, subjectType, subjectID string }{
		{"order.pay", "order", orderID},
		{"stock.update", "good", good},
		{"listing.update", "listing", listing},
		{"order.expire", "order", "999999999"},
		{"order.cancel", "order", foreignOrder},
	} {
		if _, err = pool.Exec(ctx, `INSERT INTO audit_logs(shop_id,actor_id,action,subject_type,subject_id) VALUES ($1,$2,$3,$4,$5)`, shop, actor, row.action, row.subjectType, row.subjectID); err != nil {
			t.Fatal(err)
		}
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, orderNumber, 20, 0)
	if err != nil || len(logs) != 1 || logs[0].SubjectType != "order" || logs[0].SubjectLabel != orderNumber {
		t.Fatalf("search by order number: %+v %v", logs, err)
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, "丝印海报", 20, 0)
	if err != nil || len(logs) != 2 {
		t.Fatalf("search by good name: %+v %v", logs, err)
	}
	labels := map[string]string{}
	for _, l := range logs {
		labels[l.SubjectType] = l.SubjectLabel
	}
	if labels["good"] != "丝印海报" || labels["listing"] != "丝印海报 · 活动A" {
		t.Fatalf("good/listing labels: %v", labels)
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, "999999999", 20, 0)
	if err != nil || len(logs) != 1 || logs[0].SubjectLabel != "" {
		t.Fatalf("orphan subject must have empty label: %+v %v", logs, err)
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, "取消订单", 20, 0)
	if err != nil || len(logs) != 1 || logs[0].SubjectLabel != "" {
		t.Fatalf("subject from another shop must not resolve a label: %+v %v", logs, err)
	}
	logs, _, err = store.ListAuditPage(ctx, otherID, orderNumber, 20, 0)
	if err != nil || len(logs) != 0 {
		t.Fatalf("label search tenant isolation: %d %v", len(logs), err)
	}
}
