package service

import (
	"encoding/json"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
	"net/http"
)

func NewHealth(service string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": service}); err != nil {
			logging.ErrorContext(r.Context(), "encode health", zap.Error(err))
		}
	})
}
