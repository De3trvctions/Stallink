# 手动 SQL 脚本

这里的脚本只能人工执行：本目录不在 `//go:embed migrations/*.sql` 的路径下，API 启动不会执行它们。开发只在本机 docker 的 Postgres 上执行；⚠️ 对云端或任何非本机数据库执行前，必须先由 Kel 备份并确认。

- `024a_backfill_shop_default_currency.sql`：迁移 024 之后，按每家店最近一场活动回填 `shops.default_currency`。回滚用 `024a_backfill_shop_default_currency.rollback.sql`。
