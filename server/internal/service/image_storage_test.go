package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/De3trvctions/Stallink/internal/imagestore"
)

func TestWriteImageError(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"invalid image", imagestore.ErrInvalidImage, http.StatusBadRequest},
		{"size limit", imagestore.ErrImageTooLarge, http.StatusBadRequest},
		{"drive failure", errors.New("drive unavailable"), http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeImageError(response, httptest.NewRequest("POST", "/", nil), tt.err)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
		})
	}
}
