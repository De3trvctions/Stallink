package nacos

import (
	"os"
	"strings"
	"testing"

	kconfig "github.com/go-kratos/kratos/v2/config"
)

func TestDecodeFullPublication(t *testing.T) {
	t.Setenv("STALLINK_DATABASE_SOURCE", "postgres://localhost/stallink")
	t.Setenv("STALLINK_PLATFORM_ADMIN_PASSWORD", "dev-password")
	t.Setenv("STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN", "test-token")
	doc, err := os.ReadFile("../../configs/nacos/stallink.yaml")
	if err != nil {
		t.Fatal(err)
	}
	parse := func(raw string) string {
		t.Helper()
		bc, err := decode([]*kconfig.KeyValue{{Value: []byte(raw)}})
		if err != nil {
			t.Fatal(err)
		}
		return bc.Storage.DriveOAuthWeb.RefreshToken
	}
	if parse(string(doc)) != "test-token" {
		t.Fatal("refresh token not resolved")
	}
	if parse(strings.ReplaceAll(string(doc), `refresh_token: "$STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN"`, "")) != "" {
		t.Fatal("deleted field persisted")
	}
}
