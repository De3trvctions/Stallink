# Stallink 电商平台审核报告

- 日期：2026-11-22
- 范围：`server/`（Go API，85 个文件 / 15.0k 行）、`web/`（Next.js 16）、`docs/`
- 方法：先安装三套外部 Skill，再以其中的「可执行规则」作为审核清单逐条对照代码
- 自动化验证：`go build ./...`、`go vet ./...`、`go test ./...` 全部通过（集成测试需本地 Postgres，未计入本次）

---

## 一、Skill 安装结果

三套 Skill 均装在全局，可被 Claude Code / Cursor / Codex 共用。真实目录在 `~/.agents/skills`，三个 Agent 目录下都是软链接。

| 套件 | 来源 | 安装的 Skill | 用途 |
|---|---|---|---|
| ① Anthropic Commerce Agents 蓝图 | `anthropics/commerce-agents` | commerce-architecture、commerce-trust-safety、commerce-merchant-operations、commerce-evals、commerce-ui-tools、commerce-prompt-caching | Agent 电商的架构、信任安全、商家写审批、评测、呈现契约、缓存 |
| ② 电商开发 Skill | `miles990/claude-software-skills` | e-commerce | 购物车、支付、库存预占、订单状态机 |
| ③ Stripe 官方 Skill | `docs.stripe.com` | stripe-best-practices、stripe-implementation-planner、stripe-docs、stripe-apps | 支付选型、密钥、Webhook、税务与上线检查 |

安装命令（可复现）：

```bash
npx -y skills@latest add anthropics/commerce-agents -g -y -a claude-code -a cursor -a codex -s '*'
npx -y skills@latest add miles990/claude-software-skills -g -y -a claude-code -a cursor -a codex -s e-commerce
npx -y skills@latest add https://docs.stripe.com -g -y -a claude-code -a cursor -a codex \
  -s stripe-best-practices -s stripe-implementation-planner -s stripe-docs -s stripe-apps
```

说明：另一份 `aiskillstore/marketplace` 的 `ecommerce` 只放出 SKILL.md、ZIP 需会员，无法直接安装，因此改用同为「商品/购物车/支付/订单/库存」主题的 `miles990` 版本。Stripe 另有 `stripe-pay` / `stripe-directory` / `stripe-projects` / `stripe-bfcm-dashboard` 四个偏运营/转账/季节报表的 Skill，与本项目无关未装。

---

## 二、结论速览

**资金与库存的主链路是可靠的，缺口集中在「人」和「运维」侧。**

做得扎实的部分：报价快照、幂等四元组、共享库存池的行锁与全局锁序、退款幂等、审计覆盖、CSV 公式注入防护。

值得优先处理的四件事：

1. 退款确认对 `sales` 角色开放，任何收银员都能"确认已退款并回补库存"。
2. 登录/注册没有任何限流，bcrypt cost 12 可被当作放大器。
3. 收款靠收银员手工点"已付"，没有支付凭据与对账字段。
4. 平台"自动审核"开关本身没有审计记录。

严重度分布：P0 = 0，P1 = 4，P2 = 6，P3 = 3。

---

## 三、按 Skill 维度的对照

| Skill | 核心要求 | 本项目现状 | 判定 |
|---|---|---|---|
| e-commerce | 金额整数化、库存原子预占、订单状态机、幂等下单 | 全部 int64 minor；共享池按 good 加锁；状态迁移集中在 `catalog.CanTransit`；幂等表落地 | 符合 |
| commerce-trust-safety | 身份只在服务端确定；写入要看 provenance 与上限；金额/库存等事实只能来自后端记录 | 角色网关注入 `Account`；订单金额来自服务端 quote；前端只做展示 | 基本符合，退款权限见 P1-1 |
| commerce-merchant-operations | 所有写操作先 staged、再由独立的人审批；护栏在暂存与应用各校验一次；审计留痕 | 有平台审核 + `expectedRevision` 乐观锁，但退款不属于"暂存-审批"链路；自动审核开关无审计 | 部分符合 |
| commerce-ui-tools | 组件里的价格、库存、状态由服务端 join，模型/前端不得自造数字 | `counter` 页面完全使用 `/api/staff/order-quotes`，`expectedTotalMinor` 只是回传校验 | 符合 |
| commerce-architecture / prompt-caching | 面向 LLM Agent 的循环、缓存断点 | 仓库内无任何 LLM/Agent 代码（grep 无 anthropic/openai/llm） | 不适用 |
| commerce-evals | 每个正例配一个反例；被拒的写操作必须在代码层断言 | Go 侧已有并发/原子性/退款幂等/租户迁移测试，但没有"负例清单"到用例的显式映射 | 可借鉴 |
| stripe-best-practices | 履约由 Webhook 驱动，success 页不作数；限制性 API Key；沙箱隔离 | 当前是线下收款码 + 人工确认，未接任何在线支付渠道 | 见 P1-3 |

---

## 四、问题清单

### P1-1 退款确认对 `sales` 开放，缺独立审批人

- 证据：`server/internal/service/staff.go:728` 的 `RefundConfirm` 只走 `a.staff`；第 727 行注释明确写着"no extra role gate is applied here"。`Cancel`（`staff.go:710`）在 `paid/fulfilled` 上会把订单推进到 `refund_required`，同样允许 `sales`。
- 影响：收银员可独自完成"标记退款 → 确认退款 → 库存回补 → 订单取消"整条链路。多人轮班的摊位里，这等于没有退款复核。对照 commerce-merchant-operations 第 5 条：资金类写操作必须由操作者之外的人在宿主侧批准。
- 建议：`RefundConfirm` 收紧到 `shop_admin`；或保留 `sales` 发起、强制 `shop_admin` 二次确认（订单上增加 `refund_requested_by` / `refund_confirmed_by` 两列），并拒绝同一人自审。

### P1-2 登录/注册无限流

- 证据：全仓 grep 无 limiter；README 也自述"当前代码…未验证具体反向代理是否配置了限流"。
- 影响：`/api/auth/login` 是可枚举的登录名 + bcrypt cost 12，攻击者既能撞库，也能用并发请求把 CPU 打满。
- 建议：在 API 层按"真实客户端来源 + loginName"双维度计数，失败 5 次锁定 15 分钟；`/api/auth/register` 按 IP 限速。不要直接信 `X-Forwarded-For`，要么只信可信代理注入的头，要么用连接层来源。

### P1-3 收款靠人工确认，没有支付凭据与对账字段

- 证据：`staff.go:622` 的 `Pay` 只接收 `payCodeId` 与 `cashReceivedMinor`；`order_checks.go:13` 校验现金是否够、二维码是否混入现金，但没有任何"转账参考号 / 截图 / 渠道流水号"落库。
- 影响：QR 收款无法事后证明"这笔钱确实收到"，退款与日报只能信收银员的操作。对照 stripe-best-practices：履约应由渠道回调驱动，而不是由前端成功页或人工点击决定。
- 建议（线下场景的最小闭环）：订单加 `payment_reference`、`payment_proof_image_id`、`received_at`；日报增加"按收款方式应收/实收差额"。若后续接线上支付，用 Stripe Checkout + Webhook，`checkout.session.completed` 与 `async_payment_succeeded` 都要处理并校验 `payment_status`，成功页不作为履约依据。

### P1-4 平台"自动审核"开关变更无审计

- 证据：`server/internal/service/review_switch.go:18` 的 `SetAutoApproveSetting` 只写 Redis，没有 `insertAudit`。而 `staff.go` 的商品创建/改价等路径都以 `a.autoApprove(...)` 决定是否绕过平台审核。
- 影响：谁能一键把所有商品改成免审，事后查不到人。这正是 commerce-merchant-operations 里"护栏配置本身也要留痕"的场景。
- 建议：开关变更写入平台级审计（操作人、旧值、新值、时间）；自动审核建议只对低风险字段生效（如图片排序），名称/价格/库存仍走审核。

### P2-1 过期预占仍占着可售量

- 证据：`catalog/stock.go:21` 的 `Available = Total - Reserved`，而 `Reserved` 取自 `event_listings.reserved_qty`；过期释放靠 `ExpireReservations` 批量扫（`reservation_store.go:280`，默认批 100）。
- 影响：扫之前前台会显示偏低的余量甚至"售罄"。不影响正确性，但热卖场次会让顾客看到假售罄。
- 建议：读取侧把 `expires_at <= now()` 的预占从 SUM 中剔除，或把扫描周期压到 10~30 秒并在下单时先做一次机会式释放。

### P2-2 释放预占没有走全局锁序

- 证据：`reservation_store.go:235` 的 `releaseOrderReservations` 直接改 `event_listings`，不先锁 `goods` 行；而 `reserve` / `consume` / `applyStock` 都是"先 good 后 listing"。
- 影响：方向是安全的（只减 reserved），但并发 reserve 读 `SUM(reserved_qty)` 时可能读到释放前的旧值，造成短时少卖；锁序不统一也是后续加逻辑时的隐患。
- 建议：释放路径同样按 `inventoryLockOrder` 先锁 good 再改 listing，代价很小。

### P2-3 `readGoodPool` 用 `shopID=0` 绕过店铺过滤

- 证据：`data/catalog_store.go:1214` 的查询条件是 `($2 = 0 OR shop_id = $2)`；`order_store.go:630` 的 `applyStock` 传的是 `0`。
- 影响：目前 goodID 全局唯一，不构成越权；但这是一个"传 0 就关掉租户过滤"的隐式后门，后续若按店铺复制 ID 或做分库会直接踩雷。
- 建议：内部调用显式传 `shopID`，把 `0` 分支删掉或限制在迁移脚本里。

### P2-4 缺 CSP / HSTS

- 证据：`web/next.config.ts:6` 只有 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`。
- 影响：目前 XSS 面很小（React 默认转义，唯一的 `dangerouslySetInnerHTML` 是 `layout.tsx:17` 的常量主题脚本），但没有 CSP 就少了一层兜底；上 HTTPS 后也需要 HSTS。
- 建议：加 `Content-Security-Policy`（至少 `default-src 'self'`，脚本用 nonce）与 `Strict-Transport-Security`。

### P2-5 前端有一份没人用的价格算法

- 证据：`web/src/lib/pricing.ts:25` 的 `bundleTotal` 与服务端 `catalog/pricing.go:46` 的 `BundleTotal` 是同一套 DP 算法；仓库内无任何调用点。
- 影响：两份实现迟早漂移，一旦有人接上去做"预估价"，就会出现前端显示与服务端报价不一致。
- 建议：删掉前端 `bundleTotal`，价格一律以 `/api/staff/order-quotes` 为准。

### P2-6 没有 CI

- 证据：仓库无 `.github/`、`.gitlab-ci.yml`；只有 `server/Makefile` 的 `test` 目标。
- 影响：并发、幂等、退款这几块恰恰是最需要回归的，现在只能靠人记得在本地跑。
- 建议：最小 CI = `go build` + `go vet` + `go test` + `go test -tags integration`（起 Postgres service）+ `tsc` + `eslint` + `next build`。

### P3（低）

- 会话固定 14 天、无轮换、无并发会话上限（`handlers.go:105`）：内部摊位可接受，但建议登录后轮换 token、限制同账号活跃会话数。
- 登录/登出没有审计：摊位账号被异地登录时查不到记录。
- `available` 只钳到 0，不区分"真实零库存"与"预占造成"，前端文案无法区分（可考虑返回两个字段）。

---

## 五、值得保留的设计（有证据）

- **金额**：全链路 `int64` minor，`money.go` 无浮点；DB 侧 `unit_price_minor > 0`、`line_total_minor >= 0` 等 CHECK 兜底。
- **报价**：`quote_store.go` 用 `RepeatableRead` + `catalog_revision` + TTL，下单时校验 `expectedTotalMinor`，前端无法改价。
- **幂等**：`order_commands (shop_id, actor_id, operation, key)` + `request_hash`，重放还做了一次账号/店铺有效性鉴权（`order_command_store.go:69`）。
- **库存**：共享池按 good 聚合需求、`FOR UPDATE` 锁 good、全局 ID 锁序（`inventoryLockOrder`）、`inventory_movements` 与成交/释放一一对应。
- **退款**：`refundConfirmDecision` 区分"已确认"与"待确认"，重复调用不会二次回补（`order_store.go:554`）。
- **审计**：21 处 `insertAudit`，覆盖下单、支付、履约、退款、上下架、改价、库存调整。
- **导出**：`spreadsheetText` 处理 `= + - @` 前缀，防 CSV 公式注入（`reports.go:33`）。
- **上传**：`http.DetectContentType` 白名单 + 1 MiB / 4 MiB 上限（`image_storage.go:98`、`imagestore/compress.go:18`）。
- **登录**：不存在账号也跑一次 dummy bcrypt，抹平计时差（`handlers.go:127`）。
- **越权**：收款码图片要求 `staff` 且必须是本店（`handlers.go:540`）。

---

## 六、建议的落地顺序

1. 本周：退款权限（P1-1）+ 登录限流（P1-2）。这两条不改业务逻辑，只改网关与角色判断。
2. 本周：给订单补支付凭据字段（P1-3 的前半），日报加"应收/实收"差额列。
3. 下周：自动审核开关审计（P1-4）+ 预占读取剔除（P2-1）+ 释放锁序（P2-2）。
4. 下周：CI（P2-6）与 CSP/HSTS（P2-4），顺手删掉前端 `bundleTotal`（P2-5）。
5. 下阶段：若要接线上支付，按 stripe-best-practices 走 Checkout + Webhook，并先建独立 sandbox 与限制性 Key。

---

## 七、继续使用这些 Skill

在 Cursor / Codex 里直接用自然语言触发即可，例如：

- "按 commerce-trust-safety 的写入审批规则，审一遍退款链路"
- "用 commerce-merchant-operations 的 staged change 契约，把商品改价改成暂存+审批"
- "按 stripe-best-practices 设计收银台的线上支付与对账"

在 Claude Code 里也可以用斜杠命令 `/skills` 查看已安装项；三套 Skill 都在全局作用域，换项目也能用。
