package data

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	migration024File         = "migrations/024_shop_profile_and_appearance_meta.sql"
	migration024DownFile     = "migrations/down/024_shop_profile_and_appearance_meta.down.sql"
	manual024aBackfillFile   = "manual/024a_backfill_shop_default_currency.sql"
	manual024aRollbackFile   = "manual/024a_backfill_shop_default_currency.rollback.sql"
	migration024ExpectedName = "shop_profile_and_appearance_meta"
)

func TestLoadMigrationsIncludes024ShopProfileFields(t *testing.T) {
	t.Parallel()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	var found *migration
	for i := range migrations {
		if migrations[i].version == 24 {
			found = &migrations[i]
		}
	}
	if found == nil {
		t.Fatal("expected migration 024 to be loaded")
	}
	if found.name != migration024ExpectedName {
		t.Fatalf("unexpected 024 name: %s", found.name)
	}
	sql := string(found.sql)
	for _, want := range []string{
		"contact", "booth_number", "default_currency",
		"shops_contact_check", "shops_booth_number_check", "shops_default_currency_check",
		"platform_appearance", "updated_by", "updated_at",
		"audit_logs_shop_created_id_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("024 sql missing %q", want)
		}
	}
}

// 024 runs automatically at API start, so it must only change structure;
// the default_currency backfill lives in the manual 024a script (K-Q16 = A).
func TestMigration024HasNoUpdateStatements(t *testing.T) {
	t.Parallel()
	raw, err := migrationFiles.ReadFile(migration024File)
	if err != nil {
		t.Fatalf("read 024: %v", err)
	}
	if regexp.MustCompile(`(?i)\bupdate\b`).MatchString(stripSQLComments(string(raw))) {
		t.Fatal("024 must not contain any UPDATE statement; backfill belongs in manual/024a")
	}
	if strings.Contains(strings.ToUpper(stripSQLComments(string(raw))), "CONCURRENTLY") {
		t.Fatal("024 runs inside a transaction and must not use CONCURRENTLY")
	}
}

func TestLoadMigrationsIgnoresDownDirectory(t *testing.T) {
	t.Parallel()
	if _, err := os.Stat(migration024DownFile); err != nil {
		t.Fatalf("024 rollback file must exist under migrations/down/: %v", err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	for _, m := range migrations {
		if strings.Contains(m.name, "down") {
			t.Fatalf("rollback file loaded as forward migration: %03d_%s", m.version, m.name)
		}
	}
}

func TestLoadMigrationsIgnoresManualDirectory(t *testing.T) {
	t.Parallel()
	for _, path := range []string{manual024aBackfillFile, manual024aRollbackFile} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("manual script %s must exist: %v", path, err)
		}
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	for _, m := range migrations {
		if strings.Contains(m.name, "024a") || strings.Contains(m.name, "manual") || strings.Contains(m.name, "backfill") {
			t.Fatalf("manual script loaded as forward migration: %03d_%s", m.version, m.name)
		}
	}
}

func TestMigration024DownDropsEverything024Adds(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(migration024DownFile)
	if err != nil {
		t.Fatalf("read 024 rollback: %v", err)
	}
	down := stripSQLComments(string(raw))
	for _, want := range []string{
		"DROP INDEX IF EXISTS audit_logs_shop_created_id_idx",
		"DROP COLUMN IF EXISTS updated_by", "DROP COLUMN IF EXISTS updated_at",
		"DROP CONSTRAINT IF EXISTS shops_default_currency_check",
		"DROP CONSTRAINT IF EXISTS shops_booth_number_check",
		"DROP CONSTRAINT IF EXISTS shops_contact_check",
		"DROP COLUMN IF EXISTS default_currency", "DROP COLUMN IF EXISTS booth_number", "DROP COLUMN IF EXISTS contact",
		"DELETE FROM schema_migrations WHERE version = 24",
	} {
		if !strings.Contains(down, want) {
			t.Fatalf("024 rollback missing %q", want)
		}
	}
}

// The schema bootstrap must create the same named shop constraints as 024 so a
// fresh database followed by 024 does not add duplicate checks.
func TestSchemaShopsDDLMatches024(t *testing.T) {
	t.Parallel()
	var shops string
	for _, stmt := range schema {
		if strings.Contains(stmt, "CREATE TABLE IF NOT EXISTS shops") {
			shops = stmt
		}
	}
	for _, want := range []string{
		"contact text NOT NULL DEFAULT ''", "booth_number text NOT NULL DEFAULT ''", "default_currency text",
		"shops_contact_check", "shops_booth_number_check", "shops_default_currency_check",
	} {
		if !strings.Contains(shops, want) {
			t.Fatalf("schema.go shops DDL missing %q", want)
		}
	}
}

func stripSQLComments(sql string) string {
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
