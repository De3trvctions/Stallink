# Stallink 服务端自增 ID 实施计划

> **执行说明：** 本计划仅覆盖 Go 服务、PostgreSQL、本地数据重置与种子。逐项勾选；不提交 Git，除非用户另行要求。

**目标：** 将所有业务实体主键和外键从 UUID 改为 BIGINT identity，同时保留安全的本地数据备份、重置与重新注入流程。

**架构：** HTTP 边界解析十进制实体 ID，Go 数据层使用 int64，PostgreSQL identity 自动生成主键。新迁移仅在显式清空后的本地库运行；非空旧库失败关闭。

**技术栈：** Go、Kratos HTTP、pgx/PostgreSQL、现有版本迁移器、Nacos 本地配置、Zap。

**设计依据：** [数据库自增 ID 规范](../specs/2026-11-12-stallink-bigint-identity-spec.md)。

## 文件职责

| 文件 | 职责 |
|---|---|
| server/internal/data/migrations/002_bigint_identity.sql | 验证空库并将现有 UUID 主外键列转为 BIGINT identity 及外键 |
| server/internal/data/migrations.go | 识别并报告不支持的非空 UUID schema，不静默清理 |
| server/internal/data/id.go | 定义 int64 实体 ID 类型、正数校验及 decimal-string JSON marshal/unmarshal |
| server/internal/data/*.go | 将数据模型、方法签名、扫描/插入 SQL 和关联查询统一到 BIGINT |
| server/internal/service/*.go、server/internal/server/http.go | 解析路径/JSON 实体 ID，更新 API 输出；保留 clientOrderId 和 Idempotency-Key UUID |
| server/cmd/stallink/main.go | 把正常服务启动与本地重置/种子命令分开 |
| server/internal/data/seed.go | 改为显式、事务化、可重复执行的本地 seed，不在服务启动时写 demo 数据 |
| server/internal/data/*_test.go、server/internal/service/*_test.go | 覆盖迁移、ID 解析、关系约束、幂等及隔离行为 |
| server/Makefile、根 Makefile、README.md | 增加受保护的本地备份/重置/注入命令与使用说明 |

## 任务 1：定义实体 ID 和 HTTP 校验

**文件：** 新建 server/internal/data/id.go、id_test.go；修改涉及实体 ID 的 service handlers。

- [ ] 先为 ParseID 写表驱动测试：接受 "1"、"42"、int64 最大值；拒绝空串、0、负数、空格、正号、小数、溢出和非十进制字符。
- [ ] 实现严格正数解析；命名实体 ID 类型以 int64 参与 pgx 编解码，以 JSON 字符串形式 marshal/unmarshal，保证订单命令的持久化响应也保持原样。
- [ ] 将活动、商品、上架、套餐、店铺、用户、订单、收款码、图片和报价 ID 的 URL/JSON 边界统一走该解析器。
- [ ] 保留 uuid.Parse 只用于 clientOrderId 与 Idempotency-Key；检查 order_quotes.consumed_by_client_order_id 继续绑定 UUID checkout ID。
- [ ] 运行 cd server && go test ./internal/data ./internal/service，预期新增解析器用例通过；现有集成用例在未设置 STALLINK_TEST_DATABASE_URL 时只报告跳过。

## 任务 2：为 UUID 到 BIGINT 的空库转换编写迁移

**文件：** 新建 server/internal/data/migrations/002_bigint_identity.sql；修改迁移错误映射及集成测试。

- [ ] 先写迁移集成测试：含任一业务行时返回 identity_reset_required 且行数不变；空 schema 转换后主键列是 bigint、identity 属性为 always。
- [ ] 迁移开始先检查应用表为空；检查范围包含 shops, users, sessions, events, goods, good_images, event_listings, pay_codes, orders, order_lines, audit_logs, bundles, bundle_components, bundle_images, event_bundle_listings, order_quotes, order_commands, order_payments, inventory_reservations, inventory_movements, order_pricing_groups。
- [ ] 保留已应用的 001_catalog_orders.sql 不变。新版本 002 在事务中删除会阻止类型修改的外键约束，再将实体 PK/FK、audit_logs.subject_id 和 order_pricing_groups.rule_id 转成 BIGINT；为实体 id 添加 GENERATED ALWAYS AS IDENTITY。
- [ ] 仅将真正的业务关系改为数字：orders.client_order_id、order_quotes.consumed_by_client_order_id、order_commands.key 和 schema_migrations.version 保持原类型。
- [ ] 在同一迁移末尾按 schema.go 和 migration 001 重建全部外键、复合店铺外键及原有 ON DELETE 规则；保留唯一约束、CHECK、索引和复合主键。
- [ ] 扩展 migration integration test，验证空库和重复启动都通过；非空数据只读失败。
- [ ] 将 STALLINK_TEST_DATABASE_URL 配置为专用隔离测试库后，运行 cd server && go test ./internal/data -run 'TestMigrations|TestIdentity' -count=1，预期通过。

## 任务 3：把 Go 数据层及 SQL 全部切换为 int64

**文件：** server/internal/data/shop.go, auth_store.go, catalog_store.go, bundle_store.go, order_store.go, quote_store.go, order_command_store.go, reservation_store.go, good_image_store.go, paged_lists.go 及关联 service。

- [ ] 将实体 ID 字段和数据层参数改为 int64/命名 ID 类型；删除实体 UUID 字符串扫描。
- [ ] 将 SQL 返回实体 ID 的 id::text 改为 bigint；插入继续使用 RETURNING id，不在应用中生成实体主键。
- [ ] 逐处检查 ID 数组：把 array_agg(id) 显式转为文本数组，或扫描到 []int64，保证 pgx 类型一致。
- [ ] 更新多租户 join、subject_id、rule_id、报价快照、库存预留、订单支付和幂等重放代码；保留 order_commands.key UUID 和客户端 checkout ID。
- [ ] 更新所有集成测试 fixture，使用数据库生成的 ID；禁止手动指定 identity 值。
- [ ] 运行 cd server && go test ./internal/catalog ./internal/auth ./internal/conf，再运行 cd server && go test ./...；预期纯逻辑及无数据库测试通过，数据库集成项需提供独立测试 DSN 后通过。

## 任务 4：保护本地数据重置并重做种子流程

**文件：** server/cmd/stallink/main.go, server/internal/data/seed.go, server/internal/data/reset.go, Makefiles 和 README。

- [ ] 为 reset 先写防护测试：无显式确认、目标库名不匹配或运行环境非 local 时，函数在执行任何 SQL 前拒绝。
- [ ] 实现只接受明确本地配置和数据库名确认的 reset-local 命令；只清业务表并 RESTART IDENTITY CASCADE，保留 schema_migrations，清理前输出表名和行数、不输出账号密码或 DSN。
- [ ] 添加 backup-local 命令说明/目标：先用 pg_dump --format=custom 备份目标库，并归档 storage.dir。reset 不自动删除备份。
- [ ] 把 SeedDemo 从普通服务启动路径移除；添加显式 seed-local 命令，在事务中创建 alpha/beta 两店和 6 个账号（1 platform_admin、2 shop_admin、2 sales、1 customer）、每店一场活动、demo fixture 商品、一个固定套餐和收款码。
- [ ] seed 只从本地忽略配置读取初始口令并调用 auth.HashPassword；测试并重复运行时不重复插入；不写入假订单、假审计行或占位商品图片。
- [ ] 运行 cd server && go test ./internal/data ./internal/service；在独立临时数据库上按顺序执行 backup、reset、migrate、seed，再检查每个角色绑定和商品外键。

## 任务 5：端到端验证与回滚检查

**文件：** server/internal/data/migrations_integration_test.go, server/internal/data/paged_lists_integration_test.go, server/internal/service/order_commands_integration_test.go, server/internal/service/auth_integration_test.go。

- [ ] 增加越店、无效数字 ID、两个店铺 ID 相互不可访问、订单重放仍返回相同结果的集成断言。
- [ ] 将 STALLINK_TEST_DATABASE_URL 配置为专用隔离测试库后，用新空测试库执行 cd server && go test ./... -count=1。
- [ ] 在备份副本上执行空库迁移和 seed；用 pg_restore --list 检查备份可读，不覆盖原始备份。
- [ ] 启动本地 API，确认 /api/health、登录、活动/商品查询、图片读取、报价、订单幂等和分页响应均使用十进制字符串 ID。
- [ ] 对照规范列出的所有关系检查约束；确认普通 make dev 不会清库或自动插入 demo。

## 执行边界

- 本计划不直接迁移或清除生产数据库。
- 当前请求是规划；真正执行本地 reset 时，先完成代码、备份和隔离库验收，再切换本地 Nacos 配置。
- 不执行 Git commit。
