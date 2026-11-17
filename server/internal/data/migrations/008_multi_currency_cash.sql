-- Multi-currency pay codes and cash tender tracking.
-- pay_codes.currency is nullable: NULL means the code follows the event's
-- default currency at checkout. Existing rows are left NULL on purpose.

ALTER TABLE pay_codes ADD COLUMN IF NOT EXISTS currency text;

-- Cash tender snapshot for cash pay methods. Non-cash orders keep both NULL.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cash_received_minor bigint;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cash_change_minor bigint;

-- Snapshot of the pay code's currency at the time of payment, kept alongside
-- the order currency already recorded on order_payments.
ALTER TABLE order_payments ADD COLUMN IF NOT EXISTS pay_code_currency text;
