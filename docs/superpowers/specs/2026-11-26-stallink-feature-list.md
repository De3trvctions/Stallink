# Stallink 功能清单与接口约定（PRD vNext 配套）

- **日期**：2026-11-26（已合入 Kel 2026-11-26 全部拍板口径及与 `2026-11-26-stallink-review-fixes.md` 的协调边界）
- **对应 PRD**：`docs/superpowers/specs/2026-11-26-stallink-prd-vnext.md`
- **前置修复 plan 协调边界**：`docs/superpowers/plans/2026-11-26-stallink-review-fixes.md` 先行合入并负责：**任务 1**（C1 Drive 孤儿清理跨环境误删防护与独立根目录校验）、**任务 2**（启动与 Nacos 热更新自动迁移安全守卫 + 迁移超时/校验和）、**任务 3**（迁移 **`014`** 新增 `orders.payment_ref` 等字段及 `server/internal/data/migrations/down/` 回滚目录隔离规范）、**任务 5**（移除 `readGoodPool` 的 `shopID == 0` 跳过店铺过滤）、**任务 6**（`releaseOrderReservations` 全局锁序修复）、**任务 7**（**P1-1** `cancel`/`refund-confirm` 与一步全额 `cancel-refund` 同店角色校验、`pending` 取消必填、统一 `2–500` Unicode 字符 `reason` 校验及审计）、**任务 8**（分布式/降级限流与可信代理真实 IP，含登录/注册与公开基础限流）、**任务 10**（自动审核开关与外观切换平台级审计）、**任务 11**（公开图片响应缓存头与服务端有界内存缓存）、**任务 12**（公开目录检索分页限流优化 + 图片解码并发控制与 EXIF 剥离）。**本清单及 vNext 计划对上述事项显式标注依赖的修复 plan 任务编号，不重复编写实现；vNext 迁移从 `015` 开始至 `023`，所有 `.down.sql` 回滚脚本统一放置于 `server/internal/data/migrations/down/`。**
- **性能基线**：P0 核心交付需支撑单场峰值 **1,000 QPS**、开单 **P95 < 500ms**。

---

## 一、功能统计总览

| 优先级 | 数量 | 涵盖核心内容 |
|---|---|---|
| **P0** | **14 项**（`F-01` ~ `F-14`） | 共享池+上限/净锁定保底（口径 2）、过期预留实时剔除、2s 锁超时与 `503` 重试、资格失效自动撤单、审核与套餐规则对齐、~~现金找零~~（F-07 已作废（Kel 2026-11-26））、部分退款（同店 `sales` 可退+必填原因+审计）+报损+跨日对账、活动级时区、真离线现金开单（4h 断网限制）与 `oversold` 闭环、媒体缓存防遍历、公开高频接口限流 |
| **P1** | **10 项**（`F-15` ~ `F-24`） | 全站目录 Opt-in（默认 false）与 `pg_trgm` 分页检索、活动预售开关、顾客收藏/预选清单（不占库存）/凭证夹、中英双语 i18n、前后快照审计增强、过期订单补录恢复、多终端号与 QR 流水号（复用 `014`）、超复杂购物车降级引导 |
| **P2** | **4 项**（`F-25` ~ `F-28`） | 公开目录 Redis 缓存、幂等/报价表定期清理、多币种日报 UI 调整、废弃遗留 `/api/public/landing` |
| **合计** | **28 项** | 其中 vNext 新增数据库迁移的功能共计 **15 项**（对应有序迁移脚本 `015` ~ `023`；`F-23` 的 `payment_ref` 直接复用修复 plan 的迁移 `014`） |

---

## 二、全量功能清单

### 【P0】核心交易、库存、风控与架构基线（14 项）

| 编号 | 功能名称 | 归属 | 优先级 | 前置依赖 | 涉及数据库迁移 | 核心行为与范围摘要 |
|---|---|---|---|---|---|---|
| **F-01** | 共享物理池 + 单场上限（`max_qty`）与净锁定保底（`min_reserve_qty`，口径 2） | 两者 | P0 | 无 | **是**（迁移 `015`：`event_listings` 新增 `max_qty`、`min_reserve_qty`；`inventory_reservations` 新增 `floor_reserved_qty`） | 设定 `min_reserve_qty=Y` 从设定时刻起净锁定 Y 件；全店保底之和+有效预占 > `total_stock` 或超剩余上限时拒绝设置；本场下单优先消耗自身保底并记录 `floor_reserved_qty`；活动结束/下架立即清零释放。 |
| **F-02** | 过期预留实时剔除与 `floor_reserved_qty` 退回（锁序修复依赖修复 plan 任务 6） | server | P0 | F-01，修复 plan 任务 6 | 否 | 所有可售库存读取过滤 `expires_at > now()`；复用修复 plan 任务 6 确定的 `releaseOrderReservations` 全局锁序，增量完成未结束活动 `floor_reserved_qty` 退回与小批短事务扫过期单。 |
| **F-03** | 事务 2s 锁超时、5s 上限、`503` 退避、死锁重试、去 `shops` 阻塞锁与连接池调优 | server | P0 | 无 | 否 | 写命令事务加 `lock_timeout='2s'`、`statement_timeout='5s'`；锁等待超时返回 `503 command_in_progress` + `Retry-After: 1`；死锁/串行化错误重试最多 2 次；移除下单事务对 `shops` 的 `FOR SHARE` 锁；连接池 ≥50 连接支撑 1,000 QPS / P95<500ms。 |
| **F-04** | 付款前资格失效自动取消订单并立即释放预留 | 两者 | P0 | F-02 | 否 | `Pay` 事务若检测到店铺停用、活动已结束或商品下架/撤审，在同一事务将订单转为 `cancelled`、释放预占库存、写入幂等重放记录并返回明确状态；前端自动清挂单并引导重新开单。 |
| **F-05** | 商品与套餐审核规则修正（补 `name` 重审、禁 `rejected` 复活、调价不误杀套餐） | server | P0 | 无 | 否 | 修改 `name`/`description`/图片才重审；改价格/阶梯优惠/库存/上下架不重审且不打回关联套餐；`rejected` 实体即便开启 `auto_approve` 也必须进 `pending` 人工复核。 |
| **F-06** | 固定套餐规则对齐（1 张主图、无阶梯价）与保存期连通组上限校验 | 两者 | P0 | 无 | 否 | 明确禁止套餐多图（≤1 张）与 `priceTiers`（空）；保存/上架套餐时校验无向二分图连通分量（≤20 SKU 且 ≤10 套餐），超限即拦截；前端移除套餐多图误导提示。 |
| ~~**F-07**~~ | ~~现场现金收款实收金额输入与实时找零计算~~ **已作废（Kel 2026-11-26）** | web | ~~P0~~ | 无 | 否 | ~~收银台与现场订单付款弹窗增加“实收现金”输入与快捷面额按钮，实时计算找零并把真实 `cashReceivedMinor` 传给后端（不再写死等于总额）。~~ **注：** Kel 2026-11-26 决定作废（对应 vNext web plan 的 W5）。原因：和这一轮工作台打磨去掉收银台实收、找零 UI 的方向相反。后端继续接收 `cashReceivedMinor`（F-11 离线现金同步要用），这一条不变。 |
| **F-08** | 取消、部分商品/数量退款、报损不回库与套餐破坏重算（同店角色校验依赖修复 plan 任务 7 / P1-1） | 两者 | P0 | F-01，修复 plan 任务 7（P1-1） | **是**（迁移 `017`：新增 `order_refunds`、`order_refund_lines`；`orders` 新增 `paid_at`、`cancel_reason`、`partial_refunded`、`refunded_total_minor`；`inventory_movements` 类型增加 `'damage'`） | 整单一步全额取消并退款（`/cancel-refund`，全回库）及 `pending` 取消原因强制（2–500 Unicode 字符）归修复 plan 任务 7（`P1-1`）；本项按 Q3=B 复用任务 7 的同店 `sales`/`shop_admin` 归属校验与 `reason` 口径（`reason_required`/`reason_too_short`/`reason_too_long`），扩展部分退款、逐行拆分 `restockQty` 与 `damagedQty`、破坏套餐重算上限与完整审计。 |
| **F-09** | 跨日退款日报冲减与 `paid_at`/`refunded_at` 对账改造 | 两者 | P0 | F-08, F-10 | **是**（迁移 `018`：`daily_reports` 扩展毛实收、退款冲减额、净实收及当日退款关联明细快照字段） | 按活动时区营业日切分：当日毛实收按 `paid_at` 聚合，当日退款按 `order_refunds.refunded_at` 聚合为退款冲减（关联原单号与原支付日），不修改历史旧日快照。 |
| **F-10** | 活动级 IANA 时区与营业日三端统一判定 | 两者 | P0 | 无 | **是**（迁移 `016`：`events` 新增 `timezone text NOT NULL DEFAULT 'Asia/Kuala_Lumpur'`） | 店主创建/编辑活动时选择活动时区；后端统一按 `event.timezone` 判定活动结束与日报营业日，响应下发 `isEndedNow` / `eventToday`。 |
| **F-11** | 真离线现金开单、4h 签名防篡改（HMAC + `performance.now()`）、离线同步与 Playwright E2E 验收 | 两者 | P0 | F-01, F-03, ~~F-07~~（F-07 已作废，去掉这条依赖） | **是**（迁移 `019`：放宽 `orders.number` CHECK 兼容 10 位 `OL...`；`orders` 新增 `is_offline`、`terminal_id`、`offline_created_at`、`offline_catalog_revision`、`oversold_status`、`oversold_qty`；**`payment_ref` 复用迁移 `014` 不重复添加**） | 后端签发含 `shop_id`+`terminal_id`+`lastOnlineSyncAt` 的 HMAC 租约（密钥仅存于服务端配置）；前端以 `performance.now()` + 租约校验防回拨/篡改（失败即按已超时拦截）；断网仅限现金、不设金额上限、限 4 小时，联网按 `offline_created_at` 顺序幂等同步并验签；前端 P0 含 Playwright 断网自动验收。 |
| **F-12** | 离线同步超卖（`oversold`）告警与补货核销/退款闭环 | 两者 | P0 | F-11, F-08 | **是**（迁移 `019` 新增 `oversold_order_lines` 记录逐行实际扣减与缺口） | 同步遇到共享池或上限不足时，订单照常记账但标 `oversold_status = 'pending_resolution'`、库存钳零、推送全店告警；支持店主“调增库存核销”或“联系顾客退款（仅回补实际曾扣部分）”。 |
| **F-13** | 媒体本地落盘缓存与防枚举令牌（衔接修复 plan 任务 1/11/12） | server | P0 | 修复 plan 任务 1, 11, 12 | **是**（迁移 `020`：`good_images`、`bundle_images`、`pay_codes` 新增唯一 `media_token` 字段） | 响应缓存头与有界内存缓存由修复 plan 任务 11 落地、解码并发与 EXIF 剥离由任务 12 落地、Drive 防护由任务 1 落地；vNext 增量把公开文件 URL 升级为不可枚举 `media_token` 并补齐本地磁盘二级缓存抗重启穿透 Drive，收款码支持前端 SW 缓存。 |
| **F-14** | 报价接口与高并发目录突发限流（复用修复 plan 任务 8/12 基础设施） | server | P0 | 修复 plan 任务 5, 8, 12 | 否 | 分布式/降级限流与可信代理真实 IP（含登录/注册及既有公开匿名限流）由修复 plan 任务 8 负责，公开目录分页限流由任务 12 负责，去 `shopID==0` 由任务 5 负责；vNext 直接复用任务 8 限流器增量补齐 `/api/staff/order-quotes` 报价限流与检索突发阈值。 |

---

### 【P1】双轨目录、顾客实用能力、双语与管控增强（10 项）

| 编号 | 功能名称 | 归属 | 优先级 | 前置依赖 | 涉及数据库迁移 | 核心行为与范围摘要 |
|---|---|---|---|---|---|---|
| **F-15** | 全站目录收录开关（`allow_public_catalog` 默认 `false`）、排序规则与 `pg_trgm` 分页搜索 | 两者 | P1 | 修复 plan 任务 5, 12 | **是**（迁移 `021`：`shops` 新增 `allow_public_catalog boolean NOT NULL DEFAULT false`；`goods`/`shops` 建 `pg_trgm` GIN 索引） | 承接修复 plan 任务 5（去 `shopID==0`）与任务 12（公开目录分页优化）；默认 opt-in 不收录；店主主动开启后才进入全站目录；按“进行中 > 有库存 > 开展时间/更新”排序，强制 `page`/`pageSize` 分页（每页 ≤50）。 |
| **F-16** | `scheduled` 活动显式预售开关（`allow_presale`）与开单守卫 | 两者 | P1 | F-10 | **是**（迁移 `016`：`events` 新增 `allow_presale boolean NOT NULL DEFAULT false`） | 取代服务端全局配置 `AllowScheduledOrders`；`scheduled` 活动默认仅浏览不可开单，仅当活动级 `allow_presale = true` 时才允许开单与收款。 |
| **F-17** | 顾客收藏夹（摊位与商品/套餐收藏） | 两者 | P1 | F-15 | **是**（迁移 `022`：新增 `customer_favorites` 表） | 顾客登录后可在公开摊位或详情页点击收藏/取消收藏，在“我的收藏”页统一查看实时可用库存与开展状态。 |
| **F-18** | 顾客逛展预选清单（不占库存，生成短码/QR 供收银台一键导入） | 两者 | P1 | F-01 | **是**（迁移 `022`：新增 `customer_wishlists`、`customer_wishlist_items` 表） | 顾客勾选某场活动拟购商品/套餐生成预选清单（**明确不占库存、不锁价**），展示 6 位清单码与二维码；收银台扫码/输码即可一键回填购物车并触发最新权威报价。 |
| **F-19** | 顾客电子取货凭证夹（订单关联与状态查询） | 两者 | P1 | 无 | **是**（迁移 `022`：`orders` 新增可空 `customer_user_id bigint REFERENCES users(id)` 及索引） | 收银开单时可通过导入预选清单或输入顾客绑定码关联顾客账号；顾客在“我的订单”查看取货号 `number`、取货状态与明细快照。 |
| **F-20** | 中英双语（`zh-CN` / `en`）全站 i18n 支持 | 两者 | P1 | 无 | **是**（迁移 `023`：`users` 新增 `locale text NOT NULL DEFAULT 'zh-CN'`） | 后端错误响应标准化为纯错误码；前端抽离中英双语字典，涵盖全站导航、收银台、后台、小票凭证、日报 CSV 导出表头及日期数字本地化；支持一键切换语言。 |
| **F-21** | 审计前后快照与终端/IP 增强（依赖修复 plan 任务 3/10） | 两者 | P1 | F-08，修复 plan 任务 3, 10 | **是**（迁移 `023`：`audit_logs` 新增 `before_json`、`after_json`、`reason`、`terminal_id`、`client_ip`） | 依赖修复 plan 任务 3（迁移 `014` 可空 `shop_id` 与 `detail`）及任务 10 落地的自动审核开关/外观切换平台审计；vNext 扩展变更前后 JSON 快照、原因、操作金额（`order_refunds`）、终端与 IP 留痕，后台支持筛选与对比查看。 |
| **F-22** | 过期未付订单延迟到账人工恢复收款 | 两者 | P1 | F-02, F-08 | 否 | 当顾客扫码转账晚于 15 分钟过期时，允许店员在“现场订单”对 `expired` 订单发起恢复收款：校验当前池库存充足即扣库转 `paid`（记原因与审计），不足则引导退款。 |
| **F-23** | 多终端收银标识、QR 流水参考号前端录入（字段复用 `014`）与低库存实时推送 | 两者 | P1 | F-11，修复 plan 任务 3（`014`） | 否（`payment_ref` 复用修复 plan 任务 3 迁移 `014`，`terminal_id` 在迁移 `019` 新增） | 收银设备自动持久化并可命名 `terminalName`；QR 收款弹窗可选填支付流水号透传至 `014` 字段；收银台目录轮询由 20s 缩短至 5s 并订阅 SSE 实时通知。 |
| **F-24** | 超限复杂购物车（DP > 100,000 状态）友好降级与拆单引导 | 两者 | P1 | F-06 | 否 | 若极端跨单品购物车仍触发 `422 pricing_complexity_exceeded`，报价接口返回冲突组提示及单品降级报价选项，收银台引导分两单结算或切回“单品优惠模式”。 |

---

### 【P2】性能优化、数据归档与清理（4 项）

| 编号 | 功能名称 | 归属 | 优先级 | 前置依赖 | 涉及数据库迁移 | 核心行为与范围摘要 |
|---|---|---|---|---|---|---|
| **F-25** | 公开目录 Redis 短期分层缓存与读写分离支持 | server | P2 | F-15 | 否 | 缓存热门摊位详情与全站首页目录 10~30s，高并发读取走只读连接池，巩固 1,000 QPS 场景下的低延迟响应。 |
| **F-26** | `order_quotes` 与 `order_commands` 定期过期清理与归档任务 | server | P2 | 无 | 否 | 后台定时清理超过 24 小时的过期报价及超过保留期（如 30 天）的历史命令快照，控制核心表体积。 |
| **F-27** | 多币种日报顶部并列汇总展示 & 全站报错提示对齐 | web | P2 | F-09, F-20 | 否 | 跨币种店铺的日报顶部按币种并列呈现净收款（替代单币种 Headline）；清除 `errors.ts` 中旧配额制残留错误文案。 |
| **F-28** | 下线历史遗留无效接口 `/api/public/landing` | server | P2 | F-15 | 否 | 移除多店场景下必报 404 的遗留接口，统一由 `/api/public/catalog` 承接首页入口。 |

---

## 三、涉及数据库迁移的功能汇总（vNext 迁移 `015` ~ `023`，另复用修复 plan `014`）

| 迁移文件（位于 `server/internal/data/migrations/`；对应回滚均放在 `server/internal/data/migrations/down/NNN_*.down.sql`） | 关联功能编号 | 变更对象与核心 DDL 摘要 |
|---|---|---|
| （前置依赖）`014_*.sql`（修复 plan 任务 3） | `F-21`, `F-23` 复用 | 修复 plan 任务 3 负责为 `orders` 增加支付凭证/参考号字段（`payment_ref`）及 `audit_logs` 平台审计扩展字段等，vNext 不重复创建。 |
| `015_listing_cap_and_floor.sql` | `F-01` | `event_listings` 新增 `max_qty`、`min_reserve_qty`；`inventory_reservations` 新增 `floor_reserved_qty`；加非负与上限检查约束。 |
| `016_event_timezone_and_presale.sql` | `F-10`, `F-16` | `events` 新增 `timezone`、`allow_presale`。 |
| `017_partial_refunds_and_damage.sql` | `F-08` | 新增 `order_refunds`、`order_refund_lines`；`orders` 新增 `paid_at`、`cancel_reason`、`partial_refunded`、`refunded_total_minor`；`inventory_movements` 类型新增 `'damage'`。 |
| `018_daily_report_cashflow_offset.sql` | `F-09` | `daily_reports` 扩展 `gross_paid_minor`、`refund_offset_minor`、`net_received_minor`、`refund_details jsonb`。 |
| `019_offline_cash_orders_and_oversold.sql` | `F-11`, `F-12`, `F-23` | 放宽 `orders_number_check` 兼容 `OL[A-Z0-9]{8}`；`orders` 新增 `is_offline`、`terminal_id`、`offline_created_at`、`offline_catalog_revision`、`oversold_status`、`oversold_qty`（**不再加 `payment_ref`**）；新增 `oversold_order_lines`。 |
| `020_media_tokens.sql` | `F-13` | `good_images`、`bundle_images`、`pay_codes` 新增唯一 `media_token text NOT NULL`（存量自动回填随机 hex token）。 |
| `021_public_catalog_optin_and_search.sql` | `F-15` | `shops` 新增 `allow_public_catalog boolean NOT NULL DEFAULT false`；启用 `pg_trgm` 扩展并在 `goods(name)`、`shops(name)` 创建 GIN 索引。 |
| `022_customer_favorites_wishlists_orders.sql` | `F-17`, `F-18`, `F-19` | 新增 `customer_favorites`、`customer_wishlists`、`customer_wishlist_items`；`orders` 新增 `customer_user_id`。 |
| `023_audit_snapshots_and_user_locale.sql` | `F-20`, `F-21` | `users` 新增 `locale`；`audit_logs` 新增 `before_json`、`after_json`、`reason`、`terminal_id`、`client_ip`。 |

---

## 四、前后端接口约定明细（覆盖全部「两者」功能）

> 说明：以下覆盖全部归属为 `两者` 的功能接口。所有订单交易写接口**强制要求 `Idempotency-Key`（UUID）**。

### 1. F-01｜活动上架限售上限与净锁定保底（改造现有接口）
- **方法与路径**：`POST /api/staff/goods/{goodID}/listings` 及 `PATCH /api/staff/listings/{listingID}`
- **请求关键字段**：`eventId`（创建时）、`unitPriceMinor`、`priceTiers`、`isListed`、`maxQty`（可空正整数，`null` 不限）、`minReserveQty`（非负整数，默认 `0`）
- **响应关键字段**：`id`, `eventId`, `goodId`, `unitPriceMinor`, `priceTiers`, `isListed`, `maxQty`, `minReserveQty`, `soldQty`, `reservedQty`, `availableQty`, `capReached`
- **错误码**：`invalid_listing_quota`、`stock_exceeded`
- **幂等键**：不强制（采用 `revision` 并发乐观校验，冲突返回 `409 conflict`）

### 2. F-04｜付款前资格失效自动撤单（改造现有接口）
- **方法与路径**：`POST /api/staff/orders/{orderID}/pay`
- **请求关键字段**：`payCodeId`、`cashReceivedMinor`（现金必传；**注：** F-07 已作废（Kel 2026-11-26），前端不再录入实收，后端继续接收该字段，是否改为“现金码没带就按总额记”见工作台打磨 plan K-Q10，待 Kel 确认）、`paymentRef`（QR 选填，落入修复 plan 任务 3 迁移 `014` 字段）、`terminalId`
- **响应关键字段**：
  - 正常成交：`200` 返回订单对象（`status: "paid"`, `paidAt`, `cashChangeMinor`, `paymentRef`）
  - 资格失效自动取消：`409` 返回 `{"error":"order_cancelled_eligibility_lost","order":{"id":"...","status":"cancelled","reservationMode":"released"}}`
- **错误码**：`order_cancelled_eligibility_lost`、`reservation_expired`、`invalid_cash_received`、`pay_code_currency_mismatch`、`command_in_progress`（`503` + `Retry-After: 1`）
- **幂等键**：**强制** `Idempotency-Key`

### 3. F-06｜套餐创建/编辑与连通组校验（改造现有接口）
- **方法与路径**：`POST /api/staff/bundles`、`PATCH /api/staff/bundles/{bundleID}`
- **请求关键字段**：`name`、`description`、`components: [{ goodId, quantity }]`、`listings: [{ eventId, unitPriceMinor, isListed }]`、上传图片最多 1 张；**严禁传 `priceTiers`**
- **响应关键字段**：`id`, `name`, `reviewStatus`, `imageURL`, `components`, `listings`
- **错误码**：`invalid_bundle`、`bundle_pricing_group_limit_exceeded`、`bundle_components_not_eligible`
- **幂等键**：不强制

### 4. F-08｜取消订单、一步全额取消并退款、部分退款与报损不回库（改造 + 新增接口）
- **一步取消并退款（全额+全回库，归修复 plan 任务 7）**：`POST /api/staff/orders/{orderID}/cancel-refund`
  - **请求关键字段**：`reason`（**必填**，2–500 Unicode 字符，仅限 `paid`/`fulfilled`）
  - **响应关键字段**：`order`（直接置 `status: "cancelled"` 并写入 `refundedAt`）
  - **错误码**：`reason_required`、`reason_too_short`、`reason_too_long`、`invalid_status`、`forbidden`、`idempotency_conflict`、`command_in_progress`（503）
  - **幂等键**：**强制** `Idempotency-Key`
- **取消接口（`pending` 同样必填，保留两步兼容存量 `refund_required`）**：`POST /api/staff/orders/{orderID}/cancel`
  - **请求关键字段**：`reason`（**必填**，2–500 Unicode 字符）、`terminalId`
  - **响应关键字段**：`order`（`pending` → `cancelled`；`paid`/`fulfilled` → `refund_required`）
  - **错误码**：`reason_required`、`reason_too_short`、`reason_too_long`、`invalid_status`、`forbidden`（跨店拦截归修复 plan 任务 7 / P1-1）、`idempotency_conflict`、`command_in_progress`（503）
  - **幂等键**：**强制** `Idempotency-Key`
- **部分退款试算**：`POST /api/staff/orders/{orderID}/refund-preview`
  - **请求关键字段**：`items: [{ orderLineId, refundQty }]`
  - **响应关键字段**：`maxRefundMinor`, `recommendedRefundMinor`, `remainingDueMinor`, `discountClawbackMinor`, `lines`
- **部分退款确认**：`POST /api/staff/orders/{orderID}/refunds`（并保留 `/refund-confirm` 处理存量 `refund_required`）
  - **请求关键字段**：`reason`（**必填**，2–500 Unicode 字符）、`terminalId`、`refundMinor`、`items: [{ orderLineId, restockQty, damagedQty }]`
  - **响应关键字段**：`refund: { id, orderId, refundMinor, reason, refundedBy, refundedAt, lines }`, `order`
  - **错误码**：`reason_required`、`reason_too_short`、`reason_too_long`、`invalid_refund_qty`、`refund_exceeds_max`、`invalid_status`、`forbidden`、`idempotency_conflict`、`command_in_progress`（503）
  - **幂等键**：**强制** `Idempotency-Key`

### 5. F-09｜跨日退款日报（改造现有接口）
- **方法与路径**：`GET /api/admin/daily-reports`、`POST /api/admin/daily-reports`、`GET /api/admin/daily-reports/export`
- **请求关键字段**：`date`（`YYYY-MM-DD`）、`eventId`（可选）
- **响应关键字段**：`grossPaidMinor`, `cashPaidMinor`, `qrPaidMinor`, `cashChangeMinor`, `refundOffsetMinor`, `netReceivedMinor`, `refundItems: [{ orderId, orderNumber, originalPaidAt, refundMinor, reason }]`
- **错误码**：`invalid_date`、`report_not_generated`

### 6. F-10 & F-16｜活动时区与预售开关（改造现有接口）
- **方法与路径**：`POST /api/staff/events`、`PATCH /api/staff/events/{eventID}`、`GET /api/staff/events`
- **请求关键字段**：`name`, `startsOn`, `endsOn`, `city`, `venue`, `boothNumber`, `currency`, `status`, `timezone`, `allowPresale`
- **响应关键字段**：同上 + `eventToday`, `isEndedNow`, `isOpenForOrder`
- **错误码**：`invalid_timezone`、`invalid_event`、`invalid_date`

### 7. F-11｜离线租约签发与真离线现金开单异步同步（新增接口）
- **接口 A（HMAC 离线租约下发）**：`POST /api/staff/offline-lease`
  - **请求关键字段**：`terminalId`（必填，3–40 字符）
  - **响应关键字段**：`shopId`, `terminalId`, `lastOnlineSyncAt`（ISO-8601 UTC）, `expiresAt`（`lastOnlineSyncAt + 4h`）, `signature`（`HMAC-SHA256(serverSecret, shopId|terminalId|lastOnlineSyncAt|expiresAt)` hex，密钥仅存于服务端运行配置，绝不进代码库）
  - **错误码**：`invalid_terminal`、`offline_signing_unconfigured`（503）
  - **幂等键**：不强制（每次联网刷新/同步后签发最新租约）
- **接口 B（离线现金单同步）**：`POST /api/staff/orders/offline-sync`
  - **请求关键字段**：`clientOrderId`（UUID）、`offlineNumber`（10 位 `OL...`）、`terminalId`、`offlineLease: { lastOnlineSyncAt, expiresAt, signature }`、`eventId`、`boothName`、`customerUserId`（可选）、`offlineCreatedAt`、`offlineCatalogRevision`、`payCodeId`（须为 `cash`）、`cashReceivedMinor`、`cashChangeMinor`、`totalMinor`、`lines`
  - **响应关键字段**：`order`（含 `id`, `number`, `status: "paid"`, `isOffline: true`, `oversoldStatus`, `oversoldQty`, `priceRevisionDrift`）、`refreshedLease`（同步成功后顺带下发最新签名的租约；若原租约验签异常则记高危审计并拒绝下发新租约）
  - **错误码**：`invalid_offline_order`、`invalid_pay_code`、`offline_lease_tampered`、`idempotency_conflict`、`command_in_progress`（503 + `Retry-After: 1`）
  - **幂等键**：**强制** `Idempotency-Key` + `(shop_id, client_order_id)` 唯一约束

### 8. F-12｜超卖（`oversold`）异常列表与补货核销（新增接口）
- **列表查询**：`GET /api/staff/orders/oversold?status=pending_resolution|resolved`
- **补货核销**：`POST /api/staff/orders/{orderID}/oversold/restock-resolve`
  - **请求关键字段**：`reason`（必填）
  - **响应关键字段**：`order`（`oversoldStatus: "resolved_restocked"`, `oversoldQty: 0`）
  - **错误码**：`not_enough_stock`、`invalid_status`、`invalid_reason`
  - **幂等键**：**强制** `Idempotency-Key`

### 9. F-15｜全站目录收录开关（Opt-in）与分页检索（改造现有接口）
- **店铺开关**：`PATCH /api/admin/shop` 请求新增 `allowPublicCatalog`（默认 `false`）
- **全站目录**：`GET /api/public/catalog?q=&currency=&sort=live_first&page=1&pageSize=20`
  - **响应关键字段**：`items`, `total`, `page`, `pageSize`, `hasMore`
  - **错误码**：`invalid_pagination`、`rate_limited`（429）

### 10. F-17｜顾客收藏夹（新增接口）
- **方法与路径**：`GET /api/customer/favorites`、`PUT /api/customer/favorites`、`DELETE /api/customer/favorites/{favoriteID}`
- **请求/响应**：`targetType`（`"shop" | "listing" | "bundle_listing"`）、`targetId`
- **错误码**：`unauthorized`、`not_found`、`invalid_favorite_target`

### 11. F-18｜顾客逛展预选清单（不占库存，新增接口）
- **顾客侧**：`POST /api/customer/wishlists`、`GET /api/customer/wishlists`（响应含 6 位 `shareCode`、`qrPayload`、`estimatedTotalMinor`）
- **收银台导入**：`GET /api/staff/wishlists/{shareCode}`（`eventId`, `customerUserId`, `customerDisplayName`, `items`）
- **错误码**：`not_found`、`wishlist_shop_mismatch`

### 12. F-19｜顾客电子取货凭证夹（新增 + 改造接口）
- **关联**：`POST /api/staff/orders` 请求新增可选 `customerUserId`
- **查询**：`GET /api/customer/orders`、`GET /api/customer/orders/{orderID}`

### 13. F-20｜中英双语偏好保存（改造现有接口）
- **方法与路径**：`PATCH /api/me`、`GET /api/auth/me`
- **请求/响应关键字段**：`locale`（`"zh-CN" | "en"`），错误码 `invalid_locale`

### 14. F-21｜增强审计日志查询（改造现有接口）
- **方法与路径**：`GET /api/admin/audit?action=&actorId=&page=1&pageSize=20`
- **响应关键字段**：`items: [{ id, actorId, actorName, actorRole, action, subjectType, subjectId, reason, terminalId, beforeJson, afterJson, createdAt }]`, `total`

### 15. F-22｜过期未付订单人工恢复收款（新增接口）
- **方法与路径**：`POST /api/staff/orders/{orderID}/recover-expired`
- **请求关键字段**：`reason`（必填）、`payCodeId`、`cashReceivedMinor`、`paymentRef`、`terminalId`
- **错误码**：`invalid_status`、`invalid_reason`、`not_enough_stock`、`event_not_open`、`not_listed`、`idempotency_conflict`、`command_in_progress`（503）
- **幂等键**：**强制** `Idempotency-Key`

### 16. F-23｜终端标识、QR 流水号与实时流（新增 + 改造接口）
- **实时流**：`GET /api/staff/events/{eventID}/stream`（SSE 事件：`stock_changed`、`oversold_alert`）
- **流水号**：`POST /api/staff/orders/{orderID}/pay` 透传 `paymentRef` 至修复 plan 任务 3 迁移 `014` 字段

### 17. F-24｜超限复杂购物车报价降级（改造现有接口）
- **方法与路径**：`POST /api/staff/order-quotes`
- **请求/响应新增字段**：`allowDegraded`（请求可选）、`pricingDegraded`（响应）、`422` 附 `conflictGroup` 与 `canDegradeToSingleTier`
