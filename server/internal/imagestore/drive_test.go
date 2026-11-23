package imagestore

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDriveShopImageLifecycle(t *testing.T) {
	var uploadCount atomic.Int32
	var trashCount atomic.Int32
	image := []byte("compressed PNG bytes")
	ctx := context.Background()
	client := testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("supportsAllDrives") != "true" {
			t.Error("request omitted Shared Drive support")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files":
			if strings.Contains(q.Get("q"), "appProperties") {
				if !strings.Contains(q.Get("q"), "value='root-id'") || !strings.Contains(q.Get("q"), "createdTime <") {
					t.Error("cleanup query must use the configured root and cutoff")
				}
				io.WriteString(w, `{"files":[{"id":"image-id"},{"id":"orphan-id"}]}`)
				return
			}
			t.Error("unexpected folder lookup")
		case r.Method == http.MethodPost && r.URL.Path == "/upload/drive/v3/files":
			uploadCount.Add(1)
			mimeType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mimeType != "multipart/related" || q.Get("uploadType") != "multipart" {
				t.Errorf("invalid upload content type: %s, %v", mimeType, err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reader := multipart.NewReader(r.Body, params["boundary"])
			meta, err := reader.NextPart()
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var uploaded file
			err = json.NewDecoder(meta).Decode(&uploaded)
			if err != nil || !strings.HasPrefix(uploaded.Name, "shop-42-bundle-") || uploaded.MIMEType != "image/png" || len(uploaded.Parents) != 1 || uploaded.Parents[0] != "root-id" || uploaded.AppProperties["stallinkImageRoot"] != "root-id" {
				t.Errorf("invalid image metadata: %+v, %v", uploaded, err)
			}
			media, err := reader.NextPart()
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			content, err := io.ReadAll(media)
			if err != nil || !bytes.Equal(content, image) || media.Header.Get("Content-Type") != "image/png" {
				t.Errorf("invalid upload payload: %q, %v", content, err)
			}
			io.WriteString(w, `{"id":"image-id"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files/image-id" && q.Get("alt") == "media":
			w.Write(image)
		case r.Method == http.MethodPatch && r.URL.Path == "/drive/v3/files/orphan-id":
			trashCount.Add(1)
			var update struct{ Trashed bool }
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil || !update.Trashed {
				t.Errorf("invalid trash request: %+v, %v", update, err)
			}
			io.WriteString(w, `{"id":"orphan-id","trashed":true}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	drive := &Drive{client: client, base: "https://www.googleapis.com", rootID: "root-id", sharedDriveID: "shared-id"}
	ref, err := drive.Put(ctx, 42, "bundle", "image/png", image)
	if err != nil || ref != "drive:image-id" || uploadCount.Load() != 1 {
		t.Fatalf("upload result = %q, %v", ref, err)
	}
	content, err := drive.Get(ctx, ref)
	if err != nil || !bytes.Equal(content, image) {
		t.Fatalf("download result = %q, %v", content, err)
	}
	count, err := drive.Sweep(ctx, map[string]bool{ref: true}, time.Now().Add(-24*time.Hour))
	if err != nil || count != 1 || trashCount.Load() != 1 {
		t.Fatalf("cleanup = %d, %v", count, err)
	}
}

type testDriveTransport func(http.ResponseWriter, *http.Request)

func (f testDriveTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	f(recorder, r)
	return recorder.Result(), nil
}

func testDriveHTTPClient(handler testDriveTransport) *http.Client {
	return &http.Client{Transport: handler}
}
