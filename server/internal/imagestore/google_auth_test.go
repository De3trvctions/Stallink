package imagestore

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestGoogleClientRefreshesDriveToken(t *testing.T) {
	for _, tc := range []struct {
		name    string
		useFile bool
	}{
		{name: "nacos web oauth"},
		{name: "explicit credentials file", useFile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.String() {
				case googleTokenURL:
					if err := r.ParseForm(); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if r.Method != http.MethodPost || r.Form.Get("grant_type") != "refresh_token" ||
						r.Form.Get("client_id") != "drive-client" || r.Form.Get("client_secret") != "drive-secret" ||
						r.Form.Get("refresh_token") != "drive-refresh-token" {
						t.Error("the refresh request did not use the configured OAuth client and token")
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"access_token":"drive-access-token","token_type":"Bearer","expires_in":3600}`)
				case "https://www.googleapis.com/drive/v3/files":
					if r.Header.Get("Authorization") != "Bearer drive-access-token" {
						t.Error("the Drive request did not use the refreshed token")
					}
					w.WriteHeader(http.StatusOK)
				default:
					t.Errorf("unexpected request to %s", r.URL)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			path := ""
			web := WebOAuthCredentials{ClientID: "drive-client", ClientSecret: "drive-secret", RefreshToken: "drive-refresh-token"}
			if tc.useFile {
				path = filepath.Join(t.TempDir(), "google-credentials.json")
				if err := os.WriteFile(path, []byte(`{"type":"authorized_user","client_id":"drive-client","client_secret":"drive-secret","refresh_token":"drive-refresh-token"}`), 0o600); err != nil {
					t.Fatal(err)
				}
				web = WebOAuthCredentials{ClientID: "unused-client", ClientSecret: "unused-secret", RefreshToken: "unused-token"}
			}
			client, err := GoogleClient(ctx, path, web)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Get("https://www.googleapis.com/drive/v3/files")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Drive status = %d", resp.StatusCode)
			}
		})
	}
}

func TestGoogleClientRejectsMissingCredentials(t *testing.T) {
	web := WebOAuthCredentials{ClientID: "drive-client", ClientSecret: "test-secret"}
	_, err := GoogleClient(context.Background(), "", web)
	if err == nil || !strings.Contains(err.Error(), "STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN") || strings.Contains(err.Error(), web.ClientSecret) {
		t.Fatal("expected a missing refresh token error without credential values")
	}

	web.RefreshToken = "test-refresh"
	_, err = GoogleClient(context.Background(), filepath.Join(t.TempDir(), "missing.json"), web)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("an explicitly configured credentials file must be readable: %v", err)
	}
}
