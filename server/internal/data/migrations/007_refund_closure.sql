-- Refund closure: widen the order status and stock movement type CHECKs so a
-- paid order can move to `refund_required` and then be confirmed as refunded
-- (`cancelled` with `refunded_at`). Constraint names are looked up dynamically:
-- the identity migration rebuilt constraints, so the original names are not
-- guaranteed. The block is idempotent: re-running it does not error or add a
-- duplicate constraint.

DO $$
DECLARE
	status_constraint text;
	movement_constraint text;
BEGIN
	-- orders.status: find the CHECK whose definition lists `pending` (the
	-- original status constraint), drop it, and add the widened set.
	SELECT conname INTO status_constraint
	FROM pg_constraint
	WHERE conrelid = 'orders'::regclass
		AND contype = 'c'
		AND pg_get_constraintdef(oid) LIKE '%pending%'
	LIMIT 1;

	IF status_constraint IS NOT NULL THEN
		EXECUTE format('ALTER TABLE orders DROP CONSTRAINT %I', status_constraint);
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'orders'::regclass
			AND contype = 'c'
			AND pg_get_constraintdef(oid) LIKE '%refund_required%'
	) THEN
		ALTER TABLE orders ADD CONSTRAINT orders_status_check
			CHECK (status IN ('pending', 'paid', 'fulfilled', 'cancelled', 'expired', 'refund_required'));
	END IF;

	-- inventory_movements.movement_type: add the `refund` reverse flow.
	SELECT conname INTO movement_constraint
	FROM pg_constraint
	WHERE conrelid = 'inventory_movements'::regclass
		AND contype = 'c'
		AND pg_get_constraintdef(oid) LIKE '%reserve%'
	LIMIT 1;

	IF movement_constraint IS NOT NULL THEN
		EXECUTE format('ALTER TABLE inventory_movements DROP CONSTRAINT %I', movement_constraint);
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'inventory_movements'::regclass
			AND contype = 'c'
			AND pg_get_constraintdef(oid) LIKE '%refund%'
	) THEN
		ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_movement_type_check
			CHECK (movement_type IN ('reserve', 'sell', 'release', 'refund'));
	END IF;
END $$;

-- refunded_at records when a pending refund was confirmed; NULL until then.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS refunded_at timestamptz;
