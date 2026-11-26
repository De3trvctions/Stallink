//go:build integration

package service_test

import (
	"net/http"
	"testing"
	"time"
)

type appearanceActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type appearanceMeta struct {
	Theme     string           `json:"theme"`
	UpdatedAt *string          `json:"updatedAt"`
	UpdatedBy *appearanceActor `json:"updatedBy"`
}

// Only platform admins see updatedAt/updatedBy; anonymous and shop sessions
// get exactly {theme} (appearance spec §8.1, B-11).
func TestPlatformAppearanceMetaByRole(t *testing.T) {
	f := newWorkspaceFixture(t, "appearance-meta-test")
	const endpoint = "/api/platform/appearance"
	getRaw := func(name string, cookie *http.Cookie) map[string]any {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, cookie)
		if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s GET: %d %s (Cache-Control %q)", name, res.Code, res.Body.String(), res.Header().Get("Cache-Control"))
		}
		return decodeBody[map[string]any](t, res.Body.Bytes())
	}
	getMeta := func() appearanceMeta {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, f.platform)
		if res.Code != http.StatusOK {
			t.Fatalf("platform GET: %d %s", res.Code, res.Body.String())
		}
		return decodeBody[appearanceMeta](t, res.Body.Bytes())
	}

	// Never published: the platform admin gets both keys as null.
	raw := getRaw("platform", f.platform)
	if v, ok := raw["updatedAt"]; !ok || v != nil {
		t.Fatalf("never published: updatedAt must be present and null: %v", raw)
	}
	if v, ok := raw["updatedBy"]; !ok || v != nil {
		t.Fatalf("never published: updatedBy must be present and null: %v", raw)
	}

	var platformID string
	if err := f.pool.QueryRow(f.ctx, `SELECT id::text FROM users WHERE login_name = $1`, f.platformLog).Scan(&platformID); err != nil {
		t.Fatal(err)
	}
	put := requestJSON(t, f.handler, http.MethodPut, endpoint, map[string]string{"theme": "shop"}, f.platform)
	if put.Code != http.StatusOK {
		t.Fatalf("put appearance: %d %s", put.Code, put.Body.String())
	}
	putMeta := decodeBody[appearanceMeta](t, put.Body.Bytes())
	if putMeta.Theme != "shop" || putMeta.UpdatedAt == nil || putMeta.UpdatedBy == nil || putMeta.UpdatedBy.ID != platformID || putMeta.UpdatedBy.Name != f.platformLog {
		t.Fatalf("PUT response must carry the new meta (name falls back to login): %+v", putMeta)
	}

	meta := getMeta()
	if meta.Theme != "shop" || meta.UpdatedAt == nil || meta.UpdatedBy == nil || meta.UpdatedBy.ID != platformID || meta.UpdatedBy.Name != f.platformLog {
		t.Fatalf("platform admin meta: %+v", meta)
	}
	if _, err := time.Parse(time.RFC3339, *meta.UpdatedAt); err != nil {
		t.Fatalf("updatedAt must be RFC3339 with offset: %q", *meta.UpdatedAt)
	}

	sales := f.salesCookie(t)
	bogus := &http.Cookie{Name: "stallink_session", Value: "not-a-real-session"}
	for name, cookie := range map[string]*http.Cookie{"anonymous": nil, "shop_admin": f.owner, "sales": sales, "bogus cookie": bogus} {
		body := getRaw(name, cookie)
		if len(body) != 1 || body["theme"] != "shop" {
			t.Fatalf("%s must only get {theme}, got %v", name, body)
		}
	}

	// display_name wins over login_name when set.
	if _, err := f.pool.Exec(f.ctx, `UPDATE users SET display_name = '运营 Kel' WHERE id = $1::bigint`, platformID); err != nil {
		t.Fatal(err)
	}
	if meta = getMeta(); meta.UpdatedBy == nil || meta.UpdatedBy.Name != "运营 Kel" {
		t.Fatalf("updatedBy.name must prefer display_name: %+v", meta.UpdatedBy)
	}

	// A same-theme PUT is still a publish: updatedAt moves forward.
	before, err := time.Parse(time.RFC3339, *meta.UpdatedAt)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if res := requestJSON(t, f.handler, http.MethodPut, endpoint, map[string]string{"theme": "shop"}, f.platform); res.Code != http.StatusOK {
		t.Fatalf("same-theme put: %d %s", res.Code, res.Body.String())
	}
	meta = getMeta()
	after, err := time.Parse(time.RFC3339, *meta.UpdatedAt)
	if err != nil || !after.After(before) {
		t.Fatalf("same-theme PUT must refresh updatedAt: before %v after %v (%v)", before, after, err)
	}
}
