package data

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return runMigrations(ctx, pool)
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS shops (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
		notice text NOT NULL DEFAULT '' CHECK (char_length(notice) <= 2000),
		cover_path text,
		slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])$'),
		is_enabled boolean NOT NULL DEFAULT true,
		created_at timestamptz NOT NULL DEFAULT now(),
		updated_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS users (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		login_name text NOT NULL UNIQUE CHECK (char_length(login_name) BETWEEN 3 AND 32 AND login_name !~ '\s'),
		password_hash text NOT NULL,
		role text NOT NULL CHECK (role IN ('customer', 'sales', 'shop_admin', 'platform_admin')),
		shop_id bigint REFERENCES shops (id),
		display_name text NOT NULL DEFAULT '' CHECK (char_length(display_name) <= 80),
		contact text NOT NULL DEFAULT '' CHECK (char_length(contact) <= 120),
		is_enabled boolean NOT NULL DEFAULT true,
		created_at timestamptz NOT NULL DEFAULT now(),
		CONSTRAINT users_shop_binding CHECK (
			(role IN ('sales', 'shop_admin') AND shop_id IS NOT NULL)
			OR (role IN ('customer', 'platform_admin') AND shop_id IS NULL)
		)
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
		token_hash bytea NOT NULL UNIQUE,
		expires_at timestamptz NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS events (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		shop_id bigint NOT NULL REFERENCES shops (id),
		name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
		starts_on date NOT NULL,
		ends_on date NOT NULL,
		city text NOT NULL DEFAULT '' CHECK (char_length(city) <= 80),
		venue text NOT NULL DEFAULT '' CHECK (char_length(venue) <= 120),
		booth_number text NOT NULL DEFAULT '' CHECK (char_length(booth_number) <= 40),
		currency text NOT NULL CHECK (currency IN ('MYR', 'SGD', 'CNY')),
		cover_path text,
		status text NOT NULL CHECK (status IN ('scheduled', 'live', 'ended')),
		created_at timestamptz NOT NULL DEFAULT now(),
		CONSTRAINT events_dates CHECK (ends_on >= starts_on)
	)`,
	`CREATE TABLE IF NOT EXISTS goods (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		shop_id bigint NOT NULL REFERENCES shops (id),
		name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
		description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 4000),
		total_stock int NOT NULL CHECK (total_stock >= 0),
		review_status text NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'approved', 'rejected')),
		review_note text NOT NULL DEFAULT '' CHECK (char_length(review_note) <= 500),
		reviewed_at timestamptz,
		created_at timestamptz NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS good_images (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		good_id bigint NOT NULL REFERENCES goods (id) ON DELETE CASCADE,
		path text NOT NULL,
		sort_order int NOT NULL,
		UNIQUE (good_id, sort_order)
	)`,
	`CREATE TABLE IF NOT EXISTS event_listings (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		event_id bigint NOT NULL REFERENCES events (id),
		good_id bigint NOT NULL REFERENCES goods (id),
		listed_qty int NOT NULL CHECK (listed_qty >= 0),
		sold_qty int NOT NULL DEFAULT 0 CHECK (sold_qty >= 0 AND sold_qty <= listed_qty),
		unit_price_minor bigint NOT NULL CHECK (unit_price_minor > 0),
		price_tiers jsonb NOT NULL DEFAULT '[]'::jsonb,
		is_listed boolean NOT NULL DEFAULT true,
		UNIQUE (event_id, good_id)
	)`,
	`CREATE TABLE IF NOT EXISTS pay_codes (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		shop_id bigint NOT NULL REFERENCES shops (id),
		name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
		image_path text NOT NULL,
		kind text NOT NULL DEFAULT 'qr' CHECK (kind IN ('qr', 'cash')),
		is_default boolean NOT NULL DEFAULT false,
		is_enabled boolean NOT NULL DEFAULT true
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS pay_codes_one_default ON pay_codes (shop_id) WHERE is_default`,
	`CREATE TABLE IF NOT EXISTS orders (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		shop_id bigint NOT NULL REFERENCES shops (id),
		event_id bigint NOT NULL REFERENCES events (id),
		number text NOT NULL CHECK (number ~ '^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$'),
		status text NOT NULL CHECK (status IN ('pending', 'paid', 'fulfilled', 'cancelled')),
		booth_name text NOT NULL DEFAULT '' CHECK (char_length(booth_name) <= 80),
		total_minor bigint NOT NULL CHECK (total_minor >= 0),
		currency text NOT NULL,
		pay_code_id bigint REFERENCES pay_codes (id),
		pay_code_name text NOT NULL DEFAULT '',
		opened_by bigint NOT NULL REFERENCES users (id),
		opened_by_role text NOT NULL,
		paid_by bigint REFERENCES users (id),
		created_at timestamptz NOT NULL DEFAULT now(),
		UNIQUE (shop_id, number)
	)`,
	`CREATE TABLE IF NOT EXISTS order_lines (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		order_id bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
		good_id bigint NOT NULL REFERENCES goods (id),
		listing_id bigint NOT NULL REFERENCES event_listings (id),
		good_name text NOT NULL,
		unit_price_minor bigint NOT NULL,
		line_total_minor bigint NOT NULL DEFAULT 0 CHECK (line_total_minor >= 0),
		qty int NOT NULL CHECK (qty > 0)
	)`,
	`CREATE TABLE IF NOT EXISTS audit_logs (
		id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		shop_id bigint NOT NULL REFERENCES shops (id),
		actor_id bigint NOT NULL REFERENCES users (id),
		action text NOT NULL,
		subject_type text NOT NULL,
		subject_id bigint NOT NULL,
		created_at timestamptz NOT NULL DEFAULT now()
	)`,
}
