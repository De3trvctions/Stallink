# Stallink 安全、性能与审核遗留问题修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 `docs/reports/stallink-commerce-review-2026-11-22.md` 中仍未修复的审核项，以及 `docs/reports/stallink-security-perf-review-2026-11-26.md` 中的 C1、H1、H3、H4（不动表结构部分）、全部中级项（M1–M7）合并去重，写成可直接按 TDD 执行的后端、迁移、前端与 CI 修复计划。

**架构：** 保持 Stallink 既有分层（`cmd/stallink` 启动与命令、`internal/server` HTTP 路由、`internal/service` 适配与策略、`internal/data` 事务持久化、`internal/imagestore` 媒体编解码与存储、`web/` Next.js 16）不变；以迁移 `014` 一次性落地支付凭据、结构化/平台审计与关键索引，回滚 SQL 隔离在 `migrations/down/`；在不动已有数据库主表结构的前提下落地图片内存缓存、目录分页缓存与安全防护，并与并行 `vNext` 计划（迁移 `015` 起）严格划分协作边界。

**技术栈：** Go 1.27.4+、Kratos v2、pgx/v5、PostgreSQL 16、go-redis/v9、`golang.org/x/image` v0.45.0+、`golang.org/x/sync/singleflight`、Next.js 16.4.6、React 19、TypeScript 5、GitHub Actions

---

## 一、硬性约束与边界说明

1. **数据库安全红线：** 凡涉及云端或远程数据库的步骤均标注 **`⚠️ 需要 Kel 手动确认`**；本地集成测试只允许连接 `docker-compose.yml` 起的本机 Postgres（`127.0.0.1:5432`）与 Redis（`127.0.0.1:6379`），通过 `STALLINK_TEST_DATABASE_URL` 显式注入隔离 schema。
2. **迁移编号协同：** 仓库当前最大迁移为 `013_shop_theme.sql`。并行 `vNext` 计划的迁移从 `015` 开始，**`014` 专门留给本计划**。
3. **开发规范：** 修改 Go 代码前按 `AGENTS.md` 加载 `samber/cc-skills-golang@golang-how-to` 并遵守 `.cursor/rules/server-go.mdc`；修改前端前按 `web/AGENTS.md` 核对 `web/node_modules/next/dist/docs/` 并遵守 `.cursor/rules/web-next.mdc`；严禁把真实密码、Token、Client Secret 写入任何文件。
4. **不越权提交：** 本计划仅指导研发实现；实际开发阶段在 Kel 确认前不得对远程库跑迁移或推送分支。

---

## 二、已拍板决策与待确认问题

### 2.1 已拍板项

- **C1（P0 第一项，2026-11-26 Kel 拍板）：各环境与本机共用同一个 Drive 根目录。**
  - 已确认现网、预发与本机开发环境共用同一 Google Drive 根目录，具备本机/空库运行 `runBlobSweeper` 将线上图片移入垃圾桶的最高风险，因此 **C1 定为 P0 第一项，排在自动迁移防护之前**，并**不再列为待确认问题**。
  - 代码改造严格对接报告第 10.4 节草案并补齐长期环境隔离，含四项强制防护（每项均按 TDD 写测试）：① `storage.orphan_cleanup_enabled` 默认 `false` 且支持 Nacos 热更新实时启停（绝不触发迁移）；② 参照集合为空时拒绝删除（`ErrEmptyReferenceSet`）；③ 单轮删除设上限 50（`ErrOrphanSweepAborted`）；④ 每个环境使用独立 Drive 根目录与环境标识（`storage.environment`），启动时强制校验。
  - **上线与合并顺序硬约束**：**C1（任务 1）必须与 H1 自动迁移守卫（任务 2）一起合入并同步上线**，避免因发版重启或热更新触发未受控的自动迁移。
  - **各环境先暂停清理属于运维动作，具体操作由安全与性能检查同事整理，不写进本 plan**；已有数据迁移至各环境独立根目录步骤标注 **`⚠️ 需要 Kel 手动确认`**。

- **P1-1 取消、一步取消并退款与两步退款口径（2026-11-26 Kel 拍板）：**
  - 维持 `Q3=B`：允许**订单所属店铺**的 `sales`（销售员）或 `shop_admin`（店长）直接取消已付款订单并确认退款；跨店一律返回 **`403 forbidden`**。
  - **统一原因校验（与 `vNext` 全部对齐）**：`cancel`、`refund-confirm` 与新的一步接口 `cancel-refund` 均接收 `{ "reason": "..." }`，先 `TrimSpace` 再按 `utf8.RuneCountInString`（Unicode 字符数）校验 **`2..500` 字符**：空串/空白返回 **`422 reason_required`**，仅 `1` 个字符返回 **`422 reason_too_short`**，超过 `500` 返回 **`422 reason_too_long`**；`reason` 计入 `order_commands.request_hash`（同幂等键不同 `reason` 返回 **`409 idempotency_conflict`**），并写入 `audit_logs.reason` 与 `audit_logs.detail`。
  - **未收款（`pending`）订单取消也强制填写原因（Kel 11-26 已拍板）**：与 `paid`/`fulfilled` 统一校验 `2..500` 字符并记录审计。
  - **新增一步完成接口 `POST /api/staff/orders/{orderID}/cancel-refund`（Kel 11-26 已拍板）**：保留现有两步流程（`cancel` → `refund_required` → `refund-confirm`）以兼容存量 `refund_required` 订单；前端对已收款订单默认只提供「取消并退款」，仅做全额退款且全部货品回库（部分退款与报损仍走 `vNext Task S7`）。

### 2.2 自动迁移防护方案对比与推荐（H1 / P0 第二项）

现状核实：`server/cmd/stallink/main.go:369-379` 在 `prepareRuntime` 中只要 `previous == nil`（首次启动）或 `previous.database != bc.Data.Database`（Nacos 热更新）就会直接执行 `data.Migrate`，不检查是否本机库，且 `localBootstrap`（`main.go:106-116`）也只检查 `ConnConfig.Host`，存在 `pgx` 多主机 `Fallbacks` 绕过风险。

| 方案 | 机制 | 优点 | 缺点 |
|---|---|---|---|
| ① DSN 非本机拒绝自动迁移 | 解析主主机及全部 `Fallbacks`，仅允许 `localhost` / `127.0.0.1` / `::1`（与 `stallink` 库）自动迁移 | 不改 Nacos YAML 也能拦住直连云端误操作 | 仅靠 DSN 判断仍将 DDL 与服务启动耦合 |
| ② 显式开关默认关 | `conf.Database` 新增 `auto_migrate`（默认 `false`），只有显式开启才跑迁移 | 显式可控，生产与启动彻底解耦 | 若模板误写 `true` 单开关仍可能误触远端库 |
| **推荐方案（②+①组合 + 彻底切断热更新）** | 1. **热更新（`previous != nil`）永不迁移**，DSN 变更只做只读待迁移检查；2. `auto_migrate` 默认 `false`；3. 即使开启也必须通过含 `Fallbacks` 的本地 DSN 校验，否则拒绝启动；4. 启动未自动迁移时执行只读 `PendingMigrations`，有待执行版本则拒绝启动并提示 `stallink migrate`；5. 新增显式 `stallink migrate` 子命令 | 同时封堵启动误连、热更新连锁、多主机 Fallbacks 绕过与配置误开四重风险 | 需同步在 `server/Makefile` 增加 `migrate` 目标 |

### 2.3 待确认问题（需 Kel 拍板）

- **问题 1（对应 M4，人工权限核查）：Google Drive 根目录与收款码共享权限确认**
  - **背景**：`drive.go:67-106` 上传图片不设置文件级权限，完全继承根目录共享设置；且商品图与各店收款码目前同处一个根目录。
  - **需确认**：在为各环境新建独立 Drive 根目录时，请 Kel 在 Google Drive 网页确认每个环境根目录的共享设置为「受限（Restricted）」，仅对应环境服务账号/授权 OAuth 账号可访问，非「知道链接的任何人可读」；并拍板是否需要为收款码再拆独立子目录/根目录。

- **问题 2（对应任务 7 / P1-1）：是否在已收款（`paid`/`fulfilled`）订单上禁用两步的 `cancel` 接口，以及后续何时彻底废弃两步流程？**
  - **背景**：目前已决定新增一步接口 `POST /api/staff/orders/{orderID}/cancel-refund`，并暂时保留两步接口（`cancel` → `refund_required` → `refund-confirm`）以兼容存量 `refund_required` 订单；前端对已收款订单默认只提供「取消并退款」。
  - **两种做法对前后端的影响**：
    - `a.` **暂不禁掉已收款调 `cancel`，仅前端隐藏入口（本计划默认过渡方案）**：**后端**兼容性最好，旧端/重试队列中的 `cancel` 仍能进入 `refund_required` 并由 `refund-confirm` 收尾；但需要保留 `cancelOrderTx` 中 `paid`/`fulfilled` 分支与两套测试用例。
    - `b.` **在已收款订单上直接禁止调 `cancel`（只允许 `pending` 调 `cancel`，已收款必须走 `cancel-refund`），并在存量 `refund_required` 清零后下线两步流程**：**后端**状态机更简洁，彻底消灭新产生 `refund_required` 的入口（对 `paid`/`fulfilled` 调 `cancel` 返 `409 invalid_status`），但老版前端缓存或未同步客户端若仍调 `/cancel` 会直接报错；完全废弃两步流程前必须先确认线上库已无 `status = 'refund_required'` 订单且所有客户端已升级。
- **问题 3（对应 `vNext S7` / `W6`）：`vNext Task S7` 部分退款接口的 `reason` 长度最终锁定 `2–500` 还是另有调整？**
  - **说明**：当前已按 Kel 11-26 统一要求把任务 7、迁移 `014`、`S7`、`S16`、`W6`、`feature-list` 与 `PRD` 全部同步为 **2–500 Unicode 字符**，请确认后续 `S7` 实施时无需再改回其他范围。

---

## 三、双报告全量核对与任务映射表

### 3.1 `docs/reports/stallink-security-perf-review-2026-11-26.md` 对照表

| 安全报告编号 | 问题摘要 | 对应本计划任务 | 合并 / 不纳入说明 |
|---|---|---|---|
| **C1（含第 10.4 节）** | Drive 孤儿清理跨环境误删（`orphan_cleanup_enabled` 热开关/空引用/上限 50/独立根目录） | **任务 1（P0 第一项）** | 须与 **任务 2（H1）** 一起合并上线；合并 **L3**（补齐 `shops/events.cover_path` 引用） |
| **第 10.2 / 10.3 / 10.5 节** | 本机止血、非本机临时降权/切空目录与 Drive 垃圾桶 30 天手工还原步骤 | **不纳入本 plan** | 均属人工止血与运维恢复操作，由安全与性能检查同事整理，不写进代码开发计划 |
| **第 8.2 节附带发现** | `migrations_integration_test.go` `defer adminPool.Close()` 先于 `t.Cleanup` 导致测试 schema 残留 | **任务 16** | 新增小修复放于计划末尾，保持既有任务 `1–15` 编号不变，避免影响 `vNext` 引用 |
| **H1** | 启动与 Nacos 热更新自动迁移任意库 | **任务 2（P0 第二项）** | 与 11-22 新增 P0 合并，采用报告 6.4 节草案并修正 `Fallbacks` 检查 |
| **H2** | 本机 Nacos 缓存明文密钥 & OAuth `cloud-platform` 权限过宽 | **不纳入本 plan** | 属运维与密钥轮换操作，按 Kel 指示不写入开发计划（建议 Kel 按报告第 3 节手动轮换并缩窄 scope 至 `drive.file`） |
| **H3** | Go `1.27.1` 与 `x/image v0.23.0` 15 个可达漏洞 | **任务 4**（+ **任务 15** CI） | 升级至 Go `>= 1.27.4`、`x/image >= v0.45.0`，并在 CI 固定 `govulncheck` |
| **H4** | 公开图片无缓存头、每次穿透 Drive | **任务 11**（+ **任务 8** 限流） | **与 vNext `Task S10`（迁移 `020`，需求说明中称 S11/020 重叠边界）明确分工**：本计划只做不动表结构的响应缓存头（`Cache-Control` + `ETag`/`304`）与 Go 内存 LRU + `singleflight`；防枚举 `media_token` 与磁盘二级缓存留给 `S10`/`020`；匿名 IP 限流合并至 **任务 8** |
| **M1** | 公开目录检索不分页、无上限、无缓存、SubPlan 扫表 | **任务 12**（+ **任务 3** 索引 + **任务 8** 限流） | 纳入不动主表结构的 `LIMIT 48`、`q` 截断 64、去 `description`、JOIN 聚合、10s 内存缓存与 `event_listings_good_shop_idx` 索引；`allow_public_catalog` 与 `pg_trgm` 留给 vNext `Task S13`（迁移 `021`） |
| **M2** | 图片上传解码内存/CPU 无上限 | **任务 12** | 纳入（全局解码信号量并发 2、`maxPixels` 降为 1600 万、缩放最多 4 轮、PNG 用 `DefaultCompression`） |
| **M3** | 原图直传保留 EXIF/GPS | **任务 12** | 与 M2 合并在 `imagestore/compress.go`：JPEG/收款码一律经重编码剥离 EXIF |
| **M4** | Drive 权限继承根目录、收款码与商品图同目录 | **任务 1** + **待确认问题 1** | 仓库模板去除硬编码根目录 ID 改为环境变量占位符；各环境新根目录共享权限需 Kel 人工核实 |
| **M5** | 三套主题 CSS 在全站所有路由加载 | **任务 14** | 纳入前端任务：工作台 CSS 移至 `WorkspaceShell`，公开主题 CSS 按需加载 |
| **M6** | 外观接口每次 SSR 打库、切换无审计 | **任务 10** | 审计与 **P1-4** 合并写入 `audit_logs`；缓存加在 Go 端内存原子缓存 + 变更即时失效 |
| **M7** | 迁移规范：无锁超时、无 checksum、无回滚脚本 | **任务 2** + **任务 3** + **任务 15** | 合并进迁移器（`lock_timeout=5s`、`statement_timeout=60s`、`checksum` 校验）、`migrations/down/` 隔离与 CI 校验 |
| **L1** | `docker-compose.yml` Nacos 默认 token/口令 | **不纳入**（信息记录） | 仅限 `127.0.0.1` 本机开发，若后续上服务器再改 `.env` |
| **L2** | Next.js `route.ts` 透传伪造 `X-Forwarded-*` | **任务 8** | 与登录/注册限流的可信代理 IP 提取一并修复，清洗未受信任的 `x-forwarded-*` |
| **L3** | `blob_cleanup.go` 未查 `shops/events.cover_path` | **任务 1** | 随 C1 引用集合加固一并补全，防止未来启用封面图被误删 |
| **L4–L6** | gosec 低风险项 / npm dev 依赖 / 客户端工作台守卫 | **不纳入** | 均为误报或开发态低风险项，无需在本轮阻塞修改 |

### 3.2 `docs/reports/stallink-commerce-review-2026-11-22.md` 其余未修项对照表

| 11-22 编号 | 问题摘要 | 对应本计划任务 | 处置说明 |
|---|---|---|---|
| **P1-1** | 取消、一步取消并退款与两步退款权限/原因/审计 | **任务 7**（+ **任务 3、13**） | **2026-11-26 Kel 拍板**：新增一步 `cancel-refund`（保留两步兼容存量）、`pending` 取消也必填、`reason` 统一 `2..500` Unicode 字符、同店 `sales`/`shop_admin`、跨店 403 |
| **P1-2** | 登录/注册无限流 | **任务 8** | Redis + 内存兜底，按 IP + `loginName` 双维限流返回 `429` + `Retry-After` |
| **P1-3** | 订单缺支付凭据与日报应收/实收差额 | **任务 3、9、13** | 迁移 `014` 补齐字段，日报增应收/实收差额，Web 收银台增 QR 参考号输入 |
| **P1-4** | 自动审核开关变更无审计 | **任务 3、10** | 与 M6 合并，写入平台级审计并在写审计失败时回滚 Redis |
| **P2-1** | 过期预占未实时剔除 | **不纳入（交由 vNext `S2`）** | vNext `Task S2` 已完整覆盖，避免与并行计划重复改同一批库存 SQL |
| **P2-2** | 释放预占未走全局锁序 | **任务 6** | 纳入：`releaseOrderReservations` 改为先锁 `goods` 再锁 `event_listings` |
| **P2-3** | `readGoodPool` `shopID==0` 跳过店铺过滤 | **任务 5** | 纳入：强制校验 `shopID > 0`，`applyStock` 显式传入 `shopID` |
| **P2-4** | 缺 CSP / HSTS | **任务 14** | 纳入：Next.js 16 `web/src/proxy.ts` Nonce CSP + 条件式 HSTS |
| **P2-5** | 前端多余 `bundleTotal` | **任务 14** | 纳入：直接删除 `web/src/lib/pricing.ts:25-51` |
| **P2-6** | 无 CI | **任务 15** | 纳入：新增 `.github/workflows/ci.yml` |
| **P3-a/b/c** | 会话轮换/登录登出审计/零库存语义区分 | **不纳入** | P3-a/b 延后 P3 专项；P3-c 交由 vNext 统一处理 |

## 四、文件结构总览

- **Go 配置与启动**：`server/go.mod`、`server/Makefile`、`server/configs/nacos/stallink.yaml`、`server/internal/conf/conf.go`、`server/cmd/stallink/main.go`、`server/cmd/stallink/migrate_guard.go`（新增）、`server/cmd/stallink/sweeper_test.go`（新增）
- **数据库迁移与存储**：`server/internal/data/migrations.go`、`server/internal/data/migrations/014_order_payment_reference_and_audit.sql`（新增）、`server/internal/data/migrations/down/014_order_payment_reference_and_audit.down.sql`（新增）、`server/internal/data/shop.go`、`server/internal/data/catalog_store.go`、`server/internal/data/reservation_store.go`、`server/internal/data/order_store.go`、`server/internal/data/paged_lists.go`、`server/internal/data/report_store.go`、`server/internal/data/platform_audit.go`（新增）、`server/internal/data/blob_cleanup.go`、`server/internal/data/migrations_integration_test.go`
- **媒体与限流**：`server/internal/imagestore/drive.go`、`server/internal/imagestore/compress.go`、`server/internal/imagestore/cache.go`（新增）、`server/internal/ratelimit/limiter.go`（新增）、`server/internal/reviewswitch/switch.go`
- **HTTP 服务层**：`server/internal/server/http.go`、`server/internal/service/handlers.go`、`server/internal/service/client_ip.go`（新增）、`server/internal/service/staff.go`、`server/internal/service/admin.go`、`server/internal/service/order_commands.go`、`server/internal/service/reports.go`、`server/internal/service/review_switch.go`、`server/internal/service/platform_appearance.go`、`server/internal/service/image_storage.go`
- **Web 前端与代理**：`web/src/app/api/[...path]/route.ts`、`web/src/proxy.ts`（新增）、`web/src/app/layout.tsx`、`web/src/components/workspace-shell.tsx`、`web/src/lib/api.ts`、`web/src/lib/errors.ts`、`web/src/lib/orders.ts`、`web/src/lib/pricing.ts`、`web/src/app/counter/page.tsx`、`web/src/app/counter/orders/page.tsx`、`web/src/app/admin/orders/page.tsx`、`web/src/app/admin/audit/page.tsx`
- **CI**：`.github/workflows/ci.yml`（新增）

---

## 五、分步实现任务（TDD）

### 任务 1（P0 第一项）：Drive 孤儿清理跨环境误删防护与独立根目录校验（server，C1 + 第 10.4 节 + M4 + L3）

> **依赖与合并顺序约束：** **本任务（C1）必须与任务 2（H1 自动迁移守卫）一起合并与上线**。原因是如报告第 10.3 节所述，若未带上 H1 守卫就重启或推送配置，可能顺带触发自动迁移；两者一同上线才能做到「开关默认关闭 + 热更新即时停用清理 + 热更新绝不触发迁移」。
> **运维动作边界说明：** 第 10.2 / 10.3 / 10.5 节（各环境先暂停清理、Drive 降权、垃圾桶 30 天手工还原）均属运维动作，由安全与性能检查同事整理，**不写进本 plan**。已有历史图片向各环境独立根目录迁移的步骤在步骤 5 单独标注。

**文件：**
- 修改：`server/internal/conf/conf.go:116-125, 193-207`
- 修改：`server/internal/conf/conf_test.go:28-86`
- 修改：`server/configs/nacos/stallink.yaml:17-22`
- 修改：`server/internal/imagestore/drive.go:21-186`
- 修改：`server/internal/imagestore/drive_test.go:18-98`
- 修改：`server/internal/data/blob_cleanup.go:10-31`
- 修改：`server/cmd/stallink/main.go:272-279, 346-368, 416-422, 456-488`
- 创建：`server/cmd/stallink/sweeper_test.go`

**四项核心防护设计（对齐第 10.4 节长期方案）：**
1. **孤儿清理开关 `storage.orphan_cleanup_enabled` 默认关闭，支持 Nacos 热更新实时切换**：
   - `conf.Storage` 新增 `OrphanCleanupEnabled bool`（`json:"orphan_cleanup_enabled"`，默认 `false`）；
   - `prepareRuntime`（`main.go:347`）仅在 `DriveRootFolderID`、`Environment` 或 `DriveOAuthWeb` 变化时才重建 `drive`（避免仅开关清理时重连 Drive）；`next.start()` 仅当 `drive != nil && bc.Storage.OrphanCleanupEnabled` 时才启动 `runBlobSweeper`；
   - Nacos 热更新推送 `orphan_cleanup_enabled: false` 时，`apply()` 调用旧 `active.close()`（`stopSweep(); workers.Wait();`）立即停止上一轮清理 goroutine，下一轮不再执行，且因 `previous != nil`（配合任务 2 守卫）**绝不触发任何数据库迁移**。
2. **参照集合为空时拒绝删除**：`Drive.Sweep` 入口判断 `len(referenced) == 0` 直接返回 `ErrEmptyReferenceSet`，绝不调用 Drive `Trash` 接口。
3. **单轮删除设上限 50**：定义常量 `maxOrphanTrashPerSweep = 50`（并支持配置项 `SweepMaxDelete int` 默认 `50`），当本轮待删孤儿超过上限时直接中止返回 `ErrOrphanSweepAborted`（不做部分删除）。
4. **每个环境用独立根目录，启动时校验**：`conf.Storage` 新增 `Environment string`（`json:"environment"`）；`nacos/stallink.yaml` 移除写死的根目录 ID，改为占位符 `$STALLINK_DRIVE_ROOT_FOLDER_ID` 与 `$STALLINK_STORAGE_ENVIRONMENT`，并显式配置 `orphan_cleanup_enabled: false`；启用 Drive 时启动校验 `Environment` 非空且匹配 `^[a-z0-9_-]{2,32}$`、`DriveRootFolderID` 非空；`Drive.Put` 写入 `AppProperties["stallinkEnv"] = env`，`Drive.Sweep` 查询串必须同时匹配 `stallinkImageRoot` 与 `stallinkEnv`。同时合并 **L3**，在 `blob_cleanup.go` 补入 `shops.cover_path` 与 `events.cover_path`。

- [ ] **步骤 1：为四项防护（含开关默认值与 Nacos 热更新关闭清理不触发迁移）编写失败的单元测试**

1. 在 `server/cmd/stallink/sweeper_test.go` 编写 **防护①（默认关闭 + Nacos 热更新从 `true` 改为 `false` 后下一轮不执行且不迁移）** 与 **防护④（启动校验）** 的测试：

```go
package main

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/conf"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrphanCleanupDisabledByDefault(t *testing.T) {
	t.Parallel()
	var sweepStarted bool
	cfg := conf.Storage{
		Environment:       "local",
		DriveRootFolderID: "local-root-id",
		// OrphanCleanupEnabled 零值即为 false
	}
	if startBlobSweeperIfEnabled(context.Background(), cfg, func() { sweepStarted = true }) {
		t.Fatal("expected orphan cleanup to stay disabled when orphan_cleanup_enabled is false")
	}
	if sweepStarted {
		t.Fatal("sweeper callback must not run when disabled")
	}
}

func TestHotReloadDisablesOrphanCleanupWithoutMigrating(t *testing.T) {
	t.Parallel()
	var sweepRuns atomic.Int32
	var migrateCalls atomic.Int32

	prevStorage := conf.Storage{
		Environment:          "staging",
		DriveRootFolderID:    "root-staging",
		OrphanCleanupEnabled: true,
	}
	nextStorage := conf.Storage{
		Environment:          "staging",
		DriveRootFolderID:    "root-staging",
		OrphanCleanupEnabled: false,
	}

	// 1. 初始开启时启动清理
	ctx1, cancel1 := context.WithCancel(context.Background())
	started := startBlobSweeperIfEnabled(ctx1, prevStorage, func() {
		sweepRuns.Add(1)
	})
	if !started || sweepRuns.Load() != 1 {
		t.Fatalf("expected initial sweep to run once, started=%v runs=%d", started, sweepRuns.Load())
	}

	// 2. 模拟 Nacos 热更新（previous != nil）将开关由 true 改为 false：
	//    a) 先走任务 2 迁移守卫：热更新永不触发 Migrate
	err := runMigrationGate(context.Background(), true, false, conf.Database{
		Driver: "postgres", Source: "postgres://u:p@127.0.0.1:5432/stallink", AutoMigrate: true,
	}, nil, migrationHooks{
		migrate: func(context.Context, *pgxpool.Pool) error {
			migrateCalls.Add(1)
			return nil
		},
		pending: func(context.Context, *pgxpool.Pool) ([]uint64, error) { return nil, nil },
	})
	if err != nil || migrateCalls.Load() != 0 {
		t.Fatalf("hot reload triggered migration or failed: err=%v migrateCalls=%d", err, migrateCalls.Load())
	}

	//    b) 停止旧 runtime，按新配置重建：下一轮清理不执行
	cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	restarted := startBlobSweeperIfEnabled(ctx2, nextStorage, func() {
		sweepRuns.Add(1)
	})
	time.Sleep(20 * time.Millisecond)
	if restarted || sweepRuns.Load() != 1 {
		t.Fatalf("expected no further sweeps after hot reload to false, restarted=%v totalRuns=%d", restarted, sweepRuns.Load())
	}
}

func TestValidateStorageStartupRejectsMissingEnvOrRoot(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  conf.Storage
	}{
		{"missing_environment", conf.Storage{DriveRootFolderID: "root-local", DriveClientID: "cid", DriveClientSecret: "sec", DriveRefreshToken: "rt"}},
		{"invalid_environment", conf.Storage{Environment: "Prod Env!", DriveRootFolderID: "root-prod", DriveClientID: "cid", DriveClientSecret: "sec", DriveRefreshToken: "rt"}},
		{"missing_root_id", conf.Storage{Environment: "prod", DriveClientID: "cid", DriveClientSecret: "sec", DriveRefreshToken: "rt"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateDriveStorageConfig(tc.cfg)
			if err == nil {
				t.Fatalf("expected startup validation error for %s", tc.name)
			}
			if strings.Contains(err.Error(), "sec") || strings.Contains(err.Error(), "rt") {
				t.Fatalf("validation error leaked secret: %v", err)
			}
		})
	}
}
```

2. 在 `server/internal/imagestore/drive_test.go` 编写 **防护②（空参照集拒绝 `ErrEmptyReferenceSet`）、防护③（单轮删除上限 50 `ErrOrphanSweepAborted`）与防护④（`stallinkEnv` 查询隔离）** 测试：

```go
func TestDriveSweepGuardsEmptyRefsMaxDeleteAndEnv(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("empty_reference_set_rejected", func(t *testing.T) {
		var trashCalls atomic.Int32
		client := testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				trashCalls.Add(1)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"files":[{"id":"orphan-1"}]}`)
		})
		d := &Drive{client: client, base: "https://www.googleapis.com", rootID: "root-prod", env: "prod"}
		_, err := d.Sweep(ctx, map[string]bool{}, time.Now().Add(-24*time.Hour))
		if !errors.Is(err, ErrEmptyReferenceSet) || trashCalls.Load() != 0 {
			t.Fatalf("expected ErrEmptyReferenceSet and 0 trashes, got err=%v trashes=%d", err, trashCalls.Load())
		}
	})

	t.Run("exceeds_max_delete_limit_50_aborts", func(t *testing.T) {
		var trashCalls atomic.Int32
		client := testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				trashCalls.Add(1)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files" {
				var files []map[string]string
				for i := 0; i < 51; i++ {
					files = append(files, map[string]string{"id": fmt.Sprintf("orphan-%d", i)})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
				return
			}
		})
		d := &Drive{client: client, base: "https://www.googleapis.com", rootID: "root-prod", env: "prod"}
		_, err := d.Sweep(ctx, map[string]bool{"drive:keep": true}, time.Now().Add(-24*time.Hour))
		if !errors.Is(err, ErrOrphanSweepAborted) || trashCalls.Load() != 0 {
			t.Fatalf("expected ErrOrphanSweepAborted and 0 trashes, got err=%v trashes=%d", err, trashCalls.Load())
		}
	})

	t.Run("sweep_query_requires_matching_env_and_root", func(t *testing.T) {
		var gotQuery string
		client := testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files" {
				gotQuery = r.URL.Query().Get("q")
				io.WriteString(w, `{"files":[]}`)
				return
			}
		})
		d := &Drive{client: client, base: "https://www.googleapis.com", rootID: "root-staging", env: "staging"}
		if _, err := d.Sweep(ctx, map[string]bool{"drive:keep": true}, time.Now().Add(-24*time.Hour)); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(gotQuery, "key='stallinkEnv' and value='staging'") || !strings.Contains(gotQuery, "key='stallinkImageRoot' and value='root-staging'") {
			t.Fatalf("sweep query missing env or root guard: %s", gotQuery)
		}
	})
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./cmd/stallink ./internal/imagestore -run 'TestOrphanCleanupDisabledByDefault|TestHotReloadDisablesOrphanCleanupWithoutMigrating|TestValidateStorageStartup|TestDriveSweepGuards' -v`
预期：FAIL，提示相关字段、函数与错误常量尚未定义。

- [ ] **步骤 3：实现四项防护、热更新开关与 `cover_path` 引用补全**

1. `server/internal/conf/conf.go`：`Storage` 结构体新增 `Environment string`（`json:"environment"`）、`OrphanCleanupEnabled bool`（`json:"orphan_cleanup_enabled"`）、`SweepMaxDelete int`（`json:"sweep_max_delete"`）；`ValidateBootstrap` 校验启用 Drive 时 `Environment` 匹配 `^[a-z0-9_-]{2,32}$`、`DriveRootFolderID` 非空。
2. `server/configs/nacos/stallink.yaml:17-22`：移除硬编码根目录 ID，配置 `drive_root_folder_id: "$STALLINK_DRIVE_ROOT_FOLDER_ID"`、`environment: "$STALLINK_STORAGE_ENVIRONMENT"`、`orphan_cleanup_enabled: false`。
3. `server/internal/imagestore/drive.go`：
   - 定义 `const maxOrphanTrashPerSweep = 50`、`ErrEmptyReferenceSet`、`ErrOrphanSweepAborted`；
   - `NewDrive` 接收 `env string`（与可选上限，默认 `50`），缺少 `env` 或 `rootID` 直接报错；
   - `Drive.Put` 写入 `AppProperties: map[string]string{"stallinkImageRoot": d.rootID, "stallinkEnv": d.env}`；
   - `Drive.Sweep` 入口检查 `len(referenced) == 0` 返回 `ErrEmptyReferenceSet`；查询条件同时带 `stallinkImageRoot` 与 `stallinkEnv`；待删切片长度超出上限（默认 `50`）返回 `ErrOrphanSweepAborted` 且零删除。
4. `server/internal/data/blob_cleanup.go:11-14`：补入 `UNION SELECT cover_path FROM shops WHERE cover_path IS NOT NULL UNION SELECT cover_path FROM events WHERE cover_path IS NOT NULL`（L3）。
5. `server/cmd/stallink/main.go`：
   - `prepareRuntime`（`main.go:347`）按 `DriveRootFolderID`/`Environment`/`DriveOAuthWeb` 判定是否复用 `previous.drive`，使得仅切换 `orphan_cleanup_enabled` 时只重启 worker 而不重建 OAuth 客户端；
   - `main.go:416-422` 改用 `startBlobSweeperIfEnabled`，仅 `drive != nil && bc.Storage.OrphanCleanupEnabled` 才启动 `runBlobSweeper`，否则输出 `orphan blob sweeper disabled` 日志；
   - 热更新时 `apply()` 先调用 `active.close()` 取消旧 `sweepCtx` 并等待 worker 退出，确保立即停掉清理且绝不触发任务 2 之外的迁移。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./cmd/stallink ./internal/imagestore ./internal/conf -v`
预期：PASS。

- [ ] **步骤 5：历史共用根目录文件向各环境独立根目录迁移（上线操作）**
  - **⚠️ 需要 Kel 手动确认**：在运维完成清理停用、各环境已分配独立 Drive 根目录并注入 `STALLINK_STORAGE_ENVIRONMENT` / `STALLINK_DRIVE_ROOT_FOLDER_ID` 后，由 Kel 手动批准并执行历史文件按库反查迁移/补标脚本（依据各库 `good_images`、`bundle_images`、`shops.pay_code_path`、`shops.cover_path`、`events.cover_path` 将对应的 `drive:<id>` 移至各环境独立根目录并补写 `appProperties.stallinkEnv`）。在此之前所有环境维持 `orphan_cleanup_enabled: false`。

### 任务 2（P0 第二项）：自动迁移防护、显式 `migrate` 命令与迁移超时/校验和（server，H1 + M7 锁超时与校验和）

**文件：**
- 修改：`server/internal/conf/conf.go:111-114`
- 修改：`server/internal/conf/conf_test.go:28-86`
- 修改：`server/configs/nacos/stallink.yaml:11-17`
- 修改：`server/internal/data/migrations.go:27-123`
- 创建：`server/cmd/stallink/migrate_guard.go`
- 修改：`server/cmd/stallink/main.go:40-118, 369-380`
- 修改：`server/Makefile`
- 测试：`server/cmd/stallink/migrate_guard_test.go`、`server/internal/data/migrations_checksum_test.go`

- [ ] **步骤 1：编写失败的单元测试（覆盖热更新永不迁移、`Fallbacks` 绕过拦截、`checksum` 篡改拒绝）**

创建 `server/cmd/stallink/migrate_guard_test.go`：

```go
package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/De3trvctions/Stallink/internal/conf"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationTargetAllowed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		source  string
		confirm string
		wantErr bool
	}{
		{"loopback_ipv4", "postgres://u:secret@127.0.0.1:5432/stallink?sslmode=disable", "", false},
		{"loopback_ipv6", "postgres://u:secret@[::1]:5432/stallink", "", false},
		{"wrong_db_on_localhost", "postgres://u:secret@localhost/stallink_prod", "", true},
		{"remote_without_confirm", "postgres://u:secret@10.0.0.5/stallink", "", true},
		{"remote_with_matching_confirm", "postgres://u:secret@10.0.0.5/stallink", "10.0.0.5/stallink", false},
		{"remote_with_wrong_confirm", "postgres://u:secret@10.0.0.5/stallink", "wrong/db", true},
		{"fallbacks_bypass_blocked", "host=127.0.0.1,db.prod.example dbname=stallink", "", true},
		{"unix_socket_blocked", "host=/tmp dbname=stallink", "", true},
		{"invalid_dsn_never_leaks_secret", "://bad-secret-password", "", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := migrationTargetAllowed(tc.source, tc.confirm)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Fatalf("error leaked secret: %v", err)
			}
		})
	}
}

func TestDecideRuntimeMigration(t *testing.T) {
	t.Parallel()
	var migrateCalls, pendingCalls int
	deps := migrationHooks{
		migrate: func(context.Context, *pgxpool.Pool) error {
			migrateCalls++
			return nil
		},
		pending: func(context.Context, *pgxpool.Pool) ([]uint64, error) {
			pendingCalls++
			return []uint64{14}, nil
		},
	}

	// 热更新永不调用 migrate；存在待执行迁移时拒绝热更新。
	err := runMigrationGate(context.Background(), true, true, conf.Database{
		Driver: "postgres", Source: "postgres://u:p@127.0.0.1:5432/stallink", AutoMigrate: true,
	}, nil, deps)
	if !errors.Is(err, errPendingMigrations) || migrateCalls != 0 || pendingCalls != 1 {
		t.Fatalf("reload gate got err=%v migrate=%d pending=%d", err, migrateCalls, pendingCalls)
	}

	// auto_migrate=false 时首次启动也只跑只读检查。
	err = runMigrationGate(context.Background(), false, true, conf.Database{
		Driver: "postgres", Source: "postgres://u:p@127.0.0.1:5432/stallink", AutoMigrate: false,
	}, nil, deps)
	if !errors.Is(err, errPendingMigrations) || migrateCalls != 0 || pendingCalls != 2 {
		t.Fatalf("auto_migrate=false gate got err=%v migrate=%d pending=%d", err, migrateCalls, pendingCalls)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

在 `server/` 目录执行：`go test ./cmd/stallink -run 'TestMigrationTargetAllowed|TestDecideRuntimeMigration' -v`
预期：FAIL，提示 `undefined: migrationTargetAllowed` / `migrationHooks`。

- [ ] **步骤 3：实现 `conf.Database.AutoMigrate`、`migrate_guard.go`、`PendingMigrations` 与 `checksum`/超时**

1. 修改 `server/internal/conf/conf.go:111-114`：新增 `AutoMigrate bool `json:"auto_migrate"``。
2. 修改 `server/configs/nacos/stallink.yaml:11-17`：显式写入 `auto_migrate: false`。
3. 修改 `server/internal/data/migrations.go`：
   - `migration` 结构体增加 `checksum string`（`sha256` 十六进制）；
   - `runMigrations` 建表后执行 `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum text NOT NULL DEFAULT ''`；
   - `applyMigration` 事务开始后立即执行 `SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s';`；
   - 已执行记录若 `existingChecksum != ""` 且与当前文件 `checksum` 不一致则返回错误拒绝启动；
   - 新增只读 `PendingMigrations(ctx context.Context, pool *pgxpool.Pool) ([]uint64, error)`。
4. 创建 `server/cmd/stallink/migrate_guard.go`：实现 `isLoopback`、`migrationTarget`（校验主主机与全部 `Fallbacks` 均须为回环主机且库名为 `stallink`，排除 Unix socket）、`migrationTargetAllowed`、`runMigrationGate`。
5. 修改 `server/cmd/stallink/main.go`：`localBootstrap` 复用 `migrationTarget`；`splitCommand`/`mainExitCode` 增加 `"migrate"` 子命令；`prepareRuntime` 调用 `runMigrationGate`；`server/Makefile` 新增 `migrate:` 目标。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./cmd/stallink ./internal/conf ./internal/data -v`
预期：PASS。
> **⚠️ 需要 Kel 手动确认**：今后对任何云端或远程库执行 `go run ./cmd/stallink migrate -conf <cfg> -confirm <host>/<db>` 前，必须由 Kel 人工核对目标与备份后显式授权。

---

### 任务 3：数据库迁移 `014` 与 `migrations/down/` 回滚目录隔离（server，P1-3 / P1-1 / P1-4 / M1 / M6 前置）

**文件：**
- 创建：`server/internal/data/migrations/014_order_payment_reference_and_audit.sql`
- 创建：`server/internal/data/migrations/down/014_order_payment_reference_and_audit.down.sql`
- 创建：`server/internal/data/migrations_test.go`
- 修改：`server/internal/data/migrations_integration_test.go:100-116`

**设计说明（核对 `migrations.go:16, 67-95`）：**
- `//go:embed migrations/*.sql` 与 `fs.Glob(migrationFiles, "migrations/*.sql")` 只扫描 `migrations/` 顶层 `.sql` 文件；因此回滚 SQL 放进 `server/internal/data/migrations/down/` 可保证**启动或迁移时绝不会被打包或自动执行**。
- **不要改动 `server/internal/data/schema.go`**（保持 pre-001 基线，避免破坏 `TestMigrationBackfillsLegacyLinesBySnapshotVersion`）。

- [ ] **步骤 1：编写失败的单元测试**

创建 `server/internal/data/migrations_test.go`：

```go
package data

import (
	"strings"
	"testing"
)

func TestLoadMigrationsIncludes014AndIgnoresDownDirectory(t *testing.T) {
	t.Parallel()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	var found014 bool
	for _, m := range migrations {
		if strings.Contains(m.name, "down") {
			t.Fatalf("rollback file loaded as forward migration: %d_%s", m.version, m.name)
		}
		if m.version == 14 {
			found014 = true
			if m.name != "order_payment_reference_and_audit" || len(m.checksum) != 64 {
				t.Fatalf("unexpected 014 metadata: name=%s checksumLen=%d", m.name, len(m.checksum))
			}
		}
	}
	if !found014 {
		t.Fatal("expected migration 014 to be loaded")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/data -run TestLoadMigrationsIncludes014AndIgnoresDownDirectory -v`
预期：FAIL，提示 `expected migration 014 to be loaded`。

- [ ] **步骤 3：创建迁移 `014` 升级 SQL 与 `down/` 回滚 SQL**

1. 创建 `server/internal/data/migrations/014_order_payment_reference_and_audit.sql`：

```sql
-- 014: 订单线下支付凭据与实收对账字段、结构化/平台级审计支持、M1 共享池预占覆盖索引。

ALTER TABLE orders
	ADD COLUMN IF NOT EXISTS payment_reference text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS payment_proof_image_id text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS received_at timestamptz;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'orders'::regclass AND conname = 'orders_payment_reference_len_check'
	) THEN
		ALTER TABLE orders
			ADD CONSTRAINT orders_payment_reference_len_check
			CHECK (char_length(payment_reference) <= 120);
	END IF;
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'orders'::regclass AND conname = 'orders_payment_proof_image_id_len_check'
	) THEN
		ALTER TABLE orders
			ADD CONSTRAINT orders_payment_proof_image_id_len_check
			CHECK (char_length(payment_proof_image_id) <= 128);
	END IF;
END $$;

ALTER TABLE order_payments
	ADD COLUMN IF NOT EXISTS payment_reference text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS payment_proof_image_id text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS received_at timestamptz,
	ADD COLUMN IF NOT EXISTS actual_received_minor bigint;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'order_payments'::regclass AND conname = 'order_payments_actual_received_check'
	) THEN
		ALTER TABLE order_payments
			ADD CONSTRAINT order_payments_actual_received_check
			CHECK (actual_received_minor IS NULL OR actual_received_minor >= 0);
	END IF;
END $$;

UPDATE order_payments
SET received_at = COALESCE(received_at, confirmed_at),
	actual_received_minor = COALESCE(actual_received_minor, amount_minor)
WHERE received_at IS NULL OR actual_received_minor IS NULL;

ALTER TABLE audit_logs ALTER COLUMN shop_id DROP NOT NULL;
ALTER TABLE audit_logs
	ADD COLUMN IF NOT EXISTS reason text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS detail jsonb NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'audit_logs'::regclass AND conname = 'audit_logs_reason_len_check'
	) THEN
		-- 允许非取消/退款的系统与历史审计保留默认空串 ''，故不加下限 CHECK；有原因操作由应用层强校验 2..500 Unicode 字符。
		ALTER TABLE audit_logs
			ADD CONSTRAINT audit_logs_reason_len_check
			CHECK (char_length(reason) <= 500);
	END IF;
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'audit_logs'::regclass AND conname = 'audit_logs_detail_object_check'
	) THEN
		ALTER TABLE audit_logs
			ADD CONSTRAINT audit_logs_detail_object_check
			CHECK (jsonb_typeof(detail) = 'object');
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS audit_logs_platform_created_idx
	ON audit_logs (created_at DESC, id DESC)
	WHERE shop_id IS NULL;

-- M1: 以 good_id 为前导列的覆盖索引，消除公开目录查询时 event_listings 按 good_id 汇总 reserved_qty 退化为全表 SubPlan 扫描。
-- 说明：server/internal/data/migrations.go:98-121 的 applyMigration 在 pool.Begin 事务内执行每条迁移，
-- PostgreSQL 禁止在事务块内执行 CREATE INDEX CONCURRENTLY（报 SQLSTATE 25001），因此本迁移采用普通 CREATE INDEX IF NOT EXISTS（受任务 2 的 lock_timeout=5s 保护）。
-- 若线上 event_listings 体量极大，上线前可在事务外手动预先执行 CONCURRENTLY 建同名索引后再跑 014（⚠️ 需 Kel 确认）。
CREATE INDEX IF NOT EXISTS event_listings_good_shop_idx
	ON event_listings (good_id, shop_id) INCLUDE (reserved_qty);
```

2. 创建 `server/internal/data/migrations/down/014_order_payment_reference_and_audit.down.sql`：

```sql
-- 014 手动回滚脚本（存放于 migrations/down/，不会被 //go:embed migrations/*.sql 加载执行）
BEGIN;
SET LOCAL lock_timeout = '5s';

DROP INDEX IF EXISTS event_listings_good_shop_idx;
DROP INDEX IF EXISTS audit_logs_platform_created_idx;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_detail_object_check;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_reason_len_check;
ALTER TABLE audit_logs
	DROP COLUMN IF EXISTS detail,
	DROP COLUMN IF EXISTS reason;
DELETE FROM audit_logs WHERE shop_id IS NULL;
ALTER TABLE audit_logs ALTER COLUMN shop_id SET NOT NULL;

ALTER TABLE order_payments DROP CONSTRAINT IF EXISTS order_payments_actual_received_check;
ALTER TABLE order_payments
	DROP COLUMN IF EXISTS actual_received_minor,
	DROP COLUMN IF EXISTS received_at,
	DROP COLUMN IF EXISTS payment_proof_image_id,
	DROP COLUMN IF EXISTS payment_reference;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_proof_image_id_len_check;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_reference_len_check;
ALTER TABLE orders
	DROP COLUMN IF EXISTS received_at,
	DROP COLUMN IF EXISTS payment_proof_image_id,
	DROP COLUMN IF EXISTS payment_reference;

DELETE FROM schema_migrations WHERE version = 14;
COMMIT;
```

- [ ] **步骤 4：运行单测与本机隔离集成测试验证**

- `go test ./internal/data -run TestLoadMigrationsIncludes014AndIgnoresDownDirectory -v`（预期 PASS）
- 仅使用本机 Docker Postgres：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run TestMigrationsFreshConcurrentAndIdempotent -v`
- **⚠️ 需要 Kel 手动确认**：任何对云端/远程数据库执行 `014` 升级或 `down/014` 回滚前，必须等待 Kel 手动批准。

---

### 任务 4：升级 Go 至 `1.27.4+` 与 `golang.org/x/image` 至 `v0.45.0+`（server，H3）

**文件：**
- 修改：`server/go.mod:3, 11`（与 `server/go.sum`）
- 检查：`Dockerfile`（核实当前仓库根与 `server/` 均**无 Dockerfile**；若后续构建容器镜像，必须指定 `FROM golang:1.27.4-alpine` 或更高小版本）
- 同步：`.github/workflows/ci.yml`（在任务 15 中通过 `go-version-file: server/go.mod` 与安装 `govulncheck` 强制校验）

- [ ] **步骤 1：更新 `server/go.mod` 与依赖**

在 `server/` 目录执行：
```bash
go mod edit -go=1.27.4
go get golang.org/x/image@v0.45.0
go mod tidy
```
检查 `server/go.mod`：第 3 行为 `go 1.27.4`（或当前可用的 `>= 1.27.4` 补丁版），第 11 行为 `golang.org/x/image v0.45.0`（或更高版本）。若本机 `go` 二进制低于 `1.27.4`，需先在开发机/CI 升级 Go 工具链至 `>= 1.27.4`。

- [ ] **步骤 2：重跑 `imagestore` 测试与 `govulncheck` 验证漏洞清零**

在 `server/` 目录执行：
```bash
go test ./internal/imagestore/... -v
go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
```
预期：`internal/imagestore` 全绿；`govulncheck` 输出中 **“Your code is affected” 漏洞数从 15 降为 0**（覆盖 `GO-2026-6222`、`GO-2026-5061`、`GO-2026-4961` 与 `GO-2026-6859` 等 12 个标准库漏洞）。

### 任务 5：消除 `readGoodPool` 中 `shopID == 0` 跳过店铺过滤后门（server，P2-3）

**文件：**
- 修改：`server/internal/data/catalog_store.go:1211-1238`
- 修改：`server/internal/data/order_store.go:411, 581, 600-686`
- 创建：`server/internal/data/pool_isolation_integration_test.go`

- [ ] **步骤 1：编写失败的集成测试**

创建 `server/internal/data/pool_isolation_integration_test.go`：

```go
//go:build integration

package data

import (
	"errors"
	"testing"
)

func TestReadGoodPoolRejectsZeroOrWrongShopID(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "pool_isolation")
	var shopA, shopB, goodA string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Shop A','pool-a') RETURNING id::text`).Scan(&shopA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Shop B','pool-b') RETURNING id::text`).Scan(&shopB); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'Good A',10) RETURNING id::text`, shopA).Scan(&goodA); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	goodAID := integrationID(t, goodA)
	if _, err := readGoodPool(ctx, tx, 0, goodAID); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID for shopID=0, got %v", err)
	}
	if _, err := readGoodPool(ctx, tx, integrationID(t, shopB), goodAID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-shop, got %v", err)
	}
	got, err := readGoodPool(ctx, tx, integrationID(t, shopA), goodAID)
	if err != nil || got.Total != 10 {
		t.Fatalf("expected total=10, got %+v err=%v", got, err)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run TestReadGoodPoolRejectsZeroOrWrongShopID -v`
预期：FAIL，当前 `shopID=0` 仍能读出 `Total=10`。

- [ ] **步骤 3：修改 `readGoodPool` 与 `applyStock` 强制按 `shopID` 过滤**

- `server/internal/data/catalog_store.go:1214-1225`：`if shopID <= 0 { return catalog.Pool{}, ErrInvalidID }`；SQL 去除 `$2 = 0 OR`，改为严格 `AND shop_id = $2`；未命中返回 `ErrNotFound`。
- `server/internal/data/order_store.go:411, 581, 600-686`：将 `applyStock` 签名改为 `applyStock(ctx context.Context, tx pgx.Tx, shopID, orderID ID, sell bool) error`，内部 `order_lines`、`readGoodPool`、`event_listings` 锁与更新、`goods` 更新、`assertPool` 全部显式带 `shopID` 过滤。

- [ ] **步骤 4：运行测试验证通过**

运行：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run 'TestReadGoodPoolRejectsZeroOrWrongShopID|TestConfirmRefundRestoresStockAndAggregatesMovements' -v`
预期：PASS。

---

### 任务 6：统一 `releaseOrderReservations` 按全局锁序加锁（server，P2-2）

**文件：**
- 修改：`server/internal/data/reservation_store.go:235-277`
- 创建：`server/internal/data/lock_order_integration_test.go`

- [ ] **步骤 1：编写并发释放预占死锁回归集成测试**

创建 `server/internal/data/lock_order_integration_test.go`，并发运行「多商品下单（锁 `goods` → `event_listings`）」与「释放包含多个商品的过期预占（`releaseOrderReservations`）」，要求 `err == nil` 且无 PostgreSQL `40P01` 死锁错误。

- [ ] **步骤 2：运行测试确认现状**

运行：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run TestConcurrentReserveAndReleaseNoDeadlock -v`

- [ ] **步骤 3：修改 `releaseOrderReservations`（`reservation_store.go:235-277`）**

1. 按 `id ASC` 查询预占汇总并读取涉及的 `good_id` 集合，去重后按 `good_id ASC` 升序；
2. 先对 `goods` 表执行 `SELECT id FROM goods WHERE id = ANY($1) ORDER BY id FOR UPDATE`；
3. 再对 `event_listings` 表执行 `SELECT id, good_id FROM event_listings WHERE id = ANY($1) ORDER BY id FOR UPDATE`；
4. 最后批量扣减 `event_listings.reserved_qty` 并校验 `total_stock >= reserved_qty`。

- [ ] **步骤 4：运行测试验证通过**

运行：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run TestConcurrentReserveAndReleaseNoDeadlock -v`
预期：PASS，且可复跑 `-count=5` 均无死锁。

---

### 任务 7：取消、两步退款确认与一步「取消并退款」接口契约、事务锁与审计（server，P1-1）

> **策略依据（2026-11-26 Kel 拍板）：**
> 1. 维持 `Q3=B`：同店 `sales` 或 `shop_admin` 可执行取消、一步取消并退款及两步退款确认；跨店返回 **`403 forbidden`**。
> 2. **原因口径统一为 `2..500` Unicode 字符**：先 `TrimSpace` 再按 `utf8.RuneCountInString` 计数；`0` 字符返回 `422 reason_required`，`1` 字符返回 `422 reason_too_short`，`>500` 返回 `422 reason_too_long`（理由见下文「错误码取舍说明」）。
> 3. **未收款（`pending`）订单取消也强制填写原因**（与 `paid`/`fulfilled` 统一）。
> 4. **新增一步完成接口 `POST /api/staff/orders/{orderID}/cancel-refund`**，保留现有两步接口（`cancel` → `refund_required` → `refund-confirm`）以兼容存量 `refund_required` 订单。**一步接口仅支持全额退款且货品全部完好回库；部分退款和报损仍走 `vNext Task S7`。**

#### 现状核实结果（已逐文件确认）
1. **`Cancel` 与 `RefundConfirm` 不读请求体**：`server/internal/service/staff.go:710-722` 与 `728-740` 传 `struct{}{}` 给 `a.orderActionCommand`，幂等哈希恒为 `sha256("{}")`。
2. **路由风格（`server/internal/server/http.go:74-79`）**：均使用 `route(r, http.MethodPost, "/api/staff/orders/{orderID}/<action>", api.<Handler>, "orderID")`，新接口依项目命名风格定为 **`POST /api/staff/orders/{orderID}/cancel-refund`**（`api.CancelRefund`）。
3. **审计表无原因列**：`server/internal/data/schema.go:132-140` 只有 `id, shop_id, actor_id, action, subject_type, subject_id, created_at`；由任务 3（迁移 `014`）补齐 `reason text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 500)` 与 `detail jsonb`。
4. **前端现有提示与校验（`web/src/lib/order-actions.ts:20-36`）**：`REASON_MAX_LENGTH` 现为 `200`（待前端按本契约改为 `500` 并补最短 `2` 字符校验），`validateActionReason` 在 `!reason` 时报 `"请填写原因。"`、`orderActionRequestBody` 目前返回 `{}`。

#### 错误码取舍说明（为何对 1 个字符单设 `reason_too_short`）
- 对照前端 `web/src/lib/order-actions.ts:26` 现有提示文案：`!reason` 映射为 `"请填写原因。"`；若用户输入了 1 个字（如 `"退"`）却收到 `reason_required` 并显示 `"请填写原因。"`，会给收银员造成「已填写却提示未填」的困惑。
- 因此本契约区分：`0` 字符（空串/空白）→ **`422 reason_required`**（`"请填写原因。"`）；`1` 个字符 → **`422 reason_too_short`**（`"原因至少需要 2 个字。"`）；`> 500` 字符 → **`422 reason_too_long`**（`"原因不能超过 500 个字。"`）。

#### 接口约定表（三个命令接口）

| 维度 | `POST /api/staff/orders/{orderID}/cancel` | `POST /api/staff/orders/{orderID}/refund-confirm` | `POST /api/staff/orders/{orderID}/cancel-refund`（新增） |
|---|---|---|---|
| **权限要求** | 同店 `sales` / `shop_admin`（跨店或 `shopID==nil` 返 **`403 forbidden`**） | 同店 `sales` / `shop_admin`（跨店返 **`403 forbidden`**） | 同店 `sales` / `shop_admin`（跨店返 **`403 forbidden`**） |
| **幂等操作域** | `"order.cancel:" + orderID` | `"order.refund_confirm:" + orderID` | `"order.cancel_refund:" + orderID`（独立操作域） |
| **请求体** | `{ "reason": string }` | `{ "reason": string }` | `{ "reason": string }` |
| **原因校验** | `TrimSpace` 后 `utf8.RuneCountInString` **`2..500`**（`pending` 同样必填） | `TrimSpace` 后 `utf8.RuneCountInString` **`2..500`** | `TrimSpace` 后 `utf8.RuneCountInString` **`2..500`** |
| **校验错误码** | 非法 JSON → `400 invalid_json`；`0` 字 → **`422 reason_required`**；`1` 字 → **`422 reason_too_short`**；`>500` 字 → **`422 reason_too_long`** | 同左 | 同左 |
| **幂等重放/冲突** | 归一化 `{ "reason": trimmed }` 计入 `request_hash`：同键同因 `200` + `Idempotency-Replayed: true`；同键不同因 **`409 idempotency_conflict`** | 同左 | 同左 |
| **允许前置状态** | `pending`（→ `cancelled` 释预占）；`paid`/`fulfilled`（→ `refund_required`，兼容保留）；已 `cancelled`/`expired`/`refund_required` 按幂等读取返回 | `refund_required`（→ `cancelled` + `refunded_at` + 全额回库）；已 `cancelled && refunded_at != nil` 幂等返回 | **仅 `paid` / `fulfilled`**（单事务直接置 `cancelled` + `refunded_at`、全额回库、写 `refund` 流水） |
| **非法状态返回** | 非法状态返 **`409 invalid_status`** | 非 `refund_required` 返 **`409 invalid_status`** | **`pending`、`refund_required`、`cancelled`、`expired` 统一返 `409 invalid_status`**（见下文说明） |
| **审计落库** | 写 1 条：`order.cancel` 或 `order.refund_required`，含 `reason` + `detail` | 写 1 条：`order.refund_confirm`，含 `reason` + `detail(amountMinor, currency, refundedAt)` | **只写 1 条**：`order.cancel_refund`，含 `reason` + `detail(actorId, actorRole, reason, amountMinor, currency, refundedAt)` |

#### 事务、锁机制与关键状态设计说明
1. **`cancel-refund` 单事务三件事与并发保护**：
   - 在 `runOrderCommand` / `cancelAndRefundOrderTx` 事务开启后立即设置 `SET LOCAL lock_timeout = '2s'`；若触发锁等待超时（SQLSTATE `55P03`）映射为 `503 command_in_progress` + `Retry-After: 1`；若触发 PostgreSQL 死锁（`40P01`）或串行化冲突（`40001`）在命令事务外层自动重试最多 **2 次**（在注入失败/死锁时完整回滚，不留脏数据）。
   - 事务内顺序：① `SELECT shop_id, status, total_minor, currency, refunded_at FROM orders WHERE id = $1 FOR UPDATE`（跨店返 `ErrCommandUnauthorized` → `403`，非 `paid`/`fulfilled` 返 `ErrStatus` → `409 invalid_status`）；② 调用 `applyStock(ctx, tx, shopID, orderID, false)`（严格遵循**任务 6** 的全局锁序：`goods` 按 `good_id ASC` → `event_listings` 按 `listing_id ASC`，并衔接**任务 5** 的 `shopID` 过滤），全额加回 `goods.total_stock` 并扣减 `event_listings.sold_qty`；③ 写入 `inventory_movements(..., movement_type = 'refund', qty = SUM(qty))`；④ 更新 `orders SET status = 'cancelled', refunded_at = now(), state_version = state_version + 1` 并写入 1 条 `audit_logs`。
2. **为什么 `refund_required` 调 `cancel-refund` 返回 `409 invalid_status`（而不是直接完成）**：
   - 现网状态机中 `refund_required` 已有专属出口 `POST /api/staff/orders/{orderID}/refund-confirm`（`order_store.go:554-562`），前端在 `refund_required` 状态也仅展示 `refund` 按钮（`order-actions.ts:8`）。
   - 让 `cancel-refund` 严格只接受 `paid`/`fulfilled` 并在遇到 `refund_required`（以及 `pending`、`cancelled`、`expired`）时沿用 `handlers.go:688-689` 的 **`409 invalid_status`**，可使状态边界与并发竞态完全清晰：若并发请求中 `cancel` 先赢并置 `refund_required`，后到的 `cancel-refund` 立即报 `409 invalid_status`（绝不重复/越界操作），前端刷新后按两步 `refund-confirm` 收尾；若 `cancel-refund` 先赢并置 `cancelled`，后到的 `cancel` 见 `status == "cancelled"` 直接只读返回，库存永远只回补一次。
3. **为什么 `cancel-refund` 审计只写 1 条**：
   - 一步接口在业务和数据库事务上都是不可分割的单一原子操作（单一操作人、单一原因、全额与完成时间），写 `order.cancel_refund` 1 条审计即可完整追溯；强行写两条（`order.refund_required` + `order.refund_confirm`）会伪造出不存在的中间停留态并干扰审计查询统计。
4. **范围边界**：本接口只做**全额退款 + 全部货品完好回库**；部分商品/数量退款与报损（`damaged_qty` 不回库）仍然走 `vNext Task S7`（迁移 `017`）。

**文件：**
- 修改：`server/internal/server/http.go:78-80`（新增 `POST /api/staff/orders/{orderID}/cancel-refund`）
- 修改：`server/internal/service/staff.go:710-740`（`Cancel`、`RefundConfirm` 与新增 `CancelRefund`）
- 修改：`server/internal/service/order_commands.go:20-69, 96-133`
- 修改：`server/internal/service/handlers.go:666-705`（补充 `55P03` 锁超时映射，保持 `invalid_status` / `idempotency_conflict` / `forbidden`）
- 修改：`server/internal/data/order_command_store.go:35-115`（为命令事务注入 `lock_timeout = '2s'` 及 `40P01`/`40001` 最多 2 次死锁重试）
- 修改：`server/internal/data/order_store.go:64-70, 494-598, 865-895`（新增 `CancelAndRefundOrderCommand` / `cancelAndRefundOrderTx`）
- 修改：`server/internal/data/paged_lists.go:50-75`
- 修改：`server/internal/data/catalog_store.go:1308-1315`
- 修改：`server/internal/data/refund_integration_test.go:39`
- 创建：`server/internal/service/refund_policy_test.go`
- 创建：`server/internal/data/cancel_refund_integration_test.go`

- [ ] **步骤 1：编写失败的单元测试与集成测试**

1. **HTTP 契约与边界单测（`server/internal/service/refund_policy_test.go`）**：覆盖 `cancel`、`refund-confirm` 与 `cancel-refund` 三个接口：
   - **原因长度边界（Unicode 字符数）**：
     - `""` / `"   \t\n"` / `{}` → **`422 reason_required`**（不写 `order_commands`）；
     - 恰好 **1 个字符**（`"退"`、`"😀"`、`" a "`）→ **`422 reason_too_short`**；
     - 恰好 **2 个字符**（`"退货"`、`"😀😀"`）→ **通过校验**；
     - **多字节 Unicode 校验**：`167` 个汉字（501 UTF-8 字节但仅 167 字符）与恰好 **500 个汉字/Emoji** → **通过校验**；
     - **501 个字符**（501 个汉字或 `"a"`）→ **`422 reason_too_long`**；
   - **`pending` 订单取消强制填原因**：未收款 `pending` 单传空/`1` 字被 `422` 拒绝，传有效 `2..500` 字成功取消并释放预占；
   - **跨店拦截**：店铺 B 的 `sales`/`shop_admin` 调用任一接口均返回 **`403 forbidden`**；
   - **幂等重放与同键异因**：同 `Idempotency-Key` + 同 `reason`（含 trim 归一化后一致）返回 `200` + `Idempotency-Replayed: true`；同 `Idempotency-Key` + 不同 `reason` 返回 **`409 idempotency_conflict`**；
   - **非法状态**：对 `pending`、`refund_required`、`cancelled`、`expired` 订单调用 `cancel-refund` 均返回 **`409 invalid_status`**。
2. **数据库事务、原子性与并发集成测试（`server/internal/data/cancel_refund_integration_test.go` & `refund_integration_test.go`）**：
   - **成功路径**：对 `paid` / `fulfilled` 订单调用 `CancelAndRefundOrderCommand`，验证 `status == "cancelled"`、`refunded_at != nil`、`goods.total_stock` 全额回补、`event_listings.sold_qty` 等量扣减、`inventory_movements` 写入 `movement_type = 'refund'`，且 `audit_logs` 仅写入 **1 条** `action = 'order.cancel_refund'`（`actor_id`、`reason`、`detail->>'amountMinor'`、`detail->>'currency'`、`detail->>'refundedAt'`、`created_at` 齐全）；
   - **原子性回滚**：在 `cancelAndRefundOrderTx` 的 `applyStock` 之后、提交前注入失败钩子，断言整体事务回滚、`orders.status` 仍为 `paid`、`goods.total_stock` 与 `sold_qty` 不变、无 `inventory_movements` 与 `audit_logs` 写入；
   - **并发竞争**：
     - 场景 A：同一 `paid` 订单用两个不同 `Idempotency-Key` 并发调用 `CancelAndRefundOrderCommand` → 1 个成功、1 个返 `ErrStatus`（或 `409`），`goods.total_stock` **仅回补 1 次**；
     - 场景 B：同一 `paid` 订单并发调用 `CancelAndRefundOrderCommand` 与 `CancelOrderCommand` → 若一步赢则 `Cancel` 见 `cancelled` 只读返回；若 `Cancel` 先置 `refund_required` 则一步返 `ErrStatus`，待后调 `ConfirmRefundCommand` 时累计回补 **恰好 1 次**。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run TestOrderCancelAndRefundContract -v` 与 `STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run 'TestCancelAndRefund|TestConfirmRefund' -v`
预期：FAIL。

- [ ] **步骤 3：实现路由、统一校验、一步命令事务、`2s` 锁超时/死锁重试与审计**

1. `server/internal/server/http.go:80`：注册 `route(r, http.MethodPost, "/api/staff/orders/{orderID}/cancel-refund", api.CancelRefund, "orderID")`。
2. `server/internal/service/staff.go`：
   - 新增 `canConfirmRefund(account data.Account) bool`（同店 `sales`/`shop_admin`）；
   - 更新 `readOrderActionReason(w, r)`：
     ```go
     func readOrderActionReason(w http.ResponseWriter, r *http.Request) (string, bool) {
     	var body struct {
     		Reason string `json:"reason"`
     	}
     	if err := readJSON(r, &body); err != nil {
     		writeErr(w, http.StatusBadRequest, "invalid_json")
     		return "", false
     	}
     	reason := strings.TrimSpace(body.Reason)
     	count := utf8.RuneCountInString(reason)
     	switch {
     	case count == 0:
     		writeErr(w, http.StatusUnprocessableEntity, "reason_required")
     		return "", false
     	case count < 2:
     		writeErr(w, http.StatusUnprocessableEntity, "reason_too_short")
     		return "", false
     	case count > 500:
     		writeErr(w, http.StatusUnprocessableEntity, "reason_too_long")
     		return "", false
     	default:
     		return reason, true
     	}
     }
     ```
   - `Cancel`、`RefundConfirm` 与新增 `CancelRefund` 均先校验权限（不通过返 `403 forbidden`）与 `readOrderActionReason`，将规范载荷 `struct{ Reason string `json:"reason"` }{Reason: reason}` 传给 `a.orderActionCommand(...)`。
3. `server/internal/data/order_command_store.go`：在命令事务执行前加 `SET LOCAL lock_timeout = '2s'`；对 `40P01`/`40001` 封装外层事务重试（最多 2 次），对 `55P03` 映射为 `ErrCommandInProgress`（服务层对锁等超时返回 `503` + `Retry-After: 1`）。
4. `server/internal/data/order_store.go`：
   - `cancelOrderTx` / `confirmRefundTx` 先按 `id` 锁单再比对 `orderShopID != shopID` 返回 `ErrCommandUnauthorized`；
   - 新增 `CancelAndRefundOrderCommand` / `cancelAndRefundOrderTx`：仅允许 `status == "paid" || status == "fulfilled"`，按任务 6 锁序执行 `applyStock(ctx, tx, shopID, orderID, false)`、写 `'refund'` 流水、置 `status = 'cancelled', refunded_at = now(), state_version = state_version + 1`、写 1 条 `audit_logs`（`reason` 列 + `detail` JSONB）。
   - 更新 `AuditEntry`、`order_store.go:865` 与 `paged_lists.go:53` 返回 `reason` 与 `detail`。

- [ ] **步骤 4：运行单测与本机集成测试验证通过**

运行：`go test ./internal/service -run TestOrderCancelAndRefundContract -v` 与 `STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data -run 'TestCancelAndRefund|TestConfirmRefund' -v`
预期：PASS。

#### 前端对接契约与 `vNext` 同步点（前端具体实现归前端计划）

1. **前端对接点（`web/src/lib/order-actions.ts`、`OrderActionDialog`、`errors.ts`）**：
   - `OrderAction` 新增 `"cancelRefund"`（如 `paid: ["fulfill", "cancelRefund"]`、`fulfilled: ["cancelRefund"]`、`pending: ["cancel"]`、`refund_required: ["refund"]`），即已收款订单默认只提供「取消并退款」，`refund_required` 存量订单继续走 `refund`（`/refund-confirm`）；
   - `REASON_MAX_LENGTH` 从 `200` 改为 `500`，新增 `REASON_MIN_LENGTH = 2`，`validateActionReason` 对 `0` 字 / `1` 字 / `>500` 字做一致校验；
   - `orderActionRequestBody(action, reason)` 对 `"cancel"`、`"refund"`、`"cancelRefund"` 返回 `{ reason }`（`"fulfill"` 返回 `{}`）；
   - `OrderActionDialog` 为 `"cancelRefund"` 提供「取消并退款」入口与确认弹窗，调用 `POST /api/staff/orders/${order.id}/cancel-refund`，成功后刷新订单列表与可售库存；
   - `web/src/lib/errors.ts` 补齐 `reason_required`（`"请填写原因。"`）、`reason_too_short`（`"原因至少需要 2 个字。"`）、`reason_too_long`（`"原因不能超过 500 个字。"`），复用 `invalid_status`、`forbidden`、`idempotency_conflict`、`command_in_progress`。
2. **`vNext` 需要同步的点（已同步更新对应文档，开发实施时注意对齐）**：
   - `vnext-server.md` `Task S7`（迁移 `017`）：部分退款新接口 `POST /api/staff/orders/{orderID}/refunds` 沿用本任务 `reason`（`2..500` Unicode 字符）、`422 reason_required`/`reason_too_short`/`reason_too_long` 与 `audit_logs.reason`，并与全额一步接口 `/cancel-refund` 明确分工（一步全额+全回库归任务 7，部分退款/报损归 `S7`）；
   - `vnext-server.md` `Task S16`（迁移 `023`）：迁移 `014` 已在 `audit_logs` 新增 `reason` 列，`023` 不再重复添加或回滚 `audit_logs.reason`；
   - `vnext-web.md` `Task W6`：`refund-form.ts` 与部分退款弹窗统一沿用 `2..500` Unicode 字符与 `reason_required`/`reason_too_short`/`reason_too_long` 错误码，同时区分全额「取消并退款」（`/cancel-refund`）与部分退款/报损（`/refunds`）。

### 任务 8：分布式与降级限流 + 可信代理真实 IP 提取（server + web，P1-2 + H4/M1 匿名限流 + L2）

**文件：**
- 创建：`server/internal/ratelimit/limiter.go`
- 创建：`server/internal/ratelimit/limiter_test.go`
- 创建：`server/internal/service/client_ip.go`
- 修改：`server/internal/conf/conf.go`（新增 `Security.TrustedProxyCIDRs`）
- 修改：`server/internal/service/handlers.go:132-194, 278-288, 423-456`
- 修改：`web/src/app/api/[...path]/route.ts:6-17`

**覆盖范围与限流策略（合并 P1-2 与报告 H4/M1/L2）：**
- `POST /api/auth/login`、`POST /api/auth/register`：IP 维度（如 `10 次/分钟`）+ 账号/用户名维度（`5 次/5 分钟`），任一超限返回 `429` + `Retry-After`；
- `GET /api/public/catalog`（M1）：IP 维度 `60 次/分钟`；
- `GET /api/files/goods/{imageID}`、`GET /api/files/bundles/{imageID}`（H4）：IP 维度 `180 次/分钟`（防止缓存未命中穿透 Drive）。
- Redis 不可用时自动降级为进程内令牌桶/固定窗口计数器，绝不因 Redis 抖动直接开放无限流。
- **L2 真实 IP 与防伪造**：
  - Go `clientIP(r, trustedNets)` 仅当 `r.RemoteAddr` 落在 `TrustedProxyCIDRs` 内（默认仅 `127.0.0.1/8`、`::1/128`）时，才从右向左剥离可信代理取首个不可信客户端 IP；否则无视请求自带的 `X-Forwarded-For`；
  - `web/src/app/api/[...path]/route.ts` 转发前先 `headers.delete("x-forwarded-for")` / `delete("x-forwarded-host")` / `delete("x-forwarded-proto")`，仅在信任前置代理注入时传递真实来源。

- [ ] **步骤 1：编写失败的单元测试**

创建 `server/internal/ratelimit/limiter_test.go` 与 `server/internal/service/client_ip_test.go`：
- 测试 Redis 降级内存限流：第 1..5 次放行，第 6 次返回 `allowed=false, retryAfter > 0`；
- 测试非可信直连伪造 `X-Forwarded-For: 1.2.3.4` 被忽略，仍按 `RemoteAddr` 限流；
- 测试 `Login` 连续 6 次错误返回 `429` 与 `Retry-After` 头。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/ratelimit ./internal/service -run 'TestLimiter|TestClientIP|TestLoginRateLimit' -v`
预期：FAIL，包与函数尚不存在。

- [ ] **步骤 3：实现 `ratelimit.Limiter`、`clientIP`、Handler 挂载与 Web 代理头清洗**

1. `server/internal/ratelimit/limiter.go`：定义 `Limiter` 接口与 `HybridLimiter`（优先调用 `redis.Client` 执行 Lua 滑动窗口脚本，若 `redis == nil` 或报错则降级到带定期淘汰的 `sync.Mutex` 内存窗口桶）。
2. `server/internal/service/client_ip.go`：实现 `ParseTrustedCIDRs` 与 `ClientIP(r *http.Request, trusted []*net.IPNet) string`。
3. `server/internal/service/handlers.go`：在 `Login`、`Register`、`PublicCatalog`、`GoodImage`、`BundleImage` 入口执行限流，超限写入 `w.Header().Set("Retry-After", strconv.Itoa(retrySec))` 与 `writeErr(w, http.StatusTooManyRequests, "rate_limited")`。
4. `web/src/app/api/[...path]/route.ts:6-17`：显式删除未验证的 `x-forwarded-for`、`x-forwarded-host`、`x-forwarded-proto`、`x-real-ip` 后再向 Go 上游发起转发。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/ratelimit ./internal/service -v`
预期：PASS。

---

### 任务 9：订单支付凭据落库与日报应收/实收差额（server，P1-3 后端）

**文件：**
- 修改：`server/internal/service/staff.go:622-646, 780-813`
- 修改：`server/internal/service/reports.go:91-125`
- 修改：`server/internal/data/order_store.go:31-53, 340-430, 714-800`
- 修改：`server/internal/data/paged_lists.go:10-49`
- 修改：`server/internal/data/report_store.go:73-78, 260-310`
- 创建：`server/internal/data/payment_reference_integration_test.go`

- [ ] **步骤 1：编写失败的集成测试**

创建 `server/internal/data/payment_reference_integration_test.go`：
- 创建线上扫码（`qr`）订单并确认收款时传入 `PaymentReference = "WX-20261126-0091"`、`ActualReceivedMinor = 9900`（应收 `10000`）；
- 验证 `orders`、`order_payments` 均落库 `payment_reference`、`actual_received_minor`、`received_at`；
- 验证 `DailyReport` 返回按收款方式分组的 `expectedMinor`、`actualReceivedMinor` 与 `varianceMinor`（`-100`）。

- [ ] **步骤 2：运行测试验证失败**

运行：`STALLINK_TEST_DATABASE_URL=... go test -tags integration ./internal/data -run TestOrderPaymentReferenceAndReportVariance -v`
预期：FAIL，缺少字段与报表列。

- [ ] **步骤 3：更新 Data 与 Service 层**

1. `server/internal/data/order_store.go`：为 `Order`、`CreateOrderInput`、`ConfirmPaymentInput` 增加 `PaymentReference`、`PaymentProofImageID`、`ReceivedAt`、`ActualReceivedMinor`（`actual_received_minor == nil` 时默认等于 `amount_minor`）；`ConfirmPayment` 校验参考号长度 `<= 120`、实收金额 `>= 0`。
2. `server/internal/data/paged_lists.go` 与 `server/internal/service/staff.go`：在订单详情/列表 JSON 与确认收款入参解析/回传中透出 `paymentReference`、`paymentProofImageId`、`receivedAt`、`actualReceivedMinor`。
3. `server/internal/data/report_store.go` 与 `server/internal/service/reports.go`：按 `payment_method` 聚合 `SUM(amount_minor)` 与 `SUM(COALESCE(actual_received_minor, amount_minor))`，返回 `expectedMinor`、`actualReceivedMinor` 与 `varianceMinor = actualReceivedMinor - expectedMinor`。

- [ ] **步骤 4：运行测试验证通过**

运行：`STALLINK_TEST_DATABASE_URL=... go test -tags integration ./internal/data -run TestOrderPaymentReferenceAndReportVariance -v` 与 `go test ./internal/service -v`
预期：PASS。

---

### 任务 10：平台级审计（自动审核开关 & 外观切换）与外观读取进程内缓存（server，P1-4 + M6）

**文件：**
- 创建：`server/internal/data/platform_audit.go`
- 修改：`server/internal/service/review_switch.go:18-39`
- 修改：`server/internal/reviewswitch/switch.go:63-72`
- 修改：`server/internal/data/platform_appearance.go:8-25`
- 修改：`server/internal/service/platform_appearance.go:5-36`
- 创建：`server/internal/service/platform_audit_test.go`

**设计说明（合并 P1-4 与 M6）：**
- 依赖任务 3（迁移 `014`）放宽 `audit_logs.shop_id` 为可空并新增 `detail jsonb`。
- `SetPlatformReviewSwitch` 先保存旧值，写入 Redis 后调用 `WritePlatformAudit(..., "platform.review_switch.update", ...)`；**若写审计失败则立即回滚 Redis 至旧值并向前端返回 500**（确保状态变化绝不逃逸审计）。
- `SetPlatformAppearance` 更新数据库主题后同一事务内写入平台审计 `"platform.appearance.update"`（含 `previousTheme`、`nextTheme`、`actorId`），并立即刷新/失效进程内外观缓存。
- `PlatformAppearance` 读取先读 Go 进程内缓存（带短 TTL 如 30s + `atomic.Pointer` 或 RWMutex），消除 M6 每次 SSR 打 `platform_settings` 表。

- [ ] **步骤 1：编写失败的单元测试**

创建 `server/internal/service/platform_audit_test.go`：
- 测试 `SetPlatformReviewSwitch` 写审计失败时自动回滚 Redis 值；
- 测试 `PlatformAppearance` 连续读取 10 次只查询底层存储 1 次，调用 `SetPlatformAppearance` 后立即命中新主题且写入 1 条平台审计。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run 'TestPlatformReviewSwitchAuditRollback|TestPlatformAppearanceCacheAndAudit' -v`
预期：FAIL。

- [ ] **步骤 3：实现平台审计落库与外观缓存**

按上述设计实现 `platform_audit.go`（`InsertPlatformAudit`）、修改 `review_switch.go`、`platform_appearance.go`（数据库与 service 层）。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/service ./internal/reviewswitch -v`
预期：PASS。

---

### 任务 11：公开图片响应缓存头与服务端有界内存缓存（server，H4 不动表结构部分）

> **与 vNext 计划分工说明：** 并行计划 `docs/superpowers/plans/2026-11-26-stallink-vnext-server.md` 的 **`Task S10`（迁移 `020_media_tokens.sql`，对应需求中提及的与 S11/020 协同边界）** 将引入防枚举 `media_token` 与本地磁盘二级缓存。**本任务严禁修改任何数据库表结构**，仅落地可直接上线的 HTTP 强缓存响应头（`Cache-Control` / `ETag` / `304`）与 Go 进程内有界内存 LRU + `singleflight`，待 `S10` 落地磁盘缓存与防枚举令牌后可直接平滑叠加在当前接口之上。

**文件：**
- 创建：`server/internal/imagestore/cache.go`
- 创建：`server/internal/imagestore/cache_test.go`
- 修改：`server/internal/service/image_storage.go:80-108`
- 修改：`server/internal/service/handlers.go:423-490`
- 修改：`server/cmd/stallink/main.go`（使用 `imagestore.NewMemoryCache(64<<20, driveStore)` 包装 `Drive`）
- 修改：`server/internal/service/image_storage_test.go`（或新建 `server/internal/service/image_cache_http_test.go`）

**注意：** 确认当前 `GoodImage` / `BundleImage` 的 `reference` 由 `Drive.Put` 生成不可变 `drive:<fileID>`（每次重新上传都会产生全新 `fileID`），因此公开商品图与套餐图可安全返回强缓存头；而 `PayCodeImage` 与后台私有图片仍必须返回 `Cache-Control: private, no-store`。

- [ ] **步骤 1：编写失败的单元测试**

1. `server/internal/imagestore/cache_test.go`：
   - 测试 10 个并发 `Get("drive:abc")` 仅触发底层 `upstream.Get` 1 次（`singleflight` 防击穿）；
   - 测试超过 `maxBytes` 时按 LRU 淘汰旧对象，写入总大小严格受限于上限；
   - 测试单对象大于 `maxItemBytes` 时直通返回不撑爆缓存。
2. `server/internal/service/image_cache_http_test.go`：
   - 测试 `GET /api/files/goods/{id}` 返回 `Cache-Control: public, max-age=31536000, immutable` 与带引号强 `ETag`；
   - 测试携带匹配 `If-None-Match` 请求返回 `304 Not Modified` 且**不调用 `Images.Get`**；
   - 测试收款码等私有图片保持 `Cache-Control: private, no-store`。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/imagestore ./internal/service -run 'TestMemoryCache|TestPublicImageCacheHeaders' -v`
预期：FAIL。

- [ ] **步骤 3：实现 `imagestore.MemoryCache` 与 `serveBlob` 缓存模式区分**

1. 创建 `server/internal/imagestore/cache.go`：实现带 `sync.Mutex` + `container/list` 的有界 byte-budget LRU（默认总容量 `64 << 20`，单图上限 `MaxUploadBytes`）+ `golang.org/x/sync/singleflight` 去重，实现与 `Drive` 相同的 `Get(ctx, ref)` / `Put` / `HealthCheck` 包装接口（`Put` 与 `Sweep` 透传给底层 store）。
2. 修改 `server/internal/service/image_storage.go:80-108`：
   - 拆分 `serveBlobWithCache(w, r, reference, publicCacheable bool)`（保留 `serveBlob` 默认走 `private, no-store` 给 `PayCodeImage` / 后台图片）；
   - 当 `publicCacheable == true`（仅 `GoodImage`、`BundleImage`）：
     - 以 `sha256(reference)` 生成强 `ETag`（例如 `"<hex[:24]>"`）；
     - 校验 `r.Header.Get("If-None-Match")`，命中则立即写 `Cache-Control: public, max-age=31536000, immutable` + `ETag` + `w.WriteHeader(http.StatusNotModified)` 并返回；
     - 未命中才调用 `a.Images.Get`，返回前设置 `Cache-Control: public, max-age=31536000, immutable`、`ETag` 与 `X-Content-Type-Options: nosniff`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/imagestore ./internal/service ./cmd/stallink -v`
预期：PASS。

---

### 任务 12：公开目录检索分页限流优化 + 图片解码并发控制与 EXIF 剥离（server，M1 + M2 + M3）

**文件：**
- 修改：`server/internal/data/catalog_store.go:374-420`
- 修改：`server/internal/service/handlers.go:278-295`
- 修改：`server/internal/imagestore/compress.go:16-112`
- 修改：`server/internal/imagestore/compress_test.go:11-53`
- 创建：`server/internal/service/public_catalog_cache_test.go`

**设计要点：**
1. **M1（不动核心表结构，配合任务 3 的 `event_listings_good_shop_idx` 索引）**：
   - `PublicCatalogSearch` 入参增加 `limit`（默认 `48`，最大 `48`）与 `offset`（`page >= 1`），搜索词 `q` 按 rune 截断最多 `64` 字符；
   - 列表 SQL 不查询 `g.description`（核实 `web/src/components/catalog-explorer.tsx` 列表卡片只展示名称、店铺、活动、价格、库存、图片），将每行相关子查询 `SUM(reserved_qty)` 改为一次 `LEFT JOIN (SELECT good_id, COALESCE(SUM(reserved_qty), 0) AS reserved FROM event_listings GROUP BY good_id) el ON el.good_id = g.id`，并使用 `LIMIT $n + 1` 计算 `hasMore`；
   - `service.API.PublicCatalog` 增加 10 秒进程内短 TTL 缓存（按 `q|sort|page` 键缓存，上限 200 个查询，防热点刷库）。
2. **M2（图片解码内存/CPU 上限）**：
   - 在 `compress.go` 定义全局解码信号量 `decodeSem = make(chan struct{}, 2)`（最多同时 2 张大图解码，释放采用 `defer`）；
   - `maxPixels` 从 `24_000_000` 降为 `16_000_000`；
   - 缩放循环增加轮次上限 `maxResizeAttempts = 4`（超限返回 `ErrImageTooLarge`）；
   - PNG 编码（`compress.go:106`）从 `png.BestCompression` 改为 `png.DefaultCompression`。
3. **M3（强制剥离 EXIF / GPS 元数据）**：
   - 删除 `compress.go:90-92` 直接回传含 APP1/EXIF `original` JPEG 的分支，改为所有 JPEG 均经 `orientJPEG` 校正像素方向后使用 `jpeg.Encode` 重新编码（Go 标准库 `image/jpeg` 编码输出不含任何 EXIF/GPS APP1 段）；
   - 删除 `compress.go:62-64` `preservePixels == true`（收款码）直接返回带元数据 `original` 的分支：无论原图是 JPEG/PNG/WebP，均经解码后以纯像素重编码为不含 `eXIf`/文本块的 PNG。

- [ ] **步骤 1：编写失败的单元测试**

1. 在 `server/internal/imagestore/compress_test.go` 新增：
   - `TestCompressStripsEXIFFromJPEGAndPayCode`：构造含 `FF E1 Exif\x00\x00` 段（含 GPS 标记字符串）的合法 JPEG，分别测试 `preservePixels=false` 与 `preservePixels=true`，验证返回字节中不再出现 `Exif` 字节序列；
   - `TestCompressRejectsOver16MP`：验证超出 `16_000_000` 像素的图片头直接被拒且不分配像素内存。
2. 创建 `server/internal/service/public_catalog_cache_test.go`：验证超长 `q`（200 字符）被截断为 64、默认 `limit=48`、10 秒内同参数重复请求只查 DB 1 次。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/imagestore -run 'TestCompressStripsEXIF|TestCompressRejectsOver16MP' -v` 与 `go test ./internal/service -run TestPublicCatalogCache -v`
预期：FAIL。

- [ ] **步骤 3：实现 M1 列表查询/缓存 与 M2/M3 `compress.go` 加固**

按上述设计修改 `catalog_store.go`、`handlers.go` 与 `compress.go`。

- [ ] **步骤 4：运行测试验证通过**

运行：`go test ./internal/imagestore ./internal/service ./internal/data -v`
预期：PASS。

---

### 任务 13：前端收银台支付凭据、日报差额、错误码映射与审计详情展示（web，P1-3 & 审计展示）

> **分工边界：** `cancel` / `refund-confirm` / `cancel-refund` 弹窗与 `web/src/lib/order-actions.ts`（`cancelRefund`、`orderActionRequestBody`、`REASON_MAX_LENGTH = 500`、最小 2 字符）的对接改动归独立前端计划负责，本任务仅对齐错误码映射、P1-3 支付凭据/日报差额及审计详情展示。

**文件：**
- 修改：`web/src/lib/orders.ts`
- 修改：`web/src/lib/errors.ts`
- 修改：`web/src/app/counter/page.tsx:661-669, 923-949`
- 修改：`web/src/app/admin/audit/page.tsx`
- 修改：`web/src/app/admin/reports/page.tsx`（展示按收款方式应收/实收与差额）
- 测试：`web/tests/`

- [ ] **步骤 1：编写失败的单元测试**

测试前端错误码映射与提交载荷构造：
- `reason_required` 映射「请填写原因。」；`reason_too_short` 映射「原因至少需要 2 个字。」；`reason_too_long` 映射「原因不能超过 500 个字。」；
- `rate_limited` 映射中文提示「操作过于频繁，请稍后再试。」；
- QR 收款提交载荷包含 `paymentReference`（去首尾空白）与可选 `actualReceivedMinor`。

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm run lint && npx tsc --noEmit`（及 `node --test web/tests/*.test.cjs`）。

- [ ] **步骤 3：实现前端交互更新**

1. `web/src/lib/errors.ts`：补充 `reason_required`、`reason_too_short`、`reason_too_long`、`rate_limited` 中文映射（复用已有 `forbidden`、`idempotency_conflict`）。
2. `web/src/app/counter/page.tsx`：选择非现金扫码（`qr`）收款时展示「支付凭据 / 交易参考号（选填，最多 120 字）」输入框，并在已提交订单卡片中展示 `paymentReference`。
3. `web/src/app/admin/audit/page.tsx`：可读展示后端返回的 `reason` 及 `detail` 中的 `amountMinor`、`refundedAt`、`actorId`。
4. `web/src/app/admin/reports/page.tsx`：展示各支付方式 `expectedMinor`、`actualReceivedMinor` 与 `varianceMinor`（差额非 0 时醒目标注）。

- [ ] **步骤 4：运行前端校验与构建**

运行：`cd web && npm run lint && npx tsc --noEmit && npm run build`
预期：PASS，零 TypeScript 与 Lint 错误。

---

### 任务 14：Nonce CSP + 条件式 HSTS、移除死代码 `bundleTotal` 与 CSS 按需拆分（web，P2-4 + P2-5 + M5）

> **Next.js 16 规范核实：** 已查阅 `web/node_modules/next/dist/docs/01-app/02-guides/content-security-policy.md` 与 `16-proxy.md`，Next.js 16 使用 `web/src/proxy.ts` 导出 `proxy(request)`（不再使用旧版 `middleware.ts`）。

**文件：**
- 创建：`web/src/proxy.ts`
- 创建：`web/src/lib/csp.ts`
- 修改：`web/next.config.ts:5-16`（移出静态 CSP/冲突头，保留安全基础头；按 `isHTTPS` 决定 HSTS）
- 修改：`web/src/app/layout.tsx:7-10, 20-31`
- 修改：`web/src/components/workspace-shell.tsx`（迁入 `exhibition-workspace.css`）
- 修改：`web/src/lib/pricing.ts:25-51`（删除未使用的 `bundleTotal` 及其冗余导出）

- [ ] **步骤 1：编写失败的单元测试**

创建 `web/src/lib/csp.test.ts`（或用现有 `node --test` / `tsx` 测试工具）：
- 测试生成随机 `nonce`，策略串含 `script-src 'self' 'nonce-<val>' 'strict-dynamic'`、不包含 `unsafe-eval`，且**仅在生产且为 HTTPS 请求时**附加 `Strict-Transport-Security`（防止破坏本地 `http://127.0.0.1:3000` 访问）；
- 测试 `grep -rn "bundleTotal" web/src/` 除测试外零引用。

- [ ] **步骤 2：运行测试验证失败**

- [ ] **步骤 3：实现 `web/src/proxy.ts`、移除 `bundleTotal` 与按工作台/主题拆分 CSS**

1. `web/src/proxy.ts`：为页面请求生成 `crypto.randomUUID()` base64 nonce，注入请求头 `x-nonce` 与响应头 `Content-Security-Policy`（含 `base-uri 'self'`、`form-action 'self'`、`frame-ancestors 'none'`、`object-src 'none'`、`img-src 'self' data: blob:`、`style-src 'self' 'unsafe-inline'`、`script-src 'self' 'nonce-...' 'strict-dynamic'`，开发模式按需放开热更新）。
2. `web/src/app/layout.tsx`：移除顶层无条件导入 `@/app/exhibition-workspace.css`，改为在 `web/src/components/workspace-shell.tsx` 中导入（工作台页面 `/admin`、`/counter`、`/platform` 才加载工作台样式，客户公开页面零加载）；同时将 `shop.css` / `exhibition.css` 按服务端读取的 `theme`（`await getStorefrontTheme()`）通过条件动态路由/主题外层组件按需引入，使顾客首屏只加载当前启用的一套主题 CSS。
3. 删除 `web/src/lib/pricing.ts:25-51` 的未使用函数 `bundleTotal`（`pricing.ts` 全库已核实无调用方）。

- [ ] **步骤 4：运行 Next.js 构建与类型检查**

运行：`cd web && npm run lint && npx tsc --noEmit && npm run build`
预期：PASS；通过 `curl -sI http://127.0.0.1:3000/`（或 Next 构建产物/单测）确认响应含 `Content-Security-Policy` nonce 且本机 HTTP 不返回强制 HSTS。

---

### 任务 15：建立 GitHub Actions CI 与合规门禁（ci，P2-6 + H3 + M7）

**文件：**
- 创建：`.github/workflows/ci.yml`

- [ ] **步骤 1：编写 `.github/workflows/ci.yml`**

配置在 `push`（`main`）与 `pull_request` 触发以下并行 Job：
1. **`server-unit-and-vuln`**：
   - `actions/setup-go` 读取 `go-version-file: server/go.mod`（确保 `>= 1.27.4`）；
   - 执行 `cd server && go vet ./... && go test ./...`；
   - 执行 `go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...`（H3 漏洞零容忍门禁）；
   - 执行迁移配对检查脚本：校验 `server/internal/data/migrations/0*.sql` 从 `014_*.sql` 起均在 `migrations/down/` 有同名 `.down.sql` 回滚脚本，且检查 `Dockerfile*` 若存在则基础镜像 `>= 1.27.4`。
2. **`server-integration`**：
   - `services:` 启动 `postgres:16-alpine`（`127.0.0.1:5432`）与 `redis:7-alpine`（`127.0.0.1:6379`）；
   - 注入 `STALLINK_TEST_DATABASE_URL`、`STALLINK_TEST_REDIS_ADDR` 跑 `go test -tags integration ./internal/data ./internal/service`。
3. **`web-check`**：
   - `actions/setup-node`（Node 22）+ `npm ci` + `npm run lint` + `npx tsc --noEmit` + `npm run build`。

- [ ] **步骤 2：本地逐项模拟跑通 CI 命令**

```bash
cd server && go vet ./... && go test ./... && go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
cd ../web && npm run lint && npx tsc --noEmit && npm run build
```

---

### 任务 16：修复 `migrations_integration_test.go` 连接池关闭早于 `t.Cleanup` 导致的测试 schema 残留（server，第 8.2 节附带发现）

> **编号说明：** 本任务放在修复计划末尾（任务 16），确保任务 1–15 编号与并行 `vNext` 计划的引用完全保持不变。
> **数据库约束：** 验收与测试**仅允许连接 `docker-compose.yml` 起的本机 Postgres（`127.0.0.1:5432/stallink`）**。

**文件：**
- 修改：`server/internal/data/migrations_integration_test.go:32-42, 53, 133-142, 152`

**问题核实：** 已核实 `server/internal/data/migrations_integration_test.go` 两处用例：
1. `TestMigrationsFreshConcurrentAndIdempotent`：第 `32` 行 `defer adminPool.Close()`、第 `53` 行 `defer testPool.Close()`，而 `DROP SCHEMA` 注册在第 `38-42` 行的 `t.Cleanup`；
2. `TestMigrationBackfillsLegacyLinesBySnapshotVersion`：第 `133` 行 `defer adminPool.Close()`、第 `152` 行 `defer testPool.Close()`，而 `DROP SCHEMA` 注册在第 `138-142` 行的 `t.Cleanup`。
Go 会先在函数返回时执行 `defer`，再按 LIFO 顺序执行 `t.Cleanup`；因此 `adminPool` 在 `t.Cleanup` 跑 `DROP SCHEMA` 前就已被关闭，导致静默失败（`_, _ = adminPool.Exec(...)`）并在本机 `stallink` 库残留空 `stallink_it_*` 与 `stallink_legacy_it_*` schema。

- [ ] **步骤 1：编写测试断言 `DROP SCHEMA` 失败即报错**

先确认修复前直接运行测试会在本机 Postgres 新增残留 schema：
```bash
STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable'   go test -tags integration -count=1 ./internal/data -run 'TestMigrationsFreshConcurrentAndIdempotent|TestMigrationBackfillsLegacyLinesBySnapshotVersion' -v
```

- [ ] **步骤 2：改用 LIFO `t.Cleanup` 按「关 `testPool` → `DROP SCHEMA`（失败用 `t.Errorf` 暴露）→ 关 `adminPool`」顺序清理**

在 `TestMigrationsFreshConcurrentAndIdempotent`（第 `32-53` 行）与 `TestMigrationBackfillsLegacyLinesBySnapshotVersion`（第 `133-152` 行）中：
1. 删除第 `32`、`53`、`133`、`152` 行的 `defer adminPool.Close()` 与 `defer testPool.Close()`；
2. 建立 `adminPool` 后立即注册：`t.Cleanup(adminPool.Close)`（最后执行）；
3. `CREATE SCHEMA` 后注册删库清理（使用 `t.Errorf` 暴露错误，禁止吞错）：
   ```go
   t.Cleanup(func() {
   	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
   	defer cleanupCancel()
   	if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
   		t.Errorf("drop test schema %s: %v", schema, err)
   	}
   })
   ```
4. （第二个用例 `TestMigrationBackfillsLegacyLinesBySnapshotVersion` 变量名为 `schemaName`，同样替换 `defer` 并在 `t.Errorf` 中打印 `schemaName`）。
5. 建立 `testPool` 后立即注册：`t.Cleanup(testPool.Close)`（按 LIFO 最先执行，释放占用该 schema 的连接后再执行 `DROP SCHEMA`）。

- [ ] **步骤 3：本机 docker-compose Postgres 验收（确认零新增残留）**

1. 记录测试前本机 `stallink` 库中测试 schema 数量；
2. 执行集成测试（仅本机库）：
   ```bash
   STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable'      go test -tags integration -count=1 ./internal/data -run 'TestMigrationsFreshConcurrentAndIdempotent|TestMigrationBackfillsLegacyLinesBySnapshotVersion' -v
   ```
3. 查询本机 `information_schema.schemata` 验证本次测试创建的 `stallink_it_*` 与 `stallink_legacy_it_*` 均已彻底清除（零新增残留）；经 Kel 同意后也可顺便清理第 8.2 节记录的历史 10 个残留空测试 schema。

## 六、整体验收清单与并行协作提示

1. **全量验证命令（本机）：**
   - 后端单测 & 漏洞扫描：`cd server && go vet ./... && go test -race ./... && go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...`
   - 后端集成测试（仅本机 Docker Postgres/Redis）：`STALLINK_TEST_DATABASE_URL='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable' go test -tags integration ./internal/data ./internal/service`
   - 前端静态检查与构建：`cd web && npm run lint && npx tsc --noEmit && npm run build`
2. **上线顺序与远程库人工门禁（`⚠️ 需要 Kel 手动确认`）：**
   - **任务 1（C1）与任务 2（H1）必须绑定合并、同步发布上线**；发布后先确认 `orphan_cleanup_enabled: false` 与迁移守卫生效。
   - 任何云端/远程数据库执行 `stallink migrate` 或 `014` 升级/回滚，必须经 Kel 手动确认目标 `<host>/<db>` 并先做好库级快照。
   - **C1 历史文件分环境迁移**（已确认各环境与本机此前共用同一 Drive 根目录）：在运维侧暂停清理、各环境配置好独立根目录后，由 Kel 手动批准并执行历史文件按环境迁移/打标。
   - **H2 运维操作**（本机 Nacos 缓存清理、密钥轮换与缩窄 OAuth Scope 至 `drive.file`）及 **M4 Drive 网页权限核查** 不在代码提交范围，由运维/安全侧按报告手工闭环。
3. **与 `vNext` 并行分支协同：**
   - 本分支只占用迁移号 **`014`**；`vNext` 从 **`015`** 开始。
   - 图片缓存本计划只改 HTTP 缓存头与 Go 内存 LRU；`vNext` `Task S10`（迁移 `020`）可在不冲突的前提下追加 `media_token` 与本地磁盘缓存。
   - 公开目录本计划只做 `LIMIT 48` + 去 `description` + 10s 缓存 + 联合覆盖索引；`vNext` `Task S13`（迁移 `021`）在此基础上继续叠加 `allow_public_catalog` 开关与 `pg_trgm` GIN 索引。
