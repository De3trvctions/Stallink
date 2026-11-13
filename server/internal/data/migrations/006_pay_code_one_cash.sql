-- One cash method per shop. QR codes stay unlimited.
-- Keep the earliest cash row and point existing payments at it.
WITH ranked AS (
	SELECT id, shop_id, row_number() OVER (PARTITION BY shop_id ORDER BY id) AS n
	FROM pay_codes
	WHERE kind = 'cash'
), extras AS (
	SELECT extra.id AS extra_id, keeper.id AS keeper_id
	FROM ranked extra
	JOIN ranked keeper ON keeper.shop_id = extra.shop_id AND keeper.n = 1
	WHERE extra.n > 1
)
UPDATE orders
SET pay_code_id = extras.keeper_id
FROM extras
WHERE orders.pay_code_id = extras.extra_id;

WITH ranked AS (
	SELECT id, shop_id, row_number() OVER (PARTITION BY shop_id ORDER BY id) AS n
	FROM pay_codes
	WHERE kind = 'cash'
), extras AS (
	SELECT extra.id AS extra_id, keeper.id AS keeper_id
	FROM ranked extra
	JOIN ranked keeper ON keeper.shop_id = extra.shop_id AND keeper.n = 1
	WHERE extra.n > 1
)
UPDATE order_payments
SET pay_code_id = extras.keeper_id
FROM extras
WHERE order_payments.pay_code_id = extras.extra_id;

WITH ranked AS (
	SELECT id, row_number() OVER (PARTITION BY shop_id ORDER BY id) AS n
	FROM pay_codes
	WHERE kind = 'cash'
)
DELETE FROM pay_codes
WHERE id IN (SELECT id FROM ranked WHERE n > 1);

CREATE UNIQUE INDEX IF NOT EXISTS pay_codes_one_cash ON pay_codes (shop_id) WHERE kind = 'cash';
