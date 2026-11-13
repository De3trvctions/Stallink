ALTER TABLE pay_codes ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'qr';
ALTER TABLE pay_codes DROP CONSTRAINT IF EXISTS pay_codes_kind_check;
ALTER TABLE pay_codes ADD CONSTRAINT pay_codes_kind_check CHECK (kind IN ('qr', 'cash'));
