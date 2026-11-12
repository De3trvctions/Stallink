package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeShops struct {
	bySlug map[string]Shop
}

func (f fakeShops) EnabledShop(_ context.Context, slug string) (Shop, error) {
	shop, ok := f.bySlug[slug]
	if !ok {
		return Shop{}, ErrShopNotFound
	}
	return shop, nil
}

func TestPublicShopHidesOtherBooths(t *testing.T) {
	t.Parallel()
	h := NewPublic(fakeShops{bySlug: map[string]Shop{
		"alpha": {ID: "1", Name: "甲店", Notice: "周末见", Slug: "alpha"},
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/public/shops/alpha", nil)
	req.SetPathValue("slug", "alpha")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("alpha %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "甲店" || body.Slug != "alpha" {
		t.Fatalf("%+v", body)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("乙店")) {
		t.Fatal("response leaked another shop")
	}

	for _, slug := range []string{"beta", "missing", "../alpha"} {
		req = httptest.NewRequest(http.MethodGet, "/api/public/shops/"+slug, nil)
		req.SetPathValue("slug", slug)
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound || !bytes.Contains(rec.Body.Bytes(), []byte(`"error":"no_booth"`)) {
			t.Fatalf("%s: %d %s", slug, rec.Code, rec.Body.String())
		}
	}
}

func TestErrShopNotFoundIsSentinel(t *testing.T) {
	t.Parallel()
	if !errors.Is(ErrShopNotFound, ErrShopNotFound) {
		t.Fatal("sentinel")
	}
}
