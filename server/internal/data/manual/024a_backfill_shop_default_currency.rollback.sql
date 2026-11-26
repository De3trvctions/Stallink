-- 024a 回滚：只撤销 024a 回填过、且之后店主没有保存过资料（shops.updated_at 不晚于回填时间）的店。
-- ⚠️ 开发只在本机 docker 的 Postgres 上执行；云端由 Kel 自己执行。
-- 执行：psql "$LOCAL_DB_URL" -v ON_ERROR_STOP=1 -f server/internal/data/manual/024a_backfill_shop_default_currency.rollback.sql
BEGIN;
UPDATE shops s SET default_currency = b.old_default_currency
FROM manual_024a_shop_currency_backup b
WHERE s.id = b.shop_id AND s.updated_at <= b.backfilled_at;
DROP TABLE IF EXISTS manual_024a_shop_currency_backup;
COMMIT;
