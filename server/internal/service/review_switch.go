package service

import (
	"context"
	"net/http"

	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

func (a *API) AutoApproveSetting(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": a.autoApprove(r.Context())})
}

func (a *API) SetAutoApproveSetting(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platform(w, r); !ok {
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil || body.Enabled == nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if a.ReviewSwitch != nil {
		if err := a.ReviewSwitch.Set(r.Context(), *body.Enabled); err != nil {
			logging.ErrorContext(r.Context(), "save auto approve switch", zap.Error(err))
			writeErr(w, http.StatusInternalServerError, "internal")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": *body.Enabled})
}

func (a *API) autoApprove(ctx context.Context) bool {
	if a.ReviewSwitch == nil {
		return true
	}
	return a.ReviewSwitch.Enabled(ctx)
}
