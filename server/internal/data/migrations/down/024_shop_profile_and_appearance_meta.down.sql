-- 024 手动回滚脚本：放在 migrations/down/，不会被 //go:embed migrations/*.sql 加载，API 启动不会执行。
-- ⚠️ 需 Kel 手动确认：对任何非本机数据库执行前，必须先得到 Kel 批准。
-- 如果已经执行过 manual/024a，先执行 manual/024a_backfill_shop_default_currency.rollback.sql。
-- 执行：psql "$DB_URL" -v ON_ERROR_STOP=1 -f server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql
-- 回滚会丢掉店主填写的客服联系、店铺摊位号、默认货币，以及外观的发布人和时间，执行前先导出：
--   psql "$DB_URL" -c "\copy (SELECT id, contact, booth_number, default_currency FROM shops) TO 'shops_profile_backup.csv' CSV HEADER"
--   psql "$DB_URL" -c "\copy (SELECT updated_at, updated_by FROM platform_appearance) TO 'appearance_meta_backup.csv' CSV HEADER"
BEGIN;
DROP INDEX IF EXISTS audit_logs_shop_created_id_idx;
ALTER TABLE platform_appearance
	DROP COLUMN IF EXISTS updated_by,
	DROP COLUMN IF EXISTS updated_at;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_default_currency_check;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_booth_number_check;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_contact_check;
ALTER TABLE shops
	DROP COLUMN IF EXISTS default_currency,
	DROP COLUMN IF EXISTS booth_number,
	DROP COLUMN IF EXISTS contact;
DELETE FROM schema_migrations WHERE version = 24;
COMMIT;
