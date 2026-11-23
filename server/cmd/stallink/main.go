package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/De3trvctions/Stallink/internal/conf"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/imagestore"
	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/De3trvctions/Stallink/internal/nacos"
	"github.com/De3trvctions/Stallink/internal/reviewswitch"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	logging.InitDefault()
	defer logging.Sync()

	command, args := splitCommand(os.Args[1:])
	flags := flag.NewFlagSet("stallink "+command, flag.ContinueOnError)
	configPath := flags.String("conf", "configs/local/stallink.yaml", "local file that only locates Nacos")
	confirmation := flags.String("confirm", "", "type stallink to confirm local database reset")
	backupPath := flags.String("backup", "", "non-empty database backup required by reset-local")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	var err error
	switch command {
	case "serve":
		err = run(*configPath)
	case "reset-local":
		err = resetLocal(*configPath, *confirmation, *backupPath)
	case "seed-local":
		err = seedLocal(*configPath)
	default:
		err = fmt.Errorf("unknown command %q (use serve, reset-local, or seed-local)", command)
	}
	if err != nil {
		logging.Error("stallink stopped", zap.Error(err))
		return 1
	}
	return 0
}

func splitCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "serve", args
	}
	switch args[0] {
	case "serve", "reset-local", "seed-local":
		return args[0], args[1:]
	default:
		if strings.HasPrefix(args[0], "-") {
			return "serve", args
		}
		return args[0], args[1:]
	}
}

func loadRemote(configPath string) (*conf.Startup, *nacos.Remote, error) {
	startup, err := conf.LoadStartup(configPath)
	if err != nil {
		return nil, nil, err
	}
	client, err := nacos.NewClient(startup.Nacos)
	if err != nil {
		return nil, nil, err
	}
	remote, err := nacos.Load(client, startup.Nacos.Remote)
	if err != nil {
		return nil, nil, err
	}
	return &startup, remote, nil
}

func localBootstrap(configPath string) (*nacos.Remote, error) {
	absolute, err := filepath.Abs(configPath)
	if err != nil || !strings.HasSuffix(filepath.ToSlash(absolute), "/configs/local/stallink.yaml") {
		return nil, fmt.Errorf("local database commands require configs/local/stallink.yaml")
	}
	_, remote, err := loadRemote(configPath)
	if err != nil {
		return nil, err
	}
	parsed, err := pgxpool.ParseConfig(remote.Bootstrap.Data.Database.Source)
	if err != nil {
		_ = remote.Close()
		// Do not echo the DSN here; it can contain database credentials.
		return nil, fmt.Errorf("cannot validate local database target")
	}
	host := strings.Trim(parsed.ConnConfig.Host, "[]")
	if parsed.ConnConfig.Database != "stallink" || (host != "localhost" && host != "127.0.0.1" && host != "::1") {
		_ = remote.Close()
		return nil, fmt.Errorf("local database command refused target; expected database stallink on loopback")
	}
	return remote, nil
}

func resetLocal(configPath, confirmation, backupPath string) error {
	if confirmation != "stallink" {
		return fmt.Errorf("reset-local requires -confirm stallink")
	}
	info, err := os.Stat(backupPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("reset-local requires a non-empty database backup")
	}
	remote, err := localBootstrap(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := remote.Close(); err != nil {
			logging.Error("close nacos config", zap.Error(err))
		}
	}()
	ctx := context.Background()
	pool, err := data.NewPool(ctx, remote.Bootstrap.Data.Database.Source)
	if err != nil {
		return err
	}
	defer pool.Close()
	counts, err := data.ResetLocal(ctx, pool, confirmation)
	if err != nil {
		return err
	}
	if err := data.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("database was cleared but identity migration failed; fix migration and rerun reset-local: %w", err)
	}
	logging.InfoContext(ctx, "local stallink database reset and migrated", zap.Any("deleted_rows", counts))
	return nil
}

func localSeedPassword() (string, error) {
	password := strings.TrimSpace(os.Getenv("STALLINK_LOCAL_SEED_PASSWORD"))
	if len(password) < 8 {
		return "", fmt.Errorf("STALLINK_LOCAL_SEED_PASSWORD must contain at least 8 characters")
	}
	return password, nil
}

func seedLocal(configPath string) error {
	remote, err := localBootstrap(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := remote.Close(); err != nil {
			logging.Error("close nacos config", zap.Error(err))
		}
	}()
	ctx := context.Background()
	pool, err := data.NewPool(ctx, remote.Bootstrap.Data.Database.Source)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := data.Migrate(ctx, pool); err != nil {
		return err
	}
	password, err := localSeedPassword()
	if err != nil {
		return err
	}
	if err := data.SeedLocalDemo(ctx, data.NewShopStore(pool), password); err != nil {
		return err
	}
	count := func(table string) int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			return 0
		}
		return n
	}
	logging.InfoContext(ctx, "local stallink demo data seeded",
		zap.Int("stores", count("shops")), zap.Int("accounts", count("users")),
		zap.Int("products", count("goods")), zap.Int("bundles", count("bundles")))
	return nil
}

func run(configPath string) error {
	_, remote, err := loadRemote(configPath)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer remote.Close()
	updates := make(chan conf.Bootstrap)
	go func() {
		for {
			bc, err := remote.Next()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				logging.Error("Nacos publication rejected", zap.Error(err))
				continue
			}
			select {
			case updates <- bc:
			case <-ctx.Done():
				return
			}
		}
	}()
	defer cancel() // stop the update consumer before closing its watcher
	var mu sync.RWMutex
	var active *runtimeConfig
	var listener *http.Server
	var addr string
	failures := make(chan error, 1)
	defer func() {
		if listener != nil {
			_ = listener.Shutdown(context.Background())
		}
		mu.Lock()
		defer mu.Unlock()
		if active != nil {
			active.close()
		}
	}()
	apply := func(bc conf.Bootstrap) error {
		next, err := prepareRuntime(ctx, bc, active)
		if err != nil {
			return err
		}
		if addr != bc.Server.HTTP.Addr {
			ln, err := net.Listen("tcp", bc.Server.HTTP.Addr)
			if err != nil {
				next.close()
				return err
			}
			previous := listener
			listener = next.transport
			listener.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.RLock()
				defer mu.RUnlock()
				active.handler.ServeHTTP(w, r)
			})
			addr = bc.Server.HTTP.Addr
			defer func() {
				go func(srv *http.Server) {
					if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
						failures <- err
					}
				}(listener)
				if previous != nil {
					_ = previous.Shutdown(ctx)
				}
			}()
		}
		mu.Lock()
		if active != nil {
			active.close()
		}
		active = next
		next.start()
		mu.Unlock()
		logging.InfoContext(ctx, "applied Nacos configuration", zap.String("addr", addr), zap.Bool("drive_ready", next.drive != nil))
		return nil
	}
	current := remote.Bootstrap
	if err := apply(current); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-failures:
			return err
		case bc := <-updates:
			if reflect.DeepEqual(current, bc) {
				continue
			}
			if err := apply(bc); err != nil {
				logging.Error("Nacos publication rejected; previous configuration still active")
				continue
			}
			current = bc
		}
	}
}

type runtimeConfig struct {
	handler   http.Handler
	transport *http.Server
	drive     *imagestore.Drive
	database  conf.Database
	storage   conf.Storage
	start     func()
	close     func()
}

func prepareRuntime(ctx context.Context, bc conf.Bootstrap, previous *runtimeConfig) (*runtimeConfig, error) {
	var err error
	quoteTTL := 60 * time.Second
	if configured := bc.Orders.QuoteTTL; configured != "" {
		quoteTTL, err = time.ParseDuration(configured)
		if err != nil || quoteTTL <= 0 {
			return nil, fmt.Errorf("orders.quote_ttl must be a positive duration")
		}
	}
	maxPricingStates := bc.Pricing.MaxStates
	if maxPricingStates <= 0 {
		maxPricingStates = 100000
	}
	reservationTTL := 15 * time.Minute
	if configured := bc.Orders.ReservationTTL; configured != "" {
		reservationTTL, err = time.ParseDuration(configured)
		if err != nil || reservationTTL <= 0 {
			return nil, fmt.Errorf("orders.reservation_ttl must be a positive duration")
		}
	}
	sweepInterval := 30 * time.Second
	if configured := bc.Orders.SweepInterval; configured != "" {
		sweepInterval, err = time.ParseDuration(configured)
		if err != nil || sweepInterval <= 0 {
			return nil, fmt.Errorf("orders.sweep_interval must be a positive duration")
		}
	}
	logger, err := logging.New(bc.Logging, bc.Server.ID, bc.Server.Version)
	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}
	var drive *imagestore.Drive
	if previous != nil && previous.storage == bc.Storage {
		drive = previous.drive
	} else {
		oauth := bc.Storage.DriveOAuthWeb
		token := oauth.RefreshToken
		if token == "" {
			token = os.Getenv("STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN")
		}
		path := os.Getenv("STALLINK_DRIVE_CREDENTIALS_FILE")
		if path != "" || token != "" {
			client, err := imagestore.GoogleClient(ctx, path, imagestore.WebOAuthCredentials{ClientID: oauth.ClientID, ClientSecret: oauth.ClientSecret, RefreshToken: token})
			if err != nil {
				return nil, err
			}
			drive, err = imagestore.NewDrive(ctx, client, bc.Storage.DriveRootFolderID)
			if err != nil {
				return nil, err
			}
		} else if previous != nil && previous.drive != nil {
			return nil, fmt.Errorf("Drive credentials missing")
		}
	}
	pool, err := data.NewPool(ctx, bc.Data.Database.Source)
	if err != nil {
		return nil, err
	}

	if previous == nil || previous.database != bc.Data.Database {
		if err := data.Migrate(ctx, pool); err != nil {
			pool.Close()
			return nil, err
		}
	}
	store := data.NewShopStore(pool)
	sweepCtx, stopSweep := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	dailyAt := bc.Reports.DailyAt
	if dailyAt == "" {
		dailyAt = "00:10"
	}
	if _, err := time.Parse("15:04", dailyAt); err != nil {
		stopSweep()
		pool.Close()
		return nil, fmt.Errorf("reports.daily_at must be HH:MM")
	}
	reportTZ := bc.Reports.Timezone
	reportLocation := data.ReportLocation(reportTZ)
	reviewSwitch := reviewswitch.New(bc.Redis.Addr, bc.Redis.Password, bc.Redis.DB, bc.Redis.Key)
	var images service.ImageStore
	if drive != nil {
		images = drive
	}
	httpSrv := server.NewHTTP(bc.Server.HTTP.Addr, bc.Server.Name, &service.API{
		DB: store, SecureCookies: bc.Auth.SecureCookies, Images: images,
		RequireIdempotency: bc.Orders.RequireIdempotency, BundlesEnabled: bc.Catalog.BundlesEnabled,
		QuoteTTL: quoteTTL, MaxPricingStates: maxPricingStates,
		ReservationsEnabled: bc.Orders.ReservationsEnabled, ReservationTTL: reservationTTL,
		ReviewSwitch: reviewSwitch, AllowScheduledOrders: bc.Orders.ScheduledAllowed(),
		ReportTimezone: reportTZ,
	})
	return &runtimeConfig{handler: httpSrv.Handler, transport: httpSrv.Server, storage: bc.Storage, database: bc.Data.Database, drive: drive, start: func() {
		zap.ReplaceGlobals(logger)
		if bc.Orders.ReservationsEnabled {
			workers.Add(1)
			go func() {
				defer workers.Done()
				runReservationSweeper(sweepCtx, store, sweepInterval)
			}()
		}
		if drive != nil {
			workers.Add(1)
			go func() {
				defer workers.Done()
				runBlobSweeper(sweepCtx, store, drive)
			}()
		}
		service.ConfigureReportLocation(reportTZ)
		workers.Add(1)
		go func() {
			defer workers.Done()
			runDailyReportScheduler(sweepCtx, store, dailyAt, reportLocation, reportTZ)
		}()
	}, close: func() { stopSweep(); workers.Wait(); _ = reviewSwitch.Close(); pool.Close(); _ = logger.Sync() }}, nil
}

func runReservationSweeper(ctx context.Context, store *data.ShopStore, interval time.Duration) {
	sweep := func() {
		count, err := store.ExpireReservations(ctx, 100)
		if err != nil {
			logging.ErrorContext(ctx, "reservation sweep failed", zap.Error(err))
			return
		}
		if count > 0 {
			logging.InfoContext(ctx, "reservation sweep completed", zap.Int("expired_orders", count))
		}
	}
	sweep()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}

func runBlobSweeper(ctx context.Context, store *data.ShopStore, drive *imagestore.Drive) {
	sweep := func() {
		cutoff := time.Now().Add(-24 * time.Hour)
		references, err := store.ReferencedBlobPaths(ctx)
		if err != nil {
			logging.ErrorContext(ctx, "load image references for drive cleanup", zap.Error(err))
			return
		}
		referenced := make(map[string]bool, len(references))
		for _, reference := range references {
			referenced[reference] = true
		}
		removed, err := drive.Sweep(ctx, referenced, cutoff)
		if err != nil {
			logging.ErrorContext(ctx, "orphan blob sweep failed", zap.Error(err))
			return
		}
		if removed > 0 {
			logging.InfoContext(ctx, "orphan blob sweep completed", zap.Int("removed_blobs", removed))
		}
	}
	sweep()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}

func runDailyReportScheduler(ctx context.Context, store *data.ShopStore, dailyAt string, location *time.Location, reportTZ string) {
	clock, err := time.Parse("15:04", dailyAt)
	if err != nil {
		logging.ErrorContext(ctx, "invalid daily report time", zap.Error(err))
		return
	}
	logging.InfoContext(ctx, "daily report scheduler started", zap.String("at", dailyAt), zap.String("timezone", location.String()))
	// Startup catch-up and the daily tick both call GenerateScheduledReports.
	// A manual report generated after that business day ended is kept.
	generate := func(date string) {
		reportCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		service.GenerateScheduledReports(reportCtx, store, date, reportTZ)
	}
	generate(time.Now().In(location).AddDate(0, 0, -1).Format("2006-01-02"))
	for {
		now := time.Now().In(location)
		next := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, location)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			generate(time.Now().In(location).AddDate(0, 0, -1).Format("2006-01-02"))
		}
	}
}
