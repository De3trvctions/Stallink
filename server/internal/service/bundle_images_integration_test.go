//go:build integration

package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBundleDraftImagesCreateAndEdit(t *testing.T) {
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
	schema := fmt.Sprintf("bundle_image_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") }()
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
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var shop, event string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES('shop','shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES('owner',$1,'shop_admin',$2)`, hash, shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES($1,'event',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	handler := server.NewHTTP(":0", "test", &service.API{DB: data.NewShopStore(pool), DataDir: dir, BundlesEnabled: true})
	cookie := login(t, handler, "owner", "password123")
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	draft := func(key, name string) map[string]any {
		return map[string]any{"draftKey": key, "quantity": 1, "draft": map[string]any{"name": name, "totalStock": 1, "listedQty": 1, "unitPriceMinor": 100, "isListed": true}}
	}
	send := func(method, path string, payload map[string]any, field string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		raw, _ := json.Marshal(payload)
		if err := writer.WriteField("bundle", string(raw)); err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile(field, "test.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(png); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(cookie)
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		return result
	}
	payload := map[string]any{"name": "bundle", "eventId": event, "unitPriceMinor": 150, "isListed": true, "components": []any{draft("a", "first"), draft("b", "second")}}
	created := send("POST", "/api/staff/bundles", payload, "componentImages.a")
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	var result struct {
		BundleID string `json:"bundleId"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	var good, path string
	if err := pool.QueryRow(ctx, `SELECT g.id::text,i.path FROM goods g JOIN good_images i ON i.good_id=g.id WHERE g.name='first'`).Scan(&good, &path); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "blobs", path)); err != nil || !bytes.Equal(got, png) {
		t.Fatalf("created image not saved: %v", err)
	}
	payload["expectedRevision"] = 1
	payload["components"] = []any{map[string]any{"goodId": good, "quantity": 1}, draft("c", "third")}
	updated := send("PATCH", "/api/staff/bundles/"+result.BundleID, payload, "componentImages.c")
	if updated.Code != http.StatusOK {
		t.Fatalf("update %d %s", updated.Code, updated.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT i.path FROM goods g JOIN good_images i ON i.good_id=g.id WHERE g.name='third'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "blobs", path)); err != nil || !bytes.Equal(got, png) {
		t.Fatalf("updated image not saved: %v", err)
	}
}
