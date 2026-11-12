package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
	"net/http"
	"regexp"
)

var ErrShopNotFound = errors.New("shop: not found")

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$`)

type Shop struct {
	ID     string
	Name   string
	Notice string
	Slug   string
}

type ShopReader interface {
	EnabledShop(ctx context.Context, slug string) (Shop, error)
}

func NewPublic(shops ShopReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if !slugPattern.MatchString(slug) {
			writeErr(w, http.StatusNotFound, "no_booth")
			return
		}
		shop, err := shops.EnabledShop(r.Context(), slug)
		if errors.Is(err, ErrShopNotFound) {
			writeErr(w, http.StatusNotFound, "no_booth")
			return
		}
		if err != nil {
			logging.ErrorContext(r.Context(), "load shop", zap.Error(err))
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":       shop.ID,
			"name":     shop.Name,
			"notice":   shop.Notice,
			"coverURL": nil,
			"slug":     shop.Slug,
		})
	})
}

func writeErr(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.Error("encode json", zap.Error(err))
	}
}
