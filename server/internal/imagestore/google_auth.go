package imagestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

const googleTokenURL = "https://oauth2.googleapis.com/token"

type WebOAuthCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

// GoogleClient prefers an explicitly configured credentials file over Web OAuth.
func GoogleClient(ctx context.Context, path string, web WebOAuthCredentials) (*http.Client, error) {
	const timeout = 20 * time.Second
	if client, _ := ctx.Value(oauth2.HTTPClient).(*http.Client); client == nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: timeout})
	}

	var tokens oauth2.TokenSource
	var err error
	if path == "" {
		if strings.TrimSpace(web.ClientID) == "" || strings.TrimSpace(web.ClientSecret) == "" ||
			strings.TrimSpace(web.RefreshToken) == "" {
			return nil, errors.New("set storage.drive_oauth_web client_id and client_secret and STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN, or set STALLINK_DRIVE_CREDENTIALS_FILE")
		}
		tokens = web.tokenSource(ctx)
	} else {
		tokens, err = googleFileTokenSource(ctx, path)
		if err != nil {
			return nil, err
		}
	}
	client := oauth2.NewClient(ctx, tokens)
	client.Timeout = timeout
	return client, nil
}

func (c WebOAuthCredentials) tokenSource(ctx context.Context) oauth2.TokenSource {
	config := &oauth2.Config{
		ClientID: c.ClientID, ClientSecret: c.ClientSecret,
		Endpoint: oauth2.Endpoint{TokenURL: googleTokenURL, AuthStyle: oauth2.AuthStyleInParams},
	}
	return config.TokenSource(ctx, &oauth2.Token{RefreshToken: c.RefreshToken})
}

func googleFileTokenSource(ctx context.Context, path string) (oauth2.TokenSource, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read google drive credentials: %w", err)
	}
	var credentials struct {
		Type string `json:"type"`
		WebOAuthCredentials
		ClientEmail  string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
	}
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return nil, errors.New("invalid google drive credentials JSON")
	}

	switch credentials.Type {
	case "authorized_user":
		if credentials.ClientID == "" || credentials.ClientSecret == "" || credentials.RefreshToken == "" {
			return nil, errors.New("google oauth credentials require client_id, client_secret and refresh_token")
		}
		return credentials.WebOAuthCredentials.tokenSource(ctx), nil
	case "service_account":
		if credentials.ClientEmail == "" || credentials.PrivateKey == "" {
			return nil, errors.New("google service account requires client_email and private_key")
		}
		config := &jwt.Config{
			Email: credentials.ClientEmail, PrivateKey: []byte(credentials.PrivateKey),
			PrivateKeyID: credentials.PrivateKeyID, TokenURL: googleTokenURL,
			Scopes: []string{"https://www.googleapis.com/auth/drive"},
		}
		return config.TokenSource(ctx), nil
	default:
		return nil, errors.New("google drive credential type must be authorized_user or service_account")
	}
}
