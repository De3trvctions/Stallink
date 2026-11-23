package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/imagestore"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

type ImageStore interface {
	Put(ctx context.Context, shopID int64, kind, contentType string, content []byte) (string, error)
	Get(ctx context.Context, reference string) ([]byte, error)
}

var errStorageUnavailable = errors.New("image storage is not configured")

type imageUpload struct {
	api    *API
	shopID data.ID
}

func (u *imageUpload) put(ctx context.Context, kind string, src io.Reader) (string, error) {
	content, mime, err := imagestore.Compress(src, kind == "pay-code")
	if err != nil {
		return "", err
	}
	if u.api.Images == nil {
		return "", errStorageUnavailable
	}
	return u.api.Images.Put(ctx, int64(u.shopID), kind, mime, content)
}

func (u *imageUpload) putCatalog(ctx context.Context, kind string, files []*multipart.FileHeader) ([]string, error) {
	if len(files) > maxGoodImages {
		return nil, data.ErrImageLimit
	}
	paths := make([]string, 0, len(files))
	for _, header := range files {
		if header.Size <= 0 || header.Size > maxGoodImageBytes {
			return nil, data.ErrImageLimit
		}
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		path, saveErr := u.put(ctx, kind, file)
		if err := file.Close(); err != nil {
			logging.ErrorContext(ctx, "close image upload", zap.Error(err))
		}
		if saveErr != nil {
			return nil, saveErr
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writeImageError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errStorageUnavailable):
		writeErr(w, http.StatusServiceUnavailable, "image_storage_failed")
	case errors.Is(err, data.ErrImageLimit), errors.Is(err, imagestore.ErrImageTooLarge):
		writeErr(w, http.StatusBadRequest, "image_limit")
	case errors.Is(err, imagestore.ErrInvalidImage):
		writeErr(w, http.StatusBadRequest, "invalid_image")
	default:
		logging.ErrorContext(r.Context(), "store image upload", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "image_storage_failed")
	}
}

func (a *API) serveBlob(w http.ResponseWriter, r *http.Request, reference string) {
	if reference == "" {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if a.Images == nil {
		writeErr(w, http.StatusServiceUnavailable, "image_storage_failed")
		return
	}
	content, err := a.Images.Get(r.Context(), reference)
	switch {
	case errors.Is(err, imagestore.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found")
	case err != nil:
		logging.ErrorContext(r.Context(), "read image failed", zap.Error(err))
		writeErr(w, http.StatusBadGateway, "image_storage_failed")
	default:
		w.Header().Set("X-Content-Type-Options", "nosniff")
		contentType := http.DetectContentType(content)
		switch contentType {
		case "image/jpeg", "image/png", "image/webp":
			w.Header().Set("Content-Type", contentType)
		default:
			writeErr(w, http.StatusBadGateway, "image_storage_failed")
			return
		}
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(content))
	}
}
