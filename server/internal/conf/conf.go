package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"go.uber.org/zap"
)

// Startup is the local file that only locates Nacos.
type Startup struct {
	Server Server `json:"server"`
	Nacos  Nacos  `json:"nacos"`
}

type Nacos struct {
	Host      string       `json:"host"`
	Port      uint64       `json:"port"`
	Namespace string       `json:"namespace"`
	GroupName string       `json:"group_name"`
	Username  string       `json:"username"`
	Password  string       `json:"password"`
	LogDir    string       `json:"log_dir"`
	CacheDir  string       `json:"cache_dir"`
	LogLevel  string       `json:"log_level"`
	Remote    RemoteConfig `json:"remote_config"`
}

type RemoteConfig struct {
	DataID string `json:"data_id"`
	Group  string `json:"group"`
}

// Bootstrap is the runtime config stored in Nacos.
type Bootstrap struct {
	Server  Server         `json:"server"`
	Data    Data           `json:"data"`
	Storage Storage        `json:"storage"`
	Auth    Auth           `json:"auth"`
	Orders  Orders         `json:"orders"`
	Catalog Catalog        `json:"catalog"`
	Pricing Pricing        `json:"pricing"`
	Redis   Redis          `json:"redis"`
	Reports Reports        `json:"reports"`
	Logging logging.Config `json:"logging"`
}

type Orders struct {
	RequireIdempotency  bool   `json:"require_idempotency"`
	ReservationsEnabled bool   `json:"reservations_enabled"`
	QuoteTTL            string `json:"quote_ttl"`
	ReservationTTL      string `json:"reservation_ttl"`
	SweepInterval       string `json:"sweep_interval"`
	// AllowScheduledOrders gates checkout against events still in "scheduled".
	// A nil pointer means the operator did not set it: the default keeps the
	// existing pre-sale behavior (scheduled events are orderable).
	AllowScheduledOrders *bool `json:"allow_scheduled_orders"`
}

// ScheduledAllowed reports whether orders may be placed against a scheduled
// (not yet live) event. Unset defaults to true.
func (o Orders) ScheduledAllowed() bool {
	return o.AllowScheduledOrders == nil || *o.AllowScheduledOrders
}

type Catalog struct {
	BundlesEnabled bool `json:"bundles_enabled"`
}

// Redis stores platform-level feature flags such as the auto approval switch.
type Reports struct {
	DailyAt string `json:"daily_at"`
	// Timezone is the IANA zone (e.g. "Asia/Kuala_Lumpur") used to slice
	// daily reports and order dates. Empty falls back to the KL default.
	Timezone string `json:"timezone"`
}

// DefaultReportTimezone is the report zone when no configured timezone is set.
const DefaultReportTimezone = "Asia/Kuala_Lumpur"

type Redis struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Key      string `json:"key"`
}

type Pricing struct {
	MaxStates int `json:"max_states"`
}

type Server struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Version string `json:"version"`
	HTTP    HTTP   `json:"http"`
}

type HTTP struct {
	Addr string `json:"addr"`
}

type Data struct {
	Database Database `json:"database"`
}

type Database struct {
	Driver string `json:"driver"`
	Source string `json:"source"`
}

type Storage struct {
	DriveRootFolderID string        `json:"drive_root_folder_id"`
	DriveOAuthWeb     DriveOAuthWeb `json:"drive_oauth_web"`
}

type DriveOAuthWeb struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

type Auth struct {
	PlatformAdminLogin    string `json:"platform_admin_login"`
	PlatformAdminPassword string `json:"platform_admin_password"`
	SecureCookies         bool   `json:"secure_cookies"`
}

func LoadFile(path string) (Bootstrap, error) {
	c := config.New(config.WithSource(file.NewSource(path)))
	defer closeConfig(c)
	if err := c.Load(); err != nil {
		return Bootstrap{}, fmt.Errorf("load config file: %w", err)
	}
	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		return Bootstrap{}, fmt.Errorf("scan config file: %w", err)
	}
	return bc, nil
}

func LoadStartup(path string) (Startup, error) {
	c := config.New(config.WithSource(file.NewSource(path)))
	defer closeConfig(c)
	if err := c.Load(); err != nil {
		return Startup{}, fmt.Errorf("load startup file: %w", err)
	}
	var startup Startup
	if err := c.Scan(&startup); err != nil {
		return Startup{}, fmt.Errorf("scan startup file: %w", err)
	}
	startup.Nacos.Username = os.ExpandEnv(startup.Nacos.Username)
	startup.Nacos.Password = os.ExpandEnv(startup.Nacos.Password)
	if err := ValidateStartup(startup); err != nil {
		return Startup{}, err
	}
	return startup, nil
}

func closeConfig(c config.Config) {
	if err := c.Close(); err != nil {
		logging.Error("close config", zap.Error(err))
	}
}

func ValidateStartup(c Startup) error {
	if c.Nacos.Host == "" || c.Nacos.Port == 0 || c.Nacos.Remote.DataID == "" || c.Nacos.Remote.Group == "" {
		return fmt.Errorf("startup config: missing nacos host, port, data id, or group")
	}
	if (c.Nacos.Username == "") != (c.Nacos.Password == "") {
		return fmt.Errorf("startup config: nacos username and password must be set together")
	}
	return nil
}

// ResolveEnvironment expands credentials and connection strings provided by the runtime environment.
func (c *Bootstrap) ResolveEnvironment() {
	if c == nil {
		return
	}
	c.Data.Database.Source = os.ExpandEnv(c.Data.Database.Source)
	c.Auth.PlatformAdminPassword = os.ExpandEnv(c.Auth.PlatformAdminPassword)
	c.Redis.Password = os.ExpandEnv(c.Redis.Password)
	c.Storage.DriveOAuthWeb.RefreshToken = os.ExpandEnv(c.Storage.DriveOAuthWeb.RefreshToken)
	c.Storage.DriveOAuthWeb.ClientID = os.ExpandEnv(c.Storage.DriveOAuthWeb.ClientID)
	c.Storage.DriveOAuthWeb.ClientSecret = os.ExpandEnv(c.Storage.DriveOAuthWeb.ClientSecret)
}

func ValidateBootstrap(c Bootstrap) error {
	if strings.TrimSpace(c.Server.Name) == "" || strings.TrimSpace(c.Server.HTTP.Addr) == "" {
		return fmt.Errorf("bootstrap: missing server.name or server.http.addr")
	}
	if c.Data.Database.Driver != "postgres" || strings.TrimSpace(c.Data.Database.Source) == "" {
		return fmt.Errorf("bootstrap: data.database.driver must be postgres and source is required")
	}
	if strings.TrimSpace(c.Storage.DriveRootFolderID) == "" {
		return fmt.Errorf("bootstrap: missing storage.drive_root_folder_id")
	}
	if strings.TrimSpace(c.Auth.PlatformAdminLogin) == "" || len(c.Auth.PlatformAdminPassword) < 8 {
		return fmt.Errorf("bootstrap: platform admin login is required and password must be at least 8 characters")
	}
	return nil
}
