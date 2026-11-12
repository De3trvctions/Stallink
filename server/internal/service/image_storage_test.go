package service

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestBlobStorageAndErrorClassification(t *testing.T) {
	dir := t.TempDir()
	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	name, err := saveBlob(context.Background(), dir, bytes.NewReader(image))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "blobs", name))
	if err != nil || !bytes.Equal(got, image) {
		t.Fatalf("stored image mismatch: %v", err)
	}
	_, err = saveBlob(context.Background(), dir, bytes.NewReader([]byte("not an image")))
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Fatalf("invalid image: %v", err)
	}
	broken := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(broken, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = saveBlob(context.Background(), broken, bytes.NewReader(image))
	if err == nil {
		t.Fatal("expected storage error")
	}
	response := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", nil)
	writeImageError(response, req, err)
	if response.Code != 500 {
		t.Fatalf("storage error status = %d", response.Code)
	}
}
