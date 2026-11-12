-- Catalog revisions make server quotes invalid as soon as a relevant catalog
-- write is committed. They start at 1 for shops created after this migration.
ALTER TABLE shops ADD COLUMN catalog_revision bigint NOT NULL DEFAULT 1 CHECK (catalog_revision > 0);

ALTER TABLE goods ADD COLUMN IF NOT EXISTS review_status text;
ALTER TABLE goods ADD COLUMN IF NOT EXISTS review_note text NOT NULL DEFAULT '';
ALTER TABLE goods ADD COLUMN IF NOT EXISTS reviewed_at timestamptz;
UPDATE goods SET review_status = 'approved' WHERE review_status IS NULL;
ALTER TABLE goods ALTER COLUMN review_status SET DEFAULT 'pending';
ALTER TABLE goods ALTER COLUMN review_status SET NOT NULL;

DO $migration$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'goods'::regclass AND conname = 'goods_review_status_check') THEN
		ALTER TABLE goods ADD CONSTRAINT goods_review_status_check CHECK (review_status IN ('pending', 'approved', 'rejected'));
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'goods'::regclass AND conname = 'goods_review_note_check') THEN
		ALTER TABLE goods ADD CONSTRAINT goods_review_note_check CHECK (char_length(review_note) <= 500);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'goods'::regclass AND conname = 'goods_id_shop_unique') THEN
		ALTER TABLE goods ADD CONSTRAINT goods_id_shop_unique UNIQUE (id, shop_id);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'events'::regclass AND conname = 'events_id_shop_unique') THEN
		ALTER TABLE events ADD CONSTRAINT events_id_shop_unique UNIQUE (id, shop_id);
	END IF;
END
$migration$;

CREATE TABLE bundles (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL REFERENCES shops(id),
	name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
	description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 4000),
	review_status text NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'approved', 'rejected')),
	review_note text NOT NULL DEFAULT '' CHECK (char_length(review_note) <= 500),
	reviewed_at timestamptz,
	revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
	created_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (id, shop_id)
);

CREATE TABLE bundle_components (
	bundle_id bigint NOT NULL,
	shop_id bigint NOT NULL,
	good_id bigint NOT NULL,
	quantity int NOT NULL CHECK (quantity BETWEEN 1 AND 100),
	sort_order int NOT NULL CHECK (sort_order >= 0),
	PRIMARY KEY (bundle_id, good_id),
	UNIQUE (bundle_id, sort_order),
	FOREIGN KEY (bundle_id, shop_id) REFERENCES bundles(id, shop_id) ON DELETE CASCADE,
	FOREIGN KEY (good_id, shop_id) REFERENCES goods(id, shop_id)
);

CREATE TABLE bundle_images (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	bundle_id bigint NOT NULL,
	shop_id bigint NOT NULL,
	path text NOT NULL,
	sort_order int NOT NULL CHECK (sort_order >= 0),
	UNIQUE (bundle_id, sort_order),
	FOREIGN KEY (bundle_id, shop_id) REFERENCES bundles(id, shop_id) ON DELETE CASCADE
);

ALTER TABLE event_listings ADD COLUMN IF NOT EXISTS price_tiers jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE event_listings ADD COLUMN shop_id bigint;
UPDATE event_listings l SET shop_id = e.shop_id FROM events e WHERE e.id = l.event_id AND l.shop_id IS NULL;
ALTER TABLE event_listings ALTER COLUMN shop_id SET NOT NULL;
ALTER TABLE event_listings ADD COLUMN reserved_qty int NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0);
ALTER TABLE event_listings ADD COLUMN revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0);
ALTER TABLE event_listings ADD CONSTRAINT event_listings_sold_reserved_check
	CHECK (sold_qty + reserved_qty <= listed_qty);
ALTER TABLE event_listings ADD CONSTRAINT event_listings_event_shop_fk
	FOREIGN KEY (event_id, shop_id) REFERENCES events(id, shop_id);
ALTER TABLE event_listings ADD CONSTRAINT event_listings_good_shop_fk
	FOREIGN KEY (good_id, shop_id) REFERENCES goods(id, shop_id);
ALTER TABLE event_listings ADD CONSTRAINT event_listings_id_good_event_shop_unique
	UNIQUE (id, good_id, event_id, shop_id);
ALTER TABLE event_listings ADD CONSTRAINT event_listings_id_shop_unique UNIQUE (id, shop_id);
CREATE INDEX event_listings_shop_event_idx ON event_listings(shop_id, event_id, is_listed);

CREATE TABLE event_bundle_listings (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL,
	event_id bigint NOT NULL,
	bundle_id bigint NOT NULL,
	unit_price_minor bigint NOT NULL CHECK (unit_price_minor > 0),
	price_tiers jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(price_tiers) = 'array'),
	is_listed boolean NOT NULL DEFAULT true,
	revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
	UNIQUE (event_id, bundle_id),
	UNIQUE (id, shop_id),
	UNIQUE (id, shop_id, event_id),
	FOREIGN KEY (event_id, shop_id) REFERENCES events(id, shop_id),
	FOREIGN KEY (bundle_id, shop_id) REFERENCES bundles(id, shop_id)
);
CREATE INDEX event_bundle_listings_shop_event_idx ON event_bundle_listings(shop_id, event_id, is_listed);
CREATE INDEX bundle_components_good_idx ON bundle_components(good_id, shop_id);

ALTER TABLE orders ADD COLUMN client_order_id uuid;
ALTER TABLE orders ADD COLUMN checkout_hash bytea;
ALTER TABLE orders ADD COLUMN snapshot_version int NOT NULL DEFAULT 0;
ALTER TABLE orders ALTER COLUMN snapshot_version SET DEFAULT 1;
ALTER TABLE orders ADD COLUMN reservation_expires_at timestamptz;
ALTER TABLE orders ADD COLUMN reservation_mode text NOT NULL DEFAULT 'legacy_unreserved';
ALTER TABLE orders ADD COLUMN state_version bigint NOT NULL DEFAULT 1 CHECK (state_version > 0);
ALTER TABLE orders ADD CONSTRAINT orders_checkout_hash_check CHECK (checkout_hash IS NULL OR octet_length(checkout_hash) = 32);
ALTER TABLE orders ADD CONSTRAINT orders_reservation_mode_check CHECK (reservation_mode IN ('legacy_unreserved', 'reserved', 'consumed', 'released'));
ALTER TABLE orders ADD CONSTRAINT orders_id_shop_unique UNIQUE (id, shop_id);
ALTER TABLE orders ADD CONSTRAINT orders_id_shop_event_unique UNIQUE (id, shop_id, event_id);
ALTER TABLE orders ADD CONSTRAINT orders_event_shop_fk FOREIGN KEY (event_id, shop_id) REFERENCES events(id, shop_id);
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'paid', 'fulfilled', 'cancelled', 'expired', 'refund_required'));
CREATE UNIQUE INDEX orders_client_order_unique ON orders(shop_id, client_order_id) WHERE client_order_id IS NOT NULL;
CREATE INDEX orders_expiration_idx ON orders(reservation_expires_at, id) WHERE status = 'pending' AND reservation_mode = 'reserved';

-- Snapshot version identifies legacy rows. Never identify them by amount:
-- after migration, version 1 zero-value lines are valid and stay untouched.
DO $migration$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'order_lines' AND column_name = 'line_total_minor'
	) THEN
		ALTER TABLE order_lines ADD COLUMN line_total_minor bigint NOT NULL DEFAULT 0;
	END IF;
END
$migration$;

ALTER TABLE order_lines ADD COLUMN snapshot_version int NOT NULL DEFAULT 0;
UPDATE order_lines SET line_total_minor = unit_price_minor * qty WHERE snapshot_version = 0;
ALTER TABLE order_lines ALTER COLUMN snapshot_version SET DEFAULT 1;
ALTER TABLE order_lines ADD COLUMN shop_id bigint;
ALTER TABLE order_lines ADD COLUMN event_id bigint;
UPDATE order_lines ol SET shop_id = o.shop_id, event_id = o.event_id FROM orders o WHERE o.id = ol.order_id;
ALTER TABLE order_lines ALTER COLUMN shop_id SET NOT NULL;
ALTER TABLE order_lines ALTER COLUMN event_id SET NOT NULL;
ALTER TABLE order_lines ADD COLUMN pricing_group_id bigint;
ALTER TABLE order_lines ADD COLUMN component_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(component_snapshot) = 'object');
ALTER TABLE order_lines ADD CONSTRAINT order_lines_order_shop_event_fk
	FOREIGN KEY (order_id, shop_id, event_id) REFERENCES orders(id, shop_id, event_id) ON DELETE CASCADE;
ALTER TABLE order_lines ADD CONSTRAINT order_lines_good_shop_fk FOREIGN KEY (good_id, shop_id) REFERENCES goods(id, shop_id);
ALTER TABLE order_lines ADD CONSTRAINT order_lines_listing_good_event_shop_fk
	FOREIGN KEY (listing_id, good_id, event_id, shop_id) REFERENCES event_listings(id, good_id, event_id, shop_id);
ALTER TABLE order_lines ADD CONSTRAINT order_lines_amount_check CHECK (unit_price_minor >= 0 AND line_total_minor >= 0);

CREATE TABLE order_quotes (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL REFERENCES shops(id),
	actor_id bigint NOT NULL REFERENCES users(id),
	event_id bigint NOT NULL,
	cart_version bigint NOT NULL CHECK (cart_version >= 0),
	input_hash bytea NOT NULL CHECK (octet_length(input_hash) = 32),
	catalog_revision bigint NOT NULL CHECK (catalog_revision > 0),
	snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
	total_minor bigint NOT NULL CHECK (total_minor >= 0),
	consumed_by_client_order_id uuid,
	expires_at timestamptz NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (event_id, shop_id) REFERENCES events(id, shop_id)
);
CREATE INDEX order_quotes_expiration_idx ON order_quotes(expires_at);
CREATE INDEX order_quotes_actor_event_idx ON order_quotes(shop_id, actor_id, event_id, created_at DESC);

CREATE TABLE order_commands (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL REFERENCES shops(id),
	actor_id bigint NOT NULL REFERENCES users(id),
	operation text NOT NULL CHECK (char_length(operation) BETWEEN 1 AND 100),
	key uuid NOT NULL,
	request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
	order_id bigint,
	response_status int,
	response_body jsonb,
	completed_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (shop_id, actor_id, operation, key),
	FOREIGN KEY (order_id, shop_id) REFERENCES orders(id, shop_id),
	CHECK ((completed_at IS NULL AND response_status IS NULL AND response_body IS NULL)
		OR (completed_at IS NOT NULL AND response_status BETWEEN 100 AND 599 AND response_body IS NOT NULL))
);
CREATE INDEX order_commands_order_idx ON order_commands(order_id) WHERE order_id IS NOT NULL;

CREATE TABLE order_payments (
	order_id bigint PRIMARY KEY,
	shop_id bigint NOT NULL,
	actor_id bigint NOT NULL REFERENCES users(id),
	pay_code_id bigint REFERENCES pay_codes(id),
	pay_code_name_snapshot text NOT NULL,
	amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
	currency text NOT NULL CHECK (currency IN ('MYR', 'SGD', 'CNY')),
	confirmed_at timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (order_id, shop_id) REFERENCES orders(id, shop_id)
);

CREATE TABLE inventory_reservations (
	shop_id bigint NOT NULL,
	order_id bigint NOT NULL,
	listing_id bigint NOT NULL,
	qty int NOT NULL CHECK (qty > 0),
	state text NOT NULL CHECK (state IN ('active', 'consumed', 'released')),
	expires_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (order_id, listing_id),
	FOREIGN KEY (order_id, shop_id) REFERENCES orders(id, shop_id) ON DELETE CASCADE,
	FOREIGN KEY (listing_id, shop_id) REFERENCES event_listings(id, shop_id)
);
CREATE INDEX inventory_reservations_active_expiration_idx ON inventory_reservations(expires_at, order_id) WHERE state = 'active';

CREATE TABLE inventory_movements (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL,
	order_id bigint NOT NULL,
	listing_id bigint NOT NULL,
	movement_type text NOT NULL CHECK (movement_type IN ('reserve', 'sell', 'release')),
	qty int NOT NULL CHECK (qty > 0),
	created_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (order_id, listing_id, movement_type),
	FOREIGN KEY (order_id, shop_id) REFERENCES orders(id, shop_id),
	FOREIGN KEY (listing_id, shop_id) REFERENCES event_listings(id, shop_id)
);

CREATE TABLE order_pricing_groups (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL,
	order_id bigint NOT NULL,
	rule_kind text NOT NULL CHECK (rule_kind IN ('single_tier', 'bundle', 'bundle_tier', 'regular')),
	rule_id bigint,
	rule_version bigint NOT NULL DEFAULT 1 CHECK (rule_version > 0),
	name_snapshot text NOT NULL,
	applications int NOT NULL DEFAULT 1 CHECK (applications > 0),
	total_minor bigint NOT NULL CHECK (total_minor >= 0),
	snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
	created_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (id, order_id),
	FOREIGN KEY (order_id, shop_id) REFERENCES orders(id, shop_id) ON DELETE CASCADE
);
ALTER TABLE order_lines ADD CONSTRAINT order_lines_pricing_group_order_fk
	FOREIGN KEY (pricing_group_id, order_id) REFERENCES order_pricing_groups(id, order_id);
CREATE INDEX order_pricing_groups_order_idx ON order_pricing_groups(order_id, id);

-- Pending orders from an older deployment do not own inventory reservations.
UPDATE orders SET reservation_mode = 'legacy_unreserved' WHERE status = 'pending';
