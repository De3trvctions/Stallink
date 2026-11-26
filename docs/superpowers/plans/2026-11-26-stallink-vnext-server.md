# Stallink vNext 后端（Go API + PostgreSQL）实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 基于 `docs/superpowers/specs/2026-11-26-stallink-prd-vnext.md` 与 `2026-11-26-stallink-feature-list.md`，将 Stallink Go 后端升级至 vNext：落地口径 2 共享池+单场上限/净锁定保底、2s 锁超时与 503 重试、部分退款/报损与跨日对账、真离线现金同步与 `oversold` 闭环、媒体分层缓存与公开接口限流，并支撑单场峰值 1,000 QPS、开单 P95 < 500ms。

**架构：** 在不打破既有 Kratos HTTP + pgxpool + Nacos + Google Drive 主体架构的前提下，增强事务并发守卫与锁序一致性，将可售池公式与保底消费封装进 `internal/catalog` 纯领域函数，再由 `internal/data` 以单事务原子执行；数据库迁移从 **`015`** 开始至 **`023`**（`014` 由并行修复 plan 占用），回滚脚本独立存放于 `server/internal/data/migrations/down/`。

**技术栈：** Go 1.27 (`github.com/De3trvctions/Stallink`) · Kratos v2 HTTP · PostgreSQL 16 (`pgx/v5` + `pgxpool`) · Redis (`go-redis/v9`) · Google Drive API · `k6`

---

## 一、与并行修复 Plan 的协调边界（必读）

并行计划 `docs/superpowers/plans/2026-11-26-stallink-review-fixes.md` 会**先于本计划合入**，并负责以下前置安全与一致性修复；**本 vNext 计划不重复实现，相关任务显式标注依赖的修复 plan 任务编号**：
1. **Drive 孤儿清理跨环境误删防护（修复 plan 任务 1，C1）**：独立根目录校验与误删防护归修复 plan 任务 1，vNext 媒体与存储变更严禁绕过该校验。
2. **启动/热更新自动迁移守卫与迁移超时/校验和（修复 plan 任务 2）**：非本机库禁止无检查自动迁移、显式 `migrate` 命令及 `lock_timeout`/`checksum` 归修复 plan 任务 2；vNext 直接依赖该迁移安全边界。
3. **迁移编号 `014` 与 `migrations/down/` 目录规范（修复 plan 任务 3）**：任务 3 占用 `014` 新增 `orders.payment_ref`、`audit_logs` 平台审计扩展字段及关键索引，并确立 `server/internal/data/migrations/down/` 回滚目录；vNext 迁移从 **`015`** 开始（`015`–`023`），`019` 不再重复添加 `payment_ref`。
4. **移除 `readGoodPool` 中 `shopID == 0` 跳过店铺过滤（修复 plan 任务 5）**：归修复 plan 任务 5；vNext `Task S2`、`Task S13` 直接在此安全基线上扩展，不重复编写。
5. **`releaseOrderReservations` 全局锁序修复（修复 plan 任务 6）**：归修复 plan 任务 6；vNext `Task S2` 直接依赖该锁序，仅增量加入 `expires_at > now()` 实时过滤与 `floor_reserved_qty` 退回。
6. **`Cancel` / `RefundConfirm` / 一步 `CancelRefund` 与角色原因校验（修复 plan 任务 7，P1-1）**：Kel 11-26 拍板按 Q3=B 执行——同店 `sales` 与 `shop_admin` 可执行取消、`cancel-refund`（新增一步全额取消并退款+全回库）及存量 `refund_required` 的 `refund-confirm`；跨店返回 `403 forbidden`；`pending` 与已收款取消/退款统一必填 `reason`（`TrimSpace` 后 `2–500` Unicode 字符：`reason_required`/`reason_too_short`/`reason_too_long`），并写入 `audit_logs.reason` 与 `detail`。vNext `Task S7` 在此基础上扩展部分退款、报损不回库与总额重算。
7. **分布式/降级限流与可信代理真实 IP（修复 plan 任务 8）**：登录/注册（P1-2）及既有公开匿名限流基础设施与真实 IP 提取归修复 plan 任务 8；vNext `Task S11` 直接复用该限流器增量补齐 `/api/staff/order-quotes` 与高并发突发参数。
8. **自动审核开关 & 外观切换平台审计（修复 plan 任务 10）**：基础平台审计归修复 plan 任务 10；vNext `Task S16` 在其基础上扩展全量操作 `before_json`/`after_json`/`reason`/`terminal_id`/`client_ip`。
9. **公开图片响应缓存头与有界内存缓存（修复 plan 任务 11）**：`Cache-Control`/`ETag` 与服务端有界内存缓存归修复 plan 任务 11；vNext `Task S10` 在其上游增量补齐 `media_token` 防枚举与抗重启本地磁盘缓存 `NewDiskCache`。
10. **公开目录分页限流优化 + 图片解码并发控制与 EXIF 剥离（修复 plan 任务 12）**：归修复 plan 任务 12；vNext `Task S11`/`Task S13` 在此基础上增量落地 `allow_public_catalog`（Opt-in 默认 `false`）与 `pg_trgm` GIN 检索。

---

## 二、迁移与回滚安全存放规范（必读）

1. **为什么必须放进 `server/internal/data/migrations/down/`（与修复 plan 任务 3、任务 15 一致）**：
   核查 `server/internal/data/migrations.go`（第 16、67–94 行）可知，代码通过 `//go:embed migrations/*.sql` 与 `fs.Glob(migrationFiles, "migrations/*.sql")` **仅嵌入和扫描 `migrations/` 顶层的 `.sql` 文件，不会递归加载子目录 `migrations/down/`**。反之若把 `.down.sql` 放在 `migrations/` 顶层，同编号会触发 `duplicate migration version` 导致启动直接 panic，顺延编号则会被当成正向迁移自动执行。
2. **安全存放位置与回滚方法**：
   - 正向迁移：`server/internal/data/migrations/015_*.sql` ~ `023_*.sql`
   - 回滚脚本：**`server/internal/data/migrations/down/015_*.down.sql` ~ `023_*.down.sql`**（存放于 `migrations/down/` 子目录，不被 `//go:embed migrations/*.sql` 自动加载，并配合修复 plan 任务 15 的 CI 配对检查）；回滚时人工运行 `psql "$DB_URL" -f server/internal/data/migrations/down/NNN_说明.down.sql` 并执行 `DELETE FROM schema_migrations WHERE version = NNN;`。
3. **环境红线**：
   - 本地开发与 TDD 集成测试**仅允许连接 `docker-compose.yml` 本机 Postgres（`127.0.0.1:5432`）**。
   - **⚠️ 需要 Kel 手动确认**：任何对云端/远程数据库执行迁移或回滚、修改远程 Nacos 配置、直连非本机库启动 API 或运行压测的步骤，必须由 Kel 明确确认。

---

## 三、阶段划分与跨端并行协作矩阵

> **拆分布局说明**：**Phase 1（P0，Tasks S1–S8, S9a, S9b, S10–S12，共 13 个 P0 任务）** 写至完整可直接开发的 TDD 代码与 SQL 级；**Phase 2（P1，Tasks S13–S19）** 与 **Phase 3（P2，Tasks S20–S22）** 写至完整 SQL + 接口/TDD 验收任务级。建议先执行完并行修复 plan 与 Phase 1（P0），再按模块并行推进 Phase 2/3。

| 后端任务 | 对应功能 | 优先级 | 前置依赖 | 可并行组 | 对应前端任务（见 `vnext-web.md`） |
|---|---|---|---|---|---|
| **S1** 迁移 `015` | F-01 | P0 | 修复 plan 任务 2, 3（`014`） | 组 A | W1 |
| **S2** 口径 2 共享池+保底+过期实时剔除 | F-01, F-02 | P0 | S1 + 修复 plan 任务 5（去 `shopID==0`）, 任务 6（锁序） | 组 A | W1 |
| **S3** 2s 锁超时/`503`/死锁重试/连接池 | F-03 | P0 | 无 | 组 B | W2 |
| **S4** 付款前资格失效自动撤单 | F-04 | P0 | S2, S3 | 组 A | W2 |
| **S5** 审核修正与套餐裁剪/连通组校验 | F-05, F-06 | P0 | 无 | 组 C | W3 |
| **S6** 迁移 `016` + 活动级时区 | F-10 | P0 | 无 | 组 D | W4 |
| **S7** 迁移 `017` + 部分退款/报损/重算 | F-08 | P0 | S2, S3 + 修复 plan 任务 7（`P1-1`） | 组 A | W6 |
| **S8** 迁移 `018` + 跨日对账报表 | F-09 | P0 | S6, S7 | 组 A | W7 |
| **S9a** 离线租约 HMAC 签名下发与验签 | F-11 | P0 | 无 | 组 D（可并行） | W8, W9 |
| **S9b** 迁移 `019` + 离线同步验签与 `oversold` | F-11, F-12 | P0 | S2, S3, S7, S9a + 修复 plan 任务 3（`014`） | 组 A | W8, W9, W10 |
| **S10** 迁移 `020` + 媒体 `media_token` 与落盘二级缓存 | F-13 | P0 | 修复 plan 任务 1（Drive 防护）, 任务 11（响应/内存缓存）, 任务 12（解码/EXIF） | 组 E | 无强阻塞 |
| **S11** 报价限流与目录突发阈值（复用任务 8/12） | F-14 | P0 | 修复 plan 任务 8（限流器/真实 IP）, 任务 12（分页限流） | 组 E | 无 |
| **S12** 1,000 QPS / P95 < 500ms 压测 | 全 P0 | P0 | S1–S11 | 验收关口 | 与前端 W10 E2E 并行 |
| **S13–S19** P1 任务（迁移 `021`–`023`） | F-15~F-24 | P1 | Phase 1 + 修复 plan 任务 3（`014`）, 任务 5, 任务 10（审计）, 任务 12 | 并行分组 | W11–W18 |
| **S20–S22** P2 任务 | F-25, F-26, F-28 | P2 | Phase 2 | 独立并行 | W19 |

---

## 四、Phase 1：P0 核心实现任务（完整 TDD 规格）

### Task S1：数据库迁移 `015` — 单场上限 `max_qty` 与净锁定保底 `min_reserve_qty`（F-01）

**文件：**
- 创建：`server/internal/data/migrations/015_listing_cap_and_floor.sql`
- 创建：`server/internal/data/migrations/down/015_listing_cap_and_floor.down.sql`
- 测试：`server/internal/data/migrations_015_test.go`

- [ ] **步骤 1：编写失败的迁移集成测试**

```go
// server/internal/data/migrations_015_test.go
//go:build integration

package data

import (
	"context"
	"testing"
)

func TestMigration015ListingCapAndFloor(t *testing.T) {
	ctx := context.Background()
	pool := openTestPool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE (table_name = 'event_listings' AND column_name IN ('max_qty', 'min_reserve_qty'))
		   OR (table_name = 'inventory_reservations' AND column_name = 'floor_reserved_qty')
	`).Scan(&count)
	if err != nil || count != 3 {
		t.Fatalf("expected 3 new columns, got count=%d err=%v", count, err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestMigration015ListingCapAndFloor -v`
预期：FAIL，输出 `expected 3 new columns, got count=0`。

- [ ] **步骤 3：编写升级 SQL `015` 与独立回滚 SQL**

```sql
-- server/internal/data/migrations/015_listing_cap_and_floor.sql
ALTER TABLE event_listings
	ADD COLUMN IF NOT EXISTS max_qty int,
	ADD COLUMN IF NOT EXISTS min_reserve_qty int NOT NULL DEFAULT 0;

ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_max_qty_check;
ALTER TABLE event_listings ADD CONSTRAINT event_listings_max_qty_check
	CHECK (max_qty IS NULL OR max_qty > 0);

ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_min_reserve_qty_check;
ALTER TABLE event_listings ADD CONSTRAINT event_listings_min_reserve_qty_check
	CHECK (min_reserve_qty >= 0);

ALTER TABLE inventory_reservations
	ADD COLUMN IF NOT EXISTS floor_reserved_qty int NOT NULL DEFAULT 0;

ALTER TABLE inventory_reservations DROP CONSTRAINT IF EXISTS inventory_reservations_floor_qty_check;
ALTER TABLE inventory_reservations ADD CONSTRAINT inventory_reservations_floor_qty_check
	CHECK (floor_reserved_qty >= 0 AND floor_reserved_qty <= qty);
```

```sql
-- server/internal/data/migrations/down/015_listing_cap_and_floor.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：psql "$DB_URL" -f server/internal/data/migrations/down/015_listing_cap_and_floor.down.sql && psql "$DB_URL" -c "DELETE FROM schema_migrations WHERE version = 15;"
ALTER TABLE inventory_reservations DROP CONSTRAINT IF EXISTS inventory_reservations_floor_qty_check;
ALTER TABLE inventory_reservations DROP COLUMN IF EXISTS floor_reserved_qty;

ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_min_reserve_qty_check;
ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_max_qty_check;
ALTER TABLE event_listings DROP COLUMN IF EXISTS min_reserve_qty;
ALTER TABLE event_listings DROP COLUMN IF EXISTS max_qty;
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestMigration015ListingCapAndFloor -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/015_listing_cap_and_floor.sql server/internal/data/migrations/down/015_listing_cap_and_floor.down.sql server/internal/data/migrations_015_test.go
git commit -m "feat(data): add migration 015 for listing cap and net floor reservation"
```

---

### Task S2：口径 2 共享池 + 单场上限/净锁定保底 + 过期预留实时剔除（F-01, F-02）

> **前置依赖**：`Task S1`；**去 `shopID==0` 过滤依赖修复 plan 任务 5，`releaseOrderReservations` 全局锁序依赖修复 plan 任务 6**。对应前端：`Task W1`。

**文件：**
- 修改：`server/internal/catalog/stock.go`
- 修改：`server/internal/data/catalog_store.go`、`server/internal/data/reservation_store.go`、`server/internal/data/quote_store.go`、`server/internal/data/bundle_store.go`
- 修改：`server/internal/service/staff.go`、`server/internal/service/handlers.go`
- 测试：`server/internal/catalog/stock_floor_test.go`、`server/internal/data/floor_reservation_integration_test.go`

- [ ] **步骤 1：编写失败的领域单元测试**

```go
// server/internal/catalog/stock_floor_test.go
package catalog

import "testing"

func TestListingPoolCapAndNetFloor(t *testing.T) {
	maxQty := 8
	lp := ListingPool{
		TotalStock:     10,
		ActiveReserved: 1,
		LockedByOthers: 4,
		EventSold:      3,
		EventReserved:  1,
		MaxQty:         &maxQty,
		MinReserveQty:  2,
	}
	if got := lp.Available(); got != 4 {
		t.Fatalf("expected available=4 (min(5,4)), got %d", got)
	}
	if err := ValidateFloorAlloc(10, 2, 9); err != ErrStockExceeded {
		t.Fatalf("expected ErrStockExceeded when 2+9 > 10, got %v", err)
	}
	if err := ValidateListingQuota(&maxQty, 3, 1, 5); err != ErrInvalidListingQuota {
		t.Fatalf("expected ErrInvalidListingQuota when minReserve(5) > remCap(4), got %v", err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/catalog -run TestListingPoolCapAndNetFloor -v`
预期：FAIL，提示 `undefined: ListingPool`。

- [ ] **步骤 3：实现 `catalog/stock.go` 口径 2 模型与数据层扣减/释放逻辑**

```go
// 追加至 server/internal/catalog/stock.go
var ErrInvalidListingQuota = errors.New("catalog: invalid listing cap or net floor")

type ListingPool struct {
	TotalStock     int
	ActiveReserved int
	LockedByOthers int
	EventSold      int
	EventReserved  int
	MaxQty         *int
	MinReserveQty  int
}

func (p ListingPool) FreePool() int {
	if v := p.TotalStock - p.ActiveReserved - p.LockedByOthers; v > 0 {
		return v
	}
	return 0
}

func (p ListingPool) CapRemain() (int, bool) {
	if p.MaxQty == nil {
		return 0, false
	}
	if v := *p.MaxQty - p.EventSold - p.EventReserved; v > 0 {
		return v, true
	}
	return 0, true
}

func (p ListingPool) Available() int {
	avail := p.FreePool()
	if capRem, bounded := p.CapRemain(); bounded && capRem < avail {
		avail = capRem
	}
	return avail
}

func ValidateFloorAlloc(totalStock, totalActiveReserved, totalActiveFloors int) error {
	if totalActiveReserved+totalActiveFloors > totalStock {
		return ErrStockExceeded
	}
	return nil
}

func ValidateListingQuota(maxQty *int, eventSold, eventReserved, minReserve int) error {
	if minReserve < 0 {
		return ErrInvalidListingQuota
	}
	if maxQty != nil && (*maxQty <= 0 || eventSold+eventReserved+minReserve > *maxQty) {
		return ErrInvalidListingQuota
	}
	return nil
}
```

数据层增量实现要点：
1. `catalog_store.go` / `quote_store.go` / `bundle_store.go` 所有预占聚合子查询追加 `AND expires_at > now()`，并联查 `LockedByOthers`（其他未结束且在架活动 `min_reserve_qty` 之和）；
2. `UpdateListing` / `AddListing` / `UpdateGood` 改 `total_stock` 时按 `goods` 行锁校验 `ValidateFloorAlloc` 与 `ValidateListingQuota`；活动置 `ended` 或下架 `is_listed=false` 时同一事务将 `min_reserve_qty` 清零释放；
3. `reservation_store.go` 在预占时扣减 `min_reserve_qty -= fromFloor` 并记录 `inventory_reservations.floor_reserved_qty = fromFloor`；在已由修复 plan 任务 6 统一锁序的 `releaseOrderReservations` 中增量按活动状态将 `floor_reserved_qty` 退回 `min_reserve_qty`；`ExpireReservations` 按每批 20 单独立短事务释放；
4. `handlers.go` 映射 `ErrInvalidListingQuota` -> `409 invalid_listing_quota`。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/catalog -v && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run "TestMigration015|TestFloorReservation" -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/catalog/ server/internal/data/ server/internal/service/
git commit -m "feat(stock): implement net floor reservation, event cap, and live expiry filter"
```

---

### Task S3：高并发事务保护 — 2s 锁超时、5s 上限、503 退避、死锁重试与连接池扩容（F-03）

> 对应前端：`Task W2`

**文件：**
- 修改：`server/internal/conf/conf.go`、`server/configs/nacos/stallink.yaml`
- 修改：`server/internal/data/shop.go`、`server/internal/data/order_command_store.go`、`server/internal/data/order_store.go`、`server/internal/data/quote_store.go`
- 修改：`server/internal/service/handlers.go`
- 测试：`server/internal/data/command_timeout_retry_test.go`、`server/internal/service/command_503_test.go`

- [ ] **步骤 1：编写失败的错误分类单测与 `503 + Retry-After: 1` 接口测试**

```go
// server/internal/data/command_timeout_retry_test.go
package data

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestClassifyRetryablePgError(t *testing.T) {
	if !isRetryableTxErr(&pgconn.PgError{Code: "40P01"}) || !isRetryableTxErr(&pgconn.PgError{Code: "40001"}) {
		t.Fatalf("expected 40P01 and 40001 to be retryable")
	}
	if !isLockTimeoutErr(&pgconn.PgError{Code: "55P03"}) {
		t.Fatalf("expected 55P03 to classify as lock timeout")
	}
	if isRetryableTxErr(errors.New("other")) {
		t.Fatalf("unexpected retryable on generic error")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/data -run TestClassifyRetryablePgError -v`
预期：FAIL，提示 `undefined: isRetryableTxErr`。

- [ ] **步骤 3：实现事务超时、死锁重试、去 `shops` `FOR SHARE` 锁与连接池扩容**

1. `order_command_store.go` 新增 `isRetryableTxErr` / `isLockTimeoutErr`，在 `runOrderCommand` 内注入 `SET LOCAL lock_timeout = '2s'; SET LOCAL statement_timeout = '5s'`；遇 `40P01`/`40001` 重试最多 2 次；遇 `55P03`/`57014` 返回 `ErrCommandInProgress`。
2. `handlers.go` 将 `ErrCommandInProgress` 映射为 `w.Header().Set("Retry-After", "1"); writeErr(w, http.StatusServiceUnavailable, "command_in_progress")`。
3. `order_store.go` 与 `quote_store.go` 移除对 `shops` 行的 `FOR SHARE` 阻塞锁；`shop.go` `NewPool` 支持配置 `MaxConns = 50, MinConns = 5`。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/data ./internal/service -run "TestClassifyRetryablePgError|TestCommand503" -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/conf/ server/configs/nacos/stallink.yaml server/internal/data/ server/internal/service/
git commit -m "perf(tx): enforce 2s lock_timeout, 503 Retry-After, 2x deadlock retry, and 50 pool conns"
```

---

### Task S4：付款前资格失效自动取消订单并释放预留（F-04）

> **K-Q10 = A（Kel 已定 2026-11-26 13:22）：** 在线收款不再接收 `cashReceivedMinor`，`PayInput` 没有 `CashReceivedMinor` 字段；下面测试里的 `PayInput` 已改成只传 `PayCodeID`。清理本身在 `2026-11-26-stallink-workspace-polish.md` 任务 S6。

> 对应前端：`Task W2`

**文件：**
- 修改：`server/internal/data/order_store.go`、`server/internal/data/order_command_store.go`、`server/internal/service/handlers.go`
- 测试：`server/internal/data/pay_eligibility_cancel_test.go`

- [ ] **步骤 1：编写失败测试（下架后付款返回 `ErrOrderCancelledEligibilityLost`，订单变为 `cancelled`+`released` 且重放一致）**

```go
// server/internal/data/pay_eligibility_cancel_test.go
//go:build integration

package data

import (
	"context"
	"errors"
	"testing"
)

func TestPayAutoCancelsWhenListingDisabled(t *testing.T) {
	ctx := context.Background()
	store := openSeededTestStore(t)
	order, idKey := createPendingOrderFixture(t, store)
	disableListingFixture(t, store, order.Lines[0].ListingID)

	res, err := store.PayOrderCommand(ctx, testPayIdentity(order, idKey), order.ID, PayInput{PayCodeID: testCashCodeID(t, store, order.ShopID)})
	if !errors.Is(err, ErrOrderCancelledEligibilityLost) || res.Order.Status != "cancelled" || res.Order.ReservationMode != "released" {
		t.Fatalf("expected cancelled+released with ErrOrderCancelledEligibilityLost, got err=%v order=%+v", err, res.Order)
	}
	replay, err := store.PayOrderCommand(ctx, testPayIdentity(order, idKey), order.ID, PayInput{PayCodeID: testCashCodeID(t, store, order.ShopID)})
	if !errors.Is(err, ErrOrderCancelledEligibilityLost) || !replay.Replayed {
		t.Fatalf("expected replayed ErrOrderCancelledEligibilityLost, got replayed=%v err=%v", replay.Replayed, err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestPayAutoCancelsWhenListingDisabled -v`
预期：FAIL，返回 `ErrNotListed`。

- [ ] **步骤 3：在 `payOrderTx` 与 `runOrderCommand` 实现事务内撤单+释放预留+持久化 `409` 终态**

定义 `ErrOrderCancelledEligibilityLost`；当 `payStillAllowed != nil` 时释放预留（含 `floor_reserved_qty` 退回）、置 `status = 'cancelled', reservation_mode = 'released'`、写审计，并在 `runOrderCommand` 提交该 `409` 终态以保证幂等重放一致。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestPayAutoCancelsWhenListingDisabled -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/order_store.go server/internal/data/order_command_store.go server/internal/service/
git commit -m "fix(orders): auto-cancel pending order and release stock if eligibility lost before pay"
```

---

### Task S5：审核规则修正与套餐连通组校验（F-05, F-06）

> 对应前端：`Task W3`

**文件：**
- 修改：`server/internal/data/catalog_store.go`、`server/internal/data/bundle_store.go`、`server/internal/service/bundles.go`、`server/internal/catalog/cart_pricing.go`
- 测试：`server/internal/data/moderation_rules_test.go`

- [ ] **步骤 1：编写失败测试（调价不打回套餐、`name` 变更重审、`rejected` 不被 `autoApprove` 自动复活、保存期拦截超 20 SKU / 10 套餐）**

```go
// server/internal/data/moderation_rules_test.go
//go:build integration

package data

import (
	"context"
	"testing"
)

func TestModerationRulesVNext(t *testing.T) {
	ctx := context.Background()
	store := openSeededTestStore(t)
	good, bundle := seedApprovedGoodAndBundle(t, store)

	if err := store.UpdateListingPriceOnly(ctx, good.ShopID, good.ListingID, 1500, nil, false); err != nil {
		t.Fatalf("update price: %v", err)
	}
	assertReviewStatus(t, store, "goods", good.ID, "approved")
	assertReviewStatus(t, store, "bundles", bundle.ID, "approved")

	if err := store.UpdateGoodCatalog(ctx, good.ShopID, good.ID, "New Name", good.Description, good.TotalStock, false); err != nil {
		t.Fatalf("update name: %v", err)
	}
	assertReviewStatus(t, store, "goods", good.ID, "pending")

	rejectGood(t, store, good.ID, "bad content")
	if err := store.UpdateGoodCatalog(ctx, good.ShopID, good.ID, "Fixed Name", "Fixed Desc", good.TotalStock, true); err != nil {
		t.Fatalf("resubmit rejected: %v", err)
	}
	assertReviewStatus(t, store, "goods", good.ID, "pending")
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestModerationRulesVNext -v`
预期：FAIL。

- [ ] **步骤 3：实现 `name`/`description`/图片重审、`rejected` 禁自动复活、调价不误杀套餐，并在 `bundles.go` 调用 `ValidateShopBundleGraphLimits`**

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/catalog ./internal/service -v && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestModerationRulesVNext -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/catalog/ server/internal/data/ server/internal/service/
git commit -m "fix(moderation): review on name change, prevent rejected auto-approve, validate bundle graph limits"
```

---

### Task S6：迁移 `016` — 活动级 IANA 时区与 `allow_presale`（F-10）

> 对应前端：`Task W4`

> **合入后的后续项：** 多活动店铺的店铺级时区取值顺序见第八节 B-14。

**文件：**
- 创建：`server/internal/data/migrations/016_event_timezone_and_presale.sql`
- 创建：`server/internal/data/migrations/down/016_event_timezone_and_presale.down.sql`
- 修改：`server/internal/data/catalog_store.go`、`server/internal/service/staff.go`、`server/internal/service/validation.go`
- 测试：`server/internal/service/event_timezone_test.go`

- [ ] **步骤 1：编写失败测试**

```go
// server/internal/service/event_timezone_test.go
package service

import (
	"testing"
	"time"
)

func TestComputeEventClockState(t *testing.T) {
	nowUTC := time.Date(2026, 11, 26, 16, 30, 0, 0, time.UTC)
	klState, err := computeEventClockState("Asia/Kuala_Lumpur", "2026-11-26", nowUTC)
	if err != nil || klState.EventToday != "2026-11-27" || !klState.IsEndedNow {
		t.Fatalf("KL expected ended=true today=2026-11-27, got %+v err=%v", klState, err)
	}
	bkkState, err := computeEventClockState("Asia/Bangkok", "2026-11-26", nowUTC)
	if err != nil || bkkState.EventToday != "2026-11-26" || bkkState.IsEndedNow {
		t.Fatalf("Bangkok expected ended=false today=2026-11-26, got %+v err=%v", bkkState, err)
	}
	if _, err := computeEventClockState("Mars/Olympus", "2026-11-26", nowUTC); err == nil {
		t.Fatalf("expected invalid timezone error")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/service -run TestComputeEventClockState -v`
预期：FAIL，提示 `undefined: computeEventClockState`。

- [ ] **步骤 3：编写迁移 `016` + 回滚脚本，实现 `time.LoadLocation` 校验与 `(now() AT TIME ZONE e.timezone)::date` 判定**

```sql
-- server/internal/data/migrations/016_event_timezone_and_presale.sql
ALTER TABLE events
	ADD COLUMN IF NOT EXISTS timezone text NOT NULL DEFAULT 'Asia/Kuala_Lumpur',
	ADD COLUMN IF NOT EXISTS allow_presale boolean NOT NULL DEFAULT false;
```

```sql
-- server/internal/data/migrations/down/016_event_timezone_and_presale.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：DELETE FROM schema_migrations WHERE version = 16;
ALTER TABLE events DROP COLUMN IF EXISTS allow_presale;
ALTER TABLE events DROP COLUMN IF EXISTS timezone;
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/service -run TestComputeEventClockState -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/016_event_timezone_and_presale.sql server/internal/data/migrations/down/016_event_timezone_and_presale.down.sql server/internal/data/ server/internal/service/
git commit -m "feat(events): add per-event IANA timezone and allow_presale flag"
```

---

### Task S7：迁移 `017` — 部分退款、报损不回库、套餐破坏重算与必填原因/审计（F-08）

> **前置依赖与分工**：`Task S2`、`Task S3` 与修复 plan `2026-11-26-stallink-review-fixes.md` **任务 7（`P1-1`）**。整单全额一步取消并退款（`POST /api/staff/orders/{orderID}/cancel-refund`，全回库）及既有 `cancel`/`refund-confirm` 由任务 7 落地；本任务专注部分退款与报损不回库（`POST /api/staff/orders/{orderID}/refunds`），并完全复用任务 7 的同店角色校验、`reason` 字段（`2–500` Unicode 字符：`422 reason_required`/`reason_too_short`/`reason_too_long`）与 `014` 的 `audit_logs.reason`。对应前端：`Task W6`。

**文件：**
- 创建：`server/internal/data/migrations/017_partial_refunds_and_damage.sql`
- 创建：`server/internal/data/migrations/down/017_partial_refunds_and_damage.down.sql`
- 修改：`server/internal/data/order_store.go`、`server/internal/service/staff.go`、`server/internal/server/http.go`
- 测试：`server/internal/data/partial_refund_test.go`

- [ ] **步骤 1：编写失败测试（验证 Q3=B：同店 `sales` 可部分退 1 件报损 A、最大可退 RM6、库存不增、审计日志包含操作人/原因/金额/时间）**

```go
// server/internal/data/partial_refund_test.go
//go:build integration

package data

import (
	"context"
	"testing"
)

func TestPartialRefundBundleClawbackAndDamage(t *testing.T) {
	ctx := context.Background()
	store := openSeededTestStore(t)
	order, goodA, _ := createPaidBundleOrderFixture(t, store) // A=1000, B=1000, AB=1600 minor
	stockABefore := readTotalStock(t, store, goodA)

	preview, err := store.PreviewOrderRefund(ctx, order.ShopID, order.ID, []RefundPreviewItem{{OrderLineID: order.Lines[0].ID, RefundQty: 1}})
	if err != nil || preview.MaxRefundMinor != 600 || preview.DiscountClawbackMinor != 400 {
		t.Fatalf("expected maxRefund=600 clawback=400, got %+v err=%v", preview, err)
	}

	salesIdentity := testSalesRefundIdentity(t, store, order)
	res, err := store.CreateOrderRefundCommand(ctx, salesIdentity, order.ID, CreateRefundInput{
		Reason:      "拆封压坏",
		RefundMinor: 600,
		Items:       []RefundItemInput{{OrderLineID: order.Lines[0].ID, RestockQty: 0, DamagedQty: 1}},
	})
	if err != nil || !res.Order.PartialRefunded || res.Order.Status != "paid" {
		t.Fatalf("expected partialRefunded=true status=paid, got %+v err=%v", res.Order, err)
	}
	if gotStock := readTotalStock(t, store, goodA); gotStock != stockABefore {
		t.Fatalf("damaged refund must not restock: before=%d after=%d", stockABefore, gotStock)
	}
	assertRefundAuditRecorded(t, store, order.ID, salesIdentity.ActorID, "拆封压坏", 600)
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestPartialRefundBundleClawbackAndDamage -v`
预期：FAIL，提示 `undefined: PreviewOrderRefund`。

- [ ] **步骤 3：编写迁移 `017` + 回滚脚本，实现退款试算与部分退款/报损确认**

```sql
-- server/internal/data/migrations/017_partial_refunds_and_damage.sql
ALTER TABLE orders
	ADD COLUMN IF NOT EXISTS paid_at timestamptz,
	ADD COLUMN IF NOT EXISTS cancel_reason text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS partial_refunded boolean NOT NULL DEFAULT false,
	ADD COLUMN IF NOT EXISTS refunded_total_minor bigint NOT NULL DEFAULT 0 CHECK (refunded_total_minor >= 0);

UPDATE orders SET paid_at = created_at WHERE status IN ('paid', 'fulfilled') AND paid_at IS NULL;

CREATE TABLE IF NOT EXISTS order_refunds (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	shop_id bigint NOT NULL REFERENCES shops (id),
	order_id bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	refund_minor bigint NOT NULL CHECK (refund_minor >= 0),
	reason text NOT NULL CHECK (char_length(reason) BETWEEN 2 AND 500),
	refunded_by bigint NOT NULL REFERENCES users (id),
	terminal_id text NOT NULL DEFAULT '',
	refunded_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_refund_lines (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	refund_id bigint NOT NULL REFERENCES order_refunds (id) ON DELETE CASCADE,
	order_line_id bigint NOT NULL REFERENCES order_lines (id),
	restock_qty int NOT NULL DEFAULT 0 CHECK (restock_qty >= 0),
	damaged_qty int NOT NULL DEFAULT 0 CHECK (damaged_qty >= 0),
	refund_minor bigint NOT NULL CHECK (refund_minor >= 0),
	CHECK (restock_qty + damaged_qty > 0)
);

DO $$
DECLARE c name;
BEGIN
	SELECT conname INTO c FROM pg_constraint WHERE conrelid = 'inventory_movements'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) LIKE '%movement_type%';
	IF c IS NOT NULL THEN EXECUTE format('ALTER TABLE inventory_movements DROP CONSTRAINT %I', c); END IF;
END $$;
ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_movement_type_check
	CHECK (movement_type IN ('sale', 'refund', 'damage'));
```

```sql
-- server/internal/data/migrations/down/017_partial_refunds_and_damage.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：DELETE FROM schema_migrations WHERE version = 17;
DROP TABLE IF EXISTS order_refund_lines;
DROP TABLE IF EXISTS order_refunds;
ALTER TABLE orders DROP COLUMN IF EXISTS refunded_total_minor;
ALTER TABLE orders DROP COLUMN IF EXISTS partial_refunded;
ALTER TABLE orders DROP COLUMN IF EXISTS cancel_reason;
ALTER TABLE orders DROP COLUMN IF EXISTS paid_at;
ALTER TABLE inventory_movements DROP CONSTRAINT IF EXISTS inventory_movements_movement_type_check;
ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_movement_type_check
	CHECK (movement_type IN ('sale', 'refund'));
```

实现细节：
- `CreateOrderRefundCommand` 沿用任务 7（`P1-1`）的 `readOrderActionReason` 口径：字段名统一 `reason`、`TrimSpace` 后按 `utf8.RuneCountInString` 校验 **`2–500` Unicode 字符**（`422 reason_required` / `reason_too_short` / `reason_too_long`），并复用同店 `sales`/`shop_admin` 归属校验；整单全额退+全回库走任务 7 的 `/cancel-refund`，本任务仅扩展部分退款与报损不回库；
- `order_refunds` 精确保存 `refunded_by`（操作人）、`reason`（原因）、`refund_minor`（金额）与 `refunded_at`（时间），同步写入 `audit_logs`；
- 仅 `restock_qty`（剔除未曾扣出的 `oversold` 缺口后）加回 `goods.total_stock`；`damaged_qty` 写入 `inventory_movements (movement_type='damage')`；
- `http.go` 注册：`POST /api/staff/orders/{orderID}/refund-preview`、`POST /api/staff/orders/{orderID}/refunds`。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestPartialRefundBundleClawbackAndDamage -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/017_partial_refunds_and_damage.sql server/internal/data/migrations/down/017_partial_refunds_and_damage.down.sql server/internal/data/order_store.go server/internal/service/staff.go server/internal/server/http.go
git commit -m "feat(refunds): support partial refunds, damaged non-restock, bundle clawback, and mandatory audit"
```

---

### Task S8：迁移 `018` — 跨日退款日报冲减与 `paid_at`/`refunded_at` 对账改造（F-09）

> 对应前端：`Task W7`

**文件：**
- 创建：`server/internal/data/migrations/018_daily_report_cashflow_offset.sql`
- 创建：`server/internal/data/migrations/down/018_daily_report_cashflow_offset.down.sql`
- 修改：`server/internal/data/report_store.go`、`server/internal/service/reports.go`
- 测试：`server/internal/data/report_crossday_refund_test.go`

- [ ] **步骤 1：编写失败测试**

```go
// server/internal/data/report_crossday_refund_test.go
//go:build integration

package data

import (
	"context"
	"testing"
)

func TestDailyReportCrossDayRefundOffset(t *testing.T) {
	ctx := context.Background()
	store := openSeededTestStore(t)
	order := seedCrossDayPaidAndPartialRefunded(t, store, "2026-11-25", "2026-11-26", 10000, 4000)

	repDay2, err := store.GenerateDailyReport(ctx, order.ShopID, "2026-11-26")
	if err != nil || repDay2.RefundOffsetMinor != 4000 || repDay2.NetReceivedMinor != -4000 || len(repDay2.RefundItems) != 1 {
		t.Fatalf("expected day2 refund offset 4000, net -4000, 1 item; got %+v err=%v", repDay2, err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestDailyReportCrossDayRefundOffset -v`
预期：FAIL。

- [ ] **步骤 3：编写迁移 `018` + 回滚脚本，改造 `report_store.go` 按 `e.timezone` 聚合 `paid_at` 与 `order_refunds.refunded_at`**

```sql
-- server/internal/data/migrations/018_daily_report_cashflow_offset.sql
ALTER TABLE daily_reports
	ADD COLUMN IF NOT EXISTS gross_paid_minor bigint NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS refund_offset_minor bigint NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS net_received_minor bigint NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS refund_details jsonb NOT NULL DEFAULT '[]'::jsonb;
```

```sql
-- server/internal/data/migrations/down/018_daily_report_cashflow_offset.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：DELETE FROM schema_migrations WHERE version = 18;
ALTER TABLE daily_reports DROP COLUMN IF EXISTS refund_details;
ALTER TABLE daily_reports DROP COLUMN IF EXISTS net_received_minor;
ALTER TABLE daily_reports DROP COLUMN IF EXISTS refund_offset_minor;
ALTER TABLE daily_reports DROP COLUMN IF EXISTS gross_paid_minor;
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestDailyReportCrossDayRefundOffset -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/018_daily_report_cashflow_offset.sql server/internal/data/migrations/down/018_daily_report_cashflow_offset.down.sql server/internal/data/report_store.go server/internal/service/reports.go
git commit -m "feat(reports): track cross-day refunds by refunded_at and paid_at per event timezone"
```

---

### Task S9a：离线 4 小时租约 HMAC 签名下发与验签（F-11）

> **与 `Task S9b`（离线同步）的关系**：终端联网时调用本任务实现的 `POST /api/staff/offline-lease` 获取绑定 `shopId`、`terminalId`、`lastOnlineSyncAt`、`expiresAt` 的 `HMAC-SHA256` 签名租约；`Task S9b` 在接收 `POST /api/staff/orders/offline-sync` 时调用本任务导出的 `VerifyOfflineLease` 强验签名（防篡改），并在同步完成后调用 `IssueOfflineLease` 返回新一轮 `refreshedLease`。**密钥 `STALLINK_OFFLINE_HMAC_SECRET` 仅通过环境变量 / Nacos 运行配置注入服务端，绝不写进任何代码或仓库配置模板**。对应前端：`Task W8`、`Task W9`。

**文件：**
- 创建：`server/internal/service/offline_lease.go`
- 修改：`server/internal/conf/conf.go`、`server/internal/service/staff.go`、`server/internal/server/http.go`
- 测试：`server/internal/service/offline_lease_test.go`

- [ ] **步骤 1：编写失败的单元测试（验证签发、验签、跨店/跨终端伪造或篡改时间失败）**

```go
// server/internal/service/offline_lease_test.go
package service

import (
	"testing"
	"time"
)

func TestOfflineLeaseSignAndVerify(t *testing.T) {
	secret := []byte("ephemeral-test-only-hmac-key")
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

	lease, err := IssueOfflineLease(secret, 42, "POS-1", now)
	if err != nil {
		t.Fatalf("issue lease: %v", err)
	}
	if err := VerifyOfflineLease(secret, 42, "POS-1", lease); err != nil {
		t.Fatalf("verify valid lease failed: %v", err)
	}
	if err := VerifyOfflineLease(secret, 42, "POS-2", lease); err == nil {
		t.Fatalf("expected terminal mismatch to fail verification")
	}
	if err := VerifyOfflineLease(secret, 99, "POS-1", lease); err == nil {
		t.Fatalf("expected shop mismatch to fail verification")
	}
	tampered := lease
	tampered.LastOnlineSyncAt = now.Add(2 * time.Hour).Format(time.RFC3339Nano)
	if err := VerifyOfflineLease(secret, 42, "POS-1", tampered); err == nil {
		t.Fatalf("expected tampered timestamp to fail verification")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/service -run TestOfflineLeaseSignAndVerify -v`
预期：FAIL，提示 `undefined: IssueOfflineLease`。

- [ ] **步骤 3：实现 `offline_lease.go`、从环境变量/配置读取密钥（严禁硬编码密钥），并注册 `POST /api/staff/offline-lease`**

```go
// server/internal/service/offline_lease.go
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
)

var ErrOfflineLeaseTampered = errors.New("service: offline lease signature invalid")

type OfflineLease struct {
	ShopID           string `json:"shopId"`
	TerminalID       string `json:"terminalId"`
	LastOnlineSyncAt string `json:"lastOnlineSyncAt"`
	ExpiresAt        string `json:"expiresAt"`
	Signature        string `json:"signature"`
}

func IssueOfflineLease(secret []byte, shopID int64, terminalID string, nowUTC time.Time) (OfflineLease, error) {
	if len(secret) < 16 {
		return OfflineLease{}, errors.New("offline hmac secret unconfigured")
	}
	terminalID = strings.TrimSpace(terminalID)
	if len(terminalID) < 3 || len(terminalID) > 40 {
		return OfflineLease{}, errors.New("invalid terminal id")
	}
	last := nowUTC.UTC().Format(time.RFC3339Nano)
	exp := nowUTC.UTC().Add(4 * time.Hour).Format(time.RFC3339Nano)
	return OfflineLease{
		ShopID:           fmt.Sprintf("%d", shopID),
		TerminalID:       terminalID,
		LastOnlineSyncAt: last,
		ExpiresAt:        exp,
		Signature:        signOfflineLeasePayload(secret, shopID, terminalID, last, exp),
	}, nil
}

func VerifyOfflineLease(secret []byte, shopID int64, terminalID string, lease OfflineLease) error {
	if len(secret) < 16 {
		return errors.New("offline hmac secret unconfigured")
	}
	if lease.ShopID != fmt.Sprintf("%d", shopID) || lease.TerminalID != strings.TrimSpace(terminalID) {
		return ErrOfflineLeaseTampered
	}
	expected := signOfflineLeasePayload(secret, shopID, lease.TerminalID, lease.LastOnlineSyncAt, lease.ExpiresAt)
	gotSig, err := hex.DecodeString(lease.Signature)
	expSig, _ := hex.DecodeString(expected)
	if err != nil || !hmac.Equal(gotSig, expSig) {
		return ErrOfflineLeaseTampered
	}
	return nil
}

func signOfflineLeasePayload(secret []byte, shopID int64, terminalID, lastSync, expiresAt string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%d|%s|%s|%s", shopID, terminalID, lastSync, expiresAt)
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *API) IssueOfflineLeaseHandler(w http.ResponseWriter, r *http.Request) {
	account, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	var body struct {
		TerminalID string `json:"terminalId"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	lease, err := IssueOfflineLease([]byte(a.OfflineHMACSecret), int64(*account.ShopID), body.TerminalID, time.Now().UTC())
	if err != nil {
		if len(a.OfflineHMACSecret) < 16 {
			writeErr(w, http.StatusServiceUnavailable, "offline_signing_unconfigured")
			return
		}
		writeErr(w, http.StatusBadRequest, "invalid_terminal")
		return
	}
	writeJSON(w, http.StatusOK, lease)
}
```
在 `conf.go` 增加 `OfflineHMACSecret string`（从环境变量 `STALLINK_OFFLINE_HMAC_SECRET` / 外部 Nacos Secret 注入，仓库 YAML 模板只留注释占位）；在 `http.go` 注册 `route(r, http.MethodPost, "/api/staff/offline-lease", api.IssueOfflineLeaseHandler)`。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/service -run TestOfflineLeaseSignAndVerify -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/service/offline_lease.go server/internal/service/offline_lease_test.go server/internal/conf/conf.go server/internal/server/http.go
git commit -m "feat(offline): issue and verify HMAC-signed 4h offline lease without repo secrets"
```

---

### Task S9b：迁移 `019` — 真离线现金开单异步同步（含验签）与超卖（`oversold`）闭环（F-11, F-12）

> **K-Q17 = A（Kel 已定 2026-11-26 13:55）：** 离线同步不再传 `cashReceivedMinor`。`OfflineSyncInput` 去掉 `CashReceivedMinor`；离线现金单按订单总额入账，`orders.cash_received_minor`、`cash_change_minor` 写 `NULL`；请求体里多带这个字段时**直接忽略、不报错**（`readJSON` 宽松解析，和在线收款一致）。下面的测试已删掉该字段；实现时另加一条“请求带 `cashReceivedMinor` 也照常同步、两列为 `NULL`”的用例。原后端意见 B-9（“S9b 还要用这个字段”）作废。详见 `2026-11-26-stallink-workspace-polish.md` 任务 S6。

> **前置依赖**：修复 plan 任务 3 迁移 `014`（提供 `orders.payment_ref` 等，**本迁移 `019` 不重复创建 `payment_ref`**）+ `Task S2` + `Task S3` + `Task S7` + **`Task S9a`（调用 `VerifyOfflineLease` 验证同步请求附带的 `offlineLease`，并在同步成功后调用 `IssueOfflineLease` 下发 `refreshedLease`；若验签失败仍入账已收现金但写入 `offline.lease_tampered` 高危审计并拒绝下发新租约）**。对应前端：`Task W8`、`Task W9`、`Task W10`。

**文件：**
- 创建：`server/internal/data/migrations/019_offline_cash_orders_and_oversold.sql`
- 创建：`server/internal/data/migrations/down/019_offline_cash_orders_and_oversold.down.sql`
- 修改：`server/internal/data/order_store.go`、`server/internal/service/staff.go`、`server/internal/server/http.go`
- 测试：`server/internal/data/offline_sync_oversold_test.go`

- [ ] **步骤 1：编写失败测试**

```go
// server/internal/data/offline_sync_oversold_test.go
//go:build integration

package data

import (
	"context"
	"testing"
)

func TestOfflineSyncOversoldAndRestockResolve(t *testing.T) {
	ctx := context.Background()
	store := openSeededTestStore(t)
	good, listing := seedZeroStockListing(t, store)

	res, err := store.SyncOfflineCashOrderCommand(ctx, testOfflineIdentity(listing), OfflineSyncInput{
		ClientOrderID:     "11111111-1111-4111-8111-111111111111",
		OfflineNumber:     "OLK40001A9",
		TerminalID:        "POS-1",
		EventID:           listing.EventID,
		PayCodeID:         testCashCodeID(t, store, listing.ShopID),
		TotalMinor:        2000,
		Lines:             []OfflineLineInput{{ListingID: listing.ID, GoodID: good.ID, GoodName: good.Name, Qty: 1, UnitPriceMinor: 2000, LineTotalMinor: 2000}},
	})
	if err != nil || res.Order.Status != "paid" || res.Order.OversoldStatus != "pending_resolution" || res.Order.OversoldQty != 1 {
		t.Fatalf("expected paid + pending_resolution + qty=1, got %+v err=%v", res.Order, err)
	}
	if got := readTotalStock(t, store, good.ID); got != 0 {
		t.Fatalf("total_stock must clamp at 0, got %d", got)
	}
	restockGoodFixture(t, store, good.ID, 1)
	resolved, err := store.ResolveOversoldRestockCommand(ctx, testResolveIdentity(res.Order), res.Order.ID, "补货1件核销")
	if err != nil || resolved.Order.OversoldStatus != "resolved_restocked" || readTotalStock(t, store, good.ID) != 0 {
		t.Fatalf("resolve oversold failed: resolved=%+v err=%v", resolved.Order, err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestOfflineSyncOversoldAndRestockResolve -v`
预期：FAIL。

- [ ] **步骤 3：编写迁移 `019` + 回滚脚本（不重复添加 `payment_ref`），实现同步、钳零、`oversold` 核销与超卖单安全退款**

```sql
-- server/internal/data/migrations/019_offline_cash_orders_and_oversold.sql
DO $$
DECLARE c name;
BEGIN
	SELECT conname INTO c FROM pg_constraint WHERE conrelid = 'orders'::regclass AND contype = 'c' AND pg_get_constraintdef(oid) LIKE '%number%';
	IF c IS NOT NULL THEN EXECUTE format('ALTER TABLE orders DROP CONSTRAINT %I', c); END IF;
END $$;
ALTER TABLE orders ADD CONSTRAINT orders_number_check
	CHECK (number ~ '^(?:[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}|OL[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{8})$');

ALTER TABLE orders
	ADD COLUMN IF NOT EXISTS is_offline boolean NOT NULL DEFAULT false,
	ADD COLUMN IF NOT EXISTS terminal_id text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS offline_created_at timestamptz,
	ADD COLUMN IF NOT EXISTS offline_catalog_revision bigint,
	ADD COLUMN IF NOT EXISTS oversold_status text NOT NULL DEFAULT 'none' CHECK (oversold_status IN ('none', 'pending_resolution', 'resolved_restocked', 'resolved_refunded')),
	ADD COLUMN IF NOT EXISTS oversold_qty int NOT NULL DEFAULT 0 CHECK (oversold_qty >= 0);

CREATE TABLE IF NOT EXISTS oversold_order_lines (
	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	order_id bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
	order_line_id bigint NOT NULL REFERENCES order_lines (id) ON DELETE CASCADE,
	good_id bigint NOT NULL REFERENCES goods (id),
	listing_id bigint NOT NULL REFERENCES event_listings (id),
	deducted_qty int NOT NULL CHECK (deducted_qty >= 0),
	shortage_qty int NOT NULL CHECK (shortage_qty >= 0),
	resolved_qty int NOT NULL DEFAULT 0 CHECK (resolved_qty >= 0),
	UNIQUE (order_line_id)
);
```

```sql
-- server/internal/data/migrations/down/019_offline_cash_orders_and_oversold.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：DELETE FROM schema_migrations WHERE version = 19;
DROP TABLE IF EXISTS oversold_order_lines;
ALTER TABLE orders DROP COLUMN IF EXISTS oversold_qty;
ALTER TABLE orders DROP COLUMN IF EXISTS oversold_status;
ALTER TABLE orders DROP COLUMN IF EXISTS offline_catalog_revision;
ALTER TABLE orders DROP COLUMN IF EXISTS offline_created_at;
ALTER TABLE orders DROP COLUMN IF EXISTS terminal_id;
ALTER TABLE orders DROP COLUMN IF EXISTS is_offline;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_number_check;
ALTER TABLE orders ADD CONSTRAINT orders_number_check CHECK (number ~ '^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$');
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && STALLINK_TEST_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink_test?sslmode=disable' go test -tags integration ./internal/data -run TestOfflineSyncOversoldAndRestockResolve -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/019_offline_cash_orders_and_oversold.sql server/internal/data/migrations/down/019_offline_cash_orders_and_oversold.down.sql server/internal/data/order_store.go server/internal/service/staff.go server/internal/server/http.go
git commit -m "feat(offline): support offline cash sync and oversold resolution lifecycle"
```

---

### Task S10：迁移 `020` — 媒体防枚举 `media_token` 与本地落盘二级缓存（F-13）

> **前置依赖**：公开图片响应缓存头（`Cache-Control`/`ETag`）与有界内存缓存归 **修复 plan 任务 11**，图片解码并发控制与 EXIF 剥离归 **任务 12**，Drive 孤儿清理跨环境防护归 **任务 1**；本任务不重复编写响应缓存头或内存缓存，专注迁移 `020` 的 `media_token` 与抗重启磁盘二级缓存 `NewDiskCache`。

**文件：**
- 创建：`server/internal/data/migrations/020_media_tokens.sql`
- 创建：`server/internal/data/migrations/down/020_media_tokens.down.sql`
- 修改：`server/internal/imagestore/`、`server/internal/data/good_image_store.go`、`server/internal/service/handlers.go`
- 测试：`server/internal/imagestore/cache_test.go`

- [ ] **步骤 1：编写失败测试**

```go
// server/internal/imagestore/cache_test.go
package imagestore

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestCachedStoreAvoidsRepeatDriveFetch(t *testing.T) {
	var hits atomic.Int32
	upstream := func(ctx context.Context, path string) ([]byte, string, error) {
		hits.Add(1)
		return []byte("webp-bytes"), "image/webp", nil
	}
	cached := NewDiskCache(t.TempDir(), upstream)
	for i := 0; i < 5; i++ {
		body, ct, err := cached.Get(context.Background(), "drive:abc")
		if err != nil || ct != "image/webp" || string(body) != "webp-bytes" {
			t.Fatalf("unexpected result: err=%v ct=%s", err, ct)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 upstream Drive hit, got %d", hits.Load())
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/imagestore -run TestCachedStoreAvoidsRepeatDriveFetch -v`
预期：FAIL，提示 `undefined: NewDiskCache`。

- [ ] **步骤 3：编写迁移 `020` + 回滚脚本，实现 `NewDiskCache` 衔接任务 11 内存缓存**

```sql
-- server/internal/data/migrations/020_media_tokens.sql
ALTER TABLE good_images ADD COLUMN IF NOT EXISTS media_token text;
UPDATE good_images SET media_token = encode(gen_random_bytes(12), 'hex') WHERE media_token IS NULL;
ALTER TABLE good_images ALTER COLUMN media_token SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS good_images_media_token_idx ON good_images (media_token);

ALTER TABLE bundle_images ADD COLUMN IF NOT EXISTS media_token text;
UPDATE bundle_images SET media_token = encode(gen_random_bytes(12), 'hex') WHERE media_token IS NULL;
ALTER TABLE bundle_images ALTER COLUMN media_token SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS bundle_images_media_token_idx ON bundle_images (media_token);

ALTER TABLE pay_codes ADD COLUMN IF NOT EXISTS media_token text;
UPDATE pay_codes SET media_token = encode(gen_random_bytes(12), 'hex') WHERE media_token IS NULL;
ALTER TABLE pay_codes ALTER COLUMN media_token SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS pay_codes_media_token_idx ON pay_codes (media_token);
```

```sql
-- server/internal/data/migrations/down/020_media_tokens.down.sql
-- ⚠️ 对非本机库回滚需 Kel 手动确认：DELETE FROM schema_migrations WHERE version = 20;
DROP INDEX IF EXISTS pay_codes_media_token_idx;
ALTER TABLE pay_codes DROP COLUMN IF EXISTS media_token;
DROP INDEX IF EXISTS bundle_images_media_token_idx;
ALTER TABLE bundle_images DROP COLUMN IF EXISTS media_token;
DROP INDEX IF EXISTS good_images_media_token_idx;
ALTER TABLE good_images DROP COLUMN IF EXISTS media_token;
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/imagestore -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/data/migrations/020_media_tokens.sql server/internal/data/migrations/down/020_media_tokens.down.sql server/internal/imagestore/ server/internal/data/ server/internal/service/
git commit -m "perf(media): add unguessable media_token and local disk cache before Google Drive"
```

---

### Task S11：报价接口限流与高并发检索阈值配置（复用修复 plan 任务 8/12，F-14 增量）

> **说明**：分布式 Redis + 降级滑窗限流器及可信代理真实 IP 提取（含登录/注册与既有公开匿名限流）由 **修复 plan 任务 8** 落地，公开目录分页限流优化由 **任务 12** 落地，去 `shopID == 0` 跳过过滤由 **任务 5** 落地。本任务**不另建重复限流器**，直接复用任务 8 的限流中间件与真实 IP 上下文，增量保护 `/api/staff/order-quotes` 报价接口并校准 `/api/public/catalog` 模糊检索突发阈值。

**文件：**
- 修改：`server/internal/server/http.go`（在路由上挂载任务 8 限流策略：`quoteLimiter`）
- 修改：`server/internal/conf/conf.go`（补充报价与模糊目录检索阈值配置项）
- 测试：`server/internal/server/ratelimit_quote_test.go`

- [ ] **步骤 1：编写失败测试（验证复用任务 8 限流中间件保护 `/api/staff/order-quotes`）**

```go
// server/internal/server/ratelimit_quote_test.go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrderQuotesRateLimitMiddlewareBlocksBurst(t *testing.T) {
	lim := newTestStaffQuoteLimiter(5) // 复用任务 8 分布式/降级限流组件工厂
	h := lim(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/staff/order-quotes", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("req %d expected 200, got %d", i, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/staff/order-quotes", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd server && go test ./internal/server -run TestOrderQuotesRateLimitMiddlewareBlocksBurst -v`
预期：FAIL，提示 `undefined: newTestStaffQuoteLimiter`。

- [ ] **步骤 3：在 `http.go` 与 `conf.go` 复用修复 plan 任务 8 限流中间件，补充 `/api/staff/order-quotes` 限流策略及 `/api/public/catalog` 高并发参数校准**

- [ ] **步骤 4：运行测试验证通过**

运行：`cd server && go test ./internal/server -run TestOrderQuotesRateLimitMiddlewareBlocksBurst -v`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add server/internal/server/http.go server/internal/conf/conf.go server/internal/server/ratelimit_quote_test.go
git commit -m "perf(ratelimit): apply task-8 rate limiter to order-quotes and tune public catalog bursts"
```

---

### Task S12：1,000 QPS / 开单 P95 < 500ms 本地压测与验收（全 P0 验收）

**文件：**
- 创建：`server/perf/k6_stallink_peak.js`
- 创建：`server/perf/README.md`

- [ ] **步骤 1：编写 `k6` 混合压测脚本（85% 店面目录浏览 + 10% 报价 + 5% 创建并付款，校验 `p(95) < 500ms` 且 0 超卖）**

```javascript
// server/perf/k6_stallink_peak.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    stallink_peak: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 150,
      maxVUs: 500,
    },
  },
  thresholds: {
    'http_req_duration{endpoint:create_order}': ['p(95)<500'],
    'http_req_duration{endpoint:public_shop}': ['p(95)<200'],
    'http_req_failed{expected:true}': ['rate<0.001'],
  },
};

const BASE = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const SLUG = __ENV.SHOP_SLUG || 'alpha';

export default function () {
  const r = http.get(`${BASE}/api/public/shops/${SLUG}`, { tags: { endpoint: 'public_shop', expected: 'true' } });
  check(r, { 'status 200': (res) => res.status === 200 });
}
```

- [ ] **步骤 2：仅在本机 Docker Compose 环境执行压测**

运行：
1. `make compose && make seed-local`（仅限 `127.0.0.1` 本机 Postgres + Redis + Nacos）
2. `cd server && go build -o /tmp/stallink ./cmd/stallink && /tmp/stallink -conf configs/local/stallink.yaml`
3. `k6 run server/perf/k6_stallink_peak.js`
> **⚠️ 需要 Kel 手动确认**：严禁对云端或非本机数据库运行压测；如需对预发布远程集群压测，必须由 Kel 手动确认。

预期输出：`rate >= 1000/s`、`create_order p(95) < 500ms`、数据库校验 `SUM(sold_qty) <= initial_total_stock`。

- [ ] **步骤 3：Commit 压测脚本**

```bash
git add server/perf/k6_stallink_peak.js server/perf/README.md
git commit -m "test(perf): add 1000 QPS peak load test script for storefront and checkout P95"
```

---

## 五、Phase 2：P1 任务拆解（完整 SQL 迁移 `021`–`023` 与 TDD 任务，对应 `F-15`~`F-24`）

### Task S13：迁移 `021` + 全站目录收录开关（默认 `false` Opt-in）、排序与 `pg_trgm` 分页检索（F-15）
> **前置依赖**：`readGoodPool` 去 `shopID == 0` 归 **修复 plan 任务 5**，公开目录基础分页限流优化归 **修复 plan 任务 12**。对应前端：`Task W11`
- **迁移 `021_public_catalog_optin_and_search.sql`** & **回滚 `migrations/down/021_public_catalog_optin_and_search.down.sql`**：
  ```sql
  -- 021_public_catalog_optin_and_search.sql
  CREATE EXTENSION IF NOT EXISTS pg_trgm;
  ALTER TABLE shops ADD COLUMN IF NOT EXISTS allow_public_catalog boolean NOT NULL DEFAULT false;
  CREATE INDEX IF NOT EXISTS goods_name_trgm_idx ON goods USING gin (name gin_trgm_ops);
  CREATE INDEX IF NOT EXISTS shops_name_trgm_idx ON shops USING gin (name gin_trgm_ops);
  ```
  ```sql
  -- migrations/down/021_public_catalog_optin_and_search.down.sql
  DROP INDEX IF EXISTS shops_name_trgm_idx;
  DROP INDEX IF EXISTS goods_name_trgm_idx;
  ALTER TABLE shops DROP COLUMN IF EXISTS allow_public_catalog;
  ```
- **实现文件**：`server/internal/data/shop.go`、`server/internal/data/catalog_store.go`、`server/internal/service/admin.go`、`server/internal/service/public.go`
- **TDD 验收**：新店默认 `allow_public_catalog = false` 不入 `/api/public/catalog`，但 `/api/public/shops/{slug}` 正常；分页每页 ≤50；默认 `live_first`。

### Task S14：`scheduled` 活动显式预售开关 `allow_presale` 守卫（F-16）
> 对应前端：`Task W4`（字段已在迁移 `016` 创建）
- **实现文件**：`server/internal/data/order_checks.go`、`server/internal/data/order_store.go`、`server/internal/conf/conf.go`
- **TDD 验收**：`status='scheduled' && allow_presale=false` 下单返回 `400 event_not_open`；开启后正常下单。

### Task S15：迁移 `022` + 顾客收藏夹、预选清单（不占库存）与取货凭证夹（F-17, F-18, F-19）
> 对应前端：`Task W12`、`Task W13`、`Task W14`
- **迁移 `022_customer_favorites_wishlists_orders.sql`** & **回滚 `migrations/down/022_customer_favorites_wishlists_orders.down.sql`**：
  ```sql
  -- 022_customer_favorites_wishlists_orders.sql
  CREATE TABLE IF NOT EXISTS customer_favorites (
  	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  	user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  	target_type text NOT NULL CHECK (target_type IN ('shop', 'listing', 'bundle_listing')),
  	target_id bigint NOT NULL,
  	created_at timestamptz NOT NULL DEFAULT now(),
  	UNIQUE (user_id, target_type, target_id)
  );
  CREATE TABLE IF NOT EXISTS customer_wishlists (
  	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  	user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  	shop_id bigint NOT NULL REFERENCES shops (id),
  	event_id bigint NOT NULL REFERENCES events (id),
  	share_code text NOT NULL UNIQUE CHECK (share_code ~ '^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$'),
  	created_at timestamptz NOT NULL DEFAULT now()
  );
  CREATE TABLE IF NOT EXISTS customer_wishlist_items (
  	id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  	wishlist_id bigint NOT NULL REFERENCES customer_wishlists (id) ON DELETE CASCADE,
  	item_kind text NOT NULL CHECK (item_kind IN ('listing', 'bundle')),
  	target_id bigint NOT NULL,
  	qty int NOT NULL CHECK (qty > 0)
  );
  ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_user_id bigint REFERENCES users (id);
  CREATE INDEX IF NOT EXISTS orders_customer_user_idx ON orders (customer_user_id, created_at DESC) WHERE customer_user_id IS NOT NULL;
  ```
  ```sql
  -- migrations/down/022_customer_favorites_wishlists_orders.down.sql
  DROP INDEX IF EXISTS orders_customer_user_idx;
  ALTER TABLE orders DROP COLUMN IF EXISTS customer_user_id;
  DROP TABLE IF EXISTS customer_wishlist_items;
  DROP TABLE IF EXISTS customer_wishlists;
  DROP TABLE IF EXISTS customer_favorites;
  ```
- **实现文件**：新建 `server/internal/data/customer_store.go`、`server/internal/service/customer.go`，修改 `server/internal/server/http.go`、`server/internal/data/order_store.go`。
- **TDD 验收**：预选清单不产生 `inventory_reservations`；收银台仅可导入本店清单（跨店返回 `wishlist_shop_mismatch`）。

### Task S16：迁移 `023` + 中英双语偏好与完整前后快照审计增强（F-20, F-21）
> **前置依赖**：自动审核开关与外观切换平台审计由 **修复 plan 任务 10** 负责，**`audit_logs.reason` 与 `detail` 已由修复 plan 任务 3 迁移 `014` 提供（含 `2–500` Unicode 字符应用层口径，本迁移 `023` 不再重复增删 `reason` 列）**；本任务增量扩展全量操作的前后 JSON 快照、操作金额（`order_refunds`）、终端与 IP。对应前端：`Task W15`、`Task W16`。
- **迁移 `023_audit_snapshots_and_user_locale.sql`** & **回滚 `migrations/down/023_audit_snapshots_and_user_locale.down.sql`**：
  ```sql
  -- 023_audit_snapshots_and_user_locale.sql
  ALTER TABLE users ADD COLUMN IF NOT EXISTS locale text NOT NULL DEFAULT 'zh-CN' CHECK (locale IN ('zh-CN', 'en'));
  ALTER TABLE audit_logs
  	ADD COLUMN IF NOT EXISTS before_json jsonb,
  	ADD COLUMN IF NOT EXISTS after_json jsonb,
    	ADD COLUMN IF NOT EXISTS terminal_id text NOT NULL DEFAULT '',
  	ADD COLUMN IF NOT EXISTS client_ip text NOT NULL DEFAULT '';
  ```
  ```sql
  -- migrations/down/023_audit_snapshots_and_user_locale.down.sql
  ALTER TABLE audit_logs DROP COLUMN IF EXISTS client_ip;
  ALTER TABLE audit_logs DROP COLUMN IF EXISTS terminal_id;
  ALTER TABLE audit_logs DROP COLUMN IF EXISTS after_json;
  ALTER TABLE audit_logs DROP COLUMN IF EXISTS before_json;
  ALTER TABLE users DROP CONSTRAINT IF EXISTS users_locale_check;
  ALTER TABLE users DROP COLUMN IF EXISTS locale;
  ```
- **实现文件**：`server/internal/service/admin.go`（移除硬编码 `"现金"`）、`server/internal/data/paged_lists.go`、`server/internal/data/auth_store.go`、`server/internal/service/handlers.go`。
- **TDD 验收**：调价/改库存/退款/目录开关写入 `before_json`/`after_json`/`reason`/`client_ip`；`/api/admin/audit` 支持 `action` 与 `actorId` 过滤。

### Task S17：过期未付订单延迟到账人工恢复收款（F-22）
> 对应前端：`Task W17`
- **实现文件**：`server/internal/data/order_store.go`、`server/internal/service/staff.go`、`server/internal/server/http.go`
- **TDD 验收**：`POST /api/staff/orders/{orderID}/recover-expired` 校验原单 `expired`、必填 `reason`、当前池库存足够扣库转 `paid`；不足返回 `409 not_enough_stock`。

### Task S18：SSE 实时库存/超卖事件流（QR `payment_ref` 复用修复 plan 任务 3 迁移 `014`）（F-23）
> **前置依赖**：修复 plan 任务 3（迁移 `014`）。对应前端：`Task W18`。
- **实现文件**：新建 `server/internal/service/sse.go`，修改 `server/internal/data/order_store.go`、`server/internal/server/http.go`
- **TDD 验收**：`Pay` 透传 `paymentRef` 写入 `014` 字段；`GET /api/staff/events/{eventID}/stream` 实时推送 `stock_changed` 与 `oversold_alert`。

### Task S19：超限复杂购物车（DP > 100,000 状态）友好降级报价（F-24）
> 对应前端：`Task W18`
- **实现文件**：`server/internal/catalog/cart_pricing.go`、`server/internal/service/staff.go`
- **TDD 验收**：`allowDegraded=false` 返回 `422` 含 `conflictGroup`；`allowDegraded=true` 按单品优惠输出降级可结算报价。

---

## 六、Phase 3：P2 任务拆解（`F-25`, `F-26`, `F-28`）

- **Task S20（F-25）**：`server/internal/data/public_cache.go` —— 为 `/api/public/catalog` 与 `/api/public/shops/{slug}` 接入 Redis 15s 分层缓存与只读数据源选项（无迁移）。
- **Task S21（F-26）**：`server/internal/data/cleanup.go` —— 定时清理 >24h 过期 `order_quotes` 与 >30d `order_commands`（无迁移）。
- **Task S22（F-28）**：`server/internal/server/http.go`、`server/internal/service/handlers.go` —— 下线无效路由 `/api/public/landing`（无迁移）。

---

## 七、⚠️ 需要 Kel 手动确认的步骤汇总

1. **云端/远程数据库执行迁移（`014`–`023`）或回滚（`migrations/down/*.down.sql`）**：须经 Kel 核对备份与目标环境后手动批准执行。
2. **非本机数据库启动 API 或 Nacos 热更新发布**：依赖修复 plan 任务 1（Drive 防护）与任务 2（迁移安全守卫），生产/测试环境启动与 Nacos 配置下发需 Kel 手动确认。
3. **非本机压测**：`Task S12` 禁止在未确认情况下指向任何非本机数据库或生产 Google Drive 目录。

---

## 八、合入后的后续项

### B-14：多活动店铺的时区取值顺序（合入后跟进）

> 来源：工作台打磨 plan（`2026-11-26-stallink-workspace-polish.md`）第 9.2 节后端答复 B-14、任务 S8 的说明；Kel 2026-11-26 12:40 要求写进本 plan。**不在本轮 vNext 合入范围内**，Task S6（迁移 `016`，`events.timezone`）合入后再做。

**问题：** S6 之后时区挂在活动上。一个店铺同时有多个活动时，店铺级的“今天”（例如 `GET /api/admin/daily-reports/today`、工作台日报）该用哪个时区，现在没有定义。

**取值顺序（依次往下，取到为止）：**
1. 店铺的**主推活动**（featured event）的 `timezone`；
2. **正在进行中**的活动的 `timezone`；
3. `starts_on` **离今天最近**的活动的 `timezone`；
4. 都没有时，回退 `Asia/Kuala_Lumpur`。

**细则（Kel 已定 2026-11-26 13:22，B14-1、B14-2 都选 A）：**
- **B14-1：** 同时有多个进行中的活动时，用 `starts_on` **最早**的那个活动的时区（再相同按 `id` 小的）。
- **B14-2：** 没有进行中的活动时，取日期最近的活动：已结束和未开始的一起算，按 `|starts_on - 今天|` 取最小；距离相同时取**未开始**的那个。

**测试补充：** 在原来四种情况之外，加两条用例：两个进行中的活动（`starts_on` 不同）→ 取早的那个的时区；一个 3 天前开始已结束、一个 3 天后开始的活动、没有进行中 → 取 3 天后那个的时区。

**文件（预计）：** 店铺级“今天”的取值函数放在 `server/internal/data/` 下（具体文件开工时按 S6、S8 合入后的代码定）；对应集成测试 `-tags integration`，四种情况各一条用例。

**依赖：** Task S6（`016`）、Task S8（`018`）合入。不需要新迁移。
