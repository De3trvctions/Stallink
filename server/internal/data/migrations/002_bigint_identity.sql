-- Entity identifiers are intentionally reset locally before this migration.
-- Refuse to alter a populated database: there is no safe UUID-to-integer
-- mapping that preserves external references.
DO $$
DECLARE
	app_tables text[] := ARRAY[
		'shops', 'users', 'sessions', 'events', 'goods', 'good_images',
		'event_listings', 'pay_codes', 'orders', 'order_lines', 'audit_logs',
		'bundles', 'bundle_components', 'bundle_images', 'event_bundle_listings',
		'order_quotes', 'order_commands', 'order_payments', 'inventory_reservations',
		'inventory_movements', 'order_pricing_groups'
	];
	identity_tables text[] := ARRAY[
		'shops', 'users', 'sessions', 'events', 'goods', 'good_images',
		'event_listings', 'pay_codes', 'orders', 'order_lines', 'audit_logs',
		'bundles', 'bundle_images', 'event_bundle_listings', 'order_quotes',
		'order_commands', 'inventory_movements', 'order_pricing_groups'
	];
	table_name text;
	identity_table text;
	column_row record;
	constraint_row record;
	has_rows boolean;
	schema_name text := current_schema();
BEGIN
	IF schema_name IS NULL THEN
		RAISE EXCEPTION 'identity migration requires a selected schema';
	END IF;

	FOREACH table_name IN ARRAY app_tables LOOP
		IF to_regclass(format('%I.%I', schema_name, table_name)) IS NULL THEN
			RAISE EXCEPTION 'identity migration expected table %.%', schema_name, table_name;
		END IF;
		EXECUTE format('SELECT EXISTS (SELECT 1 FROM %I.%I LIMIT 1)', schema_name, table_name) INTO has_rows;
		IF has_rows THEN
			RAISE EXCEPTION 'identity_reset_required: table %.% contains rows; back up and explicitly reset the local database before migration', schema_name, table_name;
		END IF;
	END LOOP;

	CREATE TEMP TABLE stallink_identity_foreign_keys (
		table_name text NOT NULL,
		constraint_name text NOT NULL,
		definition text NOT NULL
	) ON COMMIT DROP;

	INSERT INTO stallink_identity_foreign_keys(table_name, constraint_name, definition)
	SELECT rel.relname, con.conname, pg_get_constraintdef(con.oid)
	FROM pg_constraint con
	JOIN pg_class rel ON rel.oid = con.conrelid
	JOIN pg_namespace ns ON ns.oid = rel.relnamespace
	WHERE con.contype = 'f' AND ns.nspname = schema_name AND rel.relname = ANY(app_tables);

	FOR constraint_row IN
		SELECT f.table_name, f.constraint_name FROM stallink_identity_foreign_keys AS f
	LOOP
		EXECUTE format('ALTER TABLE %I.%I DROP CONSTRAINT %I', schema_name, constraint_row.table_name, constraint_row.constraint_name);
	END LOOP;

	-- Convert every UUID foreign/subject identifier in the application schema.
	-- These UUIDs are intentionally retained as idempotency identities.
	FOR column_row IN
		SELECT c.table_name, c.column_name
		FROM information_schema.columns AS c
		WHERE c.table_schema = schema_name
			AND c.table_name = ANY(app_tables)
			AND c.data_type = 'uuid'
			AND NOT (c.table_name = 'orders' AND c.column_name = 'client_order_id')
			AND NOT (c.table_name = 'order_quotes' AND c.column_name = 'consumed_by_client_order_id')
			AND NOT (c.table_name = 'order_commands' AND c.column_name = 'key')
	LOOP
		EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN %I DROP DEFAULT', schema_name, column_row.table_name, column_row.column_name);
		EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN %I TYPE bigint USING NULL::bigint', schema_name, column_row.table_name, column_row.column_name);
	END LOOP;

	FOREACH identity_table IN ARRAY identity_tables LOOP
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns AS c
			WHERE c.table_schema = schema_name
				AND c.table_name = identity_table
				AND c.column_name = 'id'
				AND c.is_identity = 'YES'
		) THEN
			EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY', schema_name, identity_table);
		END IF;
	END LOOP;

	FOR constraint_row IN
		SELECT f.table_name, f.constraint_name, f.definition
		FROM stallink_identity_foreign_keys AS f
		ORDER BY f.table_name, f.constraint_name
	LOOP
		EXECUTE format('ALTER TABLE %I.%I ADD CONSTRAINT %I %s', schema_name, constraint_row.table_name, constraint_row.constraint_name, constraint_row.definition);
	END LOOP;
END $$;
