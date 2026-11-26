# Stallink PRD vNext（唯一事实基线）

- **版本与日期**：vNext · 2026-11-26（已合入 Kel 2026-11-26 全部拍板口径）
- **适用范围**：东南亚（首期马来西亚、新加坡）漫展摊位线上展示、独立摊位链接、展场导流目录、现场收银（在线 + 真离线现金）与销售台账系统
- **配套文档**：功能拆解与前后端接口约定见同目录 `2026-11-26-stallink-feature-list.md`；前置安全/一致性修复基线见 `docs/superpowers/plans/2026-11-26-stallink-review-fixes.md`
- **文档收敛声明**：
  本文件合并并正式替代以下历史分散规格，凡与本文件冲突的旧口径均以本文件为准：
  - `docs/superpowers/specs/2026-11-11-bundles-pos-idempotency-design.md`
  - `docs/shared-pool-stock-design.md`
  - `docs/superpowers/specs/2026-11-11-stallink-catalog-moderation-pricing.md`
  - `docs/superpowers/specs/2026-11-11-goods-by-event-design.md`
  - `docs/superpowers/specs/2026-11-12-stallink-bigint-identity-spec.md`

---

## 一、产品目标、用户与成功指标

### 1.1 为什么做（业务背景与核心痛点）
新加坡、马来西亚同人漫展与线下市集出摊场景具有以下强特征：
1. **摊主跨多场连续出摊**：同一批周边实物往往周末在 A 展售卖，下周拉去 B 展，或同时需为重点场次/已排期场次净锁定底货不被前场卖空。
2. **现场开单极度追求快与准**：画手/摊主常由朋友或兼职 `sales` 协同收银；手工计算“单买阶梯优惠 + 固定套餐自动凑套最优价”极易算错或拉长排队时间。
3. **场馆弱网/断网是常态**：大型展馆（KLCC、MITEC、Suntec 等）人流高峰时 4G/5G 拥堵或断网，收银台必须支持无网继续收取现钞出凭证并在恢复后安全处理超卖。
4. **多摊位与多币种台账混乱**：不同摊主数据必须严格隔离；跨日退换货、报损、现金找零长短款、多币种（MYR/SGD/CNY）容易导致夜里对账混乱。

### 1.2 给谁用（用户画像与角色权限矩阵）
系统分为五类身份，严格执行店铺多租户隔离（`platform_admin` 与 `customer` 的 `shop_id IS NULL`；`shop_admin` 与 `sales` 强绑定单一 `shop_id`）：

| 角色 | 典型使用者 | 核心职责与权限边界 |
|---|---|---|
| **匿名访客（Public）** | 扫码逛展的普通漫展观众 | 访问全站展场目录（仅收录已开启公开收录的店铺）与摊位独立链接 `/c/[slug]`；按活动浏览已过审且在架商品/套餐、查看实时可用库存与摊位公告；无权操作任何写接口。 |
| **顾客（Customer）** | 注册逛展买家 | 继承访客权限；可维护个人称呼、联系方式与语言偏好（`/me`）；可**收藏摊位/商品**、创建**逛展预选清单（不占库存，生成清单码给收银台扫码/核对导入）**、查看关联自己账号的**订单与电子取货凭证夹**。 |
| **收银员（Sales）** | 现场摊位兼职/协作者 | 查看本摊位活动、商品目录与收款方式；使用在线收银台与**真离线现金开单**；对本店订单执行收款、交付、取消 `pending` 单；**可直接取消本店已付/已交付订单并确认退款（含部分退款与报损），但操作人必须属于该订单所在店铺、退款原因必填、审计日志必须记录操作人、原因、金额与时间**；无权创建/编辑商品、调库存、增删收款码、管理其他 `sales` 账号或修改店铺设置。 |
| **店主（Shop Admin）** | 摊主本人/社团负责人 | 拥有 `sales` 全部能力；管理本店活动（含活动时区、预售开关）、商品与共享总库存、单场销售上限与净锁定保底量、固定套餐、单品多买优惠、收款码、`sales` 账号启停与改密；配置**“是否允许本店被全站目录收录（`allow_public_catalog`，默认关闭）”**开关；查看本店审计日志、超卖异常单与日结报表生成/导出。 |
| **平台管理员（Platform Admin）** | Stallink 运营人员 | 开通/启停店铺；跨店审核商品与固定套餐（通过/拒绝并填原因）；配置全站视觉主题、平台级自动审核开关与全站目录排序策略；无权查看或篡改各店铺私有财务流水及替店铺开单。 |

### 1.3 成功指标（KPI 基线，2026-11-26 Kel 拍板确认）

| 指标维度 | 指标名称 | 定义口径 | 目标值 |
|---|---|---|---|
| **并发吞吐** | 单场峰值并发 | 单场漫展期间店面浏览 + 报价 + 收银核心链路系统承载能力 | **≥ 1,000 QPS** |
| **现场效率** | 在线开单响应延迟 | 点击“生成订单”到进入收款确认弹窗的服务端 + 端到端 P95 耗时 | **P95 < 500 ms** |
| **现场效率** | 离线现金开单耗时 | 断网状态下点选加购至生成离线小票 | **≤ 300 ms**（纯本地） |
| **库存准确** | 在线模式超卖率 | 在线成交量超出 `total_stock` 或单场 `max_qty`、或侵占他场净保底额的订单占比 | **0%** |
| **离线闭环** | 离线超卖异常核销率 | 联网同步产生的 `oversold` 订单在展期结束前完成补货或退款的比例 | **100%** |
| **财务对账** | 日结账实一致率 | 日报净实收（按 `paid_at` 实收 − 按 `refunded_at` 实退）与线下台账核对一致 | **100%** |
| **高并发稳定性** | 高峰接口可用性 | 展期高峰（含 `503` 重试后）核心收银接口成功率 | **≥ 99.9%** |

---

## 二、核心领域模型与业务规则

### 2.1 主键、多租户与资源标识
- 所有业务表主键沿用 `BIGINT GENERATED ALWAYS AS IDENTITY`，API JSON 统一输出十进制字符串。
- **多租户强校验**：任何店铺级读写操作必须显式校验操作人所属 `shop_id` 与目标资源 `shop_id` 一致；数据层禁止 `shopID == 0` 跳过店铺过滤（具体清理归修复 plan `2026-11-26-stallink-review-fixes.md` 任务 5）。
- **数据库迁移安全边界**：服务启动与 Nacos 热更新禁止不检查数据库归属即对非本机库自动执行迁移（具体守卫与超时/校验和归修复 plan 任务 2；回滚 SQL 统一隔离于 `server/internal/data/migrations/down/`，见任务 3）。
- **公开媒体防遍历与缓存**：公开图片响应缓存头与有界内存缓存由修复 plan 任务 11 落地、图片解码并发控制与 EXIF 剥离由任务 12 落地、Drive 孤儿清理跨环境防护由任务 1 落地；vNext 在此基础上为公开媒体接口增补不可预测 `media_token` 与本地落盘二级缓存。

### 2.2 活动、货币与活动级时区
1. **货币范围（首期新马）**：仅支持 **`MYR`、`SGD`、`CNY`**，按 `minor` 整型（`1 major = 100 minor`）存储与计算。同一场活动固定单一币种；非现金 QR 收款码必须与活动币种一致。
2. **活动级时区**：`events` 表新增 `timezone`（IANA 时区名，默认 `Asia/Kuala_Lumpur`，支持 `Asia/Singapore`、`Asia/Shanghai` 等）。营业日切分、活动结束状态与公开目录日期判断**全部统一依据该活动自身的 `timezone` 计算**。
3. **活动状态与开单限制**：`scheduled`（未开始）、`live`（进行中）、`ended`（已结束）。`scheduled` 默认仅浏览不可开单，仅当活动级 `allow_presale = true` 时才允许开单；`ended` 活动禁止创建新单与收款。

### 2.3 双轨流量入口与顾客账号能力
1. **双轨入口（独立摊位链接 + 全站展场目录）**：
   - **独立摊位链接（`/c/[slug]`）**：只要店铺 `is_enabled = true`，访客即可直接通过独立链接访问与逛展（不受全站目录开关影响）。
   - **全站目录收录开关（Opt-in）**：`shops.allow_public_catalog boolean NOT NULL DEFAULT false`（**默认不收录**）。仅店主主动开启后，店铺及旗下在架已过审商品/套餐才出现在首页全站聚合目录与搜索中。
   - **全站目录排序规则**：进行中（`live`）活动优先 > 有可售库存优先 > 活动开始日期升序 / 最近上架更新时间降序；全站搜索强制分页（每页 ≤ 50）与 `pg_trgm` GIN 索引。
2. **顾客账号实用功能**：
   - 维护称呼、联系方式与双语偏好（`/me`）；
   - **摊位/商品收藏**：收藏摊位或商品/套餐并查看实时库存；
   - **逛展预选清单（不占库存）**：生成 6 位清单短码与二维码（**不占库存、不锁价**），收银台扫码/输码一键导入购物车并按实时库存与最优计价开单；
   - **电子取货凭证夹**：关联顾客账号的订单在“我的订单”展示取货号 `number`、状态与明细。

### 2.4 商品、固定套餐与审核规则
1. **单品**：店铺级 SKU，`total_stock`、最多 8 张图（上传 ≤ 6 MiB、压缩至最长边 1600px 且 ≤ 1 MiB；并发解码限流与 EXIF 剥离由修复 plan 任务 12 保障）；活动上架 `event_listings` 配置 `unit_price_minor`、单品 `price_tiers`、`is_listed`。
2. **固定套餐**：一口价打包组合，引用同店 2–20 个不同单品；**最多 1 张主图**、**不支持套餐自身 `price_tiers`**；保存或上架套餐时构建“套餐-SKU 无向二分图”，校验任一连通分量中 **SKU ≤ 20 且 套餐 ≤ 10**，超限即拒绝保存。
3. **审核触发与不误杀规则**：
   - 新建商品/套餐，或修改 **`name`、`description`、图片** 时重置为 `pending`；
   - 修改 `unit_price_minor`、单品 `price_tiers`、`total_stock`、`is_listed`、单场上限 `max_qty`、保底 `min_reserve_qty`、以及已过审单品挂入新活动均**免审**；
   - 单品仅改价/改优惠/调库存（仍 `approved` 且 `is_listed = true`）时，**不打回依赖它的已过审套餐**；
   - **禁止 `rejected` 自动复活**：当前为 `rejected` 的商品/套餐，即便平台开启 `auto_approve = true`，重新编辑提交后也必须进入 `pending` 人工复核。

### 2.5 库存模型：全店共享物理库存池 + 单场上限与净锁定保底（口径 2）
以 `goods.total_stock` 为底层未售物理共享池，`event_listings.sold_qty` 为分场销量台账；`event_listings` 新增两个可选整数配置：
- `max_qty`（`NULL` 或 `> 0`）：**本场最多卖 X 件**（含本场 `sold_qty` + 本场未过期 `active_reserved`）。
- `min_reserve_qty`（`int NOT NULL DEFAULT 0`）：**从设定/更新时刻起，当前仍需为本场额外净锁定的件数 Y**（不给其他活动动用）。

#### 核心计算、扣减与守卫规则（以商品 `G`、目标活动 `E` 为例）：
1. **有效预占口径**：仅统计 `inventory_reservations` 中 `status = 'active' AND expires_at > now()` 的记录。
2. **净锁定保底的消费与恢复语义（口径 2 精确闭环）**：
   - 本场 `E` 下单预留（或离线同步扣减）`need` 件时，优先消耗 `E` 自身的保底额度：$\text{from\_floor} = \min(\text{need},\ \text{min\_reserve\_qty}(E, G))$；
   - 预留创建时扣减 `min_reserve_qty(E, G) -= from_floor`，并在 `inventory_reservations` 记录 `floor_reserved_qty = from_floor`；
   - 若待付单取消或过期释放（且活动 `E` 未结束仍在架），将 `floor_reserved_qty` 退回 `event_listings.min_reserve_qty`；若支付成交则 `total_stock -= need`，剩余 `min_reserve_qty` 不回退。
   - **他场被保护锁定总量**：$\text{locked\_by\_others}(E, G) = \sum_{E' \neq E,\ E'.\text{status} \neq \text{'ended'},\ E'.\text{is\_listed} = \text{true}} \text{min\_reserve\_qty}(E', G)$。
3. **自由池与本场最终可售**：
   $$\text{free\_pool}(E, G) = \text{total\_stock}(G) - \sum_{\text{all } A} \text{active\_reserved}(A, G) - \text{locked\_by\_others}(E, G)$$
   $$\text{cap\_remain}(E, G) = \begin{cases} +\infty, & \text{max\_qty}(E, G) \text{ is NULL} \\ \text{max\_qty}(E, G) - \text{sold\_qty}(E, G) - \text{active\_reserved}(E, G), & \text{otherwise} \end{cases}$$
   $$\text{available}(E, G) = \max\Big(0,\ \min\big(\text{free\_pool}(E, G),\ \text{cap\_remain}(E, G)\big)\Big)$$
4. **设置拒绝与自动释放规则**：
   - 修改 `min_reserve_qty` 或调减 `total_stock` 时，必须在 `goods` 行锁下满足：$\sum_{\text{all } A} \text{active\_reserved}(A, G) + \sum_{\text{all active } E} \text{min\_reserve\_qty}(E, G) \le \text{total\_stock}(G)$，超限拒绝返回 `409 stock_exceeded`；
   - `min_reserve_qty` 不得超过本场剩余上限（`max_qty - sold_qty - active_reserved`），否则返回 `409 invalid_listing_quota`；
   - 活动转为 `ended` 或上架改为 `is_listed = false` 时，系统在事务内将 `min_reserve_qty` 清零释放回共享池。

---

## 三、核心业务流程、异常流程与边界条件

### 3.1 自动凑套 DP 计价与分摊
- 前端显式套餐拆解为底层单品需求进入同构最优计价引擎，输出全局最低价并按比例分摊订单行金额；超过 100,000 DP 状态支持引导拆单或按单品优惠降级结算。

### 3.2 在线收银主流程、幂等与异常处理
1. **标准下单（`CreateOrder`）**：按 `good_id` 升序锁 `goods` → `listing_id` 升序锁 `event_listings`（不锁 `shops` 行）；校验 `available(E, G) >= need` 并写入 15 分钟预留。
2. **幂等与并发控制**：命令接口强制 `Idempotency-Key`（UUID）；事务 `lock_timeout = 2s`、`statement_timeout = 5s`，锁等超时返回 **`503 command_in_progress` + `Retry-After: 1`**，死锁/串行化冲突重试最多 2 次。
3. **收款确认（`Pay`）**：~~现金收款提供实收输入与快捷面额，实时算找零~~（**已作废（Kel 2026-11-26）**：对应 F-07 / vNext web W5，和这一轮去掉收银台实收、找零 UI 的方向相反；后端继续接收 `cashReceivedMinor`，离线现金同步要用）；QR 收款校验币种一致并记录支付凭证/参考号（字段 `payment_ref` 由修复 plan 任务 3 迁移 `014` 引入）。
4. **付款前资格失效自动撤单**：`Pay` 发现店铺停用、活动结束或商品撤审/下架时，在同一事务内将订单转为 `cancelled`、释放预留（按修复 plan 任务 6 统一锁序回退）并持久化幂等结果返回 `order_cancelled_eligibility_lost`。
5. **过期恢复**：对晚于 15 分钟到账的 `expired` 订单，支持校验池库存充足后填原因人工恢复收款（记强审计）。

### 3.3 真离线现金开单流程与超卖（`oversold`）风控（Q7=B）
1. **适用条件与 4 小时断网防篡改硬限制（Kel 拍板）**：
   - 仅限**现金（`cash`）**与已预缓存的 `live` 活动；
   - **不设单笔与累计金额上限，仅限制连续断网开单最长 4 小时**；
   - **双端防篡改机制**：
     1. **服务端签发签名租约**：终端每次联网刷新目录或同步完成后，由后端接口下发带签名的离线租约（`lastOnlineSyncAt`、`expiresAt = lastOnlineSyncAt + 4h`、`shopId`、`terminalId` 及 `HMAC-SHA256` 签名 `signature`；**HMAC 密钥仅保存在服务端环境变量/Nacos 运行配置中，严禁写入任何代码或配置模板仓库**）；
     2. **前端单调时钟 + 完整性校验**：页面开启期间同步记录锚点单调时钟 `performance.now()` 与系统时钟 `Date.now()`，并以浏览器 WebCrypto 不可导出密钥对本地租约镜像做完整性封印。离线期间取 `max(Date.now() - anchorWallMs, performance.now() - anchorPerfMs)` 判定是否超过 4 小时；**一旦发现本地签名缺失/损坏无法验证、或检测到系统时间回拨/与 `performance.now()` 漂移超过 60 秒，一律按“已超时”处理，立即锁定禁止继续离线开单**，必须联网重新获取有效签名租约后恢复；
     3. **同步时服务端复核验签**：恢复联网同步时将 `offlineLease` 随订单一并提交后端强验 HMAC，若验签失败记入高危审计告警（已收现金之离线单仍落库入账以便财务核对，但立即吊销并拒绝下发新租约）。
2. **离线单号与多设备隔离**：持久 `terminal_id` + 2 位短码；离线单号为 10 位编码 `OL` + `2位终端短码` + `4位序列` + `2位校验位`（如 `OLK40012A9`）。
3. **异步同步与超卖（`oversold`）闭环**：
   - 联网后按 `offline_created_at` 升序幂等调用 `POST /api/staff/orders/offline-sync`（依赖 `(shop_id, client_order_id)` 去重）；
   - 库存不足时**正常落库记账**为 `paid` + `oversold_status = 'pending_resolution'`、`oversold_qty = 缺口数`、`total_stock` 钳零不负，向全店推送 SSE 告警；
   - 处置双路径：**补货核销**（调增 `total_stock` ≥ 缺口后扣减并标 `resolved_restocked`）或 **联系退款**（标 `resolved_refunded`，且**仅回补曾经实际扣出过的部分 `max(0, ordered_qty - oversold_qty)`**）。

### 3.4 取消、部分退款、报损与跨日财务对账（Q3=B + Q5=B）
1. **权限、一步取消并退款与审计硬要求（Kel 11-26 确认）**：`sales` 与 `shop_admin` **均可直接取消本店订单并确认退款**，但操作人**必须属于该订单所在店铺**（跨店 `403`）。已收款订单前端默认只提供**一步「取消并退款」**（`cancel-refund`，全额退款+全部货品回库），同时保留两步流（`cancel` → `refund-confirm`）兼容存量 `refund_required` 订单；**未收款（`pending`）取消、两步流与一步取消并退款均强制填写原因（`TrimSpace` 后 2–500 Unicode 字符，错误码 `reason_required` / `reason_too_short` / `reason_too_long`）**，并写入 `audit_logs`（操作人、原因、金额、时间及变更前后快照）；具体边界与接口归修复 plan 任务 7 / `P1-1`。
2. **部分商品/数量退款与组合优惠重算**：支持逐行退货与拆分 `restock_qty`（完好回库）+ `damaged_qty`（报损不回库）；由同构最优计价引擎对“剩余未退商品”重算 `remaining_due_minor`，最大可退额 = `净实付余额 - remaining_due_minor`，防止拆散套餐薅差价。
3. **跨日对账口径**：营收实收按 `paid_at` 落入活动时区营业日；退款支出按 `order_refunds.refunded_at` 落入实际退款营业日作为“退款冲减项”（关联原单号与原支付日），当日净收款 = 毛实收 − 退款冲减，不篡改历史日报快照。

---

## 四、非功能需求

1. **并发与性能（1,000 QPS / 开单 P95 < 500ms）**：公开图片防遍历 `media_token` + CDN/磁盘缓存防穿透 Drive；`pgxpool` ≥ 50 连接 + 只读副本；Redis 分层缓存；全站搜索 `pg_trgm` GIN 索引与分页；事务 2s/5s 超时控制与去 `shops` 行锁。
2. **安全、权限与审计**：分布式/降级限流与可信代理真实 IP（含登录/注册与公开基础限流，归修复 plan 任务 8）、公开目录分页限流（归任务 12），vNext 增量补齐报价接口限流；Drive 跨环境清理防护（归任务 1）、去除 `shopID=0` 隐式绕过（归任务 5）、启动与热更新自动迁移守卫（归任务 2）、自动审核开关平台审计（归任务 10）；vNext `audit_logs` 扩展记录操作人、原因、金额、时间、`before_json`、`after_json`、`terminal_id`、`client_ip`。
3. **多币种、活动级时区与中英双语 i18n**：币种 `MYR / SGD / CNY`；时区按 `events.timezone`；前后端 `zh-CN` / `en` 双语覆盖界面、报错、小票与日报导出。

---

## 五、可验证的验收标准（Given-When-Then）

1. **口径 2 净锁定保底与超限拒绝**：商品 `total_stock = 10`，活动 A 设 `min_reserve_qty = 4`，活动 B 尝试设 `7` 报 `409 stock_exceeded`；B 设 `0` 时 B 可售 `6`、A 可售 `10`；A 卖 1 件后 A 剩 `min_reserve_qty = 3`、`total_stock = 9`，B 仍可售 `6`；A 结束或下架后剩余保底归零释放，B 可售立即变为 `9`。
2. **全站收录 Opt-in 与顾客预选清单**：新店默认 `allow_public_catalog = false` 不入首页目录但 `/c/{slug}` 可访问；预选清单不占库存，收银台导入才实时报价占库。
3. **审核规则与禁 `rejected` 复活**：单品改价/优惠免审且不打回关联套餐；改 `name` 重审；`rejected` 实体即使开 `auto_approve` 仍必须进 `pending`。
4. **`sales` 同店退款、部分退款、报损与审计**：同店 `sales` 可取消已付单并部分退 1 件报损 A（`damaged=1, restock=0`），AB 套餐 RM16 重算剩余 B=RM10 后最大可退 RM6，跨店 `sales` 被 `403/404` 拒绝；必填原因，`total_stock(A)` 不增，`audit_logs` 记齐操作人、原因、金额、时间。
5. **跨日退款对账**：Day 1 实收 RM100，Day 2 退款 RM100；Day 1 快照不改，Day 2 日报体现退款冲减 RM100（关联原单号）。
6. **真离线现金开单（4h 防篡改 + 超卖闭环 + Playwright E2E）**：≤4h 连续断网可开 `OL...` 现金单，>4h 自动锁定；回拨系统时钟、注入伪造 `lastOnlineSyncAt` 或破坏签名后前端均按“已超时”拦截禁开；恢复联网后按时间序幂等同步并强验后端 HMAC，冲突时正常入账 `oversold_status = 'pending_resolution'`、库存钳零、全店告警，补货核销转 `resolved_restocked`（上述 5 项均纳入 P0 Playwright 断网自动验收）。
7. **并发抢占与资格失效自动撤单**：10 并发抢最后 1 件仅 1 单成功（锁等超时 `503 + Retry-After: 1`）；付款前下架由 `Pay` 自动撤单释库。
8. **1,000 QPS / P95 < 500ms 压测达标**：模拟单场 1,000 QPS 混合流量，开单 P95 < 500ms、在线超卖 0、非业务错误率 < 0.1%。

---

## 六、已确认事项（2026-11-26 Kel 拍板）

1. **离线现金开单风控与防篡改**：**不设单笔及累计金额上限**，仅限制**连续断网开单最长 4 小时**；页面运行期间采用单调时钟 `performance.now()` 辅助校验，并由后端基于纯服务端密钥签发含 `shop_id` + `terminal_id` + `lastOnlineSyncAt` 的 HMAC 签名租约；前端验证不了签名或发现时间被篡改时按“已超时”处理禁止继续离线开单（同步纳入 P0 Playwright 断网自动验收）。
2. **单场保底 `min_reserve_qty` 口径**：采用**口径 2（额外净锁定）**——设定 `min_reserve_qty = Y` 后，从设定时刻起额外净锁定 `Y` 件不给其他活动动用；保底之和加有效预占超过 `total_stock` 时拒绝设置；活动结束或下架后立即归零释放。
3. **全站目录默认收录策略**：**默认不收录（`allow_public_catalog = false`，Opt-in）**。
4. **顾客预选清单与库存关系**：**不占库存**。
5. **高并发与性能目标**：**单场峰值 1,000 QPS**，**开单 P95 < 500 ms**。
6. **收银员取消与退款口径（11-26 再次确认）**：**按 Q3=B 执行**——同店 `sales`/`shop_admin` 可取消本店订单（`pending` 取消同样强制填原因）及直接一步取消并退款（全额+全回库，保留两步兼容存量）；原因统一为 `2–500` Unicode 字符，审计日志记录操作人、原因、金额和时间。
