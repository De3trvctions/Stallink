package service

import (
	"errors"
	"net/http"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

// optionalAccount reads the session without writing any response, for public
// endpoints whose payload only grows for some roles. Missing, invalid or
// expired sessions, and lookup failures, all read as anonymous.
func (a *API) optionalAccount(r *http.Request) (data.Account, bool) {
	cookie, err := r.Cookie("stallink_session")
	if err != nil || cookie.Value == "" {
		return data.Account{}, false
	}
	hash, err := auth.HashToken(cookie.Value)
	if err != nil {
		return data.Account{}, false
	}
	account, err := a.DB.AccountByToken(r.Context(), hash)
	if err != nil {
		if !errors.Is(err, data.ErrNotFound) {
			logging.WarnContext(r.Context(), "optional session lookup failed", zap.Error(err))
		}
		return data.Account{}, false
	}
	return account, true
}

func appearanceMetaJSON(meta data.PlatformAppearanceMeta) map[string]any {
	out := map[string]any{"theme": meta.Theme, "updatedAt": nil, "updatedBy": nil}
	if meta.UpdatedAt != nil {
		out["updatedAt"] = meta.UpdatedAt.Format(time.RFC3339)
	}
	if meta.UpdatedBy != nil {
		out["updatedBy"] = map[string]any{"id": meta.UpdatedBy.ID, "name": meta.UpdatedBy.Name}
	}
	return out
}

// PlatformAppearance is public (the root layout reads it during SSR). Only a
// platform admin session also gets updatedAt/updatedBy; everyone else gets
// exactly {theme} (appearance spec §8.1, B-11).
func (a *API) PlatformAppearance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if account, ok := a.optionalAccount(r); ok && account.Role == "platform_admin" {
		meta, err := a.DB.PlatformAppearanceMeta(r.Context())
		if err != nil {
			writeDataErr(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, appearanceMetaJSON(meta))
		return
	}
	theme, err := a.DB.PlatformTheme(r.Context())
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"theme": theme})
}

// SetPlatformAppearance publishes a theme; a PUT with the unchanged theme is
// still a publish and refreshes updatedAt/updatedBy.
func (a *API) SetPlatformAppearance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	account, ok := a.platform(w, r)
	if !ok {
		return
	}
	var body struct {
		Theme string `json:"theme"`
	}
	if err := readJSONStrict(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.Theme != "classic" && body.Theme != "exhibition" && body.Theme != "shop" {
		writeErr(w, http.StatusBadRequest, "invalid_theme")
		return
	}
	if err := a.DB.SetPlatformTheme(r.Context(), body.Theme, account.ID); err != nil {
		writeDataErr(w, r, err)
		return
	}
	meta, err := a.DB.PlatformAppearanceMeta(r.Context())
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, appearanceMetaJSON(meta))
}
