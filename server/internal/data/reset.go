package data

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var localResetTables = []string{
	"shops", "users", "sessions", "events", "goods", "good_images",
	"event_listings", "pay_codes", "orders", "order_lines", "audit_logs",
	"bundles", "bundle_components", "bundle_images", "event_bundle_listings",
	"order_quotes", "order_commands", "order_payments", "inventory_reservations",
	"inventory_movements", "order_pricing_groups",
}

// ResetLocal clears only Stallink business rows. It keeps schema_migrations,
// so the caller can apply pending schema changes immediately after the reset.
func ResetLocal(ctx context.Context, pool *pgxpool.Pool, confirmation string) (map[string]int64, error) {
	if confirmation != "stallink" {
		return nil, fmt.Errorf("local reset requires confirmation name stallink")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin local reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var databaseName string
	if err := tx.QueryRow(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return nil, fmt.Errorf("read reset database name: %w", err)
	}
	if databaseName != "stallink" {
		return nil, fmt.Errorf("local reset refuses database %q; expected stallink", databaseName)
	}
	counts := make(map[string]int64, len(localResetTables))
	for _, table := range localResetTables {
		var count int64
		if err := tx.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", pgx.Identifier{table}.Sanitize())).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s before reset: %w", table, err)
		}
		counts[table] = count
	}
	query := "TRUNCATE TABLE "
	for i, table := range localResetTables {
		if i != 0 {
			query += ", "
		}
		query += pgxIdentifier(table)
	}
	query += " RESTART IDENTITY"
	if _, err := tx.Exec(ctx, query); err != nil {
		return nil, fmt.Errorf("truncate local Stallink tables: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit local reset: %w", err)
	}
	return counts, nil
}

func pgxIdentifier(name string) string {
	return pgx.Identifier{name}.Sanitize()
}
