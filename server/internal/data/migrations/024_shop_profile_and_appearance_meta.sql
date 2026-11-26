-- 024：店铺资料按工作台模板补齐（Kel 2026-11-26，K-04），平台外观记录发布时间和发布人，操作记录分页索引。
-- 只改结构，不含任何数据修改：API 启动会自动执行本文件。
-- contact：客服联系，自由文本，顾客可见。
-- booth_number：店铺级摊位号；K-Q1 = A：新建活动的默认值，顾客端活动没填时回退到它（应用层）。
-- default_currency：店铺默认货币，NULL 表示未设置。
-- shops.updated_at 已存在，由应用层在写入时设为 now()（项目没有触发器）。
-- slug 不在这里改：仍是 NOT NULL UNIQUE，只是应用层改成店主只读。
-- 存量 default_currency 回填不在这里（K-Q16 = A）：见 server/internal/data/manual/024a_backfill_shop_default_currency.sql，
-- 该目录不会被 //go:embed migrations/*.sql 加载，必须人工执行。
-- 回滚：server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql（同样不会被加载）。
ALTER TABLE shops
	ADD COLUMN IF NOT EXISTS contact text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS booth_number text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS default_currency text;

DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_contact_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_contact_check
			CHECK (char_length(contact) <= 120 AND contact !~ '[[:cntrl:]]');
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_booth_number_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_booth_number_check
			CHECK (char_length(booth_number) <= 40 AND booth_number !~ '[[:cntrl:]]' AND booth_number = btrim(booth_number));
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_default_currency_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_default_currency_check
			CHECK (default_currency IS NULL OR default_currency IN ('MYR', 'SGD', 'CNY'));
	END IF;
END $$;

-- 平台外观元数据：历史发布时间和发布人不可考，保持 NULL。不改 012、013。
ALTER TABLE platform_appearance
	ADD COLUMN IF NOT EXISTS updated_at timestamptz,
	ADD COLUMN IF NOT EXISTS updated_by bigint REFERENCES users (id) ON DELETE SET NULL;

-- 操作记录分页索引（后端 B-6）：迁移在事务里执行，不能用并发建索引。
CREATE INDEX IF NOT EXISTS audit_logs_shop_created_id_idx ON audit_logs (shop_id, created_at DESC, id DESC);
