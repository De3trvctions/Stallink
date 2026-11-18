package conf

import (
	"strings"
	"testing"
)

func TestValidateStartupRequiresNacosLocator(t *testing.T) {
	t.Parallel()
	if err := ValidateStartup(Startup{}); err == nil {
		t.Fatal("expected missing nacos settings to fail")
	}
	err := ValidateStartup(Startup{
		Nacos: Nacos{
			Host: "127.0.0.1",
			Port: 8848,
			Remote: RemoteConfig{
				DataID: "stallink.yaml",
				Group:  "DEFAULT_GROUP",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateBootstrapRequiresRuntimeSettings(t *testing.T) {
	t.Parallel()
	if err := ValidateBootstrap(Bootstrap{}); err == nil {
		t.Fatal("expected empty bootstrap to fail")
	}
	err := ValidateBootstrap(Bootstrap{
		Server: Server{Name: "stallink", HTTP: HTTP{Addr: ":8080"}},
		Data: Data{Database: Database{
			Driver: "postgres",
			Source: "postgres://postgres:123456@127.0.0.1:5432/stallink?sslmode=disable",
		}},
		Storage: Storage{Dir: "./data"},
		Auth: Auth{
			PlatformAdminLogin:    "platform",
			PlatformAdminPassword: "change-me-please",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLocalStartupExpandsNacosCredentials(t *testing.T) {
	t.Setenv("STALLINK_NACOS_USERNAME", "nacos")
	t.Setenv("STALLINK_NACOS_PASSWORD", "nacos")
	startup, err := LoadStartup("../../configs/local/stallink.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if startup.Nacos.Username != "nacos" || startup.Nacos.Password != "nacos" {
		t.Fatalf("nacos credentials were not expanded: %#v", startup.Nacos)
	}
}

func TestPublishedNacosConfigBindsRuntimeSettings(t *testing.T) {
	t.Setenv("STALLINK_DATABASE_SOURCE", "postgres://stallink:stallink@127.0.0.1:5432/stallink?sslmode=disable")
	t.Setenv("STALLINK_PLATFORM_ADMIN_PASSWORD", "local-admin-password")
	bc, err := LoadFile("../../configs/nacos/stallink.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bc.ResolveEnvironment()
	if err := ValidateBootstrap(bc); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bc.Data.Database.Source, "127.0.0.1:5432/stallink") {
		t.Fatalf("database source = %q", bc.Data.Database.Source)
	}
	if bc.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("redis addr = %q", bc.Redis.Addr)
	}
	if bc.Auth.SecureCookies {
		t.Fatal("local template must keep secure cookies off for plain HTTP")
	}
}

func TestScanBootstrapFromYAML(t *testing.T) {
	t.Parallel()
	bc, err := LoadFile("testdata/bootstrap.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if bc.Server.Name != "stallink" || bc.Server.HTTP.Addr != ":8080" {
		t.Fatalf("%+v", bc.Server)
	}
	if bc.Data.Database.Driver != "postgres" || bc.Storage.Dir != "./data" {
		t.Fatalf("%+v", bc)
	}
	if bc.Auth.PlatformAdminLogin != "platform" {
		t.Fatal(bc.Auth.PlatformAdminLogin)
	}
}
