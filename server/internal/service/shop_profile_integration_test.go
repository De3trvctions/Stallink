//go:build integration

package service_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type adminShopProfile struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Notice          string          `json:"notice"`
	Slug            string          `json:"slug"`
	IsEnabled       bool            `json:"isEnabled"`
	Contact         string          `json:"contact"`
	BoothNumber     string          `json:"boothNumber"`
	DefaultCurrency *string         `json:"defaultCurrency"`
	FeaturedEvent   *map[string]any `json:"featuredEvent"`
	OnSaleCount     *int            `json:"onSaleCount"`
	UpdatedAt       string          `json:"updatedAt"`
}

func TestAdminShopProfileReadUpdateAndSlugReadonly(t *testing.T) {
	f := newWorkspaceFixture(t, "shop-profile-test")
	read := func(cookie *http.Cookie) adminShopProfile {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/shop", nil, cookie)
		if res.Code != http.StatusOK {
			t.Fatalf("GET /api/admin/shop: %d %s", res.Code, res.Body.String())
		}
		raw := decodeBody[map[string]any](t, res.Body.Bytes())
		for _, key := range []string{"id", "name", "notice", "slug", "isEnabled", "contact", "boothNumber", "defaultCurrency", "featuredEvent", "onSaleCount", "updatedAt"} {
			if _, ok := raw[key]; !ok {
				t.Fatalf("GET /api/admin/shop missing %q: %s", key, res.Body.String())
			}
		}
		return decodeBody[adminShopProfile](t, res.Body.Bytes())
	}
	patch := func(body map[string]any, cookie *http.Cookie) (int, string) {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodPatch, "/api/admin/shop", body, cookie)
		return res.Code, res.Body.String()
	}

	p := read(f.owner)
	if p.ID != f.shopID || p.Slug != f.shopSlug || !p.IsEnabled || p.Contact != "" || p.BoothNumber != "" || p.DefaultCurrency != nil || p.FeaturedEvent != nil || p.OnSaleCount == nil || *p.OnSaleCount != 0 {
		t.Fatalf("initial profile: %+v", p)
	}

	// 1) Saving without slug works; contact is trimmed; the response is the profile.
	code, body := patch(map[string]any{
		"name": " 晚风工作室 ", "notice": "", "isEnabled": true, "contact": "  IG @wanfeng.studio  ", "boothNumber": "A-27", "defaultCurrency": "SGD",
	}, f.owner)
	if code != http.StatusOK {
		t.Fatalf("patch without slug: %d %s", code, body)
	}
	saved := decodeBody[adminShopProfile](t, []byte(body))
	p = read(f.owner)
	if saved.ID != p.ID || saved.Name != p.Name || saved.Contact != p.Contact || saved.BoothNumber != p.BoothNumber || saved.UpdatedAt != p.UpdatedAt {
		t.Fatalf("PATCH must return the same object as GET: %+v vs %+v", saved, p)
	}
	if p.Name != "晚风工作室" || p.Contact != "IG @wanfeng.studio" || p.BoothNumber != "A-27" || p.DefaultCurrency == nil || *p.DefaultCurrency != "SGD" {
		t.Fatalf("after patch: %+v", p)
	}
	if ts, err := time.Parse(time.RFC3339, p.UpdatedAt); err != nil || ts.IsZero() {
		t.Fatalf("updatedAt must be RFC3339 with offset: %q", p.UpdatedAt)
	}

	// 1b) Owner disables the shop: flag and session revocation are one transaction.
	if code, body := patch(map[string]any{
		"name": "晚风工作室", "isEnabled": false, "contact": "IG @wanfeng.studio", "boothNumber": "A-27", "defaultCurrency": "SGD",
	}, f.owner); code != http.StatusOK {
		t.Fatalf("disable shop: %d %s", code, body)
	}
	if res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/shop", nil, f.owner); res.Code != http.StatusUnauthorized {
		t.Fatalf("old cookie after disable: want 401, got %d", res.Code)
	}
	again := login(t, f.handler, f.ownerLogin, f.password) // login still issues a cookie
	if res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/shop", nil, again); res.Code != http.StatusUnauthorized {
		t.Fatalf("re-login on a disabled shop must not reach the workspace: got %d", res.Code)
	}
	if res := requestJSON(t, f.handler, http.MethodPatch, "/api/platform/shops/"+f.shopID, map[string]any{"isEnabled": true}, f.platform); res.Code != http.StatusOK {
		t.Fatalf("platform re-enable: %d %s", res.Code, res.Body.String())
	}
	owner := login(t, f.handler, f.ownerLogin, f.password)
	f.owner = owner // the fixture cookie was revoked by the disable above

	// 2) The current slug is accepted and ignored; null currency clears it.
	if code, body := patch(map[string]any{"name": "晚风工作室", "isEnabled": true, "slug": f.shopSlug, "defaultCurrency": nil}, owner); code != http.StatusOK {
		t.Fatalf("patch with same slug: %d %s", code, body)
	}
	if p = read(owner); p.DefaultCurrency != nil {
		t.Fatalf("null defaultCurrency must clear it: %+v", p)
	}

	// 3) A different slug is 422 slug_readonly and nothing is written.
	before := read(owner)
	code, body = patch(map[string]any{"name": "改了名", "isEnabled": true, "slug": f.shopSlug + "-x", "contact": "x"}, owner)
	if code != http.StatusUnprocessableEntity || !strings.Contains(body, "slug_readonly") {
		t.Fatalf("change slug: want 422 slug_readonly, got %d %s", code, body)
	}
	if p = read(owner); p.Slug != f.shopSlug || p.Name != before.Name || p.Contact != before.Contact || p.UpdatedAt != before.UpdatedAt {
		t.Fatalf("rejected slug change must write nothing: %+v", p)
	}

	// 4) Validation.
	for name, b := range map[string]map[string]any{
		"contact too long": {"name": "晚风", "isEnabled": true, "contact": strings.Repeat("客", 121)},
		"contact control":  {"name": "晚风", "isEnabled": true, "contact": "a\nb"},
		"bad currency":     {"name": "晚风", "isEnabled": true, "defaultCurrency": "USD"},
		"booth too long":   {"name": "晚风", "isEnabled": true, "boothNumber": strings.Repeat("A", 41)},
		"booth padded":     {"name": "晚风", "isEnabled": true, "boothNumber": " A-27 "},
		"booth control":    {"name": "晚风", "isEnabled": true, "boothNumber": "A\t27"},
		"blank name":       {"name": "   ", "isEnabled": true},
		"name too long":    {"name": strings.Repeat("店", 81), "isEnabled": true},
		"notice too long":  {"name": "晚风", "isEnabled": true, "notice": strings.Repeat("公", 2001)},
	} {
		if code, body := patch(b, owner); code != http.StatusBadRequest || !strings.Contains(body, "invalid_shop") {
			t.Fatalf("%s: want 400 invalid_shop, got %d %s", name, code, body)
		}
	}
	if code, body := patch(map[string]any{"name": "晚风", "isEnabled": true, "contact": strings.Repeat("客", 120), "boothNumber": strings.Repeat("Ａ", 40)}, owner); code != http.StatusOK {
		t.Fatalf("contact 120 / booth 40 chars: %d %s", code, body)
	}

	// 5) Only the shop owner can read or write.
	sales := f.salesCookie(t)
	for _, c := range []struct {
		name   string
		cookie *http.Cookie
		want   int
	}{{"anonymous", nil, http.StatusUnauthorized}, {"sales", sales, http.StatusForbidden}, {"platform", f.platform, http.StatusForbidden}} {
		if res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/shop", nil, c.cookie); res.Code != c.want {
			t.Fatalf("%s GET: want %d, got %d", c.name, c.want, res.Code)
		}
		if code, _ := patch(map[string]any{"name": "x", "isEnabled": true}, c.cookie); code != c.want {
			t.Fatalf("%s PATCH: want %d, got %d", c.name, c.want, code)
		}
	}
}
