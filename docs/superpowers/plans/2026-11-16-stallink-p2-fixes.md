# Stallink P2 修复计划

## 目标

修复最近代码审查中确认的四项 P2 问题，并通过 Go 回归、前端构建及浏览器交互验证业务逻辑。工作仅限本地工作区，不提交代码。

## 已确认的问题

1. `BuildDailyReport` 分多条独立查询读取报表。订单在查询间变化时，同一份日报可能混合不同时间点的数据。
2. `consumeOrderReservations` 按 listing 顺序逐条锁定 listing 和 goods。多个订单跨活动占用相同商品时，反向锁序可能形成死锁。
3. 收银确认流程未对后端返回的 `reservation_expired` 做专门恢复，过期支付仍可能留在待处理 UI 状态。
4. 两个 `integration` 标签的数据层测试把数据库生成的 string ID 传给要求 `data.ID` 的接口，启用集成标签时无法编译。

## 方案

- 日报读取放进单个 PostgreSQL `REPEATABLE READ`, `READ ONLY` 事务，事务内所有查询使用同一快照，成功后提交，出错时回滚。
- 收款消费预留统一按商品 ID 顺序先锁定所有去重 goods，再按 listing ID 顺序锁定所有 listing，然后核验和更新库存、预留及流水。顺序与 `applyStock` 当前的商品优先顺序一致，避免不同订单及退款路径反向取锁。对会同时写商品和 listing 的 listing 更新路径，也按同一顺序取锁，并在锁后校验 listing 仍指向预期商品。
- 收银页识别 `reservation_expired`，清理该支付命令和订单提示状态、关闭支付弹窗、刷新商品数据，并提示操作员重新开单/选择订单。
- 修正集成测试夹具中的 ID 类型，在数据库边界解析成 `ID`，保留 SQL fixture 作为字符串参数的使用方式。
- 每项先添加或调整能暴露问题的验证，再实施最小修复。集成数据库未配置时不得改用现有业务数据库；先检查本地是否有明确隔离的测试服务。无法安全运行数据库测试时，至少执行集成标签编译和不依赖数据库的测试，并明确标注缺口。

## 影响文件

- `server/internal/data/report_store.go`：一致快照事务。
- `server/internal/data/reservation_store.go`、`server/internal/data/order_store.go`、`server/internal/data/catalog_store.go`：稳定全局锁序。
- `server/internal/data/*_integration_test.go`：集成测试 ID 类型修正，并补充必要的数据层回归用例。
- `web/src/app/counter/page.tsx` 及必要的 `web/src/lib/order-command.ts`：过期预留恢复。
- `docs/superpowers/plans/2026-11-16-stallink-p2-fixes.md`：本执行计划。

## 执行步骤

### 1. 建立验证基线

- 检查可用的隔离 Postgres、前端开发命令和浏览器状态；不得打印或复用数据库凭据。
- 运行 `cd server && go test -tags=integration ./internal/data -run '^$'`，确认当前集成测试编译失败的位置。
- 记录可用的测试命令，不重置、迁移或写入现有业务数据库。

### 2. 修正集成测试类型

- 先运行集成标签编译验证，确认 `ListOrderPage`、`ListAuditPage` 与 bundle 接口的类型错误。
- 对传入数据层 API 的 ID 在扫描后用 `ParseID` 解析并处理错误；ID 仍以字符串传给 SQL fixture。
- 重跑 `go test -tags=integration ./internal/data -run '^$'`，应编译通过；有隔离 DB 时再运行对应集成用例。

### 3. 固定日报快照

- 添加回归验证，确保构建日报的查询在一个只读、可重复读事务中执行。
- 把 `BuildDailyReport` 的查询入口统一切换到事务；保证每条错误路径回滚，成功路径提交。
- 在隔离 Postgres 上运行日报和相关数据层集成用例。若无隔离数据库，运行可执行的静态/编译验证并记录限制。

### 4. 消除预留消费的反向锁序

- 添加锁顺序回归验证，覆盖多条 listing 关联到交错排列的 goods 时的锁计划。
- 先收集仍有效预留所需的 listing/good 信息，再依照全局顺序锁定去重后的 goods 和 listings。审查 `applyStock`、listing 更新的锁序一致性，并在必要处按同一顺序调整。
- 确认失效/缺失关联会在事务中报错回滚，且库存与预留守恒断言仍覆盖全部涉及的商品。
- 在隔离 Postgres 上运行预留和并发支付相关集成用例；如当前没有相应并发用例，新增一个有超时上限的事务并发回归用例。

### 5. 修复收银过期状态恢复

- 先用隔离 mock API 在浏览器重现过期收款：确认旧逻辑保留支付弹窗/待处理订单状态，并记录实际提示。
- 对 `reservation_expired` 增加专用分支，确保只清理对应支付操作的本地状态，不误清其他待处理命令。
- 运行前端 lint/build 或仓库可用的相关验证。
- 再次在浏览器交互触发过期预留响应，检查提示、弹窗关闭、持久化标记清除与后续刷新状态。

### 6. 汇总验证

- 运行 `cd server && go test ./...` 与 `go test -tags=integration ./internal/data -run '^$'`；有隔离数据库时运行相关集成测试。
- 运行 `cd web && npm run lint` 和生产构建。
- 在浏览器中交互检查收银过期恢复和日报页面；若后端/数据库条件不足以完整验证日报，记录实际覆盖边界。
- 检查最终 diff 和工作区状态，确认旧未跟踪文档仍原样保留，且没有提交。

## 完成条件

- 四项 P2 均有针对性修复，相关验证通过或明确列出受环境限制无法完成的项目。
- 浏览器交互验证已实际执行并记录结果；无法执行时须说明阻塞原因和已完成的替代验证。
- 没有创建提交。

## 执行记录（2026-11-16）

- 集成标签初始编译复现了 ID 类型错误；修正后 `go test -tags=integration ./internal/data -run '^$'` 通过。
- `go test ./...` 通过。`go test -tags=integration ./internal/data` 通过编译；由于未配置 `STALLINK_TEST_DATABASE_URL`，数据库用例按设计跳过。没有连接本机用途未知的 5432 服务。
- 新增了重复读/只读快照与并发消费集成回归用例；它们编译通过，实际数据库执行仍需隔离测试库。
- `npx tsc --noEmit` 与 `npx next build --webpack` 通过。默认 Turbopack 构建在 CSS worker 创建本地子进程时遇到 `EPERM`，使用 webpack 构建完成生产包。
- `npm run lint` 在未修改的库存协调 effect（第 283、303 行）报两个 `react-hooks/set-state-in-effect` 错误，并提示两个 `refreshCatalog` 依赖警告。
- 浏览器使用临时 mock API 复现了原流程：过期响应后支付弹窗仍打开。修改后弹窗关闭并显示去订单列表取消旧待付款订单的提示；刷新后旧订单没有再次恢复。
- 日报页面和数据库并发用例没有连真实数据库进行交互验证；需在提供隔离 `STALLINK_TEST_DATABASE_URL` 后运行，当前已完成事务选项回归用例与并发测试的编译验证。
- 工作区未提交；原有的 `2026-11-13-stallink-quality-remediation.md` 未改动。
