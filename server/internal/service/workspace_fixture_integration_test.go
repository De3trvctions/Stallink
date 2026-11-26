//go:build integration

package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// workspaceFixture is one migrated isolated schema with a platform admin, an
// opened shop and its owner, shared by the workspace-polish HTTP tests.
type workspaceFixture struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	store       *data.ShopStore
	handler     http.Handler
	password    string
	platform    *http.Cookie
	shopSlug    string
	shopID      string
	ownerLogin  string
	owner       *http.Cookie
	prefix      string
	platformLog string
}

func newWorkspaceFixture(t *testing.T, name string) *workspaceFixture {
	t.Helper()
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	pool := isolatedSchemaPool(t, ctx, dsn)
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := data.NewShopStore(pool)
	f := &workspaceFixture{
		ctx: ctx, pool: pool, store: store, password: "test-password-123",
		handler: server.NewHTTP(":0", name, &service.API{DB: store}),
		prefix:  "p" + strconv.FormatInt(time.Now().UnixNano(), 36),
	}
	f.platformLog = f.prefix + "-platform"
	if err := data.EnsurePlatformAdmin(ctx, store, f.platformLog, f.password); err != nil {
		t.Fatal(err)
	}
	f.platform = login(t, f.handler, f.platformLog, f.password)
	f.shopSlug = f.prefix + "-shop"
	f.ownerLogin = f.prefix + "-owner"
	opened := requestJSON(t, f.handler, http.MethodPost, "/api/platform/shops", map[string]string{
		"name": "晚风", "slug": f.shopSlug, "notice": "", "adminLogin": f.ownerLogin, "adminPassword": f.password,
	}, f.platform)
	if opened.Code != http.StatusCreated {
		t.Fatalf("open shop: %d %s", opened.Code, opened.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT id::text FROM shops WHERE slug = $1`, f.shopSlug).Scan(&f.shopID); err != nil {
		t.Fatal(err)
	}
	f.owner = login(t, f.handler, f.ownerLogin, f.password)
	return f
}

func (f *workspaceFixture) salesCookie(t *testing.T) *http.Cookie {
	t.Helper()
	salesLogin := f.prefix + "-sales"
	created := requestJSON(t, f.handler, http.MethodPost, "/api/admin/sales", map[string]string{
		"loginName": salesLogin, "password": f.password,
	}, f.owner)
	if created.Code != http.StatusCreated {
		t.Fatalf("create sales: %d %s", created.Code, created.Body.String())
	}
	return login(t, f.handler, salesLogin, f.password)
}

func decodeBody[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return out
}
