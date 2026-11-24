//go:build integration

package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPlatformAppearance(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool := isolatedSchemaPool(t, ctx, dsn)
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := data.NewShopStore(pool)
	handler := server.NewHTTP(":0", "appearance-test", &service.API{DB: store})
	const endpoint = "/api/platform/appearance"
	assertTheme := func(handler http.Handler, method string, body any, cookie *http.Cookie, want string) {
		t.Helper()
		response := requestJSON(t, handler, method, endpoint, body, cookie)
		var actual struct{ Theme string }
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &actual) != nil || actual.Theme != want {
			t.Fatalf("%s appearance = %d %s, want 200 with theme %q", method, response.Code, response.Body, want)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s appearance Cache-Control = %q, want no-store", method, response.Header().Get("Cache-Control"))
		}
	}
	assertTheme(handler, http.MethodGet, nil, nil, "classic")

	const password = "test-password-123"
	if err := data.EnsurePlatformAdmin(ctx, store, "test-platform", password); err != nil {
		t.Fatal(err)
	}
	platform := login(t, handler, "test-platform", password)
	opened := requestJSON(t, handler, http.MethodPost, "/api/platform/shops", map[string]string{
		"name": "Test Shop", "slug": "test-shop", "adminLogin": "test-owner", "adminPassword": password,
	}, platform)
	if opened.Code != http.StatusCreated {
		t.Fatalf("open shop: status %d, body %s", opened.Code, opened.Body)
	}
	owner := login(t, handler, "test-owner", password)
	exhibition := map[string]string{"theme": "exhibition"}
	if response := requestJSON(t, handler, http.MethodPut, endpoint, exhibition, owner); response.Code != http.StatusForbidden {
		t.Fatalf("shop admin update: status %d, body %s", response.Code, response.Body)
	}
	assertTheme(handler, http.MethodGet, nil, nil, "classic")
	assertTheme(handler, http.MethodPut, exhibition, platform, "exhibition")

	for _, invalid := range []string{"", "Exhibition", "exhibition ", "missing"} {
		response := requestJSON(t, handler, http.MethodPut, endpoint, map[string]string{"theme": invalid}, platform)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid theme %q: status %d, body %s", invalid, response.Code, response.Body)
		}
	}

	// Rebuild the API with new database connections as a restarted process would.
	reloaded, err := pgxpool.NewWithConfig(ctx, pool.Config())
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	if err := data.Migrate(ctx, reloaded); err != nil {
		t.Fatal(err)
	}
	handler = server.NewHTTP(":0", "appearance-test", &service.API{DB: data.NewShopStore(reloaded)})
	assertTheme(handler, http.MethodGet, nil, nil, "exhibition")
	assertTheme(handler, http.MethodPut, map[string]string{"theme": "classic"}, platform, "classic")
	assertTheme(handler, http.MethodGet, nil, nil, "classic")
}
