//go:build integration

package data

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityMigrationConvertsEmptySchema(t *testing.T) {
	ctx := context.Background()
	pool, schemaName := newIdentityTestSchema(t, ctx)
	defer pool.Close()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate empty schema: %v", err)
	}

	var dataType, isIdentity, generation string
	if err := pool.QueryRow(ctx, "SELECT data_type, is_identity, COALESCE(identity_generation, '')\nFROM information_schema.columns\nWHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'id'").Scan(&dataType, &isIdentity, &generation); err != nil {
		t.Fatal(err)
	}
	if dataType != "bigint" || isIdentity != "YES" || generation != "ALWAYS" {
		t.Fatalf("shops.id = %s identity=%s generation=%s, want bigint GENERATED ALWAYS AS IDENTITY", dataType, isIdentity, generation)
	}

	var shopID ID
	if err := pool.QueryRow(ctx, "INSERT INTO shops(name,slug) VALUES ('Identity Shop','identity-shop') RETURNING id").Scan(&shopID); err != nil {
		t.Fatalf("insert generated shop ID: %v", err)
	}
	if shopID != 1 {
		t.Fatalf("first generated shop ID = %d, want 1", shopID)
	}

	var foreignKeys int
	if err := pool.QueryRow(ctx, "SELECT count(*)\nFROM information_schema.table_constraints\nWHERE constraint_schema = $1 AND constraint_type = 'FOREIGN KEY'", schemaName).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys < 20 {
		t.Fatalf("found %d foreign keys after migration, want restored relationship constraints", foreignKeys)
	}
}

func TestIdentityMigrationRejectsExistingRowsWithoutDeletingThem(t *testing.T) {
	ctx := context.Background()
	pool, _ := newIdentityTestSchema(t, ctx)
	defer pool.Close()

	for _, statement := range schema {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("create legacy schema: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, "CREATE TABLE schema_migrations (version bigint PRIMARY KEY, name text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) < 2 {
		t.Fatalf("loaded %d migrations, want version 001 and 002", len(migrations))
	}
	if err := applyMigration(ctx, pool, migrations[0]); err != nil {
		t.Fatalf("apply legacy catalog migration: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO shops(name,slug) VALUES ('Keep Me','keep-me')"); err != nil {
		t.Fatal(err)
	}

	err = applyMigration(ctx, pool, migrations[1])
	if err == nil || !strings.Contains(err.Error(), "identity_reset_required") {
		t.Fatalf("identity migration error = %v, want identity_reset_required", err)
	}
	var rows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM shops WHERE slug = 'keep-me'").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("legacy row count = %d after rejected migration, want 1", rows)
	}
}

func newIdentityTestSchema(t *testing.T, ctx context.Context) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })

	schemaName := fmt.Sprintf("stallink_identity_it_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
	})

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	return pool, schemaName
}
