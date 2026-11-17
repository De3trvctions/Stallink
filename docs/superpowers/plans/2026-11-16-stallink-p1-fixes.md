# Stallink P1 修复计划

## 目标

修复当前源码仍存在的两项高影响问题：部分库存预留过期时仍可能完成收款，以及日报活动金额混合不同币种。完成后运行单元测试、集成标签编译、前端类型检查和生产构建；不提交代码。

## 已核对的问题

1. `consumeOrderReservations` 只读取未过期的 active reservation；只要还有一条有效记录，就会消费并返回成功。订单行要求数量与有效预留数量没有先做全量比较，因此订单可能只扣部分库存，却被后续标记为已付款。
2. `BuildDailyReport` 的活动查询按 event 聚合，没有把 currency 纳入 `SELECT` / `GROUP BY`；不同币种的 minor 金额会合并。日报顶部的订单状态和售出数量也从所有币种累加，但顶部金额只选最高收入币种，口径不一致。

## 方案

- 在拿取库存/ listing 行锁和写入之前，用订单行作为预期数量，与有效预留逐 listing、逐 good、逐数量核对。任何缺失、多余、过期或不匹配都返回 `ErrReservationExpired`，保持交易全有或全无。保留上一轮已统一的商品优先全局锁序。
- 以当前收入最高的币种作为日报主币种；顶部订单总数、已付款数、待付款数、关闭数、退款待处理数和售出件数都只统计该币种。`Currencies` 仍逐币种显示。活动明细按活动和币种分行，前端用每行自己的币种格式化金额。
- 先添加单元与集成回归验证，再修改实现。Postgres 集成用例只允许使用 `STALLINK_TEST_DATABASE_URL` 指向的隔离测试库；未配置时不连接用途未知的本机数据库。

## 影响文件

- `server/internal/data/reservation_store.go`：订单行与有效预留全量匹配。
- `server/internal/data/reservation_store_test.go`：全量匹配规则单元测试。
- `server/internal/data/reservation_atomicity_integration_test.go`：一条预留过期时不得部分扣减的数据库回归测试。
- `server/internal/data/report_store.go`、`report_store_test.go`：主币种总览口径与按币种活动明细。
- `server/internal/data/report_currency_integration_test.go`：同一活动多币种统计回归测试。
- `web/src/app/admin/reports/page.tsx`：活动明细按行币种呈现，并阻止旧多币种快照被静默当成正确财务数据展示。
- `docs/superpowers/plans/2026-11-16-stallink-p1-fixes.md`：本计划与执行记录。

## 执行步骤

### 1. 预留必须全量匹配

- 增加纯函数回归用例，覆盖完整匹配、缺失 listing、额外 listing、数量不符、good 不符。
- 增加隔离 Postgres 集成用例：订单包含两个 listing，其中一个 reservation 已过期、另一个仍有效；确认返回 `ErrReservationExpired`，订单保持待付款，两个 listing 的已售/预留数与 goods 库存均未变化。
- 在 `consumeOrderReservations` 中读取订单期望行，和未过期 active reservation 逐项比较；核对失败时在商品与 listing 加锁/更新之前返回。
- 运行 `go test ./internal/data` 及 `go test -tags=integration ./internal/data -run '^$'`。数据库用例仅在隔离 DSN 可用时执行。

### 2. 日报多币种正确性

- 扩展 `TestApplyHeadlineCurrencyUsesTopCurrencyPaidOrders`，先验证主币种订单数和已付款数也应用于 headline。
- 添加隔离 Postgres 集成用例：同一活动有 MYR 与 SGD 已付款订单，另有 SGD 待付款订单；确认 headline 只统计收入最高币种，活动明细按币种拆成两行，金额不相加。
- 确保币种聚合先决定 `report.Currency`；随后按该币种查询 headline 状态、售出件数。币种明细仍保留所有币种。
- 活动 SQL 将 currency 加入投影、分组与扫描；前端 key 使用 event ID 与 currency 组合，活动金额按行币种格式化，行数标签改为“条”。
- 识别已保存但缺少活动币种的新旧版本日报；若包含多个币种则暂停展示财务数值，提示重新生成。
- 运行数据层单元测试、集成标签编译、`npx tsc --noEmit` 和 `npx next build --webpack`。

### 3. 最终核对

- 运行 `go test ./...`、`go test -tags=integration ./internal/data`、`git diff --check`。
- 集成数据库未配置时，明确标注两个数据库回归用例只完成编译，未实际执行。
- 检查 P1 计划和代码 diff；保留既有未跟踪整改文档，不提交。

## 完成条件

- 部分过期预留不会导致任何库存消费或收款成功。
- 日报任何 headline 或活动金额均不跨币种相加；页面显示每条活动的金额币种。
- 可执行验证通过，数据库验证缺口清楚记录。
- 没有创建 Git 提交。

## 执行记录

- 已在 `consumeOrderReservations` 更新库存前，将未过期预留与订单行按 listing、good 和数量全量比对；增加了匹配规则单元测试和部分过期预留数据库回归用例。
- 日报改为按收入最高币种统计总览、订单状态与售出件数；活动明细按活动和币种拆行。多币种旧快照缺少活动币种时，页面提示重新生成并隐藏旧财务数值。
- `GOCACHE=/private/tmp/stallink-gocache go test ./...`：通过。
- `go test -tags=integration ./internal/data`：通过编译；新增的部分过期预留和多币种数据库用例因未设置 `STALLINK_TEST_DATABASE_URL` 而跳过。未连接本机未知用途的数据库。
- `npx tsc --noEmit`：通过。`npx next build --webpack`：通过。
- `npm run lint`：仍有柜台页 `counter/page.tsx` 第 283、303 行的两处 `react-hooks/set-state-in-effect` 错误，以及第 349、404 行的 `refreshCatalog` 依赖警告；这些不在本次 P1 变更中。
- 浏览器交互使用仅监听本机的模拟 API：旧多币种快照显示“这份日报使用旧版币种口径”提示且不展示财务汇总；点击重新生成后，同一活动显示为 MYR `RM 100.00` 与 SGD `S$ 40.00` 两行，并显示“2 条”。此项验证了前端交互，不替代数据库集成测试。
- `git diff --check`：通过。没有创建提交。
