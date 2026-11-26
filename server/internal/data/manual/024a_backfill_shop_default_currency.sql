-- 024a：存量店铺 default_currency 回填（K-Q2 = A、K-Q16 = A）。
-- 手动执行，不随 API 启动（本目录不在 //go:embed migrations/*.sql 路径下）。
-- ⚠️ 开发只在本机 docker 的 Postgres 上执行；云端由 Kel 自己备份后执行（K-Q6 = B）。前置：024 已执行。
-- 规则（Kel 2026-11-26 13:55）：严格取这家店最近一场活动（starts_on DESC, id DESC）的货币，不看主推活动；
-- 没有活动的店保持 NULL；已经有值的店不动。可重复执行，结果不变。
-- 不改 shops.updated_at：回填不是店主的修改。回滚见同目录 024a_backfill_shop_default_currency.rollback.sql。
-- 执行：psql "$LOCAL_DB_URL" -v ON_ERROR_STOP=1 -f server/internal/data/manual/024a_backfill_shop_default_currency.sql
BEGIN;
CREATE TABLE IF NOT EXISTS manual_024a_shop_currency_backup (
	shop_id bigint PRIMARY KEY REFERENCES shops (id) ON DELETE CASCADE,
	old_default_currency text,
	backfilled_at timestamptz NOT NULL DEFAULT now()
);
WITH target AS (
	SELECT s.id,
		(SELECT e.currency FROM events e WHERE e.shop_id = s.id ORDER BY e.starts_on DESC, e.id DESC LIMIT 1) AS currency
	FROM shops s
	WHERE s.default_currency IS NULL
), saved AS (
	INSERT INTO manual_024a_shop_currency_backup (shop_id, old_default_currency)
	SELECT t.id, NULL FROM target t WHERE t.currency IS NOT NULL
	ON CONFLICT (shop_id) DO NOTHING
	RETURNING shop_id
)
UPDATE shops s SET default_currency = t.currency
FROM target t JOIN saved v ON v.shop_id = t.id
WHERE s.id = t.id;
COMMIT;
