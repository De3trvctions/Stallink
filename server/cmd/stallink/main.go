package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/De3trvctions/Stallink/internal/conf"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/De3trvctions/Stallink/internal/nacos"
	"github.com/De3trvctions/Stallink/internal/reviewswitch"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/go-kratos/kratos/v2"
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
	blobsBackupPath := flags.String("blobs-backup", "", "non-empty blob backup required by reset-local")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	var err error
	switch command {
	case "serve":
		err = run(*configPath)
	case "reset-local":
		err = resetLocal(*configPath, *confirmation, *backupPath, *blobsBackupPath)
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

func resetLocal(configPath, confirmation, backupPath, blobsBackupPath string) error {
	if confirmation != "stallink" {
		return fmt.Errorf("reset-local requires -confirm stallink")
	}
	for _, path := range []string{backupPath, blobsBackupPath} {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("reset-local requires non-empty database and blob backups")
		}
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
	password := os.Getenv("STALLINK_LOCAL_SEED_PASSWORD")
	if password == "" {
		password = "12345678"
	}
	if err := data.SeedLocalDemo(ctx, data.NewShopStore(pool), password); err != nil {
		return err
	}
	logging.InfoContext(ctx, "local stallink demo data seeded", zap.Int("stores", 2), zap.Int("accounts", 6), zap.Int("products", 8), zap.Int("bundles", 2))
	return nil
}

func run(configPath string) error {
	startup, remote, err := loadRemote(configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := remote.Close(); err != nil {
			logging.Error("close nacos config", zap.Error(err))
		}
	}()
	bc := remote.Bootstrap
	quoteTTL := 60 * time.Second
	if configured := bc.Orders.QuoteTTL; configured != "" {
		quoteTTL, err = time.ParseDuration(configured)
		if err != nil || quoteTTL <= 0 {
			return fmt.Errorf("orders.quote_ttl must be a positive duration")
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
			return fmt.Errorf("orders.reservation_ttl must be a positive duration")
		}
	}
	sweepInterval := 30 * time.Second
	if configured := bc.Orders.SweepInterval; configured != "" {
		sweepInterval, err = time.ParseDuration(configured)
		if err != nil || sweepInterval <= 0 {
			return fmt.Errorf("orders.sweep_interval must be a positive duration")
		}
	}
	logger, err := logging.New(bc.Logging, bc.Server.ID, bc.Server.Version)
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	logging.Set(logger)
	if err := os.MkdirAll(bc.Storage.Dir, 0o755); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := data.NewPool(ctx, bc.Data.Database.Source)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := data.Migrate(ctx, pool); err != nil {
		return err
	}
	store := data.NewShopStore(pool)
	sweepCtx, stopSweep := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	defer func() {
		stopSweep()
		workers.Wait()
	}()
	if bc.Orders.ReservationsEnabled {
		workers.Add(1)
		go func() {
			defer workers.Done()
			runReservationSweeper(sweepCtx, store, sweepInterval)
		}()
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		runBlobSweeper(sweepCtx, store, bc.Storage.Dir)
	}()
	reviewSwitch := reviewswitch.New(bc.Redis.Addr, bc.Redis.Password, bc.Redis.DB, bc.Redis.Key)
	defer func() {
		if err := reviewSwitch.Close(); err != nil {
			logging.ErrorContext(ctx, "close review switch redis", zap.Error(err))
		}
	}()
	httpSrv := server.NewHTTP(bc.Server.HTTP.Addr, bc.Server.Name, &service.API{
		DB: store, SecureCookies: bc.Auth.SecureCookies, DataDir: bc.Storage.Dir,
		RequireIdempotency: bc.Orders.RequireIdempotency, BundlesEnabled: bc.Catalog.BundlesEnabled,
		QuoteTTL: quoteTTL, MaxPricingStates: maxPricingStates,
		ReservationsEnabled: bc.Orders.ReservationsEnabled, ReservationTTL: reservationTTL,
		ReviewSwitch: reviewSwitch,
	})
	app := kratos.New(
		kratos.Name(bc.Server.Name),
		kratos.Version(bc.Server.Version),
		kratos.Logger(logging.KratosLogger()),
		kratos.Server(httpSrv),
	)
	logging.InfoContext(ctx, "stallink listening",
		zap.String("name", bc.Server.Name),
		zap.String("addr", bc.Server.HTTP.Addr),
		zap.String("nacos_data_id", startup.Nacos.Remote.DataID),
	)
	return app.Run()
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

func runBlobSweeper(ctx context.Context, store *data.ShopStore, dataDir string) {
	sweep := func() {
		removed, err := sweepUnreferencedBlobs(ctx, store, dataDir)
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

func sweepUnreferencedBlobs(ctx context.Context, store *data.ShopStore, dataDir string) (int, error) {
	references, err := store.ReferencedBlobPaths(ctx)
	if err != nil {
		return 0, err
	}
	referenced := make(map[string]struct{}, len(references))
	for _, path := range references {
		referenced[filepath.Clean(path)] = struct{}{}
	}
	blobDir := filepath.Join(dataDir, "blobs")
	entries, err := os.ReadDir(blobDir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read blob directory: %w", err)
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	removed := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		fullPath := filepath.Join(blobDir, entry.Name())
		if _, ok := referenced[filepath.Clean(fullPath)]; ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return removed, fmt.Errorf("stat blob %q: %w", entry.Name(), err)
		}
		if !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, fmt.Errorf("remove unreferenced blob %q: %w", entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}
