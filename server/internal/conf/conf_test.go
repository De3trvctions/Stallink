package conf

import "testing"

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
