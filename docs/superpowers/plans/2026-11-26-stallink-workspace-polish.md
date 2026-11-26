# Stallink 工作台收尾（workspace polish）实现计划

> **计划状态：已批准开工（Kel 2026-11-26 13:55）**——Kel 13:55 通过 Grok Bot 定了剩下的全部问题（C-1、C-2、K-Q5、K-Q6、K-Q16、K-Q17、回填规则），并确认“plan 没问题，开工”。本计划已没有待 Kel 决定的问题。14:13 Kel 采纳了 UIUX 的 classic 边框对比度建议（新任务 **W-C3**），原来唯一的外部依赖也已关闭，**现在没有外部依赖**。14:21 Kel 定日报 PDF 保留现金长短款（只在指标卡、分币种汇总、CSV 隐藏）；核对时发现新订单不再录实收、PDF 长短款以后恒为“持平”，列为 **K-Q18 待 Kel**（§11.6，推荐 A，不阻塞开工）。14:25 UIUX 更正：展场输入框边框 `#738b86`、`#857e6e` Kel 没定过，和展场、店面控件边框色合并为 **K-Q19 待 Kel**（§11.7）。**15:01 Kel 定了 K-Q18 = A、K-Q19 = A 和店面吸顶 96px、店面窄屏保留「退出登录」、外观页确认框焦点放「取消」（§11.9），这批没有遗留问题**；新增后端任务 S10。15:08 UIUX 交了 favicon 方案，**15:11 Kel 定 K-Q20 = A**（「S.」字标，§11.10），W19 的 D10 可以做了。15:14 新增的 K-Q22（多币种时 PDF「—（未录实收）」怎么判断）**Kel 15:15 选 A：每种币各自判断**（§11.13）。**15:13 Kel 定 K-Q21 = B**：店面吸顶跟页头高度走（>900 `96px`、641–900 `72px`、≤640 `64px`，§11.12）。**这一期已经没有待 Kel 定的项。** 另记一条下期待定 **K-Q23**（窄屏两个退出入口以后是否收成一个，§13.1，不阻塞本期）。开工顺序见 **§14 开工安排**。
>
> **开工规则（Kel 硬性要求，违反即停工）：**
> 1. **云端库一律不动**：不连、不改、不跑迁移，也不用本机 API 连云端库（API 启动会自动跑迁移）。
> 2. **024 和 024a 只在本机 `docker-compose` 起的 Postgres 上跑**（`docker compose up -d postgres`，`127.0.0.1:5432`）；集成测试用隔离 schema。云端由 Kel 自己备份后手动执行（K-Q6 = B）。
> 3. **不建 worktree**，直接在 `/Users/kelden/go/src/Stallink` 当前工作区改。
> 4. **不 commit、不 push。** 任务最后一步只列改动文件，留在工作区等 Kel。**只有 Kel 经 Grok Bot 明确批准的提交可以做，其余一律只暂存**；已批准并已提交的见 §11.11（后端 7 个 commit + S10 的 `9f7e7f4`，都在 `design/editorial-frontend`，保留，不要 reset，不 push）。
> 5. 不改本计划范围外的代码；不写密钥。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 按 Kel 2026-11-26 定下的工作台 8 件事，加上 Stallink-UIUX设计 的外观走查问题和测试缺陷 D1–D5，把店铺工作台（/admin、/counter）、平台外观页和展场、店面两套风格收尾到“以 `docs/design-v2/screens/` 编辑式图为准”的验收标准；补齐后端缺的店铺资料字段和操作记录对象信息。

**架构：** 前端只改 `web/`（App Router 页面、`workspace-shell.tsx` 导航、`globals.css` / `shop.css` / `exhibition-workspace.css` / `reports.css` 三套外观样式）。后端补这几处：新增 `GET /api/admin/shop`、扩展 `PATCH /api/admin/shop`（slug 只读，新增客服联系、店铺摊位号、默认货币）；`GET /api/admin/audit` 返回对象展示名 `subjectLabel`；平台外观返回发布时间和发布人；日报 CSV 改成 5 列（R2，Kel 已定）。数据库只新增一份迁移 `024_shop_profile_and_appearance_meta.sql`（shops 三个新字段 + platform_appearance 的发布时间和发布人 + `audit_logs_shop_created_id_idx` 索引；回滚放 `migrations/down/`；同一个 PR 对齐 `schema.go` 的 shops DDL）；操作记录对象**不新增列**：读取时按 `subject_type` 关联查询，写入时快照复用修复 plan 迁移 `014` 的 `audit_logs.detail`，和 vNext `S16` 的 `before_json/after_json` 分工，不重复。

**技术栈：** Go（`server/`，pgx v5，Postgres，`//go:embed migrations/*.sql` 启动时自动迁移）；Next.js 16.4.6 + React 19.3 + Tailwind 4（`web/`）。测试：`go test`（集成测试 `-tags integration`，`STALLINK_TEST_DATABASE_URL` 指向本机 docker-compose 库，用隔离 schema）、`node --test tests/`、`npx eslint`、`npx tsc --noEmit`、`npx next build`；视觉回归复用测试报告里的 `docs/reports/stallink-test-2026-11-26/scripts/layout-sweep.cjs`、`contrast2.cjs`、`mock-api.cjs`（不启动 Go 后端）。

---

## 0. 硬性约束（每个执行者开工前必读）

1. 开工前读：`AGENTS.md`、`web/AGENTS.md`、`.cursor/rules/project-overview.mdc`、`server-go.mdc`、`web-next.mdc`、`README.md`。写 Go 前按 `AGENTS.md` 先加载 `samber/cc-skills-golang@golang-how-to`；写 Next.js 前查 `web/node_modules/next/dist/docs/` 里对应的指南。
2. **绝不修改任何云端或远程数据库。** 后端启动会自动执行 `server/internal/data/migrations/*.sql`，所以**拿非本机数据库启动 API 也算改库**。只允许用仓库 `docker-compose.yml` 起的本机 Postgres（`docker compose up -d postgres` 或 `make compose`）；集成测试只用隔离 schema（`isolatedSchemaPool` / `newIsolatedMigratedIntegrationPool`）。云端执行 `024`、`024a` 由 Kel 本人备份后手动做（K-Q6 = B），**不属于任何开发任务**，计划里出现的 ⚠️ 步骤都是给 Kel 的交接说明。
3. 数据库改动都有 SQL 文件：升级 `server/internal/data/migrations/024_shop_profile_and_appearance_meta.sql`，回滚 `server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql`（`down/` 子目录不会被 `//go:embed migrations/*.sql` 加载）；存量回填是单独的手动脚本 `server/internal/data/manual/024a_backfill_shop_default_currency.sql`，自带 `.rollback.sql`（K-Q16 = A）。
4. **不 git commit、不 push、不建 worktree**（Kel 13:55 硬性要求）。每个任务末尾的“Commit”步骤一律写成“只暂存，不 commit、不 push”，只列拟提交的文件。
5. 不把密钥写进仓库。
6. **Kel 已于 2026-11-26 13:55 批准开工**，按 §14 的顺序做。前端本轮已完成的改动不算本计划的任务产出，只在“当前状态”里记录。

## 1. 依据与来源

| 来源 | 位置 | 用到的内容 |
|---|---|---|
| Kel 的 8 个决定 | Grok Bot 对话 11:44（UTC+8），问卷 t78s460 的答复 | 第 2 节 K-01～K-08 |
| UIUX 走查结论 | Stallink-UIUX设计 发给 Grok Bot 的“工作台子页验收结果”（Grok Bot 对话 position 457） | P0 #1、P1 #2～#6、P2 #7～#8、Q1～Q10 |
| 测试报告 | `docs/reports/stallink-test-report-2026-11-26.html` 的“缺陷列表”；截图在 `docs/reports/stallink-test-2026-11-26/screens/` | D1～D5 原文 |
| D2、实收找零的既有决定 | Grok Bot 转前端的消息（position 458）：“和之前的 D1/D3/D4/D5，还有 D2 横向滑动、订单列表去掉实收找零，一起修” | D2 改实现为横向滑动；订单列表去掉实收和找零 |
| 前端进度 | Stallink-前端开发 对话：11:44、11:45 两批派给子代理 `a249e437`，任务 STA-2 v46；Mac 上 `git status` / `grep` 快照（**2026-11-26 11:51–11:54 UTC+8**） | 第 4 节各任务的“当前状态” |
| UIUX 交付物 | 父代理 12:07、12:10 两次通知已交付；文件见第 3 节 | 第 8 节 web 任务按 spec 的改动清单拆分 |
| vNext 时区定义 | `docs/superpowers/plans/2026-11-26-stallink-vnext-server.md` Task S6（迁移 `016_event_timezone_and_presale`：`events.timezone text NOT NULL DEFAULT 'Asia/Kuala_Lumpur'`，IANA 名称）、Task S8（迁移 `018`：日报按 `e.timezone` 聚合 `paid_at`、`refunded_at`） | 日报“今天”的口径（K-Q14） |
| 修复 plan | `docs/superpowers/plans/2026-11-26-stallink-review-fixes.md` 任务 3（迁移 `014`：`audit_logs.reason`、`audit_logs.detail jsonb`、`shop_id` 可空）、任务 7（取消/退款原因写审计） | 协调操作记录对象 |
| vNext 后端 plan | `docs/superpowers/plans/2026-11-26-stallink-vnext-server.md` Task S16（迁移 `023`：`before_json`、`after_json`、`terminal_id`、`client_ip`） | 协调操作记录对象 |
| 票据式模板 | `docs/design-v2/Stallink 工作台模板.html` 第 751–775 行 `viewShopinfo()`、第 434–441 行 `LOGS` | 店铺资料字段、操作记录“对象”示例；**只参考字段，版式以 screens/ 为准**（K-01） |

> 状态快照说明：下文“当前状态”以 **Stallink-前端开发 12:21（UTC+8）对 F-1～F-9 的回复**为准，并用 12:22 的磁盘快照核对。前端说明本轮已做完 D1～D5、商品名换行、店面侧栏、`.header-loading`、订单列表去掉实收和找零、Kel 关于工作台的 6 条决定；lint、类型检查、44 个测试、build 全部通过，代码**没有提交**。早先的磁盘快照是 12:08–12:14（UTC+8）在 Mac 上用 `git status --short` 和 `grep` 看到的磁盘情况，**全部未提交**（分支 `design/editorial-frontend` @ `443bd21`）。执行前请重新跑 `git status` 核对。

## 2. Kel 已定的 8 件事（K-01～K-08）

| 编号 | 决定 | 落到任务 |
|---|---|---|
| K-01 | 验收以 `docs/design-v2/screens/` 编辑式图为准，不以票据式 HTML 模板为准 | W1、T1，以及所有 web 任务的验收标准；外观相关以 UIUX 新截图为准 |
| K-02 | 总览卡片跟侧栏走，补“销售日报”“操作记录”两张卡，编号连续 | W-C2（K-Q15 = A，Kel 12:40 已定：8 张卡 01–08，侧栏管理项同号，收银组和总览不编号） |
| K-03 | 页面名称统一为：销售日报、订单记录、本场订单 | W-C2（C15、C16）、W16（C12） |
| K-04 | 店铺资料按模板补齐（客服联系、本场摊位号、默认货币、预览）；slug 由平台分配，店主只能看、不能改 | S1、S2、W14 |
| K-05 | 操作记录显示操作对象，暂不做导出 | S3、S4、W15 |
| K-06 | 日报保持现状（日期范围、分币种、更多指标），由 UIUX 更新设计稿 | U1（已交付）、W2、W16、S7 |
| K-07 | 平台外观、展场风格、店面风格三份 spec 由 UIUX 补写 | U2（已交付）、W-C1～W-C2、W-E1～W-E4、W-S1～W-S4、W-A1～W-A3、S5 |
| K-08 | 店面（shop）的 `--warning` 改成 `#b42318` | W-S1（spec §8.3 S1） |

## 3. 引用的 UIUX spec 和设计稿（已交付）

| 内容 | 路径 | 本计划用到的章节 |
|---|---|---|
| 展场风格 + 店面风格 spec | `docs/superpowers/specs/2026-11-26-stallink-theme-exhibition-shop-design.md` | §8.1 共用 C1–C16、§8.2 展场 E1–E24、§8.3 店面 S1–S19、§9 验收清单（10 条）、§10 待确认 Q1–Q6（12:44 删除原 Q5；13:09 原 Q2“店面窄屏横滑”Kel 已定后移出，其余顺延；13:19 Kel 已全部定，见 §11.2） |
| 平台外观页 spec | `docs/superpowers/specs/2026-11-26-stallink-platform-appearance-design.md` | §4.2 状态表、§8 改动清单 A1–A18、§9 验收（7 条）、§8.2 新增 A19–A22（13:30）、§10 Q9–Q12（原 Q1–Q4，Kel 13:19 已定，13:30 已全部关闭，见 §11.2） |
| 平台外观页原型 | `docs/design-v2/平台外观-原型.html` | 十种状态的画面 |
| 展场、店面工作台原型 | `docs/design-v2/展场与店面-工作台-原型.html` | 总览、收银台、销售日报 |
| 销售日报设计说明 + 原型 | `docs/design-v2/销售日报-设计说明.md`、`docs/design-v2/销售日报-原型.html` | §4.3 工具栏、§4.4 指标卡、§4.5 分区表格、§6 三档适配、§8 接口对照、§10 问题清单 1–11 |
| 截图 | `docs/design-v2/screens/展场-{总览,收银台,销售日报}-{桌面,手机}.png`、`店面-{总览,收银台,销售日报}-{桌面,手机}.png`、`平台外观-{桌面,手机,读取中-桌面,读取失败-桌面,发布确认-桌面,发布中-手机,发布失败-手机,已发布-手机,断网-手机,无权限-桌面}.png`、`销售日报-v2-{桌面,平板,手机,空日,加载中,出错,手机-展场}.png` | 各任务的“对照截图” |

**spec 里的待确认问题怎么处理：** 展场/店面 Q1–Q6（经 12:44、13:09 两次重编号）、平台外观 Q9–Q12（原 Q1–Q4）、日报 R1–R4 和问题 2、11 都已由 Kel 定（13:19、13:32），UIUX 13:30 写进设计文件；本计划以 13:30 设计文件的编号为准，对照表见 §11.2。设计文件和 Kel 其他决定互相矛盾的地方列在 §11.4（C-1、C-2）。

## 4. 任务总表

状态取值：前端已完成、自测通过，待复测（前端 12:21 回报做完，lint、tsc、44 个测试、build 通过，但还没有测试 bot 复测）、部分完成、待开始、待后端、待 Kel 确认、待 K-Q15。依据：**Stallink-前端开发 12:21（UTC+8）对 F-1～F-9 的回复**，加上 12:08–12:22 在 Mac 上 `git status`、`git diff`、`grep` 的磁盘核对（全部未提交，分支 `design/editorial-frontend` @ `443bd21`）。前端本轮的改动都没有提交，**本计划的任务要等 Kel 通过 Grok Bot 确认后才开工**。

**server**

| 编号 | 标题 | 优先级 | 依赖 | 当前状态（依据） |
|---|---|---|---|---|
| S0 | 引用修复 plan 任务 7（原因 2–500、一步取消并退款、迁移 014） | — | 修复 plan | 由修复 plan 负责，本计划不重复 |
| S1 | 迁移 024：shops 加 contact、booth_number、default_currency；platform_appearance 加 updated_at、updated_by；audit_logs 加索引；schema.go 对齐 | P1 | 修复 plan 任务 3；K-Q1 = A、K-Q2 = A、K-Q6 = B、K-Q16 = A（Kel 已定） | 已批准开工；024 只改结构，024a 手动回填（严格按最近一场活动）+ 回滚；**只在本机 docker-compose Postgres 上跑** |
| S2 | `GET /api/admin/shop`；`PATCH` 扩字段、slug 只读、停用同事务删会话 | P1 | S1；K-Q3 = B、K-Q4 = A、K-Q9 = A（Kel 已定 13:22） | 待开始；后端已答复 B-1～B-5（`http.go:81` 只有 PATCH；`admin.go:18-38` 仍校验并写回 slug） |
| S3 | 操作记录返回 `subjectLabel` | P1 | S1 的索引；阶段 2 依赖修复 plan 迁移 014、任务 7；K-Q5 = C（Kel 已定） | 已批准开工（阶段 1）；后端已答复 B-6、B-7 |
| S4 | 操作记录 `createdAt` 带时区（含 `order_store.go:864`） | P2 | 无 | 待开始；后端已答复 B-8 |
| S5 | 平台外观 GET 只对平台管理员返回 `updatedAt`、`updatedBy {id,name}`，其他人只有 `{ theme }`（外观 Q9，Kel 已定 13:19、13:32） | P2 | S1；修复 plan 任务 10；外观 spec §8.1、A19 | 待开始；补匿名 / 店铺会话 / 平台管理员三种身份测试 |
| S6 | “实收 · 找零”前后端一起清理（K-Q10 = A） | P2 | K-Q10 = A、K-Q7 = B、K-Q17 = A（Kel 已定） | 已批准开工；不动库、不出 SQL；在线收款和 S9b 都不再收 `cashReceivedMinor`，传了就忽略 |
| S10 | 日报 | 快照根和 `currencies[]` 加 `cashTenderedOrders`（录了实收的单数，口径同 `cashOverShortMinor`；不动表、不加迁移） （多币种按币种各自判断，K-Q22 = A，Kel 15:15）| P2 | S6（已提交 `ee103c1`）；K-Q18 = A（Kel 15:01） | **已提交 `9f7e7f4`（Kel 经 Grok Bot 同意单独提交，未 push）** |
| S7 | 日报 CSV 改 5 列 `分组,币种,项目,数量,金额` 并按屏幕指标补齐（设计说明 R2 / §8.3） | P1 | R2；C-1 = A、C-2 = A（Kel 已定 13:55） | 已批准开工；5 列，无“现金长短款”行；B-12 的 6 列作废 |
| S8 | 日报“报表今天”由服务端下发 `{ today, timezone }`（R4 / §8.4） | P2 | K-Q14 = A（R4）；和 vNext S6/S8 对齐 | Kel 已定，待开工 |
| S9 | 新建活动带店铺默认摊位号和默认货币；公开接口返回 `displayBoothNumber` | P1 | S1、S2；K-Q1 = A、K-Q2 = A（Kel 已定 13:22） | Kel 已定，待开工 |

**web**

| 编号 | 组 | 标题 | 优先级 | 依赖 | 当前状态（依据） |
|---|---|---|---|---|---|
| W1 | 基线 | 验收基线：页面 × 截图对照表 | P0 | K-01 | 待开始（前端 12:21 回报的已完成项里没有这一项） |
| W-C1 | 共用 | token、遗留旧绿、焦点环、全局减少动效、主题开关（spec §8.1 C1–C7，含店面 S17） | P2 | 无 | 部分完成（C2 `.header-loading` 前端已完成、自测通过；C3、C4：`globals.css` 仍有 2 处 `rgba(49, 89, 66`；C1、C5、C6 全局规则、C7 未做） |
| W-C2 | 共用 | 侧栏编号、总览卡片、页面名称（按 K-Q15 方案 A；C8、C10、C11、C15、C16） | P1 | **K-Q15 = A（Kel 已定 12:40）**；K-02、K-03；展场 Q5 = A（Kel 已定 13:19）；UIUX 改 spec 已完成（12:44） | Kel 已定、spec 已同步，待开工（侧栏已对齐 01–08、F-10 已修；C16 两个 H1 已完成；总览 01、02 标题仍是“活动与场次”“商品与套餐”、卡片未分“经营 / 店铺”两组；C10 未做；C15 仍是“创建订单”） |
| W-C3 | 共用 | classic 交互控件边框改用 `--line-control`（`#8f8675` / 深色 `#676c78`；48 处 CSS + 1 处内联 = UIUX 口径 40 + 附录 5 + §3.2 的 4；主题块 `--line-control`：店面 `#8f8c8b`、展场顾客端 `#857e6e`（2 块）、展场工作台 `#6f8681`（K-Q19 = A，Kel 15:01），`--line-strong` 不动；展场工作台 3 处控件按 E28 改读；卡片悬停边框（13:55 第 358 行）保持 `--line-strong`，Kel 14:40） | P1 | 无（建议在 W-C1 之后）；Kel 14:13 采纳；K-Q19 = A（15:01） | 已批准开工；前端磁盘上已做完（未提交；14:42 核对 E28 的 3 处已改，358 行按 Kel 14:40 不改） |
| W2 | 日报 | D1 复测 + 指标卡改 5 张（去长短款，C-2 = A） | P0（D1 复测）/ P1（指标卡） | 和 W16 同组 | **D1 已修，待复测；5 张卡排法待做**（前端 F-5：≤900 宽 `--workspace-rail: 0px`（`exhibition-workspace.css:582`），`.report-metric` 加 `min-width: 0`、改用 `clamp` 字号，`nowrap` 有意保留以免金额折行，390 宽实测不再横向溢出；≤720 宽 `globals.css:2705` 已是 2×3；桌面仍是 4+2（`globals.css:2597`）） |
| W3 | 共用 | classic 收银台商品名一字一行 | P1 | 无 | 前端已完成、自测通过，待复测（前端 12:21：商品名换行已做完） |
| W6 | classic | D3：390 已登录页眉溢出 8px | P2 | 无 | 前端已完成、自测通过，待复测（D3；`globals.css:3065` `flex-wrap: wrap`） |
| W10 | 共用 | 订单列表去掉“实收 · 找零”；收款请求清理；日报隐藏“现金长短款” | P1 | S6（K-Q10 = A）；K-Q7 = B、C-2 = A | 展示部分前端已完成、自测通过，待复测；请求清理和日报隐藏待开工，和 S6 同批上线；**K-Q18 = A：PDF 长短款在 `cashTenderedOrders` 为 0 时显示「—（未录实收）」，依赖 S10（Kel 15:01）**；多币种按币种各自判断（K-Q22 = A，Kel 15:15） |
| W14 | 店铺页 | 店铺资料页按模板补齐 | P1 | S2；K-Q1 = A、K-Q2 = A、K-Q3 = B、K-Q4 = A（Kel 已定 13:22） | 部分完成 + 待后端（三个字段 `disabled`，提示“等后端”；“本场摊位号”按 K-Q1 = A 改名“默认摊位号”（F-8）） |
| W15 | 操作记录 | 对象列 | P1 | S3；K-Q5 = C（Kel 已定） | 部分完成 + 待后端（显示格式前端同意（F-9）：“类型 · 名称”，没有名称时“类型 #id”；现在的回退是“订单 101”或“—”） |
| W16 | 日报 | 销售日报按 v2 设计稿：标题眉标、工具栏、热销四列、分币种列模板（设计说明 §4.3–§4.5、§6；spec C12–C14；D5） | P1 | U1（已交付）；K-Q11 = A、K-Q13 = A（R1、R3）；日报问题 2、11（Kel 13:32）；C-2 = A（五列，无长短款） | 部分完成（H1 已是“销售日报”，“经营”眉标未加；**D5 前端已完成**：fixture 实测 1280、860 宽两组排成一行，手机叠成两行，真实登录截图待 T1；热销仍是 3 个单元格、序号写在名称里（`page.tsx:428-430`）） |
| W17 | 日报 | 日报默认日期改用服务端“报表今天”、日期 `max` | P2 | S8；K-Q14 = A（R4） | Kel 已定，待后端（`page.tsx:39` `localToday()` 仍用浏览器日期；失败时不退回浏览器日期） |
| W18 | 顾客端 | 摊位号显示规则（活动优先，没填用店铺的）；新建活动表单预填店铺默认值 | P1 | S9、S2；K-Q1 = A、K-Q2 = A | Kel 已定，待后端 |
| W19 | 共用 | 测试报告 D9（店面 ≤900 侧栏「退出登录」错位；按钮保留，只修底色错位，Kel 15:01）、D10（缺 favicon） | P2 | 无；D9 和 W-S2 改同一块；D10 图案 K-Q20 = A（Kel 15:11，`option-a*` 系列） | Kel 已定，待开工 |
| W-E1 | 展场 | 工作台 token、对比度、阴影、圆角、衬线、控件边框（spec §8.2 E1–E7、E13、E14、E25–E28；D4） | P2 | 展场 Q1 = A（Kel 已定 13:19）；E28 控件边框色 K-Q19 = A：`#6f8681`（Kel 15:01） | 部分完成（D4 前端已完成：`--muted: #4f696a`，在 `#d5e6df` 上 4.553:1；前端同意按 spec E1 改成 `#4b6465` 后复测（F-3）；E2 仍 `#d6ef9f`；E6、E7 仍写死 `#d8f69a`） |
| W-E2 | 展场 | 侧栏底部、触控尺寸、窄屏横滑导航（E8–E11；D2） | P1 | W-C2（C10）；展场 Q3 = A 保留 152px（Kel 已定 13:19） | 部分完成（D2 横滑前端已完成、自测通过；E8–E10 未做；E11 已定保持 `display: none`，只加防回退测试） |
| W-E3 | 展场 | 总览横幅、收银台单列（E12、E15） | P1 | W-E1；展场 Q1 = A 去海军蓝（Kel 已定 13:19） | 待开始（`exhibition-workspace.css:697` 仍是 `#0e203a, #22467c`） |
| W-E4 | 展场 | 顾客端商品页去钴蓝、token（E16–E21） | P2 | 展场 Q1 = A 去钴蓝（Kel 已定 13:19）；52px 竖栏在 W-E5 | 待开始（`exhibition.css` 仍有 6 处 `#1e36a8`） |
| W-E5 | 展场 | 窄屏退出改到 52px 竖栏底部，删 390 顶栏模式（E22–E24、E29、E30） | P1 | W-E1；展场 Q2；**Kel 14:40 选 B（竖栏底部，“顶栏退出不回退”作废）** | Kel 已定，待开工 |
| W-S1 | 店面 | token：`--warning` 等（spec §8.3 S1–S5、S18；K-08） | P1 | K-08；展场 Q4 = A 系统字体（Kel 已定 13:19） | 部分完成（S1 `--warning: #b42318`、S5 去掉 Inter 已完成；`--shop-muted: #726f6e` **已解决**（UIUX 已回填 spec 第 101 行、改动清单 S19）；S2 仍 `#ebebeb`，S3 仍 `#5433eb`，S4 未加） |
| W-S2 | 店面 | 侧栏底部固定、导航区单独滚动、窄屏横滑（S8–S12；UIUX P1 #5） | P1 | W-C2（C10）；店面窄屏横滑 Kel 已定（spec §2 第 7 条） | 部分完成（店面侧栏前端已完成**临时方案**：`shop.css:1307` 整栏 `overflow-y: auto`，底部按钮能看到；前端确认是临时方案（F-2），改成导航区单独滚动、底栏固定放在本任务；S11 仍是 860 宽折叠按钮）；**Kel 15:01 / 15:13：吸顶跟页头高度走——>900 `96px`、641–900 `72px`、≤640 `64px`（K-Q21 = B；吸顶不得用 56px（不得出现 `top: 56px`，页头 `min-height` 也不得是 56px）；与吸顶无关的 56px 保留；变量名 `--shop-header-height`）；窄屏「退出登录」保留（S11 的“隐藏底部操作”作废），D9 只修错位**；前端 15:14 已按此改成 `--shop-header-height` 变量（未提交）；**层叠 bug：`shop.css:1237` 在 ≤900 块重复声明 72px 盖掉 ≤640 的 64px；前端 15:29 已删（未提交），测试和 390 / 320 / 768 实测待补（W-S2）** |
| W-S3 | 店面 | 标题字号、≤900 顶栏（S6、S7、S13） | P2 | 无 | 待开始 |
| W-S4 | 店面 | 收银台单列、胶囊筛选、禁用按钮（S14–S16） | P1 | 无 | 待开始 |
| W-A1 | 外观页 | 状态与发布流程（外观 spec §8 A2–A11、A17、A18） | P2 | 外观 Q3 = B 全站横幅移出本期、Q4 = B 由 W-A4 实现（Kel 已定 13:19） | 待开始（`platform/appearance/page.tsx` 只改了 1 行）；**Kel 15:01：发布确认框焦点放「取消」，`ConfirmDialog` 加可选 `initialFocus`，其他页不变** |
| W-A2 | 外观页 | 双面缩略图与固定示例色（A1、A12–A16） | P2 | 外观 Q2 = A 不做预览（Kel 已定 13:19） | 待开始（`appearance.css:30` 仍写死 `#5433eb`） |
| W-A3 | 外观页 | 状态条显示发布人和时间（A19） | P2 | S5；外观 Q9（原 Q1，Kel 已定） | 待后端（S5） |
| W-A4 | 外观页 | 外观读取降级提醒（外观 Q12、A20–A22） | P2 | 外观 Q12（原 Q4，Kel 已定） | Kel 已定，待开工 |
| U1 | 设计 | 销售日报设计稿更新 | P1 | K-06 | 已完成（`docs/design-v2/销售日报-设计说明.md`、`销售日报-原型.html`、`screens/销售日报-v2-*.png` 7 张；父代理 12:10 通知） |
| U2 | 设计 | 平台外观、展场、店面 spec | P1 | K-07 | 已完成（两份 spec + 两个原型 + 22 张截图；父代理 12:07 通知） |
| T1 | 测试 | 补测与补截图（含真实登录 1280 截图） | P1 | W 系列 | 待开始 |

## 5. 外观问题和测试缺陷原文对照

### 5.1 测试缺陷 D1～D5（原文摘自 `docs/reports/stallink-test-report-2026-11-26.html`“缺陷列表”）

| 编号 | 报告定级 | 本计划优先级 | 原文要点 | 任务 |
|---|---|---|---|---|
| D1 | 中 | **P0**（UIUX 升为 P0） | exhibition 外观在 390 宽度下，/admin/reports 出现横向溢出。期望：页面宽度等于视口宽度，指标卡在内容区里自动换行。实际：`document.scrollWidth=460 > 390`；report-metric 等 12 个元素右边缘超出视口（296→460px），整页可以横向拖动，右侧卡片被截掉。截图 `screens/d-exhibition-390-reports.png` | W2（属于日报 W16 组） |
| D2 | 中 | P1 | exhibition 外观手机导航不是横向滑动，和验收口径不一致。实际：固定左侧竖向窄栏（宽约 74px，`.workspace-nav` 设了 `overflow-x:hidden`），挤占内容宽度。截图 `screens/d-exhibition-390-nav.png`。Kel 已定改实现 | W-E2 |
| D3 | 低 | P2 | classic 外观在 390 宽度下，已登录时店面页眉超出 8px。复现页：`/`、`/login`、`/c/demo`、`/c/demo/listings/500`、`/about`。实际：`nav.site-header__actions` 右边缘在 398px，“退出”按钮被截掉一半，整页可以横向滚动 8px。截图 `screens/d-classic-390-header.png` | W6 |
| D4 | 低 | P2 | exhibition 外观辅助灰字压在渐变背景上，对比度临界不达标。文字色 `#526c6d`，背景像素 `#dae8e2–#edf4f1`，对比度 4.46–5.05:1，最深处不到 4.5（`p.field-hint`，10.88px）。建议 muted 再暗约 5% | W-E1 |
| D5 | 低 | P1（UIUX 列为 P1 #4） | 日报工具栏在 1280 宽度下按钮折成三行（三套外观都有）。操作区宽约 491–517px，按钮排成“重新生成日报 / 全部币种（下拉占满整行）/ 导出 CSV + 导出 PDF”。截图 `screens/d-shop-1280-report-toolbar.png` | W16 |

### 5.2 UIUX 走查外观问题（原文摘自 UIUX 发给 Grok Bot 的验收结果）

| UIUX 编号 | 定级 | 原文要点 | 任务 |
|---|---|---|---|
| 1 | P0 | /admin/reports 在展场风格 390 宽下整页横向溢出（scrollWidth 460）。原因：`globals.css` 约 2599 行 `.report-metric strong { white-space: nowrap }`，以及展场在 390 宽仍保留 74px 左侧窄栏。改法：指标卡在内容区里换行，排成 2×2，和 `销售日报-手机.png` 一致 | W2（即 D1，属于日报 W16 组） |
| 2 | P1 | 热销商品名一个字一行：`globals.css` 第 2625 行 `.report-rank span` 用了 grid 加 `overflow-wrap: anywhere`，classic 和展场都有。改成 `break-word`，名称列不要压太窄。截图 `classic-1280-admin_reports.png` | W16（热销表四列 + C14） |
| 3 | P1 | 收银台商品名同样一个字一行，比如“亚克力立牌·海雾”。截图 `classic-1280-counter.png`、`shop-1280-counter.png` | W3（classic）、W-E3（E15）、W-S4（S14） |
| 4 | P1 | 日报工具栏 1280 宽下折成三行（D5）。`reports.css` 按视口写的 `min-width: 70rem` 不起作用；`.report-toolbar__actions` 要按自身宽度排成一行，或拆成“生成”和“导出”两组 | W16（即 D5） |
| 5 | P1 | 店面风格 1280 宽下，浮卡侧栏底部的“返回 Stallink”“退出登录”被裁掉（`shop.css` 的 `.workspace-sidebar`）。侧栏内部可滚动，或压缩高度。截图 `shop-1280-admin.png` | W-S2 |
| 6 | P1 | 补测补截图：商品页（要有数据）、收款方式、销售账号、店铺资料、操作记录（要有数据），以及平台 3 页；还有售罄、空数据、出错状态 | T1 |
| 7 | P2 | 展场 `--muted: #526c6d` 压在渐变底上对比度约 4.35:1（D4），调暗约 5%（`exhibition-workspace.css` 第 22–35 行） | W-E1（即 D4） |
| 8 | P2 | `globals.css` 第 133 行 `.header-loading` 残留旧绿色 `rgba(49,89,66,.06)` | W-C1（spec C2） |
| SU-4 摸底（UIUX 派给 spec 子代理的已知问题） | P2 | 展场顾客端赭红 `#8c3c2d` 配柠绿 `#e3f7a2`，和工作台深青 `#215e58` 不是一套色；展场 `.product-page` 把强调色改回钴蓝；展场顶栏的深浅色开关点了没反应；店面风格声明了 Inter 但没加载；店面 860 宽直接收成折叠按钮；圆角没做成 token；`appearance.css` 写死 `#5433eb` | W-E4、W-E2、W-C1（C7）、W-S1（S5）、W-S2（S11）、W-S1（S4/S18）、W-A2（A13）——目标值已由 spec 给出 |

## 6. 文件结构

**server（新增或修改）**
- 创建：`server/internal/data/migrations/024_shop_profile_and_appearance_meta.sql`、`server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql`、`server/internal/data/migrations_024_test.go`；修改 `server/internal/data/schema.go:14-23`（S1）
- 创建：`server/internal/data/shop_profile.go`、`server/internal/data/shop_profile_integration_test.go`；修改 `server/internal/service/admin.go:18-38`（`UpdateShopSettings`）并新增 `AdminShop`；修改 `server/internal/server/http.go:81` 附近注册 `GET /api/admin/shop`；创建 `server/internal/service/shop_profile_integration_test.go`（S2）
- 修改：`server/internal/data/order_store.go:64-70`（`AuditEntry.SubjectLabel`）、`server/internal/data/paged_lists.go:51-80`、`server/internal/data/paged_lists_integration_test.go`、`server/internal/service/admin.go:258-282`（S3、S4）
- 修改：`server/internal/data/platform_appearance.go`、`server/internal/service/platform_appearance.go`、`server/internal/service/platform_appearance_integration_test.go`（S5）
- 修改：`server/internal/service/reports.go:51-115`（`renderDailyReportCSV`）、`server/internal/service/report_csv_test.go`（S7）
- 修改：`server/internal/service/reports.go:117-135`（`reportDate`、新增“报表今天”接口），`server/internal/server/http.go`（S8）
- 创建：`server/internal/data/manual/024a_backfill_shop_default_currency.sql`、`.rollback.sql`、`manual/README.md`（S1 步骤 5b，不被 embed；保留与否看 K-Q16）
- 修改：`server/internal/service/staff.go`、`server/internal/service/handlers.go`、`server/internal/service/order_commands.go`、`server/internal/data/order_checks.go`、`order_store.go`、`paged_lists.go`、`shop.go`（S6 收款清理）
- 修改：`server/internal/data/catalog_store.go:225、475`、`bundle_store.go:115`、`service/handlers.go:352、583、595`；创建 `server/internal/service/event_defaults_integration_test.go`（S9）

**web（修改；多数前端已经在改）**
- 共用：`web/src/app/globals.css`、`web/src/components/app-nav.tsx`、`web/src/components/workspace-shell.tsx`、`web/src/app/admin/page.tsx`、`web/src/app/counter/page.tsx`、`web/src/app/counter/orders/page.tsx`、`web/src/app/admin/orders/page.tsx`
- 日报：`web/src/app/admin/reports/page.tsx`、`web/src/app/admin/reports/reports.css`、`web/src/app/globals.css:2597-2710`（`.report-hero`、`.report-metric`、`.report-rank`）
- 展场：`web/src/app/exhibition-workspace.css`、`web/src/app/exhibition.css`
- 店面：`web/src/app/shop.css`
- 外观页：`web/src/app/platform/appearance/page.tsx`、`web/src/app/platform/appearance/appearance.css`
- 店铺资料、操作记录、实收：`web/src/app/admin/shop/page.tsx`、`web/src/app/admin/audit/page.tsx`、`web/src/lib/audit.ts`、`web/src/lib/order-actions.ts`、`web/src/lib/errors.ts`、`web/src/lib/api.ts`
- 测试：已有 `web/tests/workspace-copy.test.cjs`、`web/tests/audit.test.cjs`、`web/tests/order-actions.test.cjs`；新增 `web/tests/workspace-nav.test.cjs`、`web/tests/theme-tokens.test.cjs`、`web/tests/report-layout.test.cjs`、`web/tests/appearance-page.test.cjs`、`web/tests/shop-settings.test.cjs`
- 顾客端摊位号、新建活动预填（W18）：创建 `web/src/lib/booth.ts`、`web/tests/booth-display.test.cjs`；修改 `c/[slug]/events/[eventId]/page.tsx`、`c/[slug]/listings/[listingId]/page.tsx`、`c/[slug]/bundles/[bundleListingId]/page.tsx`、`components/booth-view.tsx`、`components/goods-grid.tsx`、`lib/api.ts`、`admin/events/page.tsx`、`admin/goods/page.tsx`
- 外观读取失败提醒（W-A4）：`web/src/lib/api.ts`、`web/src/app/layout.tsx`、`web/src/components/workspace-shell.tsx`、新建 `web/tests/theme-read-failed.test.cjs`
- 展场退出入口（W-E5）：`web/src/app/exhibition.css`、`web/src/components/app-nav.tsx`
- 视觉复测：`docs/reports/stallink-test-2026-11-26/scripts/layout-sweep.cjs`、`contrast2.cjs`、`mock-api.cjs`（输出放 `/tmp`，不进仓库）

---

## 7. 后端任务（server）

### 任务 S0：引用修复 plan 任务 7（原因必填、一步取消并退款），本计划不重复写

> **端：** server ｜ **状态：** 由修复 plan 负责，本计划只标依赖

前端这一轮提出的“cancel / refund-confirm 接收 reason”和“一步完成取消并退款”，已经写在 `docs/superpowers/plans/2026-11-26-stallink-review-fixes.md` 任务 7：`cancel`、`refund-confirm`、新增 `cancel-refund` 都接收 `{ "reason": string }`，`TrimSpace` 后按 Unicode 字符数 2–500，`reason_required` / `reason_too_short` / `reason_too_long`，计入幂等哈希，写入 `audit_logs.reason` 和 `detail`；表结构在任务 3 的迁移 `014`。前端对接点 `web/src/lib/order-actions.ts` 的 `orderActionRequestBody` 也在那份 plan 里。本计划里依赖它的地方：
- S3 阶段 2（对象快照写进 `detail.subject`）要等任务 3、7 合入。
- S3 的中文动作名 CASE 要在任务 7 合入后补上 `order.cancel_refund`。
- W15 的对象列在任务 7 上线后会多出“取消并退款”这类记录，不需要额外改动。

---

### 任务 S1：迁移 024，shops 增加 contact、booth_number、default_currency；platform_appearance 增加 updated_at、updated_by；audit_logs 加索引（本计划唯一的一份升级 SQL + 一份回滚 SQL）

> **端：** server ｜ **优先级：** P1 ｜ **依赖：** 修复 plan 任务 3（建立 `migrations/down/` 目录和“down 不被加载”的测试）；K-Q1 = A、K-Q2 = A（Kel 已定（2026-11-26 13:22，Grok Bot 转达））；**024 只改结构**，存量回填是单独的手动脚本 024a，自带回滚（步骤 5b；**K-Q16 = A，Kel 已定 2026-11-26 13:55**）；**K-Q6 = B**（云端由 Kel 自己备份、手动执行，不在本计划里） ｜ **状态：** 已批准开工；**024、024a 只在本机 `docker-compose` 起的 Postgres 上跑**

**编号核实：** 2026-11-26 11:51 和 12:05（UTC+8）两次在 Mac 上 `ls server/internal/data/migrations`，只有 `001`～`013`，`down/` 目录还不存在。修复 plan 占 `014`（`014_order_payment_reference_and_audit`），vNext 后端 plan 占 `015`～`023`（`015_listing_cap_and_floor` … `023_audit_snapshots_and_user_locale`）。本计划用 **`024`**。**后端已答复（B-15）：** 后端两个 worktree 的迁移最高都只到 013，编号没有冲突；编号在合入时按重新 `ls` 的结果确定，024 可以先落地，**已经落地的编号不再改**。如果合入时 024 已被占，顺延编号并同步改本节所有文件名。

**字段定义：**

| 列 | 类型 | 默认值 | CHECK | 存量回填 | JSON |
|---|---|---|---|---|---|
| `contact` | `text NOT NULL` | `''` | `char_length(contact) <= 120`，且不含控制字符（`contact !~ '[[:cntrl:]]'`） | `''` | `contact` |
| `booth_number` | `text NOT NULL` | `''` | `char_length(booth_number) <= 40`（和 `events.booth_number` 一致），不含控制字符，首尾无空格（`booth_number = btrim(booth_number)`） | `''`（不从活动复制；K-Q1 = A：店铺摊位号只是新建活动的默认值，顾客端活动没填时才回退到它，见 S9、W18） | `boothNumber` |
| `default_currency` | `text NULL` | `NULL`（未设置） | `default_currency IS NULL OR default_currency IN ('MYR','SGD','CNY')`（和 `events.currency` 枚举一致） | **024 不回填**（保持 `NULL`）。回填由手动脚本 `server/internal/data/manual/024a_backfill_shop_default_currency.sql` 做（K-Q2 = A；规则见步骤 5b；⚠️ 云端执行必须 Kel 手动确认） | `defaultCurrency` |
| `updated_at` | 已存在（`schema.go:22`，`timestamptz NOT NULL DEFAULT now()`） | — | — | 不动 | `updatedAt`（RFC3339 带时区） |
| `platform_appearance.updated_at` | `timestamptz NULL` | `NULL` | — | `NULL`（历史发布时间不可考，界面显示“暂无发布记录”） | `updatedAt` |
| `platform_appearance.updated_by` | `bigint NULL REFERENCES users(id) ON DELETE SET NULL` | `NULL` | — | `NULL` | `updatedBy`（`{ id, name }`，只对平台管理员返回，见 S5） |

- **`events.booth_number` 已存在，不用加进 024**（13:22 只读核对）：`schema.go:54` `booth_number text NOT NULL DEFAULT '' CHECK (char_length(booth_number) <= 40)`，是基线建表就有的列（`migrations/001`～`013` 里没有对它的 `ADD COLUMN`）；读写在 `catalog_store.go:236、281、298、322`，接口字段 `boothNumber`（`handlers.go:599` `eventJSON`）。
- 摊位号格式只限长度和空白，不加字母数字正则：现有 `events.booth_number` 也只有长度限制，各展会编号写法不一（`A-27`、`B12/B13`、`Hall 5 · 3F`），格式收紧列为 B-3 问后端。
- **平台外观并进 024 的理由：** 现在外观配置存在单行表 `platform_appearance`（`012_platform_appearance.sql`：`singleton boolean PRIMARY KEY`、`theme text`），**没有 `updated_at`、`updated_by`**；读写在 `server/internal/data/platform_appearance.go:8-25`。平台外观 spec（`2026-11-26-stallink-platform-appearance-design.md` §10 Q1，推荐 B）要显示“由 X 于某时发布”。它和店铺资料同属本计划这一个需求，两张表互不依赖、都是加可空列，合在一份 024 里不会互相阻塞；回滚也是各自 DROP COLUMN。唯一的代价是：如果 Kel 对外观 Q1 选 A（不做），要从 024 里删掉 platform_appearance 那一段（S1 合入前删即可）。所以合并合理。
- **和修复 plan 任务 10 的关系：** 任务 10 会在 `SetPlatformAppearance` 里写平台审计 `platform.appearance.update`（依赖迁移 014 的 `audit_logs.shop_id` 可空）。审计是历史流水，`updated_at/updated_by` 是“当前这一版”的元数据，两者不重复；S5 写这两列时放进任务 10 的同一个事务。
- **`updated_at` 由应用层写，不加触发器。** 依据：仓库里没有任何触发器（`grep -rni trigger server/internal/data` 无结果），现有更新语句都在 SQL 里写 `updated_at = now()`（`catalog_store.go:157`、`:180`，`shop.go:98`）。S2 的 `UpdateShopProfile` 沿用这个写法。
- **audit_logs 索引（后端 B-6 已答复）：** 024 里加 `CREATE INDEX IF NOT EXISTS audit_logs_shop_created_id_idx ON audit_logs (shop_id, created_at DESC, id DESC);`，**不用 `CONCURRENTLY`**（迁移在事务里跑）；down 里对应 `DROP INDEX IF EXISTS`。核实：`audit_logs` 现在**没有**这类索引，早先“现有索引够不够用”的说法已删掉。
- **platform_appearance（后端 B-10 已答复）：** 024 只做 `ADD COLUMN`，`updated_by REFERENCES users(id) ON DELETE SET NULL`；**不改 `012`、`013`**。
- **schema.go 对齐（后端要求）：** 同一个 PR 里把 `server/internal/data/schema.go:14-23` 的 shops DDL 对齐到 024 之后的结构（见步骤 3b）。
- **回填不放进 024（K-Q2，13:22）：** Kel 要求云端回填必须等他手动确认后才执行，而 API 启动时会自动执行 024（`//go:embed migrations/*.sql`）。所以 **024 只改结构**；`default_currency` 的存量回填写成单独的手动 SQL，放在 `server/internal/data/manual/`（不在 embed 路径下，API 启动不会执行），并附回滚脚本（步骤 5b）。回填**不改 `shops.updated_at`**（后端确认）。**K-Q16 = A（Kel 已定 13:55）：** 024a 作为单独的手动脚本，自带回滚；024 本身仍是一份升级 + 一份回滚。
- **K-Q6 = B（Kel 已定 13:55）：** 云端由 Kel 先备份，再手动执行 024，确认没问题后再部署。**开发只在本机 `docker-compose` 的 Postgres 上跑 024 和 024a，云端库一律不动**，也不用本机 API 连云端库。

**文件：**
- 创建：`server/internal/data/migrations/024_shop_profile_and_appearance_meta.sql`
- 创建：`server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql`
- 修改：`server/internal/data/schema.go:14-23`（shops DDL 对齐，同一个 PR）
- 测试：`server/internal/data/migrations_024_test.go`、`server/internal/data/shop_profile_integration_test.go`（024 不回填、约束、手动回填脚本用例，见步骤 5、5b）

- [ ] **步骤 1：编写失败的测试**

```go
// server/internal/data/migrations_024_test.go
package data

import (
	"strings"
	"testing"
)

func TestLoadMigrationsIncludes024ShopProfileFields(t *testing.T) {
	t.Parallel()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations failed: %v", err)
	}
	var found *migration
	for i := range migrations {
		if strings.Contains(migrations[i].name, "down") {
			t.Fatalf("rollback file loaded as forward migration: %d_%s", migrations[i].version, migrations[i].name)
		}
		if migrations[i].version == 24 {
			found = &migrations[i]
		}
	}
	if found == nil {
		t.Fatal("expected migration 024 to be loaded")
	}
	if found.name != "shop_profile_and_appearance_meta" {
		t.Fatalf("unexpected 024 name: %s", found.name)
	}
	for _, want := range []string{
		"contact", "booth_number", "default_currency",
		"shops_contact_check", "shops_booth_number_check", "shops_default_currency_check",
		"platform_appearance", "updated_by",
		"audit_logs_shop_created_id_idx",
	} {
		if !strings.Contains(found.sql, want) {
			t.Fatalf("024 sql missing %q", want)
		}
	}
}
```

- [ ] **步骤 2：运行测试，确认失败**

运行：`cd server && go test ./internal/data -run TestLoadMigrationsIncludes024ShopProfileFields -v`
预期：FAIL，报 `expected migration 024 to be loaded`

- [ ] **步骤 3：创建升级 SQL 和回滚 SQL**

```sql
-- server/internal/data/migrations/024_shop_profile_and_appearance_meta.sql
-- 024：店铺资料按工作台模板补齐（Kel 2026-11-26，K-04）。
-- contact：客服联系，自由文本，顾客可见。
-- booth_number：店铺级摊位号；K-Q1 = A：新建活动的默认值，顾客端活动没填时回退到它（应用层，见 S9）。
-- default_currency：店铺默认货币，NULL 表示未设置。
-- updated_at 已存在，由应用层在 UPDATE 里写 now()（项目没有触发器）。
-- slug 不在这里改：仍是 NOT NULL UNIQUE，只是应用层改成店主只读（任务 S2）。
-- platform_appearance：记录当前风格最近一次由谁、何时发布（平台外观 spec §10 Q1 = B，Kel 13:19 已定，任务 S5）。
ALTER TABLE shops
	ADD COLUMN IF NOT EXISTS contact text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS booth_number text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS default_currency text;

DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_contact_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_contact_check
			CHECK (char_length(contact) <= 120 AND contact !~ '[[:cntrl:]]');
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_booth_number_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_booth_number_check
			CHECK (char_length(booth_number) <= 40 AND booth_number !~ '[[:cntrl:]]' AND booth_number = btrim(booth_number));
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'shops'::regclass AND conname = 'shops_default_currency_check') THEN
		ALTER TABLE shops ADD CONSTRAINT shops_default_currency_check
			CHECK (default_currency IS NULL OR default_currency IN ('MYR', 'SGD', 'CNY'));
	END IF;
END $$;

-- 存量回填不在这里（K-Q2，Kel 13:22）：024 会随 API 启动自动执行，回填必须等 Kel 手动确认，
-- 所以放到 server/internal/data/manual/024a_backfill_shop_default_currency.sql（不会被 embed）。

-- 平台外观元数据：历史值不可考，保持 NULL。
ALTER TABLE platform_appearance
	ADD COLUMN IF NOT EXISTS updated_at timestamptz,
	ADD COLUMN IF NOT EXISTS updated_by bigint REFERENCES users (id) ON DELETE SET NULL;

-- 操作记录分页索引（后端 B-6）：迁移在事务里执行，不用 CONCURRENTLY。
CREATE INDEX IF NOT EXISTS audit_logs_shop_created_id_idx ON audit_logs (shop_id, created_at DESC, id DESC);
```

```sql
-- server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql
-- 024 手动回滚脚本：放在 migrations/down/，不会被 //go:embed migrations/*.sql 加载。
-- ⚠️ 需 Kel 手动确认：对任何非本机数据库执行前，必须先得到 Kel 批准。
-- 执行：psql "$DB_URL" -f server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql
-- 回滚会丢掉店主填写的客服联系、店铺摊位号、默认货币，以及外观的发布人和时间，执行前先导出：
--   psql "$DB_URL" -c "\copy (SELECT id, contact, booth_number, default_currency FROM shops) TO 'shops_profile_backup.csv' CSV HEADER"
--   psql "$DB_URL" -c "\copy (SELECT updated_at, updated_by FROM platform_appearance) TO 'appearance_meta_backup.csv' CSV HEADER"
BEGIN;
DROP INDEX IF EXISTS audit_logs_shop_created_id_idx;
ALTER TABLE platform_appearance
	DROP COLUMN IF EXISTS updated_by,
	DROP COLUMN IF EXISTS updated_at;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_default_currency_check;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_booth_number_check;
ALTER TABLE shops DROP CONSTRAINT IF EXISTS shops_contact_check;
ALTER TABLE shops
	DROP COLUMN IF EXISTS default_currency,
	DROP COLUMN IF EXISTS booth_number,
	DROP COLUMN IF EXISTS contact;
DELETE FROM schema_migrations WHERE version = 24;
COMMIT;
```

- [ ] **步骤 3b：对齐 `schema.go` 里的 shops DDL（同一个 PR）**

`server/internal/data/schema.go:14-23` 的 `CREATE TABLE IF NOT EXISTS shops (...)` 在 `updated_at` 前加上：
```go
		contact text NOT NULL DEFAULT '' CONSTRAINT shops_contact_check CHECK (char_length(contact) <= 120 AND contact !~ '[[:cntrl:]]'),
		booth_number text NOT NULL DEFAULT '' CONSTRAINT shops_booth_number_check CHECK (char_length(booth_number) <= 40 AND booth_number !~ '[[:cntrl:]]' AND booth_number = btrim(booth_number)),
		default_currency text CONSTRAINT shops_default_currency_check CHECK (default_currency IS NULL OR default_currency IN ('MYR', 'SGD', 'CNY')),
```
约束名必须和 024 一致，这样新库（先跑 `schema`）再跑 024 时 `IF NOT EXISTS` 判断能命中，不会重复建约束。核对：`grep -n "shops_contact_check\|shops_booth_number_check\|shops_default_currency_check" server/internal/data/schema.go server/internal/data/migrations/024_*.sql` 两处都要有。`featured_event_id`（003 加的）是否也该补进 schema.go 的 shops DDL，执行时看现有习惯；本任务只要求补这三列。

- [ ] **步骤 4：运行测试，确认通过**

运行：`cd server && go test ./internal/data -run TestLoadMigrationsIncludes024ShopProfileFields -v`
预期：PASS（修复 plan 任务 3 合入后，连同 `TestLoadMigrationsIncludes014AndIgnoresDownDirectory` 一起跑）

- [ ] **步骤 5：在本机隔离 schema 上验证约束、024 不回填、回滚**

在 `shop_profile_integration_test.go` 加一个用例：建隔离 schema → 只跑到 023（或在修复/vNext 迁移还没合入时跑到 013）→ 插入一家有主推活动（`SGD`）的店和一家没有活动的店 → 再跑 024 → 断言**两家 `default_currency` 都是 `NULL`（024 不回填）**、两家 `contact = ''`、`booth_number = ''`、`updated_at` 没变。另加一个单元测试：`024_*.sql` 文件内容不含 `UPDATE shops`。如果 `Migrate` 不支持“跑到指定版本”，这个用例改成在隔离 schema 里先建表插数据，再用 `pool.Exec` 执行 024 文件内容。约束用例：`booth_number = ' A-27'`（首尾空格）、41 个字符、`contact` 121 个字符、`default_currency = 'USD'` 都应该报 CHECK 违反（`pgconn.PgError.Code == "23514"`）。

```bash
cd /Users/kelden/go/src/Stallink && docker compose up -d postgres   # 只用本机 docker-compose 的 Postgres
export STALLINK_TEST_DATABASE_URL="postgres://postgres:${LOCAL_PG_PASSWORD}@127.0.0.1:5432/stallink?sslmode=disable"
cd server && go test -tags integration ./internal/data -run 'TestShopProfile|TestMigration024' -v
```
预期：PASS。回滚 SQL 在本机隔离 schema 里手动跑一次（`SET search_path` 到该 schema 后执行 down 文件），确认 `\d shops` 里三列、`\d platform_appearance` 里两列、`audit_logs_shop_created_id_idx` 都没了、`schema_migrations` 里没有 24。**只用本机库，不启动 API。**

- [ ] **步骤 5b：存量回填手动脚本（K-Q2 = A、K-Q16 = A）**

文件（都不在 `migrations/` 下，不会被 `//go:embed migrations/*.sql` 加载，API 启动不会执行）：
- 创建：`server/internal/data/manual/024a_backfill_shop_default_currency.sql`
- 创建：`server/internal/data/manual/024a_backfill_shop_default_currency.rollback.sql`
- 创建：`server/internal/data/manual/README.md`（一句话：这里的脚本只能人工执行，云端执行前 ⚠️ 必须 Kel 确认）

回填规则（**Kel 已定 13:55：严格按最近一场活动，去掉“主推活动优先”**）：取这家店最近一场活动（`starts_on DESC, id DESC`）的货币；没有活动保持 `NULL`。

```sql
-- server/internal/data/manual/024a_backfill_shop_default_currency.sql
-- 手动执行，不随 API 启动。开发只在本机 docker-compose 的 Postgres 上执行；云端由 Kel 自己执行。前置：024 已执行。
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
-- 不更新 updated_at：回填不是店主的修改。
UPDATE shops s SET default_currency = t.currency
FROM target t JOIN saved v ON v.shop_id = t.id
WHERE s.id = t.id;
COMMIT;
```

```sql
-- server/internal/data/manual/024a_backfill_shop_default_currency.rollback.sql
-- 只撤销 024a 回填过、且之后店主没改过的值。开发只在本机库执行；云端由 Kel 自己执行。
BEGIN;
UPDATE shops s SET default_currency = b.old_default_currency
FROM manual_024a_shop_currency_backup b
WHERE s.id = b.shop_id AND s.updated_at <= b.backfilled_at;
DROP TABLE IF EXISTS manual_024a_shop_currency_backup;
COMMIT;
```
> 店主在回填后保存过资料（`updated_at` 晚于回填时间）的店不回滚，保留店主的值。备份表在确认回填无误后可以由回滚脚本以外的人工步骤删除；本计划不自动删。

测试（`shop_profile_integration_test.go`，`-tags integration`）：
- `TestManual024aBackfill`：隔离 schema 跑完 024 → 插入三家店：主推一场较早的 `SGD` 活动、但最近一场是 `MYR`（验证**不看主推**）；只有活动（最近一场 `CNY`，更早一场 `MYR`）；没有活动 → `pool.Exec` 执行 024a 文件内容 → 断言 `MYR`、`CNY`、`NULL`，三家 `updated_at` 没变；再执行一次 024a，结果不变（幂等）。
- `TestManual024aRollback`：接上一步执行 rollback 文件 → 三家都回到 `NULL`，备份表不存在；先把其中一家用 `UpdateShopProfile` 改成 `MYR` 再回滚 → 这一家保留 `MYR`。
- 单元测试 `TestLoadMigrationsIgnoresManualDirectory`：`loadMigrations()` 返回的名字里没有 `024a`、没有 `manual`。

本机验证**只用仓库根目录 `docker-compose.yml` 起的 Postgres**（服务 `postgres`，`postgres:16-alpine`，只监听 `127.0.0.1:5432`）；不连云端，不启动 API：
```bash
cd /Users/kelden/go/src/Stallink && docker compose up -d postgres
export STALLINK_TEST_DATABASE_URL="postgres://postgres:${LOCAL_PG_PASSWORD}@127.0.0.1:5432/stallink?sslmode=disable"
cd server && go test -tags integration ./internal/data -run 'TestShopProfile|TestMigration024|TestManual024a' -v
```
（测试都在隔离 schema 里建表、跑 024 / 024a，结束后删掉 schema，不碰本机 `public` 里的开发数据。`LOCAL_PG_PASSWORD` 用本机 compose 里配的值，不写进仓库。手动跑 024a 和它的回滚也只对这个本机库：`psql "$STALLINK_TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f …`。）

- [ ] **步骤 6：云端库——开发不做（K-Q6 = B，Kel 硬性规则）**

开发、测试、子代理**一律不连、不改云端库**（测试服、生产都算）。云端由 Kel 自己按下面顺序执行，记在这里只作交接：
1. ⚠️ Kel 先备份云端库。
2. ⚠️ Kel 手动执行 `024` 升级 SQL，核对 `\d shops`、`\d platform_appearance`、`audit_logs_shop_created_id_idx`。
3. ⚠️ Kel 手动执行 `manual/024a_backfill_shop_default_currency.sql`，核对 `SELECT default_currency, count(*) FROM shops GROUP BY 1`；有问题执行 `manual/024a_…rollback.sql`。
4. Kel 确认没问题后再部署新版 API（API 启动时发现 024 已执行，不会重复跑）。

- [ ] **步骤 7：只暂存，不 commit、不 push**

拟提交：`server/internal/data/migrations/024_shop_profile_and_appearance_meta.sql`、`server/internal/data/migrations/down/024_shop_profile_and_appearance_meta.down.sql`、`server/internal/data/migrations_024_test.go`、`server/internal/data/schema.go`、`server/internal/data/manual/024a_backfill_shop_default_currency.sql`、`server/internal/data/manual/024a_backfill_shop_default_currency.rollback.sql`、`server/internal/data/manual/README.md`、`server/internal/data/shop_profile_integration_test.go`

---

### 任务 S2：新增 `GET /api/admin/shop`；`PATCH /api/admin/shop` 不再校验、不再写入 slug，接收新字段

> **端：** server ｜ **优先级：** P1 ｜ **依赖：** S1；K-Q3 = B、K-Q4 = A、K-Q1 = A（Kel 已定（2026-11-26 13:22））；后端已答复 B-1～B-5（第 9.2 节） ｜ **状态：** 待开始；前端已按“等后端”占位（W14）

**现状核实（只读，2026-11-26 11:51–12:05 UTC+8）：**
- 路由里只有 `PATCH /api/admin/shop`（`server/internal/server/http.go:81`），**没有店主读取店铺资料的接口**。前端先 `GET /api/auth/me` 拿 `shopSlug`，再调公开接口 `GET /api/public/shops/{slug}`（`web/src/app/admin/shop/page.tsx`）。
- 公开接口走 `EnabledShop`（`server/internal/data/shop.go:66-72`，`WHERE slug = $1 AND is_enabled`），**店铺停用后返回 404 `no_booth`，店主打不开自己的店铺资料**；`shopJSON`（`handlers.go:586`）也不返回 `isEnabled`，前端一直按默认 `true` 显示。
- `UpdateShopSettings`（`admin.go:18-38`）要求请求体带合法 slug（不合法回 `400 invalid_shop`），再用 `UpdateShop`（`catalog_store.go:155-170`）写回 `slug`，**店主现在能改 slug**；撞到别家 slug 时回 `ErrConflict`。
- **平台侧 slug 入口：** 只有开店时手填——`POST /api/platform/shops`（`http.go:98`，`OpenShop`，`admin.go:385-410`），平台页 `web/src/app/platform/page.tsx:97` 有“专属链接”输入框（3–40 位小写字母、数字或连字符）。`PATCH /api/platform/shops/{shopID}`（`http.go:100`，`SetPlatformShop`，`admin.go:432-452`）**只接收 `isEnabled`**。**平台改 slug 的入口不存在，也没有自动生成逻辑。**
- 停用行为：`SetShopEnabled`（`catalog_store.go:172` 起）停用时在同一事务里吊销该店所有会话。
- `service.API.DB` 是具体类型 `*data.ShopStore`（`handlers.go:20-21`），新方法直接加在 `ShopStore` 上。

**权限：** 仅 `shop_admin`（沿用 `a.shopAdmin`）；`shopID` 一律取自会话里的 `account.ShopID`，路径和请求体里都不接收店铺 ID 或 slug，所以只能读写本店。`sales`、`platform_admin`、未登录分别按 `shopAdmin` 现有行为返回 403 / 401。

**`GET /api/admin/shop` 响应字段：**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 店铺 ID（字符串化 bigint，和现有接口一致） |
| `name` | string | 店名，1–80 字 |
| `notice` | string | 公告，≤2000 字 |
| `slug` | string | 平台分配的店铺地址，只读 |
| `isEnabled` | boolean | 是否公开展示 |
| `contact` | string | 客服联系，≤120 字，未填为 `""` |
| `boothNumber` | string | 店铺级摊位号，≤40 字，未填为 `""` |
| `defaultCurrency` | `"MYR" \| "SGD" \| "CNY" \| null` | 默认货币，未设置为 `null` |
| `featuredEvent` | object \| null | 预览用当前活动：`{ id, name, boothNumber, currency, status }`；取主推活动，没有就取第一个 `live`，再没有取第一个 `scheduled`；都没有返回 `null` |
| `onSaleCount` | number | 预览用在售数（后端 B-4 已答复）：当前活动里同时满足 `is_listed`、审核已通过、共享池可售量（`total_stock` − Σ`reserved_qty`）> 0 的 listing 数；不算套餐；没有当前活动时为 0 |
| `updatedAt` | string | `shops.updated_at`，RFC3339 带时区 |

```json
{
  "id": "12", "name": "晚风工作室", "notice": "手工丝网印刷……", "slug": "wanfeng", "isEnabled": false,
  "contact": "IG @wanfeng.studio", "boothNumber": "A-27", "defaultCurrency": "MYR",
  "featuredEvent": { "id": "34", "name": "吉隆坡同人展", "boothNumber": "A-27", "currency": "MYR", "status": "live" },
  "onSaleCount": 18, "updatedAt": "2026-11-26T09:40:00+08:00"
}
```

**停用店铺时的行为（后端 B-5 已答复）：** PATCH 把 `isEnabled` 改成 `false` 时，改旗标和删会话必须在**同一个 data 方法、同一个事务**里完成（`UpdateShopProfile`），不能分两次调用。停用后：这次 PATCH 仍返回 200；旧 cookie 立即失效；停用店铺的账号 **Login 仍会发 cookie**（`handlers.go:154` 只检查 `users.is_enabled`），但会话查询要求 `s.is_enabled`（`auth_store.go:64-65`），所以 `GET /api/admin/shop` 等工作台接口返回 401。前端文案写“**登录后进不了工作台**”，不要写“登录被拒绝”。恢复只能由平台管理员在店铺管理里重新启用。

**`PATCH /api/admin/shop` 请求：**
```json
{ "name": "晚风工作室", "notice": "", "isEnabled": true, "contact": "IG @wanfeng.studio", "boothNumber": "A-27", "defaultCurrency": "SGD" }
```
- 校验：`name` 去首尾空格后 1–80 字；`notice` ≤2000 字；`contact` 去首尾空格后 ≤120 字；`boothNumber` ≤40 字、不含控制字符、**首尾不能有空格（不替用户裁剪，直接 400）**（后端 B-3 已答复）；都按 Unicode 字符计，不能含控制字符；`defaultCurrency` 为 `null` 或三种货币之一。不合法返回 `400 invalid_shop`（沿用现有错误码，后端 B-3 确认不拆细）。`updatedAt` 直接复用 `shops.updated_at`，RFC3339 带时区；订单币种照旧跟活动走，`defaultCurrency` 不影响下单（后端 B-1）。
- **不再校验、不再写入 `slug`。** 请求里带了 `slug`：**K-Q3 = B（Kel 已定（2026-11-26 13:22））**——没带或和当前值相同，忽略并照常处理；和当前值不同，返回 `422 slug_readonly`，**整条请求都不写**（后端 B-2 已答复）。客服联系 **K-Q4 = A**：自由文本，最多 120 字（按字符数，和 024 的 CHECK 一致），不校验格式。平台侧也**不做**修改 slug 的功能（**K-Q9 = A**，Kel 13:22），`SetPlatformShop` 保持只改 `isEnabled`。
- 响应：`200`，返回和 GET 相同的对象，前端保存后直接用它刷新预览。

**前端对接点：** `web/src/app/admin/shop/page.tsx`
- 读取：把 `browserJSON("/api/auth/me")` + `browserJSON("/api/public/shops/...")` 两步换成 `browserJSON("/api/admin/shop")`，客服联系、摊位号、默认货币去掉 `disabled` 和“等后端”提示（现在在第 156–177 行附近）。
- 保存：第 119–120 行的 `browserJSON("/api/admin/shop", "PATCH", { name, notice, slug, isEnabled })` 改成不带 `slug`、加 `contact`、`boothNumber`、`defaultCurrency`，并删掉那段“等后端改为忽略后再拿掉”的注释。
- 预览区“最近更新等后端提供时间字段后再显示”（第 198 行附近）改为显示 `updatedAt`。
- 错误码：`422 slug_readonly` 显示“店铺地址由平台分配，不能修改”；`400 invalid_shop` 沿用 `describeRequest`；文案加到 `web/src/lib/errors.ts`。

**文件：**
- 创建：`server/internal/data/shop_profile.go`
- 创建：`server/internal/data/shop_profile_integration_test.go`
- 修改：`server/internal/service/admin.go:18-38`（替换 `UpdateShopSettings`，新增 `AdminShop`）
- 修改：`server/internal/server/http.go:81`（新增 GET 路由）
- 创建：`server/internal/service/shop_profile_integration_test.go`

- [ ] **步骤 1：编写失败的 data 层测试**

```go
// server/internal/data/shop_profile_integration_test.go
//go:build integration

package data

import (
	"errors"
	"testing"
)

func TestShopProfileReadWriteAndSlugUntouched(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "shop_profile")
	store := NewShopStore(pool)
	var rawID string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name, slug, is_enabled) VALUES ('晚风', 'wanfeng', false) RETURNING id::text`).Scan(&rawID); err != nil {
		t.Fatal(err)
	}
	shopID := integrationID(t, rawID)

	got, err := store.ShopProfile(ctx, shopID)
	if err != nil {
		t.Fatalf("read disabled shop profile: %v", err)
	}
	if got.Slug != "wanfeng" || got.Enabled || got.Contact != "" || got.BoothNumber != "" || got.DefaultCurrency != nil || got.FeaturedEvent != nil {
		t.Fatalf("unexpected initial profile: %+v", got)
	}

	sgd := "SGD"
	if err := store.UpdateShopProfile(ctx, shopID, ShopProfileInput{
		Name: "晚风工作室", Notice: "现场自取", Enabled: true, Contact: "IG @wanfeng.studio", BoothNumber: "A-27", DefaultCurrency: &sgd,
	}); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	got, err = store.ShopProfile(ctx, shopID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "晚风工作室" || !got.Enabled || got.Contact != "IG @wanfeng.studio" || got.BoothNumber != "A-27" || got.DefaultCurrency == nil || *got.DefaultCurrency != "SGD" {
		t.Fatalf("unexpected updated profile: %+v", got)
	}
	if got.Slug != "wanfeng" {
		t.Fatalf("slug must stay platform-assigned, got %q", got.Slug)
	}
	missing := integrationID(t, "999999999")
	if _, err := store.ShopProfile(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing shop: want ErrNotFound, got %v", err)
	}
}
```

- [ ] **步骤 2：编写失败的 HTTP 测试**

```go
// server/internal/service/shop_profile_integration_test.go
//go:build integration

package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
)

func TestAdminShopProfileReadUpdateAndSlugReadonly(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := isolatedSchemaPool(t, ctx, dsn)
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := data.NewShopStore(pool)
	handler := server.NewHTTP(":0", "shop-profile-test", &service.API{DB: store})

	prefix := "p" + strconv.FormatInt(time.Now().UnixNano(), 36)
	password := "test-password-123"
	platformLogin := prefix + "-platform"
	if err := data.EnsurePlatformAdmin(ctx, store, platformLogin, password); err != nil {
		t.Fatal(err)
	}
	platformCookie := login(t, handler, platformLogin, password)
	shopSlug := prefix + "-shop"
	ownerLogin := prefix + "-owner"
	if res := requestJSON(t, handler, http.MethodPost, "/api/platform/shops", map[string]string{
		"name": "晚风", "slug": shopSlug, "notice": "", "adminLogin": ownerLogin, "adminPassword": password,
	}, platformCookie); res.Code != http.StatusCreated {
		t.Fatalf("open shop: %d %s", res.Code, res.Body.String())
	}
	owner := login(t, handler, ownerLogin, password)

	type profile struct {
		ID              string  `json:"id"`
		Slug            string  `json:"slug"`
		IsEnabled       bool    `json:"isEnabled"`
		Contact         string  `json:"contact"`
		BoothNumber     string  `json:"boothNumber"`
		DefaultCurrency *string `json:"defaultCurrency"`
		UpdatedAt       string  `json:"updatedAt"`
	}
	read := func() profile {
		t.Helper()
		res := requestJSON(t, handler, http.MethodGet, "/api/admin/shop", nil, owner)
		if res.Code != http.StatusOK {
			t.Fatalf("GET /api/admin/shop: %d %s", res.Code, res.Body.String())
		}
		var p profile
		if err := json.Unmarshal(res.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	if p := read(); p.Slug != shopSlug || !p.IsEnabled || p.Contact != "" || p.DefaultCurrency != nil {
		t.Fatalf("initial profile: %+v", p)
	}

	shopID := read().ID

	// 1) 不带 slug 也能保存（仍是启用状态）
	if res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风工作室", "notice": "", "isEnabled": true, "contact": "  IG @wanfeng.studio  ", "boothNumber": "A-27", "defaultCurrency": "SGD",
	}, owner); res.Code != http.StatusOK {
		t.Fatalf("patch without slug: %d %s", res.Code, res.Body.String())
	}
	p := read()
	if !p.IsEnabled || p.Contact != "IG @wanfeng.studio" || p.BoothNumber != "A-27" || p.DefaultCurrency == nil || *p.DefaultCurrency != "SGD" {
		t.Fatalf("after patch: %+v", p)
	}
	if _, err := time.Parse(time.RFC3339, p.UpdatedAt); err != nil {
		t.Fatalf("updatedAt must be RFC3339 with offset: %q", p.UpdatedAt)
	}

	// 1b) 店主自己停用（B-5）：改旗标和删会话在同一个事务里；旧 cookie 立即失效。
	//     停用店铺的账号 Login 仍会发 cookie（handlers.go:154 只看 users.is_enabled），
	//     但会话查询要求 s.is_enabled（auth_store.go:64-65），所以“登录后进不了工作台”。
	if res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风工作室", "notice": "", "isEnabled": false, "contact": "IG @wanfeng.studio", "boothNumber": "A-27", "defaultCurrency": "SGD",
	}, owner); res.Code != http.StatusOK {
		t.Fatalf("disable shop: %d %s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, handler, http.MethodGet, "/api/admin/shop", nil, owner); res.Code != http.StatusUnauthorized {
		t.Fatalf("old cookie after disable: want 401, got %d", res.Code)
	}
	again := login(t, handler, ownerLogin, password) // 仍会发 cookie
	if res := requestJSON(t, handler, http.MethodGet, "/api/admin/shop", nil, again); res.Code != http.StatusUnauthorized {
		t.Fatalf("re-login on disabled shop must not reach the workspace: got %d", res.Code)
	}
	// 平台管理员重新启用后，店主重新登录继续后面的用例
	if res := requestJSON(t, handler, http.MethodPatch, "/api/platform/shops/"+shopID, map[string]any{"isEnabled": true}, platformCookie); res.Code != http.StatusOK {
		t.Fatalf("platform re-enable: %d %s", res.Code, res.Body.String())
	}
	owner = login(t, handler, ownerLogin, password)

	// 2) 带原 slug 兼容通过
	if res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风工作室", "notice": "", "isEnabled": true, "slug": shopSlug, "contact": "", "defaultCurrency": nil,
	}, owner); res.Code != http.StatusOK {
		t.Fatalf("patch with same slug: %d %s", res.Code, res.Body.String())
	}

	// 3) 改 slug 被拒（K-Q3 = B，Kel 已定 13:22），slug 不变
	res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风工作室", "notice": "", "isEnabled": true, "slug": shopSlug + "-x",
	}, owner)
	if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "slug_readonly") {
		t.Fatalf("change slug: want 422 slug_readonly, got %d %s", res.Code, res.Body.String())
	}
	if p := read(); p.Slug != shopSlug {
		t.Fatalf("slug changed to %q", p.Slug)
	}

	// 4) 校验
	for name, body := range map[string]map[string]any{
		"contact too long": {"name": "晚风", "isEnabled": true, "contact": strings.Repeat("客", 121)},
		"bad currency":     {"name": "晚风", "isEnabled": true, "defaultCurrency": "USD"},
		"booth too long":   {"name": "晚风", "isEnabled": true, "boothNumber": strings.Repeat("A", 41)},
		"booth padded":     {"name": "晚风", "isEnabled": true, "boothNumber": " A-27 "},
		"booth control":    {"name": "晚风", "isEnabled": true, "boothNumber": "A\t27"},
		"blank name":       {"name": "   ", "isEnabled": true},
	} {
		if res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", body, owner); res.Code != http.StatusBadRequest {
			t.Fatalf("%s: want 400, got %d %s", name, res.Code, res.Body.String())
		}
	}
	if res := requestJSON(t, handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风", "isEnabled": true, "contact": strings.Repeat("客", 120),
	}, owner); res.Code != http.StatusOK {
		t.Fatalf("contact exactly 120: %d %s", res.Code, res.Body.String())
	}

	// 5) 非店主不能读
	if res := requestJSON(t, handler, http.MethodGet, "/api/admin/shop", nil, platformCookie); res.Code != http.StatusForbidden && res.Code != http.StatusUnauthorized {
		t.Fatalf("platform admin GET /api/admin/shop: want 401/403, got %d", res.Code)
	}
}
```
> 第 1b 段按后端 B-5 答复写：停用店铺的账号 Login 仍会发 cookie，但会话查询要求店铺启用，所以工作台接口返回 401。平台重新启用接口 `PATCH /api/platform/shops/{shopID}`（`server/internal/server/http.go:100`）返回码以现有实现为准（如果是 204，断言跟着改）。

- [ ] **步骤 3：运行测试，确认失败**

运行：`cd server && go test -tags integration ./internal/data ./internal/service -run 'TestShopProfile|TestAdminShopProfile' -v`
预期：FAIL：编译错误 `store.ShopProfile undefined`；补上 data 层后，HTTP 用例报 `GET /api/admin/shop: 404`

- [ ] **步骤 4：实现 data 层**

```go
// server/internal/data/shop_profile.go
package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ShopProfileEvent struct {
	ID          ID
	Name        string
	BoothNumber string
	Currency    string
	Status      string
}

type ShopProfile struct {
	ID              ID
	Name            string
	Notice          string
	Slug            string
	Enabled         bool
	Contact         string
	BoothNumber     string
	DefaultCurrency *string
	FeaturedEvent   *ShopProfileEvent
	OnSaleCount     int
	UpdatedAt       time.Time
}

type ShopProfileInput struct {
	Name            string
	Notice          string
	Enabled         bool
	Contact         string
	BoothNumber     string
	DefaultCurrency *string
}

// ShopProfile 按 shopID 读取店主可见的店铺资料。本身不过滤 is_enabled；停用店铺的会话在 auth 层就失效了（B-5）。
func (s *ShopStore) ShopProfile(ctx context.Context, shopID ID) (ShopProfile, error) {
	var p ShopProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, notice, slug, is_enabled, contact, booth_number, default_currency, updated_at
		FROM shops WHERE id = $1`, shopID).
		Scan(&p.ID, &p.Name, &p.Notice, &p.Slug, &p.Enabled, &p.Contact, &p.BoothNumber, &p.DefaultCurrency, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShopProfile{}, ErrNotFound
	}
	if err != nil {
		return ShopProfile{}, fmt.Errorf("query shop profile: %w", err)
	}
	var ev ShopProfileEvent
	err = s.pool.QueryRow(ctx, `
		SELECT e.id::text, e.name, e.booth_number, e.currency, e.status
		FROM events e JOIN shops s ON s.id = e.shop_id
		WHERE e.shop_id = $1 AND e.status <> 'ended'
		ORDER BY (e.id = s.featured_event_id) DESC NULLS LAST, (e.status = 'live') DESC, e.starts_on, e.id
		LIMIT 1`, shopID).Scan(&ev.ID, &ev.Name, &ev.BoothNumber, &ev.Currency, &ev.Status)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ShopProfile{}, fmt.Errorf("query shop profile event: %w", err)
	}
	if err == nil {
		p.FeaturedEvent = &ev
		// 口径（后端 B-4 已答复）：当前活动里同时满足 is_listed、审核已通过、
		// 共享池可售量（total_stock − Σreserved_qty）> 0 的 listing 数；不算套餐；没有当前活动时为 0。
		if err := s.pool.QueryRow(ctx, `
			SELECT count(*) FROM event_listings el JOIN goods g ON g.id = el.good_id
			WHERE el.event_id = $1 AND el.shop_id = $2 AND el.is_listed
			  AND g.review_status = 'approved'
			  AND g.total_stock - COALESCE((SELECT sum(x.reserved_qty) FROM event_listings x WHERE x.good_id = g.id), 0) > 0`,
			ev.ID, shopID).Scan(&p.OnSaleCount); err != nil {
			return ShopProfile{}, fmt.Errorf("count on-sale listings: %w", err)
		}
	}
	return p, nil
}

// UpdateShopProfile 不写 slug：slug 由平台分配（Kel 2026-11-26，K-04）。
// isEnabled 改成 false 时，改旗标和删会话在同一个方法、同一个事务里完成（后端 B-5）。
func (s *ShopStore) UpdateShopProfile(ctx context.Context, shopID ID, in ShopProfileInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update shop profile: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE shops SET name = $2, notice = $3, is_enabled = $4, contact = $5, booth_number = $6, default_currency = $7, updated_at = now()
		WHERE id = $1`, shopID, in.Name, in.Notice, in.Enabled, in.Contact, in.BoothNumber, in.DefaultCurrency)
	if err != nil {
		return fmt.Errorf("update shop profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !in.Enabled {
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE shop_id = $1)`, shopID); err != nil {
			return fmt.Errorf("revoke shop sessions: %w", err)
		}
	}
	return tx.Commit(ctx)
}
```
> `goods.total_stock`、`goods.review_status` 已在 `schema.go:66-67` 核实；`event_listings.is_listed`、`reserved_qty`、`shop_id` 见 `001_catalog_orders.sql:68-79`。删会话的 SQL 和 `SetShopEnabled`（`catalog_store.go:174-193`）一致；service 层**不再另外调用** `SetShopEnabled`（后端 B-5：不能分两次调用）。

- [ ] **步骤 5：实现 service 层和路由**

```go
// server/internal/service/admin.go：替换 UpdateShopSettings（原 18-38 行），新增 AdminShop
// import 需包含 "strings"、"time"、"unicode"、"unicode/utf8"
var shopCurrencies = map[string]bool{"MYR": true, "SGD": true, "CNY": true}

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

func shopProfileJSON(p data.ShopProfile) map[string]any {
	out := map[string]any{
		"id": p.ID, "name": p.Name, "notice": p.Notice, "slug": p.Slug, "isEnabled": p.Enabled,
		"contact": p.Contact, "boothNumber": p.BoothNumber, "defaultCurrency": p.DefaultCurrency, "featuredEvent": nil,
		"onSaleCount": p.OnSaleCount, "updatedAt": p.UpdatedAt.Format(time.RFC3339),
	}
	if p.FeaturedEvent != nil {
		out["featuredEvent"] = map[string]any{
			"id": p.FeaturedEvent.ID, "name": p.FeaturedEvent.Name, "boothNumber": p.FeaturedEvent.BoothNumber,
			"currency": p.FeaturedEvent.Currency, "status": p.FeaturedEvent.Status,
		}
	}
	return out
}

func (a *API) AdminShop(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	profile, err := a.DB.ShopProfile(r.Context(), *account.ShopID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, shopProfileJSON(profile))
}

func (a *API) UpdateShopSettings(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name            string  `json:"name"`
		Notice          string  `json:"notice"`
		Slug            *string `json:"slug"`
		IsEnabled       bool    `json:"isEnabled"`
		Contact         string  `json:"contact"`
		BoothNumber     string  `json:"boothNumber"`
		DefaultCurrency *string `json:"defaultCurrency"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_shop")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Contact = strings.TrimSpace(body.Contact)
	if n := utf8.RuneCountInString(body.Name); n < 1 || n > 80 ||
		utf8.RuneCountInString(body.Notice) > 2000 || utf8.RuneCountInString(body.Contact) > 120 || utf8.RuneCountInString(body.BoothNumber) > 40 ||
		hasControl(body.Name+body.Contact+body.BoothNumber) ||
		body.BoothNumber != strings.TrimSpace(body.BoothNumber) || // B-3：首尾有空格直接 400，不替用户裁剪
		(body.DefaultCurrency != nil && !shopCurrencies[*body.DefaultCurrency]) {
		writeErr(w, http.StatusBadRequest, "invalid_shop")
		return
	}
	current, err := a.DB.ShopProfile(r.Context(), *account.ShopID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	if body.Slug != nil && *body.Slug != current.Slug {
		// 后端 B-2 已答复：整条请求都不写（K-Q3 = B，Kel 已定 13:22）
		writeErr(w, http.StatusUnprocessableEntity, "slug_readonly")
		return
	}
	if err := a.DB.UpdateShopProfile(r.Context(), *account.ShopID, data.ShopProfileInput{
		Name: body.Name, Notice: body.Notice, Enabled: body.IsEnabled, Contact: body.Contact, BoothNumber: body.BoothNumber, DefaultCurrency: body.DefaultCurrency,
	}); err != nil {
		writeDataErr(w, r, err)
		return
	}
	updated, err := a.DB.ShopProfile(r.Context(), *account.ShopID)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, shopProfileJSON(updated))
}
```
```go
// server/internal/server/http.go：加在第 81 行 PATCH 之前
route(r, http.MethodGet, "/api/admin/shop", api.AdminShop)
```
> 旧的 `ShopStore.UpdateShop` 改完后是否还有引用，用 `grep -rn "UpdateShop(" server/` 复核。没有引用也先保留；删除文件或函数前按规则问 Kel。

- [ ] **步骤 6：运行测试，确认通过**

运行：`cd server && go test ./... && go test -tags integration ./internal/data ./internal/service -run 'TestShopProfile|TestAdminShopProfile|TestAuthenticationAndRoleLifecycle' -v`
预期：全部 PASS；`TestAuthenticationAndRoleLifecycle` 里“停用店铺吊销会话”的断言仍然通过；第 1b 段证明停用和删会话在同一事务（旧 cookie 立即 401）。

- [ ] **步骤 7：只暂存，不 commit、不 push**

拟提交：`server/internal/data/shop_profile.go`、`server/internal/data/shop_profile_integration_test.go`、`server/internal/service/admin.go`、`server/internal/server/http.go`、`server/internal/service/shop_profile_integration_test.go`

---

### 任务 S3：操作记录返回对象展示名 `subjectLabel`

> **端：** server ｜ **优先级：** P1 ｜ **依赖：** 阶段 1 依赖 S1 的 `audit_logs_shop_created_id_idx`（后端 B-6）；阶段 2 依赖修复 plan 任务 3（迁移 014 的 `audit_logs.detail`）和任务 7（写审计时带 `detail`）；**K-Q5 = C（Kel 已定 2026-11-26 13:55）**：阶段 1 先用实时名称，阶段 2 等修复 plan 任务 7 完成后补快照，读取时快照优先 ｜ **状态：** 已批准开工；阶段 1 现在做，阶段 2 等修复 plan 任务 7

**现状核实（只读）：**
- `audit_logs`（`schema.go:132-140`）只有 `id, shop_id, actor_id, action, subject_type, subject_id, created_at`，**没有对象名称或快照**。
- 写入只有一个入口 `insertAudit`（`catalog_store.go:1308-1316`），全仓 21 处调用。实际出现的 `subject_type` 只有 4 种：`order`（`order_store.go`、`reservation_store.go`）、`good`、`listing`（`catalog_store.go`）、`bundle`（`bundle_store.go`）。店铺资料、收款码、销售账号、活动的修改**目前都不写审计**。
- 读取 `ListAuditPage`（`paged_lists.go:51-80`）和 `AdminAudit`（`admin.go:258-282`）只返回 `action, actorLogin, subjectType, subjectId, createdAt`；`createdAt` 是 `to_char(a.created_at,'YYYY-MM-DD HH24:MI')`，**不带时区**。
- 能关联出展示名的现有列：`orders.number`（6 位订单号，`schema.go:106`）、`goods.name`、`bundles.name`（`001_catalog_orders.sql:32`）、`event_listings` 关联 `goods.name` 和 `events.name`。

**三种方案比较：**

| 方案 | 做法 | 优点 | 缺点 | 和 014 / S16 的关系 |
|---|---|---|---|---|
| A. 查询时 join 实时名称 | `ListAuditPage` 按 `subject_type` 关联 `orders.number`、`goods.name`、`bundles.name`、`event_listings→goods/events` | 不改表、不改写入路径，马上能做；旧记录也有名称 | 显示的是**当前**名称；每页多 4 个子查询；对象如果被物理删除就查不到 | 无冲突 |
| B. 写入时快照进 `audit_logs.detail` | `insertAudit` 写 `detail.subject = {"label": "..."}`，读取 `detail->'subject'->>'label'` | 显示操作当时的名称；读取快 | 依赖修复 plan 迁移 014 和任务 7 改 `insertAudit`；014 之前的旧记录没有快照 | 复用 014 的 `detail`，不新增列；S16 的 `before_json/after_json` 是整行快照，`detail.subject.label` 只是展示名，不重复 |
| C. 新增一列 `subject_label` | 迁移里给 `audit_logs` 加 `subject_label text`，写入时填 | 查询和索引最直接 | 又多一列，和 014 的 `detail`、S16 的 `after_json` 三处都能放名称，重复；要再出一份迁移（或塞进 024） | 和 014 / S16 重复，不推荐 |

**推荐：A 先上，B 随修复 plan 任务 7 一起补，读取用 `COALESCE(detail->'subject'->>'label', join 结果, '')`（即 K-Q5 方案 C）。不选表里的 C 列（新增 `subject_label` 列）。** **Kel 已定 13:55：K-Q5 = C**，和后端 B-6、B-7 一致。

**对象被删除或改名后显示什么：**
- 核实：`server/internal/data` 里没有对 `goods`、`bundles`、`orders`、`events`、`event_listings` 的 `DELETE`（只有 `reset.go` 的本地重置会清表），商品和套餐下架是改状态，不删行。所以正常使用下“对象被删”不会发生，join 总能查到。
- 改名：阶段 1 显示**当前**名称（例如商品改名后，旧的“调整商品库存”记录也显示新名）；阶段 2 之后，有快照的记录显示**操作当时**的名称，014 之前的旧记录仍显示当前名称。
- 查不到（本地重置后的残留、未知 `subject_type`）：`subjectLabel` 返回 `""`，前端显示“类型 #内部编号”，例如“订单 #1045”；类型和编号都没有时显示“—”。
- 订单号不会变（`orders.number` 生成后不改），所以订单类记录 A、B 两种方案显示一样。

**接口约定：** `GET /api/admin/audit` 每条记录新增 `subjectLabel`
```json
{ "action": "order.cancel", "actorLogin": "kai", "subjectType": "order", "subjectId": "1045",
  "subjectLabel": "A7K2QM", "createdAt": "2026-11-26T13:21:00+08:00" }
```
- `subjectLabel`：order 为订单号；good 为商品名；bundle 为套餐名；listing 为“商品名 · 活动名”；查不到（对象已删、未知类型）返回 `""`，前端此时退回显示“类型 + 内部编号”。
- `createdAt` 的格式变化单列在任务 S4，两者可以一起上线。
- 搜索：`search` 关键字也匹配 `subjectLabel`（订单号、商品名）。前端搜索框占位文字从“搜索操作、账号或对象编号”改成“搜索操作、账号或对象”。

**文件：**
- 修改：`server/internal/data/order_store.go:64-70`（`AuditEntry` 加 `SubjectLabel string`）
- 修改：`server/internal/data/paged_lists.go:51-80`
- 修改：`server/internal/data/paged_lists_integration_test.go`（`TestPagedListsSearchAndShopIsolation`，第 106 行附近）
- 修改：`server/internal/service/admin.go:274-280`

- [ ] **步骤 1：编写失败的测试**

追加在 `TestPagedListsSearchAndShopIsolation` 现有 `ListAuditPage` 断言之后，沿用该函数已有的 `shopID`、`otherID`、`store`、`pool`、`ctx` 夹具（订单夹具已经写入了 `order.create` 审计）：

```go
	// 对象展示名：订单审计要带订单号，按订单号能搜到
	var orderNumber string
	if err := pool.QueryRow(ctx, `SELECT number FROM orders WHERE shop_id = $1 ORDER BY id LIMIT 1`, shopID).Scan(&orderNumber); err != nil {
		t.Fatal(err)
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, orderNumber, 20, 0)
	if err != nil || len(logs) == 0 {
		t.Fatalf("search by order number: %d %v", len(logs), err)
	}
	for _, l := range logs {
		if l.SubjectType == "order" && l.SubjectLabel != orderNumber {
			t.Fatalf("order audit label: want %q, got %q", orderNumber, l.SubjectLabel)
		}
	}
	// 关联不到对象时返回空串，不报错
	if _, err := pool.Exec(ctx, `INSERT INTO audit_logs (shop_id, actor_id, action, subject_type, subject_id)
		SELECT $1, actor_id, 'order.expire', 'order', 999999999 FROM audit_logs WHERE shop_id = $1 LIMIT 1`, shopID); err != nil {
		t.Fatal(err)
	}
	logs, _, err = store.ListAuditPage(ctx, shopID, "999999999", 20, 0)
	if err != nil || len(logs) != 1 || logs[0].SubjectLabel != "" {
		t.Fatalf("orphan subject: %+v %v", logs, err)
	}
	// 跨店隔离：按本店订单号在别家店搜不到
	logs, _, err = store.ListAuditPage(ctx, otherID, orderNumber, 20, 0)
	if err != nil || len(logs) != 0 {
		t.Fatalf("label search tenant isolation: %d %v", len(logs), err)
	}
```

- [ ] **步骤 2：运行测试，确认失败**

运行：`cd server && go test -tags integration ./internal/data -run TestPagedListsSearchAndShopIsolation -v`
预期：FAIL，编译错误 `l.SubjectLabel undefined`

- [ ] **步骤 3：实现**

```go
// server/internal/data/order_store.go:64
type AuditEntry struct {
	Action       string
	ActorLogin   string
	SubjectType  string
	SubjectID    ID
	SubjectLabel string
	CreatedAt    string
}
```

`paged_lists.go` 里 `ListAuditPage` 的 SQL 改为：先在 CTE 里算出 `label`，再在外层过滤。原来的中文动作名 CASE 原样搬进来：

```go
	rows, err := s.pool.Query(ctx, `
WITH labelled AS (
  SELECT a.id, a.action, u.login_name, a.subject_type, a.subject_id, a.created_at,
    COALESCE(CASE a.subject_type
      WHEN 'order'   THEN (SELECT o.number FROM orders o WHERE o.id = a.subject_id AND o.shop_id = a.shop_id)
      WHEN 'good'    THEN (SELECT g.name FROM goods g WHERE g.id = a.subject_id AND g.shop_id = a.shop_id)
      WHEN 'bundle'  THEN (SELECT b.name FROM bundles b WHERE b.id = a.subject_id AND b.shop_id = a.shop_id)
      WHEN 'listing' THEN (SELECT g.name || ' · ' || e.name FROM event_listings el
                             JOIN goods g ON g.id = el.good_id JOIN events e ON e.id = el.event_id
                            WHERE el.id = a.subject_id AND el.shop_id = a.shop_id AND e.shop_id = a.shop_id)
    END, '') AS label
  FROM audit_logs a JOIN users u ON u.id = a.actor_id
  WHERE a.shop_id = $1
)
SELECT action, login_name, subject_type, subject_id::text, label, to_char(created_at, 'YYYY-MM-DD HH24:MI')
FROM labelled
WHERE ($2 = '' OR strpos(lower(action), lower($2)) > 0 OR strpos(lower(login_name), lower($2)) > 0
   OR strpos(lower(subject_type), lower($2)) > 0 OR strpos(subject_id::text, lower($2)) > 0
   OR strpos(lower(label), lower($2)) > 0
   OR strpos(CASE action
        WHEN 'bundle.approve' THEN '审核通过套餐' WHEN 'bundle.create' THEN '创建套餐' WHEN 'bundle.update' THEN '编辑套餐'
        WHEN 'bundle.reject' THEN '拒绝套餐' WHEN 'good.approve' THEN '审核通过商品' WHEN 'good.reject' THEN '拒绝商品'
        WHEN 'good.image.add' THEN '添加商品图片' WHEN 'good.image.delete' THEN '删除商品图片'
        WHEN 'good.image.reorder' THEN '调整商品图片顺序' WHEN 'listing.create' THEN '上架商品'
        WHEN 'listing.update' THEN '修改商品上架信息' WHEN 'stock.update' THEN '调整商品库存'
        WHEN 'order.create' THEN '创建订单' WHEN 'order.pay' THEN '确认收款' WHEN 'order.cancel' THEN '取消订单'
        WHEN 'order.refund_required' THEN '收款后取消待退款' WHEN 'order.refund_confirm' THEN '确认退款'
        WHEN 'order.fulfill' THEN '完成交付' WHEN 'order.expire' THEN '订单预留过期' ELSE '' END, $2) > 0)
ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, shopID, search, limit+1, offset)
```
> **后端已答复（B-6、B-7）：** listing 子查询加 `el.shop_id = a.shop_id`（已写进上面的 SQL）；分页沿用 `pageParams`（`service/pagination.go:10-21`，默认 20 条，最多 50 条）；排序 `created_at DESC, id DESC` 由 S1 迁移 024 新增的 `audit_logs_shop_created_id_idx (shop_id, created_at DESC, id DESC)` 支撑（`audit_logs` 原来**没有**这类索引）。读取用 `COALESCE(detail->'subject'->>'label', 关联名称, '')`，**不新增列**；快照写入放在修复 plan 任务 7，**本计划不改 `insertAudit`**。
>
> 上面的中文动作名 CASE 抄自 2026-11-26 的 `paged_lists.go:54`。执行前和当前代码比对一次：修复 plan 任务 7 会新增 `order.cancel_refund`，那时一并加进来。Scan 增加 `&e.SubjectLabel`（放在 `SubjectID` 之后）。
>
> 阶段 2（修复 plan 迁移 014 合入后）：把 `COALESCE(CASE ...)` 改为 `COALESCE(a.detail->'subject'->>'label', CASE ... END, '')`，`subject.label` 由修复 plan 任务 7 改造 `insertAudit` 时随 `detail` 一起写入。以修复 plan 里 `insertAudit` 的新签名为准，本计划不预先改签名。

```go
// server/internal/service/admin.go:276 起
		out = append(out, map[string]string{
			"action": entry.Action, "actorLogin": entry.ActorLogin,
			"subjectType": entry.SubjectType, "subjectId": entry.SubjectID.String(),
			"subjectLabel": entry.SubjectLabel, "createdAt": entry.CreatedAt,
		})
```

- [ ] **步骤 4：运行测试，确认通过**

运行：`cd server && go test ./... && go test -tags integration ./internal/data -run TestPagedListsSearchAndShopIsolation -v`
预期：PASS。另外在本机库（已跑过 024）造 1 万条审计，用 `EXPLAIN ANALYZE` 跑一次（只读、只在本机），确认外层排序走 `audit_logs_shop_created_id_idx`，耗时写进交付报告。

- [ ] **步骤 5：只暂存，不 commit、不 push**

拟提交：`server/internal/data/order_store.go`、`server/internal/data/paged_lists.go`、`server/internal/data/paged_lists_integration_test.go`、`server/internal/service/admin.go`

---

### 任务 S4：操作记录 `createdAt` 带时区（P2）

> **端：** server ｜ **优先级：** P2 ｜ **依赖：** 无（建议和 S3 同批） ｜ **后端已答复（B-8）：** 改成 RFC3339，`order_store.go` 里审计查询的 `to_char`（第 864 行）一起改 ｜ **状态：** 待开始

**问题：** `ListAuditPage` 用 `to_char(created_at,'YYYY-MM-DD HH24:MI')` 输出，按数据库会话时区；前端 `audit/page.tsx` 再用 `new Date(entry.createdAt)` 按浏览器本地时区解析。服务器和摊主不在同一时区时（比如服务器 UTC、摊主在吉隆坡），显示会差 8 小时。

**文件：** `server/internal/data/paged_lists.go`、`server/internal/data/order_store.go:64-70`、`server/internal/service/admin.go:276-279`

- [ ] **步骤 1：编写失败的测试**（追加到 `TestPagedListsSearchAndShopIsolation`，需要 `import "time"`）

```go
	logs, _, err = store.ListAuditPage(ctx, shopID, "", 1, 0)
	if err != nil || len(logs) != 1 {
		t.Fatal(err)
	}
	if logs[0].CreatedAt.IsZero() {
		t.Fatal("audit createdAt must be a real timestamp")
	}
	if got := logs[0].CreatedAt.Format(time.RFC3339); !strings.ContainsAny(got[len(got)-6:], "+-Z") {
		t.Fatalf("audit createdAt must carry an offset, got %q", got)
	}
```
- [ ] **步骤 2：运行测试，确认失败**：命令同 S3 步骤 2；预期 FAIL，编译错误 `logs[0].CreatedAt.IsZero undefined (type string)`。
- [ ] **步骤 3：实现**：SQL 最后一列直接返回 `created_at`（`timestamptz`）；`AuditEntry.CreatedAt` 改为 `time.Time`；`AdminAudit` 输出 `entry.CreatedAt.Format(time.RFC3339)`。用 `grep -n "to_char(a.created_at" server/internal/data/*.go` 找到 `order_store.go:865` 附近另一处审计查询，同步检查。
- [ ] **步骤 4：运行测试，确认通过**：同上，PASS。
- [ ] **步骤 5：只暂存，不 commit、不 push**：文件同 S3。

---

### 任务 S5：平台外观返回 `updatedAt`、`updatedBy`（外观 spec Q1 = B，Kel 已定 2026-11-26 13:19）

> **端：** server ｜ **优先级：** P2 ｜ **依赖：** S1（024 里的 `platform_appearance.updated_at/updated_by`）；修复 plan 任务 10（同样改 `platform_appearance.go`、`service/platform_appearance.go`，加平台审计和进程内缓存，S5 必须在它之后、写进同一个事务）；平台外观 spec §10 Q1 = B（Kel 已定 13:19；和 024、W-A3 已核对一致） ｜ **状态：** 待开始

**现状核实：** 外观存在单行表 `platform_appearance`（`012`/`013`）；`GET /api/platform/appearance`（`server/internal/server/http.go:108`，**不需要登录**，根布局 SSR 也调它）返回 `{ "theme": "..." }`；`PUT`（`http.go:109`，平台管理员）只写 `theme`，返回请求体。没有时间、没有操作人。

**接口约定：**
- **以外观 spec §8.1（UIUX 13:30）为准，Kel 已定（13:19 Q9；13:32 再次确认“只对平台管理员返回”）。** `updatedBy` 是对象 `{ id, name }`，不是登录名字符串（12:25 版写的是字符串，已按 §8.1 改）；`name` = `COALESCE(NULLIF(users.display_name, ''), users.login_name)`。
- `PUT /api/platform/appearance`：写 `theme` 的同时写 `updated_at = now()`、`updated_by = 会话里的用户 ID`；**风格没变的 PUT 也算一次发布**（§8.1），照样覆盖两列。响应 `{ "theme": "shop", "updatedAt": "2026-11-26T21:30:00+08:00", "updatedBy": { "id": "7", "name": "运营 Kel" } }`。
- `GET /api/platform/appearance`：**只有平台管理员会话**返回 `{ theme, updatedAt, updatedBy: { id, name } | null }`；未登录、顾客、店主、销售员都**只返回 `{ "theme" }`**，`updatedAt` 和 `updatedBy` 两个键都不出现（§8.1 原文“未登录或非平台管理员只返回 `{ theme }`”，和后端 B-11 一致：同一个接口按会话返回不同内容，**不新建 `/meta`**，也**不走 `a.platform()`**，因为它会直接写 401/403）。从没发布过时两个字段都是 `null`。`updatedAt` 用 RFC3339 带时区偏移。**B-10：** 024 只对 `platform_appearance` 做 `ADD COLUMN`，不改 `012`、`013`；写入、审计和缓存失效放在同一个事务里。
- **核对结论（13:31）：** 12:25 版 S5 已经覆盖“匿名只返回 theme”（B-11），差两处，已补：`updatedBy` 的形状改成 `{ id, name }`；测试从“匿名 + 平台管理员”两种身份补成三种（加普通店铺会话）。

**文件：** `server/internal/data/platform_appearance.go`、`server/internal/service/platform_appearance.go`、`server/internal/service/platform_appearance_integration_test.go`（已存在，追加用例）

- [ ] **步骤 1：编写失败的测试**（追加到 `platform_appearance_integration_test.go`，沿用文件里已有的平台管理员登录夹具）

```go
	// Kel 已定（13:19 Q9、13:32）：只有平台管理员能看到 updatedAt/updatedBy；匿名和店铺会话只拿到 { theme }
	put := requestJSON(t, handler, http.MethodPut, "/api/platform/appearance", map[string]string{"theme": "shop"}, platform)
	if put.Code != http.StatusOK {
		t.Fatalf("put appearance: %d %s", put.Code, put.Body.String())
	}
	type actor struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var meta struct {
		Theme     string  `json:"theme"`
		UpdatedAt *string `json:"updatedAt"`
		UpdatedBy *actor  `json:"updatedBy"`
	}
	// 1) 平台管理员
	got := requestJSON(t, handler, http.MethodGet, "/api/platform/appearance", nil, platform)
	if err := json.Unmarshal(got.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Theme != "shop" || meta.UpdatedAt == nil || meta.UpdatedBy == nil || meta.UpdatedBy.ID == "" || meta.UpdatedBy.Name == "" {
		t.Fatalf("platform admin meta: %+v", meta)
	}
	if _, err := time.Parse(time.RFC3339, *meta.UpdatedAt); err != nil {
		t.Fatalf("updatedAt must be RFC3339 with offset: %q", *meta.UpdatedAt)
	}
	// 2) 匿名、3) 普通店铺会话（店主）：只有 theme
	owner := login(t, handler, "test-owner", password) // 按文件里已有的店主夹具名调整
	for name, cookie := range map[string]*http.Cookie{"anonymous": nil, "shop_admin": owner} {
		res := requestJSON(t, handler, http.MethodGet, "/api/platform/appearance", nil, cookie)
		var body map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || res.Code != http.StatusOK {
			t.Fatalf("%s GET: %d %s", name, res.Code, res.Body.String())
		}
		if len(body) != 1 || body["theme"] != "shop" {
			t.Fatalf("%s must only get {theme}, got %v", name, body)
		}
	}
	// 风格没变的 PUT 也算一次发布：updatedAt 前进
	before := *meta.UpdatedAt
	time.Sleep(1100 * time.Millisecond)
	requestJSON(t, handler, http.MethodPut, "/api/platform/appearance", map[string]string{"theme": "shop"}, platform)
	again := requestJSON(t, handler, http.MethodGet, "/api/platform/appearance", nil, platform)
	_ = json.Unmarshal(again.Body.Bytes(), &meta)
	if *meta.UpdatedAt == before {
		t.Fatal("same-theme PUT must still refresh updatedAt")
	}
```
> `login` 返回值的类型按 `auth_integration_test.go` 里的实际定义改写 map 的值类型（示例写成 `*http.Cookie`）；文件顶部补 `time` 导入。
> 追加在 `TestPlatformAppearance`（`platform_appearance_integration_test.go:19`）里，`platform := login(t, handler, "test-platform", password)` 是该函数第 50 行已有的夹具；`requestJSON` 在 `auth_integration_test.go`。

- [ ] **步骤 2：运行测试，确认失败**：`cd server && go test -tags integration ./internal/service -run PlatformAppearance -v`；预期 FAIL `platform admin meta`。
- [ ] **步骤 3：实现**：`SetPlatformTheme(ctx, theme string, actorID ID)` 的 SQL 改为 `INSERT INTO platform_appearance (singleton, theme, updated_at, updated_by) VALUES (true, $1, now(), $2) ON CONFLICT (singleton) DO UPDATE SET theme = EXCLUDED.theme, updated_at = now(), updated_by = EXCLUDED.updated_by`；新增 `PlatformAppearanceMeta(ctx) (theme string, updatedAt *time.Time, updatedByID *ID, updatedByName *string, err error)`（`LEFT JOIN users u ON u.id = updated_by`，`name` 取 `COALESCE(NULLIF(u.display_name, ''), u.login_name)`）。service 层：`PlatformAppearance` 判断会话是否平台管理员（用不写错误响应的会话读取，不能用会直接写 401 的 `a.platform`），是就带上两个字段。和修复 plan 任务 10 的平台审计、缓存放在同一个事务和同一次缓存失效里。
- [ ] **步骤 4：运行测试，确认通过**：同上 PASS；`go test ./...` 全绿。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`server/internal/data/platform_appearance.go`、`server/internal/service/platform_appearance.go`、`server/internal/service/platform_appearance_integration_test.go`

**前端对接点：** 任务 W-A3（外观 spec A19：状态条“上次发布：{name} · {时间}”，为 `null` 时“尚无记录”，PUT 成功后显示“你 · 刚刚”）。

---

### 任务 S6：“实收 · 找零”前后端一起清理（K-Q10 = A）

> **端：** server（前端部分在 W10） ｜ **优先级：** P2 ｜ **依赖：** **K-Q10 = A（Kel 已定 2026-11-26 13:22）**；**K-Q7 = B**（日报“现金长短款”隐藏，历史只在 PDF 保留）；vNext W5 已作废（12:40）；**K-Q17 = A（Kel 已定 2026-11-26 13:55）：S9b 也不再传 `cashReceivedMinor`，接口忽略这个字段** ｜ **后端原意见（B-9）：** 接口继续接收 `cashReceivedMinor`——**已被 K-Q17 = A 取代，作废** ｜ **状态：** 已批准开工

**核实结果（2026-11-26 12:05 UTC+8 首次核实，13:22 只读复核，以代码为准）：**

| 位置 | 现状 |
|---|---|
| 数据库列 | `orders.cash_received_minor bigint`、`orders.cash_change_minor bigint`（`schema.go:117-118`，`008_multi_currency_cash.sql:8-9`），可空 |
| 收款接口 | `POST /api/staff/orders/{id}/pay` 请求体 `cashReceivedMinor *int64`（`service/staff.go:629`），负数回 `400 invalid_cash_received`（`staff.go:635-636`） |
| 收款校验 | `data/order_checks.go:13-27` `cashSettlement`：**现金码必须带 `cashReceivedMinor` 且不少于总额**，否则 `ErrInvalidCashReceived`；二维码码带了也报错；计算 `change = received - total` 写库（`order_store.go:416`） |
| 错误码映射 | `service/handlers.go:695`、`service/order_commands.go:125` → `invalid_cash_received` |
| 订单返回 | `staff.go:807-808` 返回 `cashReceivedMinor`、`cashChangeMinor`（注意是 `cashChangeMinor`，不是 `changeMinor`）；`order_store.go:720/780`、`paged_lists.go:15` 都读这两列 |
| 日报 | `report_store.go:52、143-144、178-179`：`cashOverShortMinor` = 现金单 `sum(cash_received_minor) - sum(total_minor)`；前端日报和 PDF 都显示“现金长短款”（`admin/reports/page.tsx:377-392`，`lib/reports/document-model.ts:84、97、109`） |
| CSV 导出 | `service/reports.go` 没有用到这两个字段 |
| 前端 | `web/src/lib/order-actions.ts:44-45` `payCommandBody`：现金码一律传 `cashReceivedMinor: orderTotalMinor`；`web/src/lib/api.ts:161` 类型里还有 `cashReceivedMinor`；`web/src/lib/errors.ts:16` 还有 `invalid_cash_received` 文案“现金收款需要实收不少于订单总额；二维码收款不要填写现金金额。”；`counter/page.tsx:260、295、667` 待提交命令类型里有 `cashReceivedMinor?` |
| vNext | 后端 S9b 离线现金同步 `OfflineSyncInput.CashReceivedMinor`（`vnext-server.md:947`），S1 支付测试也传 `CashReceivedMinor`（`:375`）；前端 W5 / F-07“现金实收输入与找零计算器”（`vnext-web.md:26、307`）和这次“收银台去掉实收找零”的决定方向相反——**Kel 已定作废 W5（2026-11-26 12:40）**，vNext web plan 已标注；后端继续接收 `cashReceivedMinor`（S9b 要用） |

**13:22 复核补充：** 库里和实收、找零有关的只有 `orders.cash_received_minor`、`orders.cash_change_minor` 两列（`008_multi_currency_cash.sql:8-9`、`schema.go:117-118`），没有别的表或列；`daily_reports` 快照不单独存长短款（`report_store.go` 每次按订单现算）。收款接口用的是宽松的 `readJSON`（`staff.go:631`，不是 `readJSONStrict`），所以结构体里删掉字段后，请求里多带的 `cashReceivedMinor` 会被**直接忽略**，不会报错。

**Kel 的决定（K-Q10 = A）：** 前后端一起清理；W5 已作废；如果涉及库表，一个需求只出一个升级 SQL 和一个回滚 SQL。

**后端清理清单：**

| 位置 | 改法 |
|---|---|
| `service/staff.go:629` 收款请求体 | 删掉 `CashReceivedMinor` 字段；`staff.go:635-636` 的负数校验一起删。请求里再带这个字段时被忽略（`readJSON` 宽松解析）。 |
| `data/order_checks.go:13-27` `cashSettlement` | 只校验码的类型，不再计算实收和找零；签名改成不返回 `received, change`（或整个函数删掉，收款时两列写 `NULL`）。 |
| `data/order_store.go:340、346、403、416-418` | `PayOrderCommand`、`payOrderTx` 去掉 `cashReceivedMinor` 参数；`UPDATE orders` 不再写 `cash_received_minor`、`cash_change_minor`（新订单这两列保持 `NULL`）。 |
| `data/shop.go:19` `ErrInvalidCashReceived`、`service/handlers.go:694-695`、`service/order_commands.go:124-125` | 删掉错误和 `invalid_cash_received` 映射。 |
| `service/staff.go:807-808` 订单 JSON | 不再返回 `cashReceivedMinor`、`cashChangeMinor`。 |
| `data/order_store.go:49-50、720、729、780`、`data/paged_lists.go:15、29` | 读取订单时不再 select 这两列，`Order` 结构体去掉两个字段。 |
| `data/report_store.go:51-54、94、143-144、158、178-179、183、305` | **保留不改**：`cashOverShortMinor` 继续按 `cash_received_minor IS NOT NULL` 的历史现金单计算。K-Q7 = B 要求历史数据在 PDF 里保留；新订单两列为 `NULL`，不会进入这个合计。屏幕上隐藏由前端 W10 做。 |
| 测试 | `order_checks_test.go:53-60` 改写；`reservation_atomicity_integration_test.go:65-66` 去掉 `&cashReceived` 参数；`service` 层集成测试里收款请求体删掉该字段。 |

**要不要动库：不动，不出 SQL。** 理由：
1. K-Q7 = B 要求历史长短款在 PDF 里保留，数据来源就是这两列；删列会丢历史数据，而且不可逆。
2. 方案 A 原文是“数据库列保留或删除”，保留时没有表结构变化，“一个需求一份升级 SQL + 一份回滚 SQL”的要求不触发。
3. `schema.go:117-118` 也不改。
（如果 Kel 以后要删列，按“一个需求一份 SQL”应**另开一个迁移**（下一个可用编号），不放进 024：024 是店铺资料和外观元数据，和收款清理不是同一个需求。）

**vNext 的调整（K-Q17 = A，Kel 已定 13:55；13:55 已同步改 vNext 两份 plan）：**
- vNext **S4**（`vnext-server.md` Task S4 的测试，12:25 版误写成 S1）的 `PayInput{..., CashReceivedMinor: order.TotalMinor}`：`PayInput` 去掉 `CashReceivedMinor`，测试改成只传 `PayCodeID`。不论 K-Q17 选哪个都要改（在线收款按 K-Q10 = A 不再接收）。
- vNext S9b（`vnext-server.md:949`）的 `OfflineSyncInput.CashReceivedMinor: 2000`：
  - **K-Q17 = A（Kel 已定）：** S9b 也不再传。`OfflineSyncInput` 去掉 `CashReceivedMinor`，离线现金单按总额入账，`cash_received_minor`、`cash_change_minor` 写 `NULL`；同步接口请求里多带这个字段时**直接忽略**（和在线收款一致，靠 `readJSON` 宽松解析）。测试改成不传该字段，并加一条“带了也忽略、不报错”的用例。
- 已同步（13:55）：`2026-11-26-stallink-vnext-server.md` Task S4 测试的 `PayInput` 去掉 `CashReceivedMinor`、Task S9b 加 K-Q17 说明并删掉测试里的 `CashReceivedMinor: 2000`；`2026-11-26-stallink-vnext-web.md` W5 作废说明和 W8 依赖说明改为“离线单和同步请求都不带 `cashReceivedMinor`”。

**⚠️ 幂等哈希（后端 B-9）：** 收款请求的幂等哈希按**整个请求体**计算。字段删掉后，新版前端发的请求体不再带它；旧标签页里还没完成的收款如果用同一个幂等键重发**原来的请求体**，哈希不变，正常回放；只有“同一个键、前后两次请求体不同”才会 `idempotency_conflict`。前端一次切换（W10），上线说明写“刷新后重试”。

- [ ] **步骤 1：编写失败的测试**
  - `order_checks_test.go`：现金码、二维码码都不再需要也不接受实收参数；`cashSettlement`（如保留）对现金码返回 `nil, nil, nil`。
  - `service` 集成测试（`-tags integration`）：现金码收款请求体 `{ "payCodeId": "..." }` → 200，返回体不含 `cashReceivedMinor`、`cashChangeMinor`，库里两列为 `NULL`；请求体多带 `"cashReceivedMinor": 1` → 同样 200 且两列为 `NULL`（被忽略）。
  - `report_store` 集成测试：一张历史现金单（直接 INSERT `cash_received_minor = total + 100`）+ 一张新收款单 → `cashOverShortMinor = 100`（新单不影响）。
- [ ] **步骤 2：** `cd server && go test ./internal/data ./internal/service -run 'Cash|Pay|Report' -v`，确认失败。
- [ ] **步骤 3：实现**：按上面的后端清理清单。
- [ ] **步骤 4：** `go test ./...` 全绿；集成测试用本机 docker Postgres（同 S1 步骤 5b 的命令），不启动 API。
- [ ] **步骤 4b：PDF 数据仍在（Kel 14:21 定 PDF 保留长短款；K-Q18 = A 的“录了实收的单数”见新任务 S10，Kel 15:01）**：快照的 `cashOverShortMinor`（根和 `currencies[]`）**继续计算、继续返回**，不能随实收清理一起删；`report_store.go:143-144、178-179` 的算法不改。测试：`report_currency_integration_test.go` 里造一张历史现金单（`cash_received_minor` 有值）和一张新单（为 `NULL`），断言快照 `cashOverShortMinor` = 历史单的 `cash_received_minor - total_minor`，新单不参与。只在本机 docker-compose 的 Postgres 上跑。注意矛盾见 §11.6 K-Q18（新订单没有实收，这个数对之后的日子恒为 0）。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`server/internal/service/staff.go`、`server/internal/service/handlers.go`、`server/internal/service/order_commands.go`、`server/internal/data/order_checks.go`、`server/internal/data/order_checks_test.go`、`server/internal/data/order_store.go`、`server/internal/data/paged_lists.go`、`server/internal/data/shop.go`、`server/internal/data/reservation_atomicity_integration_test.go` 及相关集成测试

**上线顺序：** S6（后端）和 W10 的请求清理同一次上线；如果做不到，先上后端（多带的字段被忽略，旧前端不受影响），再上前端。

---

### 任务 S10：日报快照加 `cashTenderedOrders`，区分“持平”和“未录实收”（K-Q18 = A）

> **端：** server（前端部分在 W10） ｜ **优先级：** P2 ｜ **后端 Mac 现状（15:07 只读）：** 已在做，未提交：`report_store.go`、`report_store_test.go`、`report_currency_integration_test.go` 已出现 `CashTenderedOrders`（不带 `omitempty`）和旧快照补键逻辑，`service/reports.go` 只改了 `GetDailyReport` 的调用参数，CSV 没加这一项 ｜ **依赖：** S6（后端已在 `design/editorial-frontend` 分支提交为 `ee103c1`，所以 K-Q18 单独成任务，不再改 S6 的提交）；**K-Q18 = A，Kel 2026-11-26 15:01 定（Grok Bot 转达）** ｜ **状态：** **已提交**：`design/editorial-frontend` 分支 `9f7e7f4`“feat(reports): add cashTenderedOrders to daily report snapshot (K-Q18=A)”，**Kel 经 Grok Bot 同意单独提交一次（15:11 转达），未 push**

**旧快照：** 读取 K-Q18 之前存下的快照时，由后端按订单现算计数补上缺的键（`report_store.go` 的 `fillMissingCashTenderedOrders`、`cashTenderedOrdersByCurrency`），**不写库**；所以接口总会带 `cashTenderedOrders`，前端“缺字段照旧显示”只是兜底。

**背景：** S6 步骤 4b 让 `cashOverShortMinor` 继续算、继续返回（PDF 保留长短款，Kel 14:21）。K-Q10 / K-Q17 之后新订单 `cash_received_minor` 都是 `NULL`，所以新日子的长短款恒为 0，PDF 会显示“MYR 0.00 持平”，像是点过钱箱刚好对上。K-Q18 = A：快照多给一个“当天录了实收的单数”，前端为 0 时显示「—（未录实收）」，历史日期照常。

**现状（15:0x 只读核对 Mac，`server/internal/data/report_store.go`）：** `DailyReportCurrency`（第 40–55 行）、`DailyReport`（第 79–100 行）都没有这个字段；分币种查询第 137–160 行、根查询第 172–186 行用 `FILTER (WHERE cash_received_minor IS NOT NULL)` 算 `cashOverShortMinor`；`applyHeadlineCurrency`（第 293–311 行）把主币种行复制到根；第 172 行的根查询随后按 `currency = report.Currency` 再算一遍并覆盖。

**接口约定（K-Q18 = A，前后端共用）：**
- 字段：日报快照根对象 `cashTenderedOrders`（主币种）和 `currencies[i].cashTenderedOrders`（该币种），整数，≥0，**总是输出**（Go 不加 `omitempty`）。
- 含义：当天（报表时区）、本店、该币种下 `cash_received_minor IS NOT NULL` 的订单数，口径和 `cashOverShortMinor` 完全一样（同一组 `WHERE`、同一个 `FILTER`），只是计数。
- 前端判断：
  - `cashTenderedOrders === 0` → 长短款显示「—（未录实收）」，不显示“{金额} 持平”；
  - `cashTenderedOrders > 0` → 照常显示长款 / 短款 / 持平（`cashOverShortMinor`）；
  - **字段不存在**（`daily_reports.payload` 里 K-Q18 之前存下的旧快照）→ 按旧逻辑显示数值，**不当成 0**。后端 15:07 的工作区（未提交）在读取时用订单现算的计数把缺的键补上（`fillMissingCashTenderedOrders`、`cashTenderedOrdersByCurrency`，`GetDailyReport` 多了时区参数），所以接口正常情况下总会带这个字段；前端的“缺字段照旧显示”保留作兜底。两边不冲突。
- 按币种（**K-Q22 = A，Kel 15:15 定：每种币各自判断**，§11.13；对应销售日报设计说明 §10 第 13 条（已关闭，第 540–546 行）、§4.6 第 198–210、220–221 行）：
  - 分币种汇总每一行用 `currencies[]` 里**该行自己**的 `cashTenderedOrders` 判断；`zeroRow()` 补出来的空币种行计数是 0，显示「—（未录实收）」。
  - 指标卡用根对象的 `cashTenderedOrders`，**根对象只代表主币种**（`report_store.go` `applyHeadlineCurrency` 约第 313 行把主币种行复制到根），不是整天所有币种合计；筛了非主币种时用该币种行。
  - 前端 `document-model.ts` 现在（15:07）就是这样做的（第 87、100、112 行传所在对象的计数）。
  - 导出非主币种时，指标卡的金额和「未录实收」判断都用该币种行（产品经理对齐，K-Q22 = A 的推论）；后端不用改。设计说明 15:17 版已对齐。
- 不改数据库表、不加迁移；CSV 不输出这个字段（长短款在 CSV 里本来就隐藏，C-2 = A）。

**文件：**
- 修改：`/Users/kelden/go/src/Stallink/server/internal/data/report_store.go`
- 测试：`/Users/kelden/go/src/Stallink/server/internal/data/report_currency_integration_test.go`（追加，`-tags integration`）

- [ ] **步骤 1：编写失败的测试**（`report_currency_integration_test.go`，只连本机 docker 库）：同一店铺、同一天、同一币种造两张单：
  - 历史单：直接 INSERT，`cash_received_minor = total_minor + 100`；
  - 新单：走现在的收款路径（或 INSERT），`cash_received_minor IS NULL`。
  - 断言：根 `CashTenderedOrders == 1`，该币种 `Currencies[i].CashTenderedOrders == 1`，`CashOverShortMinor == 100`（不变）。
  - 再造一天只有新单（`NULL`）：根和币种行的 `CashTenderedOrders == 0`，`CashOverShortMinor == 0`。
  - JSON 断言：`json.Marshal(report)` 含 `"cashTenderedOrders":0`（0 也要输出，不能被省略）。
- [ ] **步骤 2：运行测试，确认失败**：`cd /Users/kelden/go/src/Stallink/server && STALLINK_TEST_DATABASE_URL="postgres://stallink:${LOCAL_PG_PASSWORD}@127.0.0.1:5432/stallink?sslmode=disable" go test -tags integration ./internal/data -run 'Report.*Tendered|ReportCurrency' -v`（同 S1 步骤 5b 的本机库；**不连云端，不启动 API**）。
- [ ] **步骤 3：实现**：
  - 两个结构体各加 `CashTenderedOrders int `json:"cashTenderedOrders"``（紧挨 `CashOverShortMinor`）；
  - 分币种查询、根查询各加一列 `count(*) FILTER (WHERE cash_received_minor IS NOT NULL)`（分币种查询里写 `o.cash_received_minor`），`Scan` 对应加；
  - `applyHeadlineCurrency` 一起复制 `CashTenderedOrders`；
  - 不改表、不加迁移；`service/reports.go` 的 CSV 不加这一项。
- [ ] **步骤 4：运行测试，确认通过**；`go test ./...` 全绿；`gofmt`、`go vet ./...` 无新增问题。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`server/internal/data/report_store.go`、`server/internal/data/report_currency_integration_test.go`。（例外：Kel 经 Grok Bot 批准本任务单独提交一次，已提交为 `9f7e7f4`，不 push。）

**备注：** K-Q22 = A（Kel 15:15），不需要整天合计字段，本任务已提交的内容不用再改。

**验收：** 上面两种日子的计数分别是 1 和 0；`cashOverShortMinor` 算法和结果不变；旧快照（payload 里没有这个字段）读取接口原样返回、不报错；没有新迁移文件。

---

### 任务 S7：日报 CSV 按屏幕指标补齐，计数列统一叫“数量”（设计说明 §8.3，R2）

> **端：** server ｜ **优先级：** P1（导出用于对账，表头和内容不符属于缺陷） ｜ **依赖：** **设计说明 R2（Kel 已定 2026-11-26 13:19，UIUX 13:30 写进 §8.3）**；**C-1 = A、C-2 = A（Kel 已定 2026-11-26 13:55）** ｜ **后端原答复（B-12）：** 同意改成 6 列——**作废**，按 R2 出 5 列 ｜ **状态：** 已批准开工

**核实：** `server/internal/service/reports.go:51-115` `renderDailyReportCSV` 的表头固定为 `分组, 币种, 项目, 订单数, 金额`（第 108 行）；“按商品”行把 `g.Quantity`（件数）写进“订单数”列（第 89 行）；“退款待处理”行在“订单数”列写 `0`（第 102 行附近）。文件名由服务端决定：`reports.go:223-228`，`currency` 为空时取快照主币种，`Content-Disposition: attachment; filename="daily-report-{date}-{主币种}.csv"`；前端 `admin/reports/page.tsx:180-181` 先读这个响应头，读不到才用同样格式兜底。

**冲突 C-1（列结构）：** 本计划 12:25 版按 K-Q12 推荐方案 A 写的是 6 列（`订单数`、`件数` 分开，后端 B-12 已同意）。设计说明 R2（Kel 13:19 已定）是 **5 列**：`分组,币种,项目,数量,金额`，“数量”由“项目”区分是笔数还是件数，相当于 K-Q12 的方案 B 再加上“按屏幕指标补齐”。**Kel 已定 13:55：C-1 = A，按 R2 出 5 列，B-12 的 6 列作废。**

**冲突 C-2（现金长短款）：** 设计说明 §8.3 的总览和按币种段都有“现金长短款”行，§4.4 六张指标卡、§4.5 分币种汇总也保留“长短款”；而 Kel 13:22 定的 K-Q7 = B 是“隐藏，历史数据只在 PDF 里保留”。两条都是 Kel 的决定，内容相反。**Kel 已定 13:55：C-2 = A，“现金长短款”在 CSV、分币种汇总、指标卡三处全部隐藏**（PDF 历史保留，见 K-Q7 = B）。本任务不写这两行。

**按 R2 / §8.3 的目标（现金长短款行按 C-2 = A 不写）：**
- 表头固定五列：`分组,币种,项目,数量,金额`。数量不适用的行留空，不写 0。
- **总览**（主币种取快照根字段；选了非主币种时用 `currencies[]` 里该币种有的字段，没有的字段不出行）：收款总额（数量 `paidOrders`，金额 `revenueMinor`）、订单数（`orders`）、待付款订单（`pendingOrders`）、已关闭订单（`closedOrders`）、退款订单（`refundOrders`）、售出件数（`itemsSold`）、平均每单（金额 `averageMinor`）、优惠金额（金额 `discountMinor`）、待退金额（金额 `refundMinor`，0 也写）。不再单独写“已付款订单”行；取消原来只在 `refundMinor != 0` 时出现的“退款待处理”段。
- **按币种**（`currencies[]`，每个币种一组）：全部订单（`orders` / `revenueMinor`）、已付款订单（`paidOrders`）、优惠金额（`discountMinor`）、待退金额（`refundMinor`）。
- **按活动**：每场两行——活动名（`orders` / `revenueMinor`）、“{活动名} 优惠”（数量空，`discountMinor`，0 也写）。
- **按商品**：商品名（`quantity` / `revenueMinor`）。**按收款方式**：名称（`orders` / `revenueMinor`）。
- 商品名等仍经 `spreadsheetText`；金额主单位两位小数；UTF-8 BOM。
- **币种筛选（设计说明问题 2，Kel 13:32 已定：维持现状）：** `currency` 为空（“全部币种”）时，CSV 仍只导出主币种（明细只出主币种）；文件名写明主币种（服务端 `Content-Disposition`，现有逻辑已满足，本任务加测试锁住）。页面提示归前端 W16。
- **非主币种的件数和各类订单笔数（设计说明问题 11，Kel 13:32 已定：维持现状）：** 不在服务端按币种补统计；导出非主币种时总览不出这些没有字段的行（§8.3 原文）。界面说明归前端 W16。

**文件：** `server/internal/service/reports.go:51-115、223-228`、`server/internal/service/report_csv_test.go`

- [ ] **步骤 1：编写失败的测试**（改 `report_csv_test.go`）
  - `TestRenderDailyReportCSVHeaderAndGroups`：表头等于 `[]string{"分组", "币种", "项目", "数量", "金额"}`。
  - `TestRenderDailyReportCSVOverviewRows`：总览依次有“收款总额、订单数、待付款订单、已关闭订单、退款订单、售出件数、平均每单、优惠金额、待退金额”；“收款总额”行数量 = `paidOrders`；“平均每单”“优惠金额”“待退金额”数量列为空；`refundMinor = 0` 时“待退金额”仍出现、金额 `0.00`；没有“已付款订单”“退款待处理”行；**没有“现金长短款”行（C-2 = A）**。
  - `TestRenderDailyReportCSVEventDiscountRow`：每个活动后面跟一行“{活动名} 优惠”，数量空。
  - `TestRenderDailyReportCSVGoodsQuantity`：按商品行数量 = `quantity`，不再出现在“订单数”含义的列。
  - `TestRenderDailyReportCSVNonPrimaryCurrency`：导出非主币种时总览没有“待付款订单、已关闭订单、退款订单、售出件数”行（问题 11）。
  - `TestRenderDailyReportCSVCurrencyFilter`（已有）：`currency` 为空时明细只有主币种（问题 2）；读“金额”的列下标保持 `row[4]`（5 列，B-12 说的改 `row[5]` 不再需要）。
  - service 层 `TestExportDailyReportFilenameUsesPrimaryCurrency`：`GET /api/admin/daily-reports/export?date=…`（不带 currency）的 `Content-Disposition` 是 `daily-report-{date}-{主币种}.csv`。
- [ ] **步骤 2：运行测试，确认失败**：`cd server && go test ./internal/service -run 'RenderDailyReportCSV|ExportDailyReport' -v`。
- [ ] **步骤 3：实现**：`add(group, curr, item string, quantity *int64, amountMinor *int64)`，`nil` 写空字符串；按上面的行顺序重写各段；表头 5 列。
- [ ] **步骤 4：运行测试，确认通过**：同上 PASS；`go test ./...` 全绿。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`server/internal/service/reports.go`、`server/internal/service/report_csv_test.go`

**前端对接点：** CSV 由服务端生成，前端只下载；“全部币种”提示和非主币种说明见 W16。

---

### 任务 S8：日报“报表今天”由服务端下发（设计说明 R4 / §8.4；K-Q14 = A；和 vNext 时区定义对齐）

> **端：** server ｜ **优先级：** P2 ｜ **依赖：** **K-Q14 = A（Kel 已定 2026-11-26 13:19，设计说明 R4）**；vNext server plan Task S6（迁移 `016_event_timezone_and_presale`）、Task S8（迁移 `018`，日报按 `e.timezone` 聚合） ｜ **状态：** Kel 已定，待开工

**核实：**
- 前端 `web/src/app/admin/reports/page.tsx:39` `localToday()` 用浏览器本地日期，`:109` 作为默认起止日期。
- 服务端 `server/internal/service/reports.go:117-135`：`reportLocation` 来自配置 `reports.timezone`（`server/internal/conf/conf.go:77-83`，IANA 名称，缺省 `DefaultReportTimezone = "Asia/Kuala_Lumpur"`），`reportDate` 拒绝晚于该时区“今天”的日期，返回 `invalid_date`。人不在吉隆坡时区时，默认查询可能直接失败。
- vNext Task S6 给 `events` 加 `timezone text NOT NULL DEFAULT 'Asia/Kuala_Lumpur'`（IANA，`time.LoadLocation` 校验），Task S8 让日报按 `e.timezone` 切日。也就是说 vNext 合入后，“一天”的边界会从**全局配置**变成**按活动**。

**本计划的定义（避免和 vNext 冲突）：**
1. 本计划**不新增任何时区字段或配置**，只把服务端已有的“报表今天”暴露给前端。
2. “报表今天” = 服务端用“报表时区”算出的日期。报表时区读配置 `reports.timezone`，默认 `Asia/Kuala_Lumpur`（后端 B-13）。vNext S6/S8 合入后怎么取由 vNext 定义；本计划只要求接口字段名不变。
   - **给 vNext 的后续建议（后端 B-14，记录在此，本计划不实现）：** 店铺的报表时区依次取主推活动、进行中的活动、`starts_on` 最近的活动的 `events.timezone`，都没有就用 `Asia/Kuala_Lumpur`。请 vNext server plan 的维护者在 Task S6/S8 里采纳或另行说明。细则 Kel 13:22 已定（B14-1 = A：多个进行中取 `starts_on` 最早的；B14-2 = A：没有进行中取日期最近的，已结束和未开始一起算，距离相同取未开始的），已同步写进 `docs/superpowers/plans/2026-11-26-stallink-vnext-server.md` 第八节 B-14。
3. 接口：`GET /api/admin/daily-reports/today`（权限和日报其他接口一样：`shopAdmin`；未登录 `401`，无权限 `403`），响应 **`{ "today": "2026-11-26", "timezone": "Asia/Kuala_Lumpur" }`**——字段名按设计说明 §8.4 用 `today`（12:25 版写的是 `date`，已改，需要通知后端）。读、生成、导出接口继续拒绝晚于 `today` 的日期（`invalid_date`）。`timezone` 一律是 IANA 名称，和 vNext `events.timezone` 同一种写法；配置为空时返回 `"Asia/Kuala_Lumpur"`。**后端已答复（B-13）：** 新增这个接口，S8 按此写；不在现有日报响应里附带。

**TDD 要点：** `ConfigureReportLocation` 现在只存 `*time.Location`，需要同时记下配置的 IANA 名称（空或非法时记 `conf.DefaultReportTimezone`），接口才能返回 `timezone`。`reports_test.go` 加 `TestReportTodayUsesReportLocation`：`ConfigureReportLocation("Asia/Tokyo")` 后接口返回的 `today` 等于 `time.Now().In(Asia/Tokyo)` 的日期、`timezone == "Asia/Tokyo"`；空配置返回 `"Asia/Kuala_Lumpur"`。

**前端对接点：** 任务 W17。

---

### 任务 S9：新建活动带店铺默认值；顾客端接口返回生效摊位号（K-Q1 = A、K-Q2 = A）

> **端：** server ｜ **优先级：** P1 ｜ **依赖：** S1（`shops.booth_number`、`shops.default_currency`）、S2（`data.Shop` 读出这两个字段）；**K-Q1 = A、K-Q2 = A（Kel 已定 2026-11-26 13:22）** ｜ **前端对接：** W18 ｜ **状态：** Kel 已定，待开工

**Kel 的决定：**
- K-Q1 = A：店铺摊位号是**新建活动时的默认值**；顾客端**优先显示活动自己的摊位号**，活动没填才用店铺的。
- K-Q2 = A：默认货币是**新建活动的默认货币**；存量店铺回填见 S1 步骤 5b（手动脚本 024a，K-Q16 = A；严格按最近一场活动）。

**核实（13:22 只读）：** `events.booth_number` 已存在（`schema.go:54`，`<= 40` 字符，默认 `''`），**不需要加进 024**。新建活动有三条路径：`catalog_store.go:225` `CreateEvent`（活动页）、`catalog_store.go:475` 和 `bundle_store.go:115`（商品页、套餐页里“顺手新建活动”的 `NewEvent`）。三条都先经过 `service/staff.go` 的 `readEvent` → `validEvent`（`staff.go:752-777`），`validEvent` 现在要求 `currency` 合法，空串会报 `400 invalid_event`。公开接口里活动 JSON 由 `handlers.go:595` `eventJSON` 生成，`handlers.go:352`（活动详情）、`:583`（店铺页）用它；工作台 `staff.go:40` 也用它。

**接口约定：**
- 新建活动（三条路径）：请求里 `boothNumber` 为空串时，用店铺的 `booth_number`；`currency` 为空串时，用店铺的 `default_currency`；店铺也没设默认货币时仍返回 `400 invalid_event`（和现在一样）。**只在新建时生效**，编辑活动（`catalog_store.go:281`）不带默认值，店主清空摊位号就保存为空。
- 公开接口 `GET /api/public/shops/{slug}`、`GET /api/public/shops/{slug}/events/{eventID}` 的每个活动多返回 `displayBoothNumber`：活动自己的 `boothNumber` 非空就用它，否则用店铺的 `boothNumber`，都没有是 `""`。原来的 `boothNumber`（活动自己的值）照样返回。工作台接口（`staff.go:40`）不加这个字段。
- `/api/public/catalog`（全站目录）如果也展示活动摊位号，执行时核对它的数据来源，按同一规则补 `displayBoothNumber`；核对不到就在 W18 里只改店铺页。

**文件：** `server/internal/service/staff.go`（`readEvent`、新建活动的 handler）、`server/internal/data/catalog_store.go:225、475`、`server/internal/data/bundle_store.go:115`、`server/internal/service/handlers.go:352、583、595`、新建 `server/internal/service/event_defaults_integration_test.go`

- [ ] **步骤 1：编写失败的测试**（`-tags integration`，用 `login`/`requestJSON`）
  - 店铺 `boothNumber = "A-27"`、`defaultCurrency = "SGD"` → `POST` 新建活动，请求体 `boothNumber: ""`、`currency: ""` → 201，读回活动 `boothNumber = "A-27"`、`currency = "SGD"`。
  - 同一店铺，请求体 `boothNumber: "B-01"`、`currency: "MYR"` → 保存的是请求里的值。
  - 店铺没设默认货币，请求体 `currency: ""` → `400 invalid_event`。
  - 编辑活动把 `boothNumber` 改成 `""` → 保存为 `""`（不再套默认值）。
  - 商品页“顺手新建活动”（`NewEvent`）走同样的默认值：至少一条用例。
  - 公开 `GET /api/public/shops/{slug}`：活动 `boothNumber = "B-01"` → `displayBoothNumber = "B-01"`；活动 `boothNumber = ""`、店铺 `"A-27"` → `displayBoothNumber = "A-27"`；两边都空 → `""`。活动详情接口同样断言。
  - 工作台活动列表不含 `displayBoothNumber`。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：新建路径里先读店铺的 `booth_number`、`default_currency`（和插入在同一个事务），空串时代入，再做 `validEvent` 的货币校验（新建时把货币校验挪到代入之后；编辑路径不变）。公开接口加 `publicEventJSON(event, shop)`，在 `eventJSON` 的基础上加 `displayBoothNumber`。
- [ ] **步骤 4：** `go test ./...` 全绿；集成测试用本机 docker Postgres（同 S1 步骤 5b），不启动 API。
- [ ] **步骤 5：只暂存，不 commit、不 push**：上面列的文件。

---
## 8. 前端任务（web）

### 8.0 通用约定（每个 web 任务都适用）

- **spec 推荐方案：** 涉及 spec §10 待确认问题的地方，原先按推荐方案写。**13:30 起展场/店面 Q1–Q6、外观 Q9–Q12、日报 R1–R4 和问题 2、11 都已由 Kel 定，相关任务已改成“Kel 已定”。**
- **不逐条抄 spec：** 每个任务写明“spec 第几节、条目范围”，实现时逐条对照 spec 表格的“现在 → 改成”两列。本计划只补 spec 没写的部分：测试、依赖、验收和磁盘现状。
- **通用验证命令**（下文简称“通用验证”）：
  ```bash
  cd web && node --test tests/ && npx eslint && npx tsc --noEmit && npx next build
  # 视觉：mock 后端 + 生产构建（不启动 Go 后端，不碰数据库）
  node ../docs/reports/stallink-test-2026-11-26/scripts/mock-api.cjs &          # 监听 18080
  API_ORIGIN=http://127.0.0.1:18080 npx next start -p 3100 &
  NODE_PATH=/tmp/stallink-pw/node_modules node ../docs/reports/stallink-test-2026-11-26/scripts/layout-sweep.cjs   # 三套风格 × 390/820/1280，scrollWidth 必须等于视口宽度
  NODE_PATH=/tmp/stallink-pw/node_modules node ../docs/reports/stallink-test-2026-11-26/scripts/contrast2.cjs      # 文字 ≥4.5:1，边框 ≥3:1
  ```
  截图输出放 `/tmp/stallink-shots/`，不进仓库。
- **最后一步一律是“只暂存，不 commit、不 push”**，只列拟提交的文件。**只有 Kel 经 Grok Bot 明确批准的提交可以做，其余一律只暂存**；push 一律不做（§11.11）。
- 前端这一轮改动还在主目录、全部未提交。开工前先 `git status --short` 和 `git diff --stat`，**已经改好的条目只补测试，不重做**。

---

### 8.1 基线

#### 任务 W1：验收基线：页面 × 截图对照表

> **端：** web ｜ **优先级：** P0 ｜ **依赖：** K-01 ｜ **状态：** 待开始（前端 12:21 回报的已完成项里没有这一项）

- [ ] 在 `docs/reports/` 下新建对照表（前端执行时建，本计划不建）：每行“页面路径 × 风格 × 宽度 → 目标截图”。目标截图：
  - classic：`docs/design-v2/screens/` 原有编辑式图。
  - 展场 / 店面：`展场-*`、`店面-*`（总览、收银台、销售日报 × 桌面、手机）。
  - 平台外观页：`平台外观-*` 10 张。
  - 日报：`销售日报-v2-*` 7 张。
- [ ] 没有目标截图的页面（商品、收款方式、销售账号、店铺资料、操作记录、平台店铺管理、商品审核）在表里标“无目标图，按 classic 编辑式风格 + spec §4 token 验收”，交给 T1 补截图。
- **验收：** 表里每个工作台页面都有一行；没有空的“目标”格。

---

### 8.2 共用（展场/店面 spec §8.1 C1–C16）

#### 任务 W-C1：token、遗留旧绿、焦点环、全局减少动效、主题开关

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 无 ｜ **spec：** `2026-11-26-stallink-theme-exhibition-shop-design.md` §8.1 **C1–C7**；§8.3 **S17**（交给 C6 统一处理）；§4.5–§4.7 token 表 ｜ **前端已改：** C2（`.header-loading` 改成 `color-mix(in srgb, var(--ink) 6%, transparent)`，和 spec 目标 `rgba(22, 23, 26, 0.06)` 等效，保留即可） ｜ **状态：** 部分完成

**文件：** `web/src/app/globals.css`、`web/src/components/app-nav.tsx`、`web/src/app/shop.css`（S17 删除）、新建 `web/tests/theme-tokens.test.cjs`

- [ ] **步骤 1：编写失败的测试**

```js
/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const globals = fs.readFileSync('src/app/globals.css', 'utf8');

test('C1 shared radius/shadow/motion tokens exist', () => {
  for (const token of ['--radius-control', '--radius-card', '--radius-panel']) {
    assert.match(globals, new RegExp(`${token}:`), `${token} missing`);
  }
});

test('C2-C4 no legacy green left in globals.css', () => {
  assert.doesNotMatch(globals, /rgba\(49,\s*89,\s*66/);
});

test('C5 focus ring uses --focus', () => {
  assert.match(globals, /:focus-visible[^}]*outline:\s*2px solid var\(--focus\)/);
});

test('C6 one global reduced-motion rule', () => {
  assert.match(globals, /@media \(prefers-reduced-motion: reduce\)\s*{\s*\*,\s*\*::before,\s*\*::after/);
});

test('C7 theme toggle only renders for classic', () => {
  const nav = fs.readFileSync('src/components/app-nav.tsx', 'utf8');
  assert.match(nav, /storefrontTheme\s*(!==|===)\s*"classic"/);
});
```

- [ ] **步骤 2：** `cd web && node --test tests/theme-tokens.test.cjs`；预期 C1、C3/C4、C5、C6、C7 失败，C2 那一处已通过。
- [ ] **步骤 3：实现**：逐条按 spec §8.1 C1、C3–C7 的“改成”列修改；C1 的 token 取值按 §4.5–§4.7 的 classic 一列（classic 圆角是展场 spec Q6（13:09 重编号前是 Q7，12:44 前是 Q8）= K-Q8，**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：B 保持圆角**——classic 保留磁盘上的 10–20px 柔化圆角，不回直角、不另开 classic 任务；本任务不动 classic 的圆角值；spec 13:30 §2 第 16 条：design-v2 稿是直角、实现是圆角，**这一处差异不算验收不通过**，不用改 design-v2 稿）。删掉 `shop.css` 第 3690 行附近的 S17 局部规则。
- [ ] **步骤 4：** 通用验证；另外用系统“减少动效”打开 `/admin`，加载圈不转、无位移动画。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/globals.css`、`web/src/components/app-nav.tsx`、`web/src/app/shop.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 6 条后半（展场顶栏没有深浅色开关）、第 9 条（减少动效）、第 10 条（焦点环清晰）。

---

#### 任务 W-C2：侧栏编号、总览卡片、页面名称（按 K-Q15 方案 A）

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** **K-Q15 = A（Kel 已定，2026-11-26 12:40，Grok Bot 转达）**；K-02、K-03；展场/店面 spec **Q5**（H1“现场开单”）——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：A**（13:09 重编号前是 Q6，12:44 前是 Q7）；UIUX 改 spec（**已完成**，12:44） ｜ **spec：** §5.2 编号规则、§8.1 **C8、C9、C10、C11、C15、C16**，§8.2 **E11**，§8.3 **S10**，§9 第 2、3 条（均为 UIUX 12:44 按方案 A 改过的版本） ｜ **状态：** Kel 已定方案 A，spec 已同步，待开工

**Kel 的决定（K-Q15 = A），spec 已按此改写（§5.2、C8、C9、C11）：**

| 分组 | 编号 | 名称（侧栏名 = 卡片标题） | 路径 |
|---|---|---|---|
| 经营 | — | 总览（不编号，没有卡片） | `/admin` |
| 经营 | 01 | 活动 | `/admin/events` |
| 经营 | 02 | 商品 | `/admin/goods` |
| 经营 | 03 | 订单记录 | `/admin/orders` |
| 经营 | 04 | 销售日报 | `/admin/reports` |
| 现场收银 | — | 现场开单（不编号，没有卡片） | `/counter` |
| 现场收银 | — | 本场订单（不编号，没有卡片） | `/counter/orders` |
| 店铺 | 05 | 收款方式 | `/admin/pay-codes` |
| 店铺 | 06 | 销售账号 | `/admin/sales` |
| 店铺 | 07 | 店铺资料 | `/admin/shop` |
| 店铺 | 08 | 操作记录 | `/admin/audit` |

- 侧栏：三组，顺序是经营、现场收银、店铺。
- 总览卡片：只有 8 个管理项，分“经营”（01–04）和“店铺”（05–08）两组，**同序、同号、同名、同分组**；收银组不出卡片，入口是横幅下的“打开现场收银”按钮（C11）。
- `cashierGroup` 摊主和销售员共用一个常量，**不改成工厂函数**（C9）。
- 侧栏编号**只在 classic 显示**；展场、店面的 `.workspace-nav__index` 保持 `display: none`（spec §2 第 8 条、E11、S10，Kel 已定 2026-11-26，UIUX 13:09 写进 spec）。`index` 数据照样保留，收银组两项为空不渲染（C9）。测试锁的是数据，不受这条影响。

**12:45 磁盘现状（只读核对）：**
- **侧栏已经对了（F-10 已修）**：`workspace-shell.tsx:59-70` 是总览（无 `index`）、01–04、`cashierGroup`（两项无 `index`）、05–08，`/admin/orders` 已叫“订单记录”。
- **总览卡片还差两处**（`admin/page.tsx:3-12`）：
  - 01、02 的标题仍是“活动与场次”“商品与套餐”，要改成“活动”“商品”；
  - 8 张卡没有分组，要分成“经营”“店铺”两组。
- 还没做：C10 `workspace-sidebar__foot`；C15 `counter/page.tsx:780` 仍是 `<h1>创建订单</h1>`。
- “开摊准备”这类小标签（`label` 字段）spec C11 只说“描述见原型”，去留以原型为准，测试不锁定。

**文件：** `web/src/app/admin/page.tsx`（改标题、分组）、`web/src/components/workspace-shell.tsx`（C10；侧栏只核对）、（E11、S10 已定保持 `display: none`，CSS 不用改）、`web/src/app/counter/page.tsx:780`、`web/tests/workspace-copy.test.cjs`（已有，核对断言不冲突）、新建 `web/tests/workspace-nav.test.cjs`

- [ ] **步骤 1：编写失败的测试**（`workspace-nav.test.cjs`）

```js
/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
const overviewSrc = fs.readFileSync('src/app/admin/page.tsx', 'utf8');

// spec §5.2 / C8 / C11（12:44 版）：同序、同号、同名、同分组
const business = [
  ['/admin/events', '01', '活动'], ['/admin/goods', '02', '商品'],
  ['/admin/orders', '03', '订单记录'], ['/admin/reports', '04', '销售日报'],
];
const shop = [
  ['/admin/pay-codes', '05', '收款方式'], ['/admin/sales', '06', '销售账号'],
  ['/admin/shop', '07', '店铺资料'], ['/admin/audit', '08', '操作记录'],
];
const managed = [...business, ...shop];

// 取出所有含 href 的对象字面量，读 href / index / label|title
function items(src) {
  return [...src.matchAll(/\{[^{}]*href: "([^"]+)"[^{}]*\}/g)].map((m) => ({
    href: m[1],
    index: (m[0].match(/index: "([^"]*)"/) || [, ''])[1],
    name: (m[0].match(/title: "([^"]*)"/) || m[0].match(/label: "([^"]*)"/) || [, ''])[1], // 卡片用 title，侧栏用 label
    at: m.index,
  }));
}
const shopDesk = shell.slice(shell.indexOf('const cashierGroup'), shell.indexOf('sales: {'));
const nav = items(shopDesk);
const cards = items(overviewSrc.slice(0, overviewSrc.indexOf('export default')));
const find = (list, href) => list.find((i) => i.href === href);

test('sidebar: 8 managed items 01-08 with spec names', () => {
  for (const [href, index, name] of managed) {
    const it = find(nav, href);
    assert.ok(it, `sidebar ${href}`);
    assert.equal(it.index, index, `sidebar index ${href}`);
    assert.equal(it.name, name, `sidebar name ${href}`);
  }
});

test('sidebar: 总览 and cashier group are unnumbered', () => {
  for (const href of ['/admin', '/counter', '/counter/orders']) {
    assert.equal(find(nav, href).index, '', `${href} must have no index`);
  }
  assert.equal(find(nav, '/counter').name, '现场开单');
  assert.equal(find(nav, '/counter/orders').name, '本场订单');
  assert.doesNotMatch(shell, /index: "(POS|ORD|00|09|10)"/);
});

test('sidebar: groups 经营(总览, 01-04) → 现场收银 → 店铺(05-08)', () => {
  const desk = shopDesk.slice(shopDesk.indexOf('groups:'));
  const pos = (s) => { const i = desk.indexOf(s); assert.ok(i >= 0, s); return i; };
  const order = [
    pos('label: "经营"'), pos('href: "/admin",'),
    ...business.map(([h]) => pos(`href: "${h}"`)),
    pos('cashierGroup'), pos('label: "店铺"'),
    ...shop.map(([h]) => pos(`href: "${h}"`)),
  ];
  assert.deepEqual(order, [...order].sort((x, y) => x - y));
  assert.match(shell, /const cashierGroup: NavGroup = \{\s*label: "现场收银"/); // C9：一个常量，不是工厂
});

test('overview: exactly the 8 managed cards, same index and name as sidebar', () => {
  assert.equal(cards.length, 8);
  assert.deepEqual(cards.map((c) => [c.href, c.index, c.name]), managed);
  assert.ok(!cards.some((c) => c.href.startsWith('/counter')), 'no cashier cards');
  assert.doesNotMatch(overviewSrc, /活动与场次|商品与套餐/);
});

test('overview: cards grouped 经营 01-04, 店铺 05-08', () => {
  const head = overviewSrc.slice(0, overviewSrc.indexOf('export default'));
  const b = head.indexOf('"经营"'); const s = head.indexOf('"店铺"');
  assert.ok(b >= 0 && s > b, 'group labels 经营 then 店铺');
  for (const c of cards) {
    const inBusiness = c.at > b && c.at < s;
    assert.equal(inBusiness, Number(c.index) <= 4, `${c.href} group`);
  }
});

test('C10 sidebar foot wraps back link and logout', () => {
  assert.match(shell, /className="workspace-sidebar__foot"/);
});

test('C15 counter H1 is 现场开单 (spec Q5 = A, Kel decided 13:19)', () => {
  const counter = fs.readFileSync('src/app/counter/page.tsx', 'utf8');
  assert.match(counter, /<h1>现场开单<\/h1>/);
});
```
> 分组的数据结构（例如 `const groups = [{ label: "经营", items: [...] }, { label: "店铺", items: [...] }]`）由前端定；测试只要求组名字符串“经营”在“店铺”前、卡片按编号落在对应组里。若前端把组名写成别的形式（例如 JSX 里的 `<h2>经营</h2>`），按实际调整 `indexOf` 的写法，判定逻辑不变。

- [ ] **步骤 2：** `node --test tests/workspace-nav.test.cjs`；按 12:45 磁盘，预期侧栏三条通过；总览两条（01/02 标题、分组）、C10、C15 失败。
- [ ] **步骤 3：实现**：`admin/page.tsx` 把 01、02 标题改为“活动”“商品”并分成“经营”“店铺”两组（描述按原型）；C10 加 `workspace-sidebar__foot`；E11、S10 不改（保持 `display: none`）；C15 把 H1 改成“现场开单”，页面 `<title>` 同步（spec 13:30 C15，原 Q5，Kel 已定）。
- [ ] **步骤 4：** 通用验证；`workspace-copy.test.cjs` 里卡片相关断言如写死“活动与场次”等旧标题，同步改成新名称。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/admin/page.tsx`、`web/src/components/workspace-shell.tsx`、`web/src/app/counter/page.tsx`、相关 CSS、`web/tests/workspace-nav.test.cjs`、`web/tests/workspace-copy.test.cjs`

**验收：** spec §9 第 2、3 条（12:44 版）；总览和侧栏对照上表逐项一致；对照 UIUX 12:44 重新渲染的原型和 12 张截图（包括 `展场-总览-桌面.png`、`店面-总览-桌面.png`）。

---

#### 任务 W-C3：classic 交互控件边框改用 `--line-control`（对比度 ≥3:1；Kel 2026-11-26 14:13 采纳）

> **端：** web ｜ **优先级：** P1（WCAG 2.2 SC 1.4.11 不达标：现在 `#c6bdaa` 只有 1.58–1.86:1） ｜ **依赖：** 无（不依赖后端）；建议在 W-C1 之后做（同改 `globals.css` `:root`，避免冲突） ｜ **依据：** `docs/design-v2/classic-边框对比度-建议.md`（UIUX，13:37）§1、§2、§3.1、§3.2、§4、附录；**Kel 2026-11-26 14:13 选 A 采纳，附录 5 个用 `--line` 的控件一起改** ｜ **前端已改（14:26 只读核对 Mac 磁盘，未提交，待前端回报）：** 看起来已经按本任务做了大部分——`globals.css:55` 是 `#8f8675`，两个深色块有 `#676c78`，`var(--line-control)` 共 48 处，`admin/goods/page.tsx:61` 已改，5 个主题块（含 `.product-page`）都加了 `--line-control: var(--line-strong)`，`web/tests/classic-line-control.test.cjs` 已存在；**E28：** 14:42 再核对，第 407、427、470 行（现第 439、459、502 行）3 处控件已改 `var(--line-control)`；第 358 行（现第 390 行，卡片悬停）没改，**Kel 14:40 定就保持 `var(--line-strong)`**，前端不用再动 ｜ **状态：** 已批准开工

**目标：** classic 主题下，所有靠边框识别的交互控件边框和相邻背景的对比度 ≥3:1（浅色、深色都要）；版面分割线、装饰框、hover、禁用态保持原色；shop、exhibition（顾客端和工作台）两套主题的控件边框颜色**完全不变**。

**改动范围（2026-11-26 14:1x 在 Mac 上 grep `web/src` 核对；建议文档的行号基于 13:33 快照，现在整体后移 24–31 行，以下面“现行号”为准；现行号是声明所在行，多行规则的选择器行会早几行）：**

1. token（`web/src/app/globals.css`）：
   - `:55` `--line-control: var(--line-strong);` → `--line-control: #8f8675;`（`--line-strong: #c6bdaa` 第 22 行**不改**）。
   - 深色两个 token 块：`:1311`（`@media (prefers-color-scheme: dark)` 里的 `:root:not(...)`）和 `:1394`（`:root:has(> head > meta[name="color-scheme"][content="dark"])`）在 `--line-strong: #40444f;` 后各加 `--line-control: #676c78;`（`#40444f` 本身不改）。
2. 选择器 `var(--line-strong)` / `var(--line)` → `var(--line-control)`，**共 48 处声明**：建议文档 §3.1 交互控件 39 处（浅色 33 + 深色 3 组 × 2 块）、§3.2 建议同步 4 处、附录用 `--line` 的 5 处。
   - **和 UIUX 口径（“40 处 + 附录 5 处”，spec C17、建议文档 §4 第 2 条）的对应：** UIUX 的 40 = §3.1 的 39 处 CSS + `admin/goods/page.tsx:61` 内联 1 处；附录 5 处相同。本任务的“48 处声明 + 1 处内联” = §3.1 39 + §3.2 4 + 附录 5 = 48，再加同一处内联 1。**差别只在 §3.2 的 4 处**（`.goods-event-option`、`.counter-product--bundle`、`.workspace-shell .stock-meter__bar`、`.text-link`），建议文档写“视情况一起改”，本任务按父代理 14:13 的范围（39+4+5）一起改。合计：UIUX 45 处 = 本任务 49 处 − §3.2 4 处。
   - 逐条：

| 来源 | 文档行号 | 现行号 | 选择器 |
|---|---|---|---|
| 3.1 | 190 | 214 | `.button--outline` |
| 3.1 | 422 | 446 | `.field input` |
| 3.1 | 423 | 447 | `.field textarea` |
| 3.1 | 1137 | 1165 | `.field select` |
| 3.1 | 500 | 525 | `.workspace-toolbar button:not(.button)` |
| 3.1 | 508 | 533 | `.workspace-page input:not([type="checkbox"]):not([type="file"]), .workspace-page select, .workspace-page textarea` |
| 3.1 | 523 | 548 | `.workspace-page form button.button--outline, .workspace-page form .select-menu__trigger, .workspace-page form .counter-sheet__close, .workspace-page form .goods-modal__close` |
| 3.1 | 527 | 552 | `.workspace-page li button` |
| 3.1 | 581 | 606 | `.pay-code-form input[type="file"]` |
| 3.1 | 2198 | 2229 | `.workspace-page input[type="file"]` |
| 3.1 | 761 | 789 | `.tier-editor__remove, .tier-editor__add` |
| 3.1 | 1010 | 1038 | `.workspace-page form button.tier-editor__remove, .workspace-page form button.tier-editor__add` |
| 3.1 | 798 | 826 | `.goods-search` |
| 3.1 | 903 | 931 | `.counter-catalog-search input` |
| 3.1 | 1222 | 1250 | `.workspace-page form .counter-catalog-filter button` |
| 3.1 | 973 | 1001 | `.quantity-control` |
| 3.1 | 1122 | 1150 | `.header-account` |
| 3.1 | 1495 | 1523 | `.draft-gallery__upload` |
| 3.1 | 1502 | 1530 | `.image-picker` |
| 3.1 | 2054 | 2085 | `.counter-page .counter-sheet__close` |
| 3.1 | 2259 | 2290 | `.pay-dialog__close` |
| 3.1 | 2431 | 2462 | `.goods-view-tabs` |
| 3.1 | 2497 | 2528 | `.good-listing-editor__release` |
| 3.1 | 2558 | 2589 | `.bundle-existing-picker__head button` |
| 3.1 | 2574 | 2605 | `.workspace-page form .bundle-existing-picker__head button` |
| 3.1 | 2559 | 2590 | `.bundle-existing-picker input` |
| 3.1 | 2592 | 2623 | `.review-switch` |
| 3.1 | 2599 | 2630 | `.review-switch__track` |
| 3.1 | 2614 | 2645 | `.report-date input` |
| 3.1 | 2776 | 2812 | `.select-menu__trigger` |
| 3.1 | 2838 | 2874 | `.workspace-page .select-menu__trigger` |
| 3.1 | 2992 | 3028 | `.slug-input > span` |
| 3.1 | 3022 | 3058 | `.catalog__box` |
| 3.1 深色 | 1307、1390 | 1335、1418 | `& .button--outline`（两个深色块各一次） |
| 3.1 深色 | 1312、1395 | 1340、1423 | `& .workspace-page form button.button--outline, …select-menu__trigger, …counter-sheet__close, …goods-modal__close` |
| 3.1 深色 | 1322、1405 | 1350、1433 | `& .workspace-page form .counter-catalog-filter button` |
| 3.2 | 565 | 590 | `.goods-event-option` |
| 3.2 | 913 | 941 | `.counter-product--bundle` |
| 3.2 | 2312 | 2343 | `.workspace-shell .stock-meter__bar` |
| 3.2 | 192 | 216 | `.text-link` |
| 附录 | 790 | 818 | `.product-gallery__thumbs button` |
| 附录 | 834 | 862 | `.bundle-component-draft__images button` |
| 附录 | 859 | 887 | `.review-filter button` |
| 附录 | 905 | 933 | `.counter-catalog-filter button` |
| 附录 | 1012 | 1040 | `.workspace-page form button.counter-product` |

   - 表里同一条规则里只改 `border` / `border-color` / `text-decoration-color` 里的颜色，宽度、虚线样式不动。
   - **附录行 `:1040` 只改默认态**；`:1043`、`:1044`（`:disabled`、`--unavailable`）是禁用态，WCAG 豁免，保留 `var(--line)`。深色块 `:1376` 的 `:disabled` 描边同理保留。
3. **建议文档没列、grep 多出来的 1 处：** `web/src/app/admin/goods/page.tsx:61` 的内联 `pickerStyle`（商品页图片选择按钮，`minHeight: 44`）写的是 `border: "1px solid var(--line-strong)"`，属于交互控件，一起改成 `var(--line-control)`（建议文档只查了 CSS 文件）。
4. **保持其他主题不变（spec 14:25 C18；建议文档 §4 第 4 条）：** shop、exhibition 是在下层元素（`body:has(...)`、`[data-storefront-theme]`）或 `:root:has(...)` 上覆盖 `--line-strong` 的，`:root` 上的 `--line-control` 会把 classic 的值继承下去。所以每个重新定义 `--line-strong` 的块都要紧跟着加 `--line-control: var(--line-strong);`，在那一层重新取值。**2026-11-26 14:26 在 Mac 上 grep `--line-strong:` 核实（不含 `globals.css` 的 classic 三处）共 5 个块**，行号写“声明行（块起始行）”：

| 文件 | `--line-strong` 声明行（块起始行） | 块 | 现值 | UIUX C18 | 处理 |
|---|---|---|---|---|---|
| `web/src/app/shop.css` | 13（3） | `body:has(#storefront-shell[data-storefront-theme="shop"])` | `#ebebeb` | 列了（第 13 行） | 加 |
| `web/src/app/exhibition.css` | 32（23） | `[data-storefront-theme="exhibition"] :is(.exhibition-home, .ed-about, .ed-booth, .exhibition-event, .product-page, .auth-page)` | `#999281` | 列了（第 32 行） | 加 |
| `web/src/app/exhibition.css` | 930（921） | `:root:has(#storefront-shell[data-storefront-theme="exhibition"])` | `#999281` | 列了（第 928 行；前端在上面插了两行，现在是 930） | 加 |
| `web/src/app/exhibition-workspace.css` | 35（20，`@media screen` 里） | `[data-storefront-theme="exhibition"][data-surface="workspace"], body:has(> …workspace…) > .select-menu__panel` | `#8ca29d` | 列了（第 35 行） | 加 |
| `web/src/app/exhibition.css` | 547（540） | `[data-storefront-theme="exhibition"] .product-page`（classic token 覆盖块） | `#c6bdaa` | 列为“不要加” | ~~按条件~~ **15:0x 已被 W-E4 删掉，不要加回来**（见下面 K-Q19 段） |

   - 更正：13:55 版 W-C3 写的是块起始行（`shop.css:3`、`exhibition.css:23/919/538–546`、`exhibition-workspace.css:39`），UIUX 写的是声明行；其中 `exhibition-workspace.css:39` 是写错了（第 39 行起是另一个不含 token 的规则，token 块从第 20 行开始）。块本身和 UIUX C18 一致，差别只在 `.product-page` 块。
   - **`.product-page` 块的条件（已过时：W-E4 先合入，这块已删；现行规则见下面 K-Q19 段，不要加回来）：** 如果 W-C3 **先于** W-E4（E16 删掉这整块）合入，就给这块也加 `--line-control: var(--line-strong);`；如果 **E16 先合入**，这块已经没了，不用加。说明：第 23 行的展场块选择器里本来就含 `.product-page`，两块命中同一个元素、特异性相同，后写的第 540 行块赢，所以这个元素上的 `--line-strong` 是 `#c6bdaa`；`var(--line-control)` 在本元素上按本元素的 `--line-strong` 计算，**加或不加，商品页控件边框都是 `#c6bdaa`，和改动前一样**（UIUX 说的“加了会变浅”、13:55 版说的“不加会继承展场色”都不会发生）。按条件加只是让“每个重新定义 `--line-strong` 的块都定义 `--line-control`”这条测试规则不出例外。
   - **K-Q19 = A（Kel 2026-11-26 15:01 定，Grok Bot 转达）：三处控件边框都加深，分割线和装饰框保持原色。** 只改各主题块里的 `--line-control`，**`--line-strong` 一个都不动**。15:0x 在 Mac 上重新 grep 核实（W-E4 已把 `.product-page` 的 classic 覆盖块删掉，行号已变）：

     | 文件 | `--line-control` 声明行（块起始行，15:0x） | 块 | `--line-strong`（不动） | 改成 | 各背景对比度（≥3:1） |
     |---|---|---|---|---|---|
     | `web/src/app/shop.css` | 14（3） | `body:has(#storefront-shell[data-storefront-theme="shop"])` | `#dedfe0` | **`#8f8c8b`** | 白 3.34、canvas `#f2f4f5` 3.03 |
     | `web/src/app/exhibition.css` | token 在第 41–42 行（`--line-strong` 41、`--line-control` 42；块从第 32 行开始） | 展场顾客端页面块 `[data-storefront-theme="exhibition"] :is(.exhibition-home, .ed-about, .ed-booth, .exhibition-event, .product-page, .auth-page)` | `#999281` | **`#857e6e`** | canvas `#f4f0e6` 3.54、surface `#fcf9f0` 3.83、surface-soft `#e8e2d5` 3.12、白 4.03 |
     | `web/src/app/exhibition.css` | token 在第 884–885 行（`--line-strong` 884、`--line-control` 885；块从第 875 行开始） | 根元素块 `:root:has(#storefront-shell[data-storefront-theme="exhibition"])` | `#999281` | **`#857e6e`** | 同上 |
     | `web/src/app/exhibition-workspace.css` | 36（20–21，`@media screen` 里） | `[data-storefront-theme="exhibition"][data-surface="workspace"], body:has(> …) > .select-menu__panel` | `#8ca29d` | **`#6f8681`**（不是旧建议 `#738b86`） | 白 3.89、面板 `#f4faf6` 3.67、canvas `#edf3f1` 3.46、surface-soft `#e2ece8` 3.22、渐变最深处 `#d5e6df` 3.00 |

     对比度按 WCAG 相对亮度公式复算，和建议文档 §5.3 一致。**前端 Mac 现状（15:07 只读）：** 四处已经改成上表的值，`--line-strong` 没动；下面 K-Q19 测试我在盒子上用 15:07 的 CSS 副本跑过，通过，把店面值换回 `#ebebeb` 会失败。**商品页 `[data-storefront-theme="exhibition"] .product-page` 的 classic token 覆盖块已被 W-E4 删掉，不要加回来。** 如果这一块又出现（比如合并冲突带回来），里面也**不能**写 `--line-control: var(--line-strong);`：那块的 `--line-strong` 是 classic 的 `#c6bdaa`，商品页控件边框会退回 `#c6bdaa`，对比度只有约 1.64（UIUX 口径），不到 3:1。商品页应直接继承上面第 32 行展场块的 `#857e6e`（该块选择器本来就含 `.product-page`）。店面“浅灰填充加 `#8f8c8b` 描边”的备选**不采用**。行号会漂，实现和测试按选择器定位。
   - **展场工作台 E28（spec 14:25；Kel 14:40 改范围）：** `exhibition-workspace.css` 原来列了 4 处 `border-color: var(--line-strong)`（13:55 快照的第 358、407、427、470 行）。**Kel 2026-11-26 14:40 定：第 358 行保持 `var(--line-strong)`，不改。** 父代理 14:28 让前端把它也改成 `--line-control` 的那条指令作废。14:42 只读核对 Mac 磁盘（文件 14:38 改过，行号已后移），四处按选择器对应如下：

     | 13:55 行号 | 14:38 磁盘行号 | 选择器 | 是不是交互控件 | 怎么做 |
     |---|---|---|---|---|
     | 358 | 390 | `.workspace-shell .workspace-card:hover` | **不是**：总览卡片的悬停边框，属于建议文档 §3.3 的 hover 态，E28 自己也说卡片悬停这类非控件保留更深的现有边框 | **保持 `border-color: var(--line-strong)`**（Kel 14:40） |
     | 407 | 439 | `.workspace-page :is(.counter-catalog-filter button[data-active="true"], .review-filter button[data-active="true"])` | 是：收银台商品筛选、审核筛选按钮的选中态 | 改 `var(--line-control)`（磁盘上已改） |
     | 427 | 459 | `.counter-page form button.counter-product:has(.counter-product__count)` | 是：收银台已加购的商品按钮（`<button>`） | 改 `var(--line-control)`（磁盘上已改） |
     | 470 | 502 | `.workspace-shell .review-switch__track` | 是：审核开关的轨道 | 改 `var(--line-control)`（磁盘上已改） |

     行号会继续漂移，实现和测试一律按选择器定位，不按行号。K-Q19 = A 之后这 3 处控件跟着变成 `#6f8681`；卡片悬停边框不跟，仍是 `#8ca29d`。测试见下面 `E28` 用例：只允许 `.workspace-card:hover` 这一处留 `var(--line-strong)`。
5. 不动：建议文档 §3.3 的分割线、容器、状态胶囊、图片框、小图标、hover、禁用态；`exhibition.css:389` `.product-card__tag` 写死的 `#c6bdaa`（装饰标签）。

**文件：**
- 修改：`/Users/kelden/go/src/Stallink/web/src/app/globals.css`
- 修改：`/Users/kelden/go/src/Stallink/web/src/app/admin/goods/page.tsx`
- 修改：`/Users/kelden/go/src/Stallink/web/src/app/shop.css`
- 修改：`/Users/kelden/go/src/Stallink/web/src/app/exhibition.css`
- 修改：`/Users/kelden/go/src/Stallink/web/src/app/exhibition-workspace.css`
- 创建：`/Users/kelden/go/src/Stallink/web/tests/classic-line-control.test.cjs`

- [ ] **步骤 1：编写失败的测试**（照 `web/tests/theme-tokens.test.cjs` 的 `node:test` 写法）

```js
/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const read = (p) => fs.readFileSync(p, 'utf8');
const globals = read('src/app/globals.css');

// WCAG 相对亮度和对比度
const lum = (hex) => {
  const c = hex.replace('#', '').match(/../g).map((h) => parseInt(h, 16) / 255)
    .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4));
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
};
const ratio = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((m, n) => n - m); return (x + 0.05) / (y + 0.05); };

// 取出所有“选择器 { 声明 }”块（不处理嵌套 @media 的外层，只看最内层）
const blocks = (src) => [...src.matchAll(/([^{}]+)\{([^{}]*)\}/g)].map((m) => ({ sel: m[1].replace(/\/\*[\s\S]*?\*\//g, '').trim(), body: m[2] }));
const rule = (src, sel) => blocks(src).filter((b) => b.sel.split(',').map((s) => s.replace(/^&\s*/, '').trim()).includes(sel));

test('classic light: --line-control is #8f8675 and passes 3:1 on every classic surface', () => {
  assert.equal((globals.match(/--line-control:\s*#8f8675;/g) || []).length, 1, ':root defines --line-control #8f8675 once');
  assert.ok(globals.indexOf('--line-control: #8f8675') < globals.indexOf('prefers-color-scheme: dark'), 'defined in the light :root block');
  assert.match(globals, /--line-strong:\s*#c6bdaa;/, 'divider token unchanged');
  for (const bg of ['#faf8f3', '#fffdf8', '#ffffff', '#fffefa', '#f0ece2']) {
    assert.ok(ratio('#8f8675', bg) >= 3, `#8f8675 on ${bg}: ${ratio('#8f8675', bg).toFixed(2)}`);
  }
});

test('classic dark: both dark token blocks set --line-control #676c78 and pass 3:1', () => {
  assert.equal((globals.match(/--line-control:\s*#676c78;/g) || []).length, 2);
  assert.equal((globals.match(/--line-strong:\s*#40444f;/g) || []).length, 2, 'dark divider token unchanged');
  for (const bg of ['#0e1014', '#16181d', '#1e2129']) {
    assert.ok(ratio('#676c78', bg) >= 3, `#676c78 on ${bg}: ${ratio('#676c78', bg).toFixed(2)}`);
  }
});

const controls = [
  // §3.1（浅色 33）
  '.button--outline', '.field input', '.field textarea', '.field select', '.workspace-toolbar button:not(.button)',
  '.workspace-page select', '.workspace-page form .select-menu__trigger', '.workspace-page li button',
  '.pay-code-form input[type="file"]', '.workspace-page input[type="file"]', '.tier-editor__remove',
  '.workspace-page form button.tier-editor__remove', '.goods-search', '.counter-catalog-search input',
  '.workspace-page form .counter-catalog-filter button', '.quantity-control', '.header-account', '.draft-gallery__upload',
  '.image-picker', '.counter-page .counter-sheet__close', '.pay-dialog__close', '.goods-view-tabs',
  '.good-listing-editor__release', '.bundle-existing-picker__head button', '.workspace-page form .bundle-existing-picker__head button',
  '.bundle-existing-picker input', '.review-switch', '.review-switch__track', '.report-date input', '.select-menu__trigger',
  '.workspace-page .select-menu__trigger', '.slug-input > span', '.catalog__box',
  // §3.2
  '.goods-event-option', '.counter-product--bundle', '.workspace-shell .stock-meter__bar', '.text-link',
  // 附录（--line）
  '.product-gallery__thumbs button', '.bundle-component-draft__images button', '.review-filter button',
  '.counter-catalog-filter button', '.workspace-page form button.counter-product',
];

test('every listed control uses var(--line-control), none keeps --line-strong / --line on its border', () => {
  for (const sel of controls) {
    const hits = rule(globals, sel).filter((b) => /border|text-decoration-color/.test(b.body));
    assert.ok(hits.length, `rule not found: ${sel}`);
    assert.ok(hits.some((b) => /var\(--line-control\)/.test(b.body)), `${sel} should use --line-control`);
    for (const b of hits) {
      assert.doesNotMatch(b.body, /(border(-color)?|text-decoration-color):[^;]*var\(--line(-strong)?\)/, `${sel} still uses --line/--line-strong`);
    }
  }
});

test('dark blocks: 3 control groups use --line-control in both dark blocks (6)', () => {
  const dark = blocks(globals).filter((b) => /^&\s*\.(button--outline|workspace-page form)/.test(b.sel) && /border-color:\s*var\(--line-control\)/.test(b.body));
  assert.ok(dark.length >= 6, `dark control rules using --line-control: ${dark.length}`);
});

test('dividers, disabled and hover keep their original tokens', () => {
  for (const [sel, token] of [['.section-heading', '--line-strong'], ['.report-hero', '--line-strong'], ['.order-status', '--line-strong'],
    ['.confirm-dialog', '--line-strong'], ['.event-card:hover', '--line-strong'], ['.workspace-page form button.counter-product--unavailable', '--line']]) {
    assert.ok(rule(globals, sel).some((b) => b.body.includes(`var(${token})`)), `${sel} must keep ${token}`);
  }
});

test('goods page inline picker border uses --line-control', () => {
  assert.match(read('src/app/admin/goods/page.tsx'), /pickerStyle[^;]*border:\s*"1px solid var\(--line-control\)"/);
});

test('E28: exhibition workspace controls read --line-control; card hover keeps --line-strong (Kel 14:40)', () => {
  const ws = read('src/app/exhibition-workspace.css');
  const strong = blocks(ws).filter((b) => /border-color:\s*var\(--line-strong\)/.test(b.body)).map((b) => b.sel.trim());
  // 唯一例外：13:55 快照第 358 行的卡片悬停边框，不是控件，Kel 14:40 定保持 --line-strong
  assert.deepEqual(strong, ['.workspace-shell .workspace-card:hover'], 'only the card hover border may stay on --line-strong');
  for (const sel of [
    '.counter-catalog-filter button[data-active="true"]',
    'button.counter-product:has(.counter-product__count)',
    '.workspace-shell .review-switch__track',
  ]) {
    const b = blocks(ws).find((x) => x.sel.includes(sel));
    assert.ok(b, `missing block ${sel}`);
    assert.match(b.body, /border-color:\s*var\(--line-control\)/, `${sel} should use --line-control`);
  }
  assert.match(ws, /--line-control:\s*#6f8681/, 'K-Q19 = A (Kel 15:01): workspace control border #6f8681');
});

test('K-Q19 = A (Kel 15:01): theme control borders darkened, --line-strong untouched, all >= 3:1', () => {
  const cases = [
    ['src/app/shop.css', 'data-storefront-theme="shop"])', '#8f8c8b', '#dedfe0', ['#ffffff', '#f2f4f5']],
    ['src/app/exhibition.css', '.exhibition-home', '#857e6e', '#999281', ['#f4f0e6', '#fcf9f0', '#e8e2d5', '#ffffff']],
    ['src/app/exhibition.css', ':root:has(#storefront-shell[data-storefront-theme="exhibition"])', '#857e6e', '#999281', ['#f4f0e6', '#fcf9f0', '#e8e2d5', '#ffffff']],
    ['src/app/exhibition-workspace.css', '[data-surface="workspace"]', '#6f8681', '#8ca29d', ['#ffffff', '#f4faf6', '#edf3f1', '#e2ece8', '#d5e6df']],
  ];
  for (const [file, selPart, control, strong, bgs] of cases) {
    const b = blocks(read(file)).find((x) => x.sel.includes(selPart) && /--line-strong:/.test(x.body));
    assert.ok(b, `${file}: token block containing ${selPart}`);
    assert.match(b.body, new RegExp(`--line-control:\\s*${control}`), `${file}: --line-control ${control}`);
    assert.match(b.body, new RegExp(`--line-strong:\\s*${strong}`), `${file}: --line-strong stays ${strong} (dividers unchanged)`);
    for (const bg of bgs) assert.ok(ratio(control, bg) >= 3, `${control} on ${bg}: ${ratio(control, bg).toFixed(2)}`);
  }
  assert.doesNotMatch(read('src/app/exhibition.css'), /\.product-page\s*\{[^}]*--line-strong:/, '.product-page classic override stays deleted');
});

test('other themes unchanged: every block that redefines --line-strong also defines --line-control', () => {
  for (const file of ['src/app/shop.css', 'src/app/exhibition.css', 'src/app/exhibition-workspace.css']) {
    for (const b of blocks(read(file)).filter((x) => /--line-strong:/.test(x.body))) {
      assert.match(b.body, /--line-control:/, `${file} block "${b.sel.slice(0, 60)}" must define --line-control`);
    }
  }
});
```

- [ ] **步骤 2：运行测试，确认失败**：`cd /Users/kelden/go/src/Stallink/web && node --test tests/classic-line-control.test.cjs`。预期：除“dividers…keep”一条外都失败。
- [ ] **步骤 3：实现**：按上面改动范围第 1–4 条改。`exhibition-workspace.css` 第 34–35 行在嵌套块里，`blocks()` 只看最内层，所以 `--line-control` 要加在和 `--line-strong` 同一个最内层块里。
- [ ] **步骤 4：运行测试，确认通过**：`node --test tests/*.cjs` 全绿（含 `theme-tokens.test.cjs`）；`npm run lint`、`npx tsc --noEmit` 无新增问题。
- [ ] **步骤 5：浏览器验收**（本机 `npm run dev`，**后端只连本机 docker-compose 的 Postgres**）：
  1. classic 浅色、classic 深色（主题开关切深色）各打开 `/admin/goods`（商品编辑，含图片选择、阶梯价增删、搜索框）、`/counter`（搜索、筛选、商品卡、数量步进、结算抽屉关闭按钮）、`/admin/reports`（日期输入、下拉）、`/admin/shop`（店铺链接前缀）、`/platform/goods`（审核开关、审核筛选）、店铺首页（作品搜索框、顶栏胶囊链接）。用 DevTools 的对比度检查或取色计算：控件边框和它外侧背景 ≥3:1。
  2. 版面分割线（`.section-heading`、`.report-hero`、表格线）颜色不变。
  3. **其他主题不变**：切到 shop、exhibition（顾客端和工作台）各打开同一批页面，在改动前后各跑一次 `getComputedStyle(el).borderTopColor`（`.field input`、`.button--outline`、`.workspace-page select`、`.select-menu__trigger`、`.counter-catalog-search input`），两次结果完全一样。
- [ ] **步骤 6：只暂存，不 commit、不 push**：上面“文件”列的 6 个文件。

**验收：** WCAG 2.2 SC 1.4.11 非文本对比度——classic 浅色下控件边框 `#8f8675` 在 canvas `#faf8f3` 3.39、surface `#fffdf8` 3.54、白 3.60、顶栏 `#fffefa` 3.57、surface-soft `#f0ece2` 3.05，深色下 `#676c78` 在 `#0e1014` 3.62、`#16181d` 3.38、`#1e2129` 3.06，**都 ≥3:1**；48 处 CSS 声明 + 1 处内联样式都改用 `--line-control`；`--line-strong`（`#c6bdaa` / `#40444f`）和 §3.3 的分割线、hover、禁用态不变；shop、exhibition 两套主题控件边框计算值前后完全一致。后续通知 UIUX：展场/店面 spec §4.2 classic 一列的 `--line-control` 改成 `#8f8675`（建议文档 §4 第 5 条）。

---

#### 任务 W3：classic 收银台商品名一字一行

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** 无 ｜ **来源：** UIUX P1 #3（截图 `classic-1280-counter.png`）。展场、店面的收银台分别在 W-E3（E15）和 W-S4（S14） ｜ **前端已改：** 前端 12:21 回报“商品名换行”已做完、自测通过 ｜ **状态：** 前端已完成，待复测（下面的步骤只补防回退测试和复测）

- [ ] **步骤 1：编写失败的测试**（加进 `web/tests/report-layout.test.cjs`，见 W16）：断言 `globals.css` 里 `.counter-product` 名称相关规则不含 `overflow-wrap: anywhere`，且含 `overflow-wrap: break-word`。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：名称列 `min-width: 0; overflow-wrap: break-word; word-break: normal`，价格列 `white-space: nowrap`。只改收银台商品名相关的规则，其余 18 处不顺手改。
- [ ] **步骤 4：** 通用验证；classic 1280 收银台截图里“亚克力立牌·海雾”一行显示。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/globals.css`、`web/tests/report-layout.test.cjs`

---

#### 任务 W6：D3 classic 390 已登录页眉溢出 8px

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 无 ｜ **来源：** 测试报告 D3（不在 spec 改动清单里） ｜ **前端已改：** D3 已做完、自测通过（`globals.css:3065` classic `.site-header__actions` 加了 `flex-wrap: wrap`） ｜ **状态：** 前端已完成，待复测

- [ ] 用 `layout-sweep.cjs` 跑 classic 390 已登录的 `/`、`/login`、`/c/demo`、`/c/demo/listings/500`、`/about`，`scrollWidth` 必须是 390，“退出”按钮完整可见。
- [ ] 通过就只补一条 CSS 断言到 `theme-tokens.test.cjs`；没通过再修。
- [ ] 只暂存，不 commit、不 push：`web/src/app/globals.css`、`web/tests/theme-tokens.test.cjs`

---

#### 任务 W10：订单列表去掉“实收 · 找零”

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** 展示部分无；PDF“未录实收”依赖 S10（K-Q18 = A，Kel 15:01）；请求和类型的清理依赖 S6（**K-Q10 = A，Kel 已定（2026-11-26 13:22）**）；日报“现金长短款”隐藏（**K-Q7 = B（13:22）；C-2 = A（13:55）：CSV、分币种汇总、指标卡三处全部隐藏**） ｜ **前端已改：** 订单列表去掉实收和找零已做完、自测通过 ｜ **状态：** 展示部分已完成，待复测；请求清理和日报隐藏待开工（和 S6 同批上线）

- [ ] 补测试：`web/tests/order-pages.test.cjs`（已有，前端本轮改过）断言 `src/app/counter/orders/page.tsx`、`src/app/admin/orders/page.tsx` 不含“实收”“找零”。
- [ ] **请求清理（K-Q10 = A，前后端一起清理）**：
  - 测试（`web/tests/order-actions.test.cjs`）：`payCommandBody(cashCode, 1200)` 和 `payCommandBody(qrCode, 1200)` 都只返回 `{ payCodeId }`，不含 `cashReceivedMinor`。
  - 实现：删掉 `order-actions.ts:44-45` 里的 `cashReceivedMinor`；删掉 `api.ts:161` 类型里的 `cashReceivedMinor`、`cashChangeMinor`；`counter/page.tsx:260、295、667` 待提交命令里不再带这个字段。
  - `errors.ts:16` 的 `invalid_cash_received` 文案：S6 上线后后端不再返回这个错误码，**和 S6 同一次上线时删除**；如果前端先上线，保留到 S6 上线。
  - **幂等一次切换（后端 B-9）**：收款幂等哈希按整个请求体算。同一个幂等键重发**原来的请求体**会正常回放；只有同一个键前后两次请求体不同（一次带字段、一次不带）才会 `idempotency_conflict`。所以前端一次切换，不做“重试时去掉字段”；上线说明写“遇到冲突提示就刷新后重新收款”。
- [ ] **日报“现金长短款”隐藏（K-Q7 = B、C-2 = A）**：CSV 那一处由后端 S7 负责，这里只管屏幕；
  - 测试（`web/tests/report-pages.test.cjs` 或现有日报测试）：`admin/reports/page.tsx` 不再渲染“现金长短款”（第 377–378 行指标、第 392 行分币种列）。
  - **PDF 仍显示长短款（Kel 14:21 再确认）**：`lib/reports/document-model.ts` 第 84、97 行指标里的“现金长短款”和第 109 行 `summaryFor` 分币种汇总里的 `overShortLabel(...)` 单元格都**保留不改**；PDF 由浏览器用 pdfmake 生成（`lib/reports/pdf/export-report-pdf.ts`、`pdf-definition.ts`），不走服务端。屏幕隐藏只改 `admin/reports/page.tsx`，不许动 `document-model.ts` 里的长短款。
  - **PDF 回归测试**：`web/tests/report-pdf.test.cjs`（已有）第 93、133 行断言 PDF 指标标签含“现金长短款（MYR）”“现金长短款（SGD）”，保留；再加一条：分币种汇总行的单元格含 `overShortLabel` 的输出（fixture `cashOverShortMinor: -50` → 含“短款”）。跑 `cd /Users/kelden/go/src/Stallink/web && node --test tests/report-pdf.test.cjs tests/report-layout.test.cjs`：PDF 测试断言有长短款，页面测试断言没有长短款，两边同时通过。
  - 原 K-Q7 B 原文是“隐藏，历史数据只在 PDF 里保留”。后端继续返回 `cashOverShortMinor`（见 S6），新订单不再记实收，所以它只反映历史现金单。
  - `report-types.ts:5、13` 的字段保留（PDF 还要用）。
- [ ] **K-Q18 = A：PDF 长短款区分“持平”和“未录实收”（Kel 15:01；后端字段在 S10）**（**前端 Mac 现状 15:07：** `document-model.ts:53-56` 已加第三个参数，`=== 0` 返回「—（未录实收）」，第 87、100、112 行已传计数，`report-types.ts:5、13` 已加可选字段；`report-pdf.test.cjs` 的新用例待前端回报确认）——这一项**解除“不许动 `document-model.ts` 长短款”的限制**，只改显示逻辑，长短款标签和位置不变。
  **接口约定（K-Q18 = A，前后端共用）：**
  - 字段：日报快照根对象 `cashTenderedOrders`（主币种）和 `currencies[i].cashTenderedOrders`（该币种），整数，≥0，**总是输出**（Go 不加 `omitempty`）。
  - 含义：当天（报表时区）、本店、该币种下 `cash_received_minor IS NOT NULL` 的订单数，口径和 `cashOverShortMinor` 完全一样（同一组 `WHERE`、同一个 `FILTER`），只是计数。
  - 前端判断：
    - `cashTenderedOrders === 0` → 长短款显示「—（未录实收）」，不显示“{金额} 持平”；
    - `cashTenderedOrders > 0` → 照常显示长款 / 短款 / 持平（`cashOverShortMinor`）；
    - **字段不存在**（`daily_reports.payload` 里 K-Q18 之前存下的旧快照）→ 按旧逻辑显示数值，**不当成 0**。后端 15:07 的工作区（未提交）在读取时用订单现算的计数把缺的键补上（`fillMissingCashTenderedOrders`、`cashTenderedOrdersByCurrency`，`GetDailyReport` 多了时区参数），所以接口正常情况下总会带这个字段；前端的“缺字段照旧显示”保留作兜底。两边不冲突。
  - 按币种（**K-Q22 = A，Kel 15:15 定：每种币各自判断**，§11.13；对应销售日报设计说明 §10 第 13 条（已关闭，第 540–546 行）、§4.6 第 198–210、220–221 行）：
    - 分币种汇总每一行用 `currencies[]` 里**该行自己**的 `cashTenderedOrders` 判断；`zeroRow()` 补出来的空币种行计数是 0，显示「—（未录实收）」。
    - 指标卡用根对象的 `cashTenderedOrders`，**根对象只代表主币种**（`report_store.go` `applyHeadlineCurrency` 约第 313 行把主币种行复制到根），不是整天所有币种合计；筛了非主币种时用该币种行。
    - 前端 `document-model.ts` 现在（15:07）就是这样做的（第 87、100、112 行传所在对象的计数）。
    - 设计说明 15:17 版已按 K-Q22 = A 改好（§4.6 第 198–210、220–221 行；第 13 条已关闭，第 540–546 行；开放问题全部关闭）。
  - 不改数据库表、不加迁移；CSV 不输出这个字段（长短款在 CSV 里本来就隐藏，C-2 = A）。
  - **测试先写**（追加到 `/Users/kelden/go/src/Stallink/web/tests/report-pdf.test.cjs`）：
    - `overShortLabel('MYR', 0, 0)` → `—（未录实收）`；`overShortLabel('MYR', 0, 2)` → 含“持平”；`overShortLabel('MYR', -50, 1)` → 含“短款”；`overShortLabel('MYR', 0, undefined)` → 含“持平”（旧快照）。
    - fixture 根 `cashTenderedOrders: 0`、`currencies` 里 MYR `cashTenderedOrders: 0`、SGD `cashTenderedOrders: 1, cashOverShortMinor: -50`：PDF 指标“现金长短款（MYR）”的值是 `—（未录实收）`；分币种汇总 MYR 行长短款单元格是 `—（未录实收）`，SGD 行含“短款”。
    - 原有断言“现金长短款（MYR）”“现金长短款（SGD）”标签和 W10 上一条“分币种汇总含短款”继续通过。
  - **实现**：`/Users/kelden/go/src/Stallink/web/src/lib/reports/document-model.ts` 的 `overShortLabel(currency, minor, tendered?)` 加第三个参数，`tendered === 0` 返回 `—（未录实收）`；第 84、97 行指标和第 109 行 `summaryFor` 传入对应对象（根 / 该币种行）的 `cashTenderedOrders`；`zeroRow()`（第 71 行）补 `cashTenderedOrders: 0`；`/Users/kelden/go/src/Stallink/web/src/lib/reports/report-types.ts` 第 5、13 行加可选字段 `cashTenderedOrders?: number`。指标注释“正数长款 · 负数短款”不变。屏幕页面不受影响（长短款屏幕上本来就隐藏）。
  - **备注：** K-Q22 = A（Kel 15:15），按币种各自判断，前端现在的写法就是这样，不用相加；测试 fixture 里“MYR 计数 0、SGD 计数 1”时 MYR 行显示「—（未录实收）」、SGD 行照常显示。
  - **导出非主币种时的口径（产品经理对齐，K-Q22 = A 的直接推论，不是 Kel 新定的）：** 指标卡的长短款金额取该币种那一行的 `cashOverShortMinor`，「—（未录实收）」也用**同一行**的 `cashTenderedOrders` 判断，两边口径一致；选「全部币种」或主币种时金额和判断都用根对象；分币种汇总每一行都用它自己那一行。**字段缺失的旧快照不当成 0**，仍显示长款、短款或「RM 0.00 持平」。前端 `document-model.ts` 第 53–59 行（`overShortLabel`）、第 87 行（根对象）、第 100 行（非主币种行）、第 112 行（`summaryFor` 每行）已经这样做，**不用改**；`report-pdf.test.cjs` 补一条“导出 SGD（非主币种）且 SGD 行 `cashTenderedOrders: 0` → 指标卡显示「—（未录实收）」；SGD 行有计数 → 显示该行的长款 / 短款 / 持平”。依据：销售日报设计说明（15:17 版）§4.6 第 198–210 行、第 220–221 行。
  - **验证**：`cd /Users/kelden/go/src/Stallink/web && node --test tests/report-pdf.test.cjs tests/report-layout.test.cjs`；后端 S10 没合入时用 fixture 测，接上后端后用本机 API 导一份当天 PDF 看到「—（未录实收）」，再导一份有历史实收的日期看到长款 / 短款 / 持平。
  - **只暂存，不 commit、不 push**：`web/src/lib/reports/document-model.ts`、`web/src/lib/reports/report-types.ts`、`web/tests/report-pdf.test.cjs`。
- [ ] 只暂存，不 commit、不 push：`web/tests/order-pages.test.cjs`、`web/tests/order-actions.test.cjs`、`web/src/lib/order-actions.ts`、`web/src/lib/api.ts`、`web/src/app/counter/page.tsx`、`web/src/lib/errors.ts`、`web/src/app/admin/reports/page.tsx`、日报测试文件

---

### 8.3 销售日报（设计说明 v2 + spec C12–C14 + D1、D5）

> 日报的更新稿已交付：`docs/design-v2/销售日报-设计说明.md`、`docs/design-v2/销售日报-原型.html`、`docs/design-v2/screens/销售日报-v2-{桌面,平板,手机,空日,加载中,出错,手机-展场}.png`。**P0 的 D1（展场 390 宽溢出）属于指标卡网格这一项**：前端已修好（F-5），W2 负责复测并补做桌面 3×2，和 W16 放在同一组。

#### 任务 W2：D1 复测 + 指标卡改成 5 张（长短款隐藏，C-2 = A）

> **端：** web ｜ **优先级：** P0（D1 复测）/ P1（桌面 3×2） ｜ **依赖：** 和 W16 同组 ｜ **设计：** 设计说明 §4.4 指标卡、§6 三档适配；测试报告 D1；UIUX P0 #1 ｜ **状态：D1 已修，待复测；桌面 3×2 待做**

**前端已改（Stallink-前端开发 12:21 回复 F-5，12:22 磁盘核对）：**
- 指标卡是 6 张（`reports/page.tsx:351-376`）。
- **D1 溢出已经修好**：≤900 宽 `--workspace-rail: 0px`（`exhibition-workspace.css:582`，展场 390 不再保留侧栏轨道）；`.report-metric` 加了 `min-width: 0`（`globals.css:2601`、`:2707`），数值改用 `clamp` 字号（`:2602`、`:2710`）；`white-space: nowrap` 是**有意保留**的，避免金额折行；390 宽实测已经不再横向溢出。
- ≤720 宽已经是 2×3（`globals.css:2705-2706` `.report-hero { grid-template-columns: 1fr 1fr; }`）。
- **桌面仍是 4+2**（`globals.css:2597` `.report-hero { grid-template-columns: 1.6fr 1fr 1fr 1fr; }`），设计说明 §4.4 要的 3×2 还没做。

**和设计说明的差异：** 设计说明 §4.4 写“数值允许在格内换行，但优先缩小字号”。前端保留 `nowrap`、只靠 `clamp` 缩字号，只要三档都不溢出就符合“优先缩小字号”。最长的金额（如“短款 S$ 1,280.50”）在 390 宽是否完整可见，放进步骤 4 的复测；如果被截断，再把 `nowrap` 改成允许换行。

**文件：** `web/src/app/globals.css:2597`（桌面列模板）、新建 `web/tests/report-layout.test.cjs`

- [ ] **步骤 1：编写失败的测试**

```js
/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const globals = fs.readFileSync('src/app/globals.css', 'utf8');
const expo = fs.readFileSync('src/app/exhibition-workspace.css', 'utf8');
const page = fs.readFileSync('src/app/admin/reports/page.tsx', 'utf8');

test('five metric cards, no 现金长短款 (C-2 = A, Kel 13:55)', () => {
  assert.equal((page.match(/className="report-metric( report-metric--lead)?"/g) || []).length, 5);
  assert.doesNotMatch(page, /现金长短款|长短款/);
});

test('D1 guards stay in place (already fixed, prevent regression)', () => {
  assert.match(expo, /--workspace-rail:\s*0px/);
  assert.match(globals, /\.report-metric\s*{\s*min-width:\s*0/);
  assert.match(globals, /\.report-metric strong\s*{[^}]*font-size:\s*clamp\(/);
});

test('narrow metric grid: 2 columns, lead card spans both (already 2 columns)', () => {
  assert.match(globals, /@media \(max-width: 720px\)\s*{\s*\.report-hero\s*{\s*grid-template-columns:\s*1fr 1fr/);
  assert.match(globals, /\.report-metric--lead\s*{[^}]*grid-column:\s*span 2/);
});

test('desktop metric grid is 3 columns, lead spans 2 → 2+1 / 3 (to do)', () => {
  assert.match(globals, /\.report-hero\s*{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\)/);
});
```

- [ ] **步骤 2：** `node --test tests/report-layout.test.cjs`；预期只有最后一条失败（前三条锁住前端已做的修复）。
- [ ] **步骤 3：实现**：先删掉**屏幕上**的“现金长短款”指标卡（C-2 = A，和 W10 的日报隐藏是同一处改动，谁先做谁改；**只改 `admin/reports/page.tsx`，PDF 的 `document-model.ts` 不动，Kel 14:21 定 PDF 保留**）；把 `globals.css:2597` 改成 `repeat(3, minmax(0, 1fr))`，收款总额卡 `.report-metric--lead { grid-column: span 2; }`，桌面排成“收款总额 + 1 / 3 张”，≤720 宽排成“收款总额整行 + 2×2”，5 张卡都不留空格。`globals.css:2844` 那条 `1.45fr 1fr 1fr 1fr` 覆盖一起改掉；第二行的左边框（`:nth-child`）按 3 列重排。
- [ ] **步骤 4：** 通用验证；`layout-sweep.cjs` 跑三套风格 × 390/768/1280，`/admin/reports` 的 `scrollWidth` 都等于视口宽度（**D1 复测**：展场 390 原来是 460）；用最长金额的 fixture 确认数值不被截断；对照 `销售日报-v2-桌面.png`、`-平板.png`、`-手机.png`、`-手机-展场.png`。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/globals.css`、`web/tests/report-layout.test.cjs`

**验收：** 测试报告 D1 复测关闭；页面上没有“长短款”；1280 宽两行（2+1 / 3），768、390 宽三行（收款总额整行 / 2 / 2）；三档都没有横向滚动。设计说明 §4.4 原来画的是 6 张卡 3×2，C-2 = A 后少一张，这个排法已通知 UIUX 同步设计稿（不阻塞开工）。

---

#### 任务 W16：销售日报按 v2 设计稿：标题、工具栏、热销四列、分币种列模板

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** U1（已交付）；W2；**K-Q11 = A（R1）、K-Q13 = A（R3），Kel 已定 2026-11-26 13:19**；设计说明问题 2、11 **Kel 13:32 已定：维持现状、用文案说明**；分币种汇总不显示“长短款”列（**C-2 = A，Kel 已定 13:55**） ｜ **设计：** 设计说明 §3 信息层级、§4.3 工具栏、§4.5 分区表格、§5 状态一览、§6 三档适配；spec §8.1 **C12–C14**；测试报告 D5；UIUX P1 #2、#4 ｜ **状态：** 部分完成

**前端已改（12:12 核实）：**
- H1 已是“销售日报”（`reports/page.tsx:228`，`tests/workspace-copy.test.cjs` 已断言）；**C12 要求的“经营”眉标还没加**。
- **D5 前端已完成**：工具栏拆成两个 `.report-toolbar__group`（`page.tsx:285` 生成组、`:292` 导出组；`reports.css:117`）。前端用 fixture 量过（F-4）：1280、860 宽两组排成一行，手机上叠成两行，符合设计说明 §4.3；**还没在真实登录状态下截图**，放进 T1。
- 热销商品名换行已修（`.report-rank span` 改成 `overflow-wrap: break-word`，`globals.css:2628`），但仍是 3 个单元格、序号写在名称里（`page.tsx:428-430`：`{index + 1}. {good.name}`、`{good.quantity} 件`、金额），**不是四列**。

**目标：**
1. **标题**：`<p className="eyebrow">经营</p>` + `<h1>销售日报</h1>`（C12）。
2. **工具栏**（设计说明 §4.3、spec C13）：左边日期；右边“生成”“导出（币种、CSV、PDF）”两组，组间距大于组内间距，可加 1px 竖线；**1280 宽排成一行**，不用“视口 ≥ 70rem 才 nowrap”这种条件；390 宽按组换行，按钮 ≥44px；币种下拉不 `width: 100%`；未生成时隐藏导出组，查询失败时只留“重试”。
3. **热销商品四列**（设计说明 §4.5、spec C14）：序号、名称、件数、金额各一列；名称列 `minmax(0, 1fr)`、`overflow-wrap: break-word`、`word-break: normal`，件数和金额 `white-space: nowrap`。
4. **分币种汇总**（设计说明 §4.5，R3；C-2 = A）：单独定义**五列**模板，列顺序为币种、订单（含已收款）、优惠、待退、收款——设计说明的六列去掉“长短款”（Kel 13:55 定指标卡、分币种汇总、CSV 三处隐藏；**Kel 14:21 再确认 PDF 保留长短款**）；不再套默认三列的 `.report-table`；390 宽改两行（第一行币种 + 收款）。活动表名称、笔数、优惠、收款，用 `.report-table--four`。
5. **币种下拉**（R1）：可见标签“导出币种”，无障碍名称也改成“导出币种”（原来只有无障碍名称“选择导出币种”）；只决定导出范围，不改屏幕报表。
   - **PDF 不受影响（Kel 14:21）：** 屏幕分币种汇总去掉长短款列，但 PDF 的分币种汇总（`document-model.ts:109` `summaryFor`）**保留长短款单元格**；如果为了统一列定义改 `summaryFor`，长短款那一格必须留下，`report-pdf.test.cjs` 要继续通过。
6. **“全部币种”提示**（设计说明问题 2，Kel 13:32 已定）：保留并核对 `page.tsx:316` 的提示“选择「全部币种」时，PDF 包含全部币种，CSV 只导出主币种 {主币种} 的明细。”；CSV 文件名由服务端 `Content-Disposition` 决定（`reports.go:228`，已带主币种），前端 `page.tsx:180-181` 读响应头、兜底格式相同，不用改，只加测试锁住。
7. **非主币种说明**（设计说明问题 11，Kel 13:32 已定：维持现状；**说明文字 Kel 14:40 定用设计长句**）：非主币种没有件数和各类订单笔数，界面和 PDF 用文案说明，不借用主币种的数字。
   - **屏幕（分币种汇总下方）：** 现在 `page.tsx:407` 是短句 `件数仅统计主币种。`，**改成设计说明的长句**。原句摘自 `docs/design-v2/销售日报-设计说明.md` 第 185 行（§4.5，第 507 行问题 11 的 Kel 决定里是同一句），一字不差：

     > 「售出件数、待付款、已关闭和退款笔数只按主币种 MYR 统计，不按其他币种拆分。」

     同一行设计说明还写了：“主币种取 `report.currency`”，所以句中的 `MYR` 渲染时换成 `report.currency`，其余文字不改；只在出现分币种汇总（快照含两种及以上币种）时显示；样式与导出提示相同：`--muted`、至少 0.8rem、行高 1.55、AA 对比度、窄屏正常换行、不横向溢出。
   - **PDF 不改：** `document-model.ts:94` 的 `件数仅统计主币种 ${headline}` 是 PDF 指标卡注释（设计说明第 346 行“PDF 若筛到非主币种……注明「件数仅统计主币种」”），不换成长句；`report-pdf.test.cjs:140` 继续通过。

**文件：** `web/src/app/admin/reports/page.tsx`、`web/src/app/admin/reports/reports.css`、`web/src/app/globals.css:2624-2630`、`web/tests/report-layout.test.cjs`（追加）、`web/tests/workspace-copy.test.cjs`（追加眉标断言）

- [ ] **步骤 1：编写失败的测试**（追加到 `report-layout.test.cjs`）

```js
const reportsCss = fs.readFileSync('src/app/admin/reports/reports.css', 'utf8');

test('C12 eyebrow 经营 above 销售日报', () => {
  assert.match(page, /<p className="eyebrow">经营<\/p>\s*<h1>销售日报<\/h1>/);
});

test('C13 toolbar has filters/actions groups and no viewport-only nowrap', () => {
  assert.equal((page.match(/className="report-toolbar__group"/g) || []).length >= 2, true);
  assert.doesNotMatch(reportsCss, /min-width:\s*70rem/);
  assert.match(reportsCss, /\.report-range-detail-toolbar \.report-toolbar__actions\s*{[^}]*flex-wrap:\s*nowrap/);
});

test('C14 top goods has four cells: index, name, quantity, revenue', () => {
  assert.doesNotMatch(page, /{index \+ 1}\. {good\.name}/);
  assert.match(globals, /\.report-rank li\s*{[^}]*grid-template-columns:[^;]*minmax\(0,\s*1fr\)[^;]*max-content[^;]*max-content/);
});

test('R1 export currency has a visible label 导出币种', () => {
  assert.match(page, />导出币种</);
});

test('issue 2: all-currency hint and filename fallback stay', () => {
  assert.match(page, /CSV 只导出主币种/);
  assert.match(page, /filename="\?\(\[\^";\]\+\)"\?/);
});

test('issue 11: non-primary currency note uses the design long sentence (Kel 14:40)', () => {
  // 设计说明 §4.5 第 185 行原句：售出件数、待付款、已关闭和退款笔数只按主币种 MYR 统计，不按其他币种拆分。（MYR 取 report.currency）
  assert.match(page, /售出件数、待付款、已关闭和退款笔数只按主币种\s*\{[^}]*currency[^}]*\}\s*统计，不按其他币种拆分。/);
  assert.doesNotMatch(page, /件数仅统计主币种。/, 'old short sentence should be gone from the screen');
});

test('currency summary has its own 5-column template, no 长短款', () => {
  assert.match(reportsCss + globals, /\.report-table--currency\s*{[^}]*grid-template-columns/);
  assert.doesNotMatch(page, /长短款/);
});
```
> 工具栏外层改成 `nowrap` 后 390 宽要靠媒体查询改回按组换行，第三条断言只锁桌面规则；如果实现改用容器查询，断言跟着改。

- [ ] **步骤 2：** 运行，确认 C12、C14、分币种列模板失败；C13 第三条如果因为前端用了别的写法（例如外层保持 `wrap`、靠组宽度排成一行）而失败，按实际写法改断言，不为了过测试改样式。
- [ ] **步骤 3：实现**：按上面 1–7 修改。热销 `li` 改成 `<em>{index + 1}</em><span>{good.name}</span><small>{good.quantity} 件</small><strong>金额</strong>`，列模板 `2ch minmax(0, 1fr) max-content max-content`。
- [ ] **步骤 4：** 通用验证；用 `toolbar-crop.cjs` 截三套风格 1280 的工具栏，必须一行（D5 原来是三行）；对照 `销售日报-v2-*` 7 张截图，包括空日、加载中、出错三种状态。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/admin/reports/page.tsx`、`web/src/app/admin/reports/reports.css`、`web/src/app/globals.css`、`web/tests/report-layout.test.cjs`、`web/tests/workspace-copy.test.cjs`

**验收：** spec §9 第 3、4 条；测试报告 D5 关闭；设计说明 §5 状态一览逐项对上。

---

#### 任务 W17：日报默认日期改用服务端“报表今天”，日期 `max`（设计说明 R4 / §8.4）

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** S8；**K-Q14 = A（Kel 已定 2026-11-26 13:19，R4）** ｜ **前端已改：** 无（`page.tsx:39` `localToday()` 用浏览器日期，日期输入 `max="9999-12-31"`） ｜ **状态：** Kel 已定，待后端（S8）

- [ ] 页面加载先请求 `GET /api/admin/daily-reports/today`，用返回的 **`today`** 作为默认起止日期和两个日期输入的 `max`，再自动查询这一天；区间不得晚于 `today`。
- [ ] **请求失败时不退回 `localToday()`**（设计说明 §8.4：“禁止退回浏览器本地日期”）：显示错误和“重试”按钮。12:25 版写的“失败时退回 `localToday()`”已作废。
- [ ] 提示文案可用 `timezone`：“今天 {today} 由服务端按店铺时区 {timezone} 提供”；前端不拿 `timezone` 自己算今天。
- [ ] 测试：`web/tests/report-dates.test.cjs`（已有）加用例：给定 `{ today: "2026-11-25", timezone: "Asia/Kuala_Lumpur" }`，默认日期和 `max` 都是 `2026-11-25`；请求失败时不调用 `localToday()`、进入出错状态；源码不再有 `max="9999-12-31"`。
- [ ] 只暂存，不 commit、不 push：`web/src/app/admin/reports/page.tsx`、`web/tests/report-dates.test.cjs`

---

### 8.4 展场（spec §8.2 E1–E24）

#### 任务 W-E1：展场工作台 token、对比度、阴影、圆角、衬线、控件边框

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 展场 spec Q1——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：A**（工作台保留深青，两面共用炭黑 `#262820`、柠绿 `#e3f7a2`、衬线标题；去掉海军蓝横幅（E12，W-E3）和钴蓝商品页（E16–E21，W-E4）） ｜ **spec：** §8.2 **E1–E7、E13、E14，以及 13:30 新增 E25（`--font-display` 改成宋体优先、新增 `--expo-charcoal: #262820`）、E26（筛选激活态文字用炭黑）、E27（品牌标识衬线 + 赭红角标）、E28（表单控件边框改用 `var(--line-control)`；**颜色 K-Q19 = A，Kel 15:01 定：工作台 `#6f8681`**，不是旧建议 `#738b86`）**；§4.2 颜色、§7.3 对比度；测试报告 D4；UIUX P2 #7 ｜ **前端已改：** D4 已修：`--muted` 从 `#526c6d` 改成 `#4f696a`（`exhibition-workspace.css:29`），在 `#d5e6df` 上 4.553:1；前端同意（F-3）按 spec E1 改成 `#4b6465`，改完复测对比度 ｜ **状态：** 部分完成（D4 待复测；E1 终值、E2–E7、E13、E14 待做）

**文件：** `web/src/app/exhibition-workspace.css`、`web/tests/theme-tokens.test.cjs`（追加）

- [ ] **步骤 1：编写失败的测试**

```js
const expo = fs.readFileSync('src/app/exhibition-workspace.css', 'utf8');
test('E1/E2 exhibition workspace muted and accent-light', () => {
  assert.match(expo, /--muted:\s*#4b6465/);
  assert.match(expo, /--accent-light:\s*#e3f7a2/);
});
test('E3-E7 shadows and selected state use tokens, no hard-coded lime', () => {
  assert.match(expo, /--shadow-rail:/);
  assert.doesNotMatch(expo, /#d8f69a/);
});
test('E13 report page no longer overrides --muted', () => {
  assert.doesNotMatch(expo, /\.report-page[^{]*{[^}]*--muted:/);
});
```
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：逐条按 E1–E7、E13、E14、E25–E28 的“改成”列。字体只用系统字体栈，不自托管 Inter（spec Q4，Kel 已定）。测试追加：`expo` 含 `--expo-charcoal: #262820`、`--line-control: #6f8681`（**K-Q19 = A，Kel 15:01：恢复断言具体值**；工作台块的 `--line-strong: #8ca29d` 不变）；`--font-display` 以 `"Songti SC"` 开头；`.workspace-brand__mark` 用 `var(--font-display)`。
- [ ] **步骤 4：** 通用验证；`contrast2.cjs` 里 `p.field-hint` 在渐变最深处 ≥4.5（D4 原来 4.46）。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/exhibition-workspace.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 6 条前半（工作台没有海军蓝和钴蓝，和 W-E3 一起验）、第 8 条（对比度）；测试报告 D4 关闭。

---

#### 任务 W-E2：展场侧栏底部、触控尺寸、窄屏横滑导航

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** W-C2（C10 的 `workspace-sidebar__foot` 结构） ｜ **spec：** §8.2 **E8–E11**；§3.2 窄屏导航、§5.4 适配；测试报告 D2（Kel 已定改为横向滑动） ｜ **前端已改：** D2 横滑已做完、自测通过（删掉 ≤900 的 74px 窄栏，`--workspace-rail: 0px`，`.workspace-nav` 改 `overflow-x: auto`，`workspace-shell.tsx` 用 `scrollIntoView` 把当前项滚进可视区）；E8–E11 未做 ｜ **状态：** 部分完成（D2 待复测）

- [ ] **步骤 1：编写失败的测试**（追加到 `theme-tokens.test.cjs`）：断言 `exhibition-workspace.css` 里 `.workspace-sidebar__logout` 用 `min-height: var(--touch)`（E8、E9）；`.workspace-nav` 有 `flex: 1 1 auto` 和 `overflow-y: auto`（E10）；`.workspace-nav__index` 仍是 `display: none`（E11 已改为：Kel 已定展场侧栏不显示编号，只有 classic 显示；这条用来防回退，现在就通过）；≤900 的 `.workspace-nav` 有 `overflow-x: auto`（D2，已通过，用来防回退）。
- [ ] **步骤 2：** 运行，确认 E8–E10 失败（E11、D2 两条已通过）。
- [ ] **步骤 3：实现**：按 E8–E10 的“改成”列；E11 不改。展场 901–1080 宽的 152px 窄侧栏保留（展场 spec Q3 = A，Kel 已定（2026-11-26 13:19，Grok Bot 转达）），本任务不动这一档。
- [ ] **步骤 4：** 通用验证；展场 390 当前项在可视区内、导航可横滑；1280 宽侧栏底部“返回 Stallink”“退出登录”不用滚动就能看到。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/exhibition-workspace.css`、`web/src/components/workspace-shell.tsx`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 1 条（展场部分）；测试报告 D2 关闭；对照 `展场-总览-手机.png`。

---

#### 任务 W-E3：展场总览横幅、收银台单列

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** W-E1（token）；展场 spec Q1 = A（Kel 已定（2026-11-26 13:19，Grok Bot 转达）：去掉海军蓝横幅） ｜ **spec：** §8.2 **E12、E15**；侧栏宽度保留 152px（spec Q3，Kel 已定，不改）；§7.2 收银台单手操作 ｜ **前端已改：** 无（`exhibition-workspace.css:697` 仍是 `linear-gradient(115deg, #0e203a, #22467c)`） ｜ **状态：** 待开始

- [ ] **步骤 1：编写失败的测试**：断言 `exhibition-workspace.css` 不含 `#0e203a`、`#22467c`（E12）；`.counter-products` 在默认规则里就是 `grid-template-columns: minmax(0, 1fr)`，而不是只在 1100–1330 区间（E15）。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：按 E12、E15。
- [ ] **步骤 4：** 通用验证；收银台商品名不竖排，≤1099 宽底部结算栏可见，“生成订单”≥52px 高，加减按钮 ≥44px。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/exhibition-workspace.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 5 条、第 6 条前半；对照 `展场-总览-桌面.png`、`展场-收银台-桌面.png`、`展场-收银台-手机.png`。

---

#### 任务 W-E4：展场顾客端商品页去钴蓝、补 token

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 展场 spec Q1——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：A**（去掉钴蓝 `#1e36a8`，商品详情页强调色用赭红）；顾客端手机竖栏是展场 spec Q2（13:09 前是 Q3），Kel 已定缩到 52px，**不在本任务**，由 W-E5 做 ｜ **spec：** §8.2 **E16–E21** ｜ **前端已改：** 无（`exhibition.css` 第 355、547、592、616、626、691 行仍是 `#1e36a8`） ｜ **状态：** 待开始

- [ ] **步骤 1：编写失败的测试**：`exhibition.css` 不含 `#1e36a8`；`[data-storefront-theme="exhibition"]` token 块里有 `--radius-control: 0`（E21）。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：按 E16–E21；E16 删掉 `.product-page` 那组 classic token 覆盖。
- [ ] **步骤 4：** 通用验证；展场商品详情页强调色是赭红。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/exhibition.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 6 条中段（商品详情页强调色是赭红）。

---

#### 任务 W-E5：展场窄屏退出入口改到 52px 竖栏底部（spec E22–E24、E29–E30；Kel 14:40 选 B）

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** W-E1（前端排法：W-E1 之后做）；展场 spec §2 第 10、12 条，§5.3（第 308–314 行“顾客端手机竖栏”“390 宽的退出入口”）、§9 第 12 条（“展场顾客端 390 宽：竖栏 52px，退出在竖栏底部、52×≥44px、不用滚动就能看到”） ｜ **Kel 决定：** 原 Q2 = C（13:19）；**2026-11-26 14:40 再确认选 B：展场窄屏的「退出」改到竖栏底部**（前端 14:33 提的问题 3：A 保留 390 顶栏里的退出，B 按 plan 改竖栏）。之前“390 顶栏退出不得回退”的要求**作废** ｜ **spec：** §8.2 **E22–E24、E29（`--expo-rail: 52px`、`.exhibition-nav { padding: .5rem 0 }`；顾客端控件边框色 **K-Q19 = A（Kel 15:01）：`#857e6e`**，值由 W-C3 改在 token 块里，本任务的测试断言它）、E30（品牌拆成全称 / 短称，≤620 宽只显示 “S.”）** ｜ **状态：** Kel 已定，待开工

**前端 Mac 现状（14:42 只读核对，未提交）：**
- `web/src/app/exhibition.css` 第 800–805 行 `@media (max-width: 620px)` 里还是 `--expo-rail: 67px`；
- 第 807 行起是前端先前按“390 页眉要有退出”做的 `@media (max-width: 390px)`：注释写“左侧窄栏改为顶栏，「退出」始终可见”，`--expo-rail: 0px`、`#main-content` 去掉左边距、`.exhibition-nav` 改 `position: sticky; top: 0` 横排；
- `web/src/components/app-nav.tsx:148` 的退出 `<button>` 在 `exhibition-nav__settings` 里，只有文字，没有图标；
- `web/tests/workspace-copy.test.cjs:125-133`（“exhibition storefront header includes a logout action when signed in”）断言 `exhibition.css` 有 `@media (max-width: 390px)`，这条要跟着改。

**要做的事：**
1. ≤620 宽（**含 390**）竖栏 52px，不改横条：删掉 `exhibition.css` 里“390 店面页眉：左侧窄栏改为顶栏”那段 `@media (max-width: 390px)` 顶栏模式（`--expo-rail: 0px`、去左边距、`.exhibition-nav` 横排吸顶），390 不再有顶栏里的「退出」。这段里如果有和顶栏模式无关的规则，保留并移到 620 块里，前端在回报里列出来。
2. E22：≤620 时竖栏项和退出按钮 `font-size: .75rem; width: 52px; min-height: 44px; padding-inline: 0; border-radius: 0`；文字和图标 `#f4f0e6`；`:focus-visible { outline: 2px solid var(--expo-lime); outline-offset: -2px }`。
3. E23：≤620 时 `.exhibition-nav__settings { position: sticky; bottom: 0; background: var(--expo-charcoal); padding-bottom: max(.5rem, env(safe-area-inset-bottom)) }`，退出钉在竖栏底部，不用滚动就能看到。
4. E24：`app-nav.tsx` 退出按钮加 18px `RailIcon name="logout"`（`railIcons` 新增门加箭头路径），文字「退出」保留。
5. E29、E30：按 spec“改成”列；控件边框色按 K-Q19 = A 是 `#857e6e`（W-C3 改 token 块，本任务只断言）。
6. 展场**工作台**不在本任务：≤900 横向吸顶条右上角的「退出登录」（spec §9 第 11 条后半）照旧。

**文件：** `/Users/kelden/go/src/Stallink/web/src/app/exhibition.css`（第 800–850 行一带）、`/Users/kelden/go/src/Stallink/web/src/components/app-nav.tsx`（第 137–150 行一带、`railIcons`）、`/Users/kelden/go/src/Stallink/web/tests/exhibition-shell.test.cjs`（新建）、`/Users/kelden/go/src/Stallink/web/tests/workspace-copy.test.cjs`（改第 125–133 行的用例）

- [ ] **步骤 1：编写失败的测试**（新建 `web/tests/exhibition-shell.test.cjs`）：
  - 读 `exhibition.css`，`@media (max-width: 620px)` 块里断言 `--expo-rail: 52px`；整个文件**不再有** `--expo-rail: 0px`（390 顶栏模式删掉）；
  - 620 块里 `.exhibition-nav__settings` 含 `position: sticky` 和 `bottom: 0`；退出按钮所在选择器含 `min-height: 44px`、`width: 52px`；
  - `app-nav.tsx` 的 `exhibition-nav__settings` 里退出按钮含 `RailIcon name="logout"`，`railIcons` 有 `logout`；仍含 `{loading ? "退出中" : "退出"}` 和 `onClick={logout}`；
  - `app-nav.tsx` 含 `exhibition-nav__brand-full` 和 `exhibition-nav__brand-short`，后者带 `aria-hidden="true"`；
  - 断言 `exhibition.css` 两个展场 token 块里 `--line-control: #857e6e`（K-Q19 = A，Kel 15:01）；**不出现** `#738b86`。
  - 改 `workspace-copy.test.cjs:125-133`：去掉 `assert.match(exhibitionCss, /@media \(max-width: 390px\)/)`，换成断言 620 块里竖栏 52px、退出在 `.exhibition-nav__settings` 里；用例名改成“exhibition storefront rail keeps logout at the bottom when signed in”。
- [ ] **步骤 2：运行测试，确认失败**：`cd web && node --test tests/exhibition-shell.test.cjs tests/workspace-copy.test.cjs`。
- [ ] **步骤 3：实现**：按上面 1–5。
- [ ] **步骤 4：运行测试，确认通过**；通用验证；浏览器在 390 和 620 宽各看一次展场顾客端（登录态）：竖栏 52px，没有顶栏，「退出」在竖栏底部，不滚动就能看到，点击区 ≥ 52×44，键盘 Tab 到它时焦点环可见；页面没有横向滚动。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/exhibition.css`、`web/src/components/app-nav.tsx`、`web/tests/exhibition-shell.test.cjs`、`web/tests/workspace-copy.test.cjs`。

**验收：** spec §9 第 12 条；390 / 620 宽展场顾客端只有竖栏一个退出入口，位置在竖栏底部；`node --test tests/*.cjs` 全绿。

---

### 8.5 店面（spec §8.3 S1–S18）

#### 任务 W-S1：店面 token（`--warning` 等）

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** K-08；展场/店面 spec Q4（店面字体；13:09 前是 Q5，12:44 前是 Q6）——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：A 系统字体，不自托管 Inter**（S5 前端已完成） ｜ **spec：** §8.3 **S1–S5、S18** ｜ **前端已改：** S1 `shop.css:18` `--warning: #b42318`（另加了 `--shop-danger` 一组同色变量）；S5 已删掉 `Inter`，只用系统字体；另外前端把 `--muted` 改成 `--shop-muted: #726f6e`：从 `#787574` 调来以达到 AA，在 `#f2f4f5` 上 4.52:1、白底 4.98:1（F-7）。**已解决**：UIUX 已回填进 spec（第 101 行 token 表，改动清单 §8.3 **S19**，“已修：保持”） ｜ **状态：** 部分完成（`--shop-muted` 已解决；S2–S4、S18 待做）

- [ ] **步骤 1：编写失败的测试**（追加到 `theme-tokens.test.cjs`）

```js
const shop = fs.readFileSync('src/app/shop.css', 'utf8');
test('S1-S4 shop tokens', () => {
  assert.match(shop, /--warning:\s*#b42318/);       // 已通过，防回退
  assert.match(shop, /--line-strong:\s*#dedfe0/);
  assert.match(shop, /--accent-dark:\s*#4524db/);
  assert.match(shop, /--line-control:\s*#8f8c8b/);   // K-Q19 = A（Kel 15:01）；--line-strong #dedfe0 不变
  for (const t of ['--radius-control', '--radius-card', '--radius-panel', '--radius-pill']) {
    assert.match(shop, new RegExp(`${t}:`));
  }
});
test('S5 no Inter webfont', () => {
  assert.doesNotMatch(shop, /font-(display|ui|mono):\s*Inter/);
});
```
- [ ] **步骤 2：** 运行，确认 S2–S4 失败。
- [ ] **步骤 3：实现**：按 S2–S5；S18 的字面圆角**分批替换**，本任务只换侧栏、卡片、按钮三类，并在文件头注释“新代码不要写字面圆角”。
- [ ] **步骤 4：** 通用验证；“仅剩 N 件”和报错文字是 `#b42318`。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/shop.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 7 条前半。

---

#### 任务 W-S2：店面侧栏底部、窄屏横滑导航

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** W-C2（C10）；店面 ≤900 改横滑胶囊条、去掉折叠按钮——**Kel 已定**（spec §2 第 7 条、S11、S12，UIUX 13:09 写进 spec 并关闭原 Q2） ｜ **spec：** §8.3 **S8–S12**；§6.4 适配；UIUX P1 #5（`shop-1280-admin.png` 底部被裁） ｜ **前端已改：** 店面侧栏底部被裁已用**临时方案**修好：`shop.css:1293` `max-height: calc(100dvh - 96px - 16px)`、`:1307` 整栏 `overflow-y: auto`，底部按钮能看到。前端确认这是临时方案（F-2）。本任务把它改成**导航区单独滚动、底栏固定**（spec S8）。S11 仍是 860 宽折叠按钮（`shop.css:668`、`:1175`、`:1543`） ｜ **Kel 15:01 改 S11 两处（前端 14:5x 提的 Q1、Q2，Grok Bot 转达）：** ①~~侧栏吸顶保持 `top: 96px`~~（**已被 K-Q21 = B 细化，见下**）；S11 的 `top: 56px` **作废**；②**≤900 不隐藏侧栏底部的「退出登录」**，S11 原文“隐藏品牌、身份、底部操作和分组标签”里的“底部操作”这一项作废，品牌、身份、分组标签照旧隐藏。spec S11、§6.3 第 357 行（“品牌和退出在这个宽度下由顶栏承担”、`top: 56px`）、§6.4 第 383 行由 UIUX 改，本计划不动 spec ｜ **K-Q21 = B（Kel 2026-11-26 15:13，Grok Bot 转达；§11.12）：吸顶位置跟着页头高度走**：宽度 >900 `top: 96px`；641–900 `top: 72px`（页头 72px）；≤640 `top: 64px`（页头 64px，在 `@media (max-width: 640px)` 里）。**页头高度本身不改**；吸顶不得用 56px（不得出现 `top: 56px`，页头 `min-height` 也不得是 56px）。`shop.css` 里和吸顶无关的 56px 保留（15:25 grep 核对）：`.catalog__box` 的 `min-height: 56px`（第 251 行）和 `padding: 4px 56px 4px 20px`（第 252 行）、`.event-card` 的 `grid-template-columns: 56px minmax(0, 1fr) auto`（第 2260 行）、`.event-card__date` 的 `width: 56px`（第 2270 行）、约第 2351 行 `.event-card` 日期列 `grid-template-columns: 56px minmax(0, 1fr)`。变量名统一用 `--shop-header-height`（前端已经在用）；UIUX 原型 `docs/design-v2/展场与店面-工作台-原型.html` 第 276、299、308、349 行里的 `--shop-header-h` 是旧名，以代码为准。spec 交叉引用：展场/店面 spec `docs/superpowers/specs/2026-11-26-stallink-theme-exhibition-shop-design.md`（15:15 版行号；**15:28 新版 md5 `822af34f3bb2131eef7c30a752c43e99`**，原型 `docs/design-v2/展场与店面-工作台-原型.html` 新版 md5 `2bea9105e36ef9836908ae32db213328`，两边变量名都已统一成 `--shop-header-height`，原型里 `--shop-header-h` 已 0 处）：§2 决定 17 第 63 行、§6.3 第 360 行、§6.4 第 384–387 行、S11 第 512 行、§9 验收第 14 项第 539 行。行号说明：Kel 原话引的是 `shop.css` 第 87 行（72px）、第 735 行（64px）；前端 15:14 已改成变量，现在磁盘上是 `--shop-header-height: 72px`（第 51 行）、`@media (max-width: 640px)` 里 `--shop-header-height: 64px`（第 729 行起，第 732 行），页头 `min-height: var(--shop-header-height)`（第 88、740 行），≤900 吸顶 `top: var(--shop-header-height)`（第 1246 行），>900 仍是 `top: 96px`（第 1376 行）——值和 Kel 的决定一致，实现和测试按选择器和值核对，不按行号 ｜ **状态：** 部分完成（临时方案已上，S8 正式方案待做）

- [ ] **步骤 1：编写失败的测试**：`shop.css` 的 `.workspace-sidebar` 有 `display: flex` 和 `flex-direction: column`、**自身不再 `overflow-y: auto`**；`.workspace-nav` 有 `flex: 1 1 auto; min-height: 0; overflow-y: auto`；`.workspace-sidebar__foot` 有 `flex: 0 0 auto`，不在滚动区（S8，替换 `shop.css:1307` 的临时方案）；`.workspace-nav__group` `margin-bottom: 16px`（S9）；没有 `.workspace-nav-toggle` 规则，`@media (max-width: 900px)` 下 `.workspace-nav` 有 `overflow-x: auto`（S11、S12）；**吸顶分档（K-Q21 = B）**：店面 `.workspace-sidebar` 基础规则 `top: 96px`；`@media (max-width: 900px)` 里店面 `.workspace-sidebar` 的 `top` 等于页头高度（`72px`，或 `var(--shop-header-height)` 且该变量在店面根块是 `72px`）；`@media (max-width: 640px)` 里页头高度是 `64px`，吸顶跟着是 `64px`（直接写 `top: 64px`，或变量在 640 块里是 `64px`）；页头 `.site-header__inner` 的高度仍是 72px / 64px，不改；**吸顶不得用 56px（不得出现 `top: 56px`，页头 `min-height` 也不得是 56px）**：断言店面 `.workspace-sidebar` 相关规则里没有 `top: 56px`、`.site-header__inner` 的 `min-height` 不是 56px（按选择器定位，**不要**对全文件做“不含 56px”的断言——`shop.css` 里和吸顶无关的 56px 保留（15:25 grep 核对）：`.catalog__box` 的 `min-height: 56px`（第 251 行）和 `padding: 4px 56px 4px 20px`（第 252 行）、`.event-card` 的 `grid-template-columns: 56px minmax(0, 1fr) auto`（第 2260 行）、`.event-card__date` 的 `width: 56px`（第 2270 行）、约第 2351 行 `.event-card` 日期列 `grid-template-columns: 56px minmax(0, 1fr)`）；页头高度和 ≤900 吸顶都读 `--shop-header-height`（店面根块 72px，640 块 64px）；**按源码顺序检查最终值**：把 `shop.css` 里所有作用在店面根选择器上的 `--shop-header-height` 声明按出现顺序列出来（记下各自所在的 `@media` 条件），模拟 ≤640 宽度（同时命中无条件、`max-width: 900px`、`max-width: 640px`）时**最后一次生效的值必须是 `64px`**，模拟 768 宽度时最后生效的是 `72px`；并断言 `@media (max-width: 900px)` 块里没有 `--shop-header-height` 声明，或者它出现在 ≤640 块之前（UIUX 15:2x 发现的层叠 bug，现第 1237 行）；≤900 块里 `.workspace-sidebar__logout` 不是 `display: none`（Kel 15:01）。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：按 S8–S12；`workspace-shell.tsx` 里店面专用的折叠按钮一起删掉（S11），`scrollIntoView` 逻辑和展场共用。吸顶按 K-Q21 = B 分三档：>900 `top: 96px`；641–900 `top: 72px`；≤640 `top: 64px`。用页头高度变量 `--shop-header-height` 让页头 `min-height` 和 ≤900 吸顶 `top` 取同一个值，避免两处数字不同步；页头高度不改。**层叠顺序（UIUX 15:2x 发现的 bug）：** `--shop-header-height` 只能在**店面根块**（`body:has(#storefront-shell[data-storefront-theme="shop"])`，72px）和 **≤640 块**（64px）里各声明一次；**不能**在 `@media (max-width: 900px)` 里再声明。现在磁盘上（15:28 grep，`shop.css` 15:22 版）第 1235 行的 `@media (max-width: 900px)` 里第 1237 行又写了 `--shop-header-height: 72px`，它在 ≤640 块（第 730 行起，第 732 行 64px）**后面**、选择器特异性相同，于是把 64px 盖掉，所有宽度页头都是 72px。修法：删掉第 1237 行这次重复声明（推荐）；**15:29 再看：前端已删掉这次重复声明（未提交），现在全文件只有第 51 行 72px 和第 732 行 64px 两处声明，≤900 吸顶 `top: var(--shop-header-height)` 在第 1246 行，桌面 `top: 96px` 在第 1381 行；测试和实测验收仍要做，防止回退**；如果一定要在 ≤900 里声明，就把 ≤640 的块挪到它后面。桌面端 `.workspace-sidebar` 的 `top: 96px`（现第 1385 行）是写死的、不用变量，这样是对的，不要改成变量。变量名统一用 `--shop-header-height`（前端已经在用）；UIUX 原型 `docs/design-v2/展场与店面-工作台-原型.html` 第 276、299、308、349 行里的 `--shop-header-h` 是旧名，以代码为准。
- [ ] **步骤 4：** 通用验证；店面 1280 侧栏底部两个按钮不用滚动就能看到；820、390 横滑，当前项在可视区；**吸顶无空带（K-Q21 = B）**：店面登录态在 **1280、900、768、640、390、320** 六个宽度下打开 `/admin` 并向下滚动，吸顶栏上沿都紧贴页头下沿，上方**不露空带**、也不钻到页头下面（1280 是 `top: 96px` 的侧栏白卡；900、768 贴 72px 页头；640、390、320 贴 64px 页头；用 DevTools 量吸顶栏 `getBoundingClientRect().top` 等于页头 `offsetHeight`，1280 档等于 96）；**层叠 bug 回归**：390 和 320 宽度实测页头 `.site-header__inner` 高 64px、吸顶栏 `top` 64px；768 宽度两者都是 72px；1280 宽度侧栏 `top: 96px`（写死，不读变量）；「退出登录」在窄屏仍可见、可点（位置和底色由 W19 D9 修）。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/shop.css`、`web/src/components/workspace-shell.tsx`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 1 条（店面部分）、**第 14 项（第 539 行，K-Q21 = B）**；1280 / 900 / 768 / 640 / 390 / 320 六个宽度吸顶栏上方都不露空带；390、320 下页头和吸顶都是 64px、768 下都是 72px（`--shop-header-height` 在 ≤900 块里不重复声明）；对照 `店面-总览-桌面.png`、`店面-总览-手机.png`。

---

#### 任务 W-S3：店面标题字号、≤900 顶栏

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 无 ｜ **spec：** §8.3 **S6、S7、S13** ｜ **前端已改：** 无 ｜ **状态：** 待开始

- [ ] 测试：`shop.css` 里 S6、S7 两组 h1 规则是 `font-size: 28px`；≤900 宽隐藏 `.header-link[href="/about"]` 和“查看店铺”（S13）。
- [ ] 实现按 S6、S7、S13；通用验证，390 顶栏不竖排、不溢出。
- [ ] 只暂存，不 commit、不 push：`web/src/app/shop.css`、`web/tests/theme-tokens.test.cjs`

---

#### 任务 W-S4：店面收银台单列、胶囊筛选、禁用按钮

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** W-S1（token） ｜ **spec：** §8.3 **S14–S16**；§7.2；UIUX P1 #3（`shop-1280-counter.png`） ｜ **前端已改：** 无 ｜ **状态：** 待开始

- [ ] 测试：`.counter-products { grid-template-columns: minmax(0, 1fr) }`（S14）；筛选按钮外层 `border-radius: var(--radius-pill)`（S15）；`:is(.button--dark, .ed-btn):disabled` 背景 `#ebebeb`、字色 `var(--shop-muted-strong)`（S16）。
- [ ] 实现按 S14–S16；通用验证，`contrast2.cjs` 禁用主按钮字色 ≥4.5。
- [ ] 只暂存，不 commit、不 push：`web/src/app/shop.css`、`web/tests/theme-tokens.test.cjs`

**验收：** spec §9 第 5 条、第 7 条后半；对照 `店面-收银台-桌面.png`、`店面-收银台-手机.png`。

---

### 8.6 平台外观页（外观 spec §8 A1–A18）

> 接口现状：`GET /api/platform/appearance` 公开、返回 `{ theme }`；`PUT` 仅平台管理员。A1–A18 **不需要改接口**；发布人和时间（外观 spec Q1，Kel 已定（2026-11-26 13:19，Grok Bot 转达）：B）需要 S5。

#### 任务 W-A1：外观页状态与发布流程

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 外观 spec Q3——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：B，单独排期**：全站断网横幅移出本期，写进 §13 后续项；本任务只做本页的断网横幅（A5）。外观 spec Q4（读取失败提醒平台管理员）——Kel 已定（2026-11-26 13:19，Grok Bot 转达）：B，**由新任务 W-A4 实现**，不在本任务 ｜ **spec：** `2026-11-26-stallink-platform-appearance-design.md` §3 方案 A（卡片 + 发布确认）、§4.2 状态表、§8 **A2–A11、A17、A18** ｜ **发布确认框焦点（Kel 15:01，前端 14:5x 提的 Q3）：** 外观 spec §4.2 第 70 行、§6 第 132 行、§9 第 5 条要求打开时焦点在「取消」；共用的 `web/src/components/confirm-dialog.tsx` 第 44–45 行在 `danger=false` 时把 `autofocus` 给「确认」。Kel 定：给 `ConfirmDialog` 加一个可选参数（例如 `initialFocus?: "cancel" | "confirm"`，不传时保持现在的规则：危险操作焦点在取消、非危险在确认），**只有外观页传 `initialFocus="cancel"`**，其他页面一律不变（**前端 Mac 现状 15:07：** `confirm-dialog.tsx:14、24` 已加 `initialFocus` 和 `initialFocus ?? (danger ? "cancel" : "confirm")`，`appearance/page.tsx:273` 已传 `"cancel"`，未提交） ｜ **前端已改：** `page.tsx` 只有 1 行改动，状态和发布确认都还没做 ｜ **状态：** 待开始

**文件：** `/Users/kelden/go/src/Stallink/web/src/app/platform/appearance/page.tsx`、`/Users/kelden/go/src/Stallink/web/src/app/platform/appearance/appearance.css`、`/Users/kelden/go/src/Stallink/web/src/components/confirm-dialog.tsx`（加可选 `initialFocus`，Kel 15:01）、新建 `/Users/kelden/go/src/Stallink/web/tests/appearance-page.test.cjs`

- [ ] **步骤 1：编写失败的测试**

```js
/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const page = fs.readFileSync('src/app/platform/appearance/page.tsx', 'utf8');
const css = fs.readFileSync('src/app/platform/appearance/appearance.css', 'utf8');

test('A2/A3 publish goes through a confirm dialog', () => {
  assert.match(page, /confirmOpen/);
  assert.match(page, /ConfirmDialog/);
});
test('A3 publish confirm opens with focus on 取消 (Kel 15:01); other pages keep the default', () => {
  const dialog = fs.readFileSync('src/components/confirm-dialog.tsx', 'utf8');
  assert.match(dialog, /initialFocus\?:\s*"cancel"\s*\|\s*"confirm"/);
  // 不传 initialFocus 时：danger → 取消，非 danger → 确认（和现在一样）
  assert.match(dialog, /initialFocus\s*\?\?\s*\(danger \? "cancel" : "confirm"\)/);
  assert.match(page, /<ConfirmDialog[\s\S]*?initialFocus="cancel"/);
  for (const other of require('node:child_process').execSync('grep -rl "<ConfirmDialog" src --include=*.tsx').toString().trim().split('\n')) {
    if (other.endsWith('platform/appearance/page.tsx')) continue;
    assert.doesNotMatch(fs.readFileSync(other, 'utf8'), /initialFocus=/, `${other} must keep the default focus`);
  }
});
test('A5 offline banner listens to online/offline', () => {
  assert.match(page, /addEventListener\("offline"/);
  assert.match(page, /addEventListener\("online"/);
});
test('A6 401 and 403 have their own blocks', () => {
  assert.match(page, /登录已过期/);
  assert.match(page, /只有平台管理员可以修改平台外观/);
});
test('A7 revert link and A11 two tags', () => {
  assert.match(page, /还原为当前风格/);
  assert.match(page, /已选 · 未发布/);
});
test('A8 view storefront opens in new tab', () => {
  assert.match(page, /target="_blank" rel="noopener"/);
});
test('A10/A17/A18 status bar, message blocks, reduced motion', () => {
  assert.match(page, /appearance-status/);
  assert.match(css, /\.appearance-offline/);
  assert.match(css, /prefers-reduced-motion: reduce/);
});
```
- [ ] **步骤 2：** `node --test tests/appearance-page.test.cjs`；预期全部失败。
- [ ] **步骤 3：实现**：按 §4.2 的 13 种状态和 A2–A11、A17、A18 的“改成”列；文案以 §4.2 为准。
- [ ] **步骤 4：** 通用验证，并按外观 spec §9 第 1–6 条手工走一遍：DevTools Offline 出横幅、发布禁用、恢复后自动重读；mock 让 GET 返回 500 时没有卡片显示“使用中”；Tab 进卡片组、方向键切换、确认框焦点在“取消”、Esc 后焦点回发布按钮；390/820/1280 无横向滚动，手机发布按钮在底部固定栏且 ≥48px。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/platform/appearance/page.tsx`、`web/src/app/platform/appearance/appearance.css`、`web/tests/appearance-page.test.cjs`；确认框焦点单独一条拟提交：`web/src/components/confirm-dialog.tsx`（加 `initialFocus`）＋外观页传参＋上面那条测试

**外观 Q12（原 Q4）= B 的落点：** Kel 已定（2026-11-26 13:19，Grok Bot 转达），由**任务 W-A4** 实现（改根布局和平台工作台，不在 A1–A18 里，所以单独成任务，不再作为本任务的追加步骤）。**全站断网横幅（外观 Q3 = B）移出本期**，见 §13 后续项；本任务的 A5 只做本页。

**验收：** 外观 spec §9 第 1–6 条；对照 `平台外观-读取中-桌面.png`、`-读取失败-桌面.png`、`-发布确认-桌面.png`、`-发布中-手机.png`、`-发布失败-手机.png`、`-已发布-手机.png`、`-断网-手机.png`、`-无权限-桌面.png`。

---

#### 任务 W-A2：外观页双面缩略图与固定示例色

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 外观 spec Q2——**Kel 已定（2026-11-26 13:19，Grok Bot 转达）：A，不做实时预览** ｜ **spec：** §8 **A1、A12–A16** ｜ **前端已改：** 无（`appearance.css:30` 仍写死 `#5433eb`） ｜ **状态：** 待开始

- [ ] 测试（追加到 `appearance-page.test.cjs`）：`page.tsx` 的 `options` 每项有 `swatches` 和 `swatchLabel`（A1）；`appearance.css` 不含 `#5433eb`（A13），十六进制只出现在 `.appearance-options` 上集中声明的 `--sample-*` 变量里（A14）；`.appearance-preview` 是 `grid-template-columns: 1.15fr 1fr`（A15）；没有 `.appearance-label small { color: var(--accent) }`，改为 `.appearance-tag--live`、`.appearance-tag--draft`（A16）。
- [ ] 实现按 A1、A12–A16；通用验证。
- [ ] 只暂存，不 commit、不 push：`web/src/app/platform/appearance/page.tsx`、`web/src/app/platform/appearance/appearance.css`、`web/tests/appearance-page.test.cjs`

**验收：** 外观 spec §9 第 7 条（三种风格下缩略图颜色都是固定示例色）；对照 `平台外观-桌面.png`、`平台外观-手机.png`。

---

#### 任务 W-A3：外观页状态条显示发布人和时间（外观 spec A19）

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** S5；外观 spec Q9（原 Q1）——**Kel 已定 2026-11-26（13:19；13:32 再确认：只对平台管理员返回）** ｜ **spec：** 外观 spec §8.2 **A19**、§8.1 ｜ **前端已改：** 无 ｜ **状态：** 待后端（S5）

- [ ] 测试：状态条渲染函数传 `{ updatedAt: "2026-11-26T21:30:00+08:00", updatedBy: { id: "7", name: "运营 Kel" } }`，输出“上次发布：运营 Kel · 11/26 21:30”；`updatedAt: null` 输出“尚无记录”；PUT 成功后显示“你 · 刚刚”（§8.1）。
- [ ] 实现：读 `GET /api/platform/appearance` 的 `updatedAt`、`updatedBy`（只有平台管理员会话才有；字段不存在按“尚无记录”处理）；PUT 成功后用响应里的值刷新状态条。
- [ ] 只暂存，不 commit、不 push。

---

#### 任务 W-A4：外观读取降级提醒（外观 spec A20–A22，Q12）

> **端：** web ｜ **优先级：** P2 ｜ **依赖：** 外观 spec Q12（原 Q4）——**Kel 已定 2026-11-26**：只提醒平台管理员，顾客、摊主、销售员看不到；不依赖后端改接口 ｜ **spec：** 外观 spec §4.2“读取降级提醒”状态、§5、§6 `.platform-theme-fallback`、§8.2 **A20–A22**、§9 第 9 条；截图 `平台外观-读取降级提醒-桌面.png`（13:30 新增）；原型 `?state=fallback` ｜ **不在本期：** 外观实时预览（Q10）、全站断网横幅（Q11）——和收银台离线一起另外排期 ｜ **前端已改：** 无 ｜ **状态：** Kel 已定，待开工

**文件：** `web/src/lib/api.ts`（`getStorefrontTheme()`）、`web/src/app/layout.tsx`、`web/src/components/workspace-shell.tsx`、`web/src/app/globals.css` 或 `web/src/app/platform/appearance/appearance.css`、`web/tests/appearance-fallback.test.cjs`（新建）

- [ ] **步骤 1：编写失败的测试**：`getStorefrontTheme()` 在请求失败或 2.5 秒超时时返回 `{ theme: "classic", fallback: true }`，成功时 `fallback: false`（A20）；`layout.tsx` 把 `fallback` 传给 `SiteFrame`（如 `data-theme-fallback`），顾客端不渲染任何提示；`workspace-shell.tsx` 只在 `fallback && account.role === "platform_admin"` 时渲染 `.platform-theme-fallback`，“重新检查”调用 `router.refresh()`（A21）；样式复用 `.appearance-message--error`，按钮 `min-height: 44px`（A22）。
- [ ] **步骤 2：运行测试，确认失败**：`cd web && node --test tests/appearance-fallback.test.cjs`。
- [ ] **步骤 3：实现**：按 A20–A22 的“改成”列；提醒文案用外观 spec §4.2 的原文。
- [ ] **步骤 4：运行测试，确认通过**；对照截图。
- [ ] **步骤 5：只暂存，不 commit、不 push**。

---

### 8.7 店铺资料与操作记录

#### 任务 W14：店铺资料页按模板补齐

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** S2；K-Q1 = A、K-Q2 = A、K-Q3 = B、K-Q4 = A（Kel 已定（2026-11-26 13:22）） ｜ **前端已改：** `admin/shop/page.tsx:158-170` 已加“客服联系”“本场摊位号”“默认货币”三个字段，全部 `disabled`，占位“等后端”；`:78` 仍从 `/api/public/shops/{slug}` 读；`:119-120` 的 PATCH 仍带 slug ｜ **状态：** 部分完成 + 待后端

- [ ] **步骤 1：编写失败的测试**（新建 `web/tests/shop-settings.test.cjs`）：`page.tsx` 读 `/api/admin/shop`，不再读 `/api/public/shops/`；PATCH 请求体不含 `slug`；三个字段没有 `disabled` 和“等后端”；slug 显示为只读文本并带“由平台分配”说明；页面显示 `updatedAt`（`:198` 附近）。
- [ ] **步骤 2：** 运行，确认失败。
- [ ] **步骤 3：实现**：字段名按 S2 的接口约定（`contact`、`boothNumber`、`defaultCurrency`、`updatedAt`）；按 K-Q1 = A，“本场摊位号”改名为“默认摊位号”，说明文字“新建活动时自动带入；顾客端优先显示活动自己的摊位号，活动没填才用这里的”（F-8）；“默认货币”说明“新建活动时自动带入”（K-Q2 = A）；“客服联系”是自由文本输入框，`maxLength={120}`（K-Q4 = A）；按 S2 的错误码：`422 slug_readonly` 在 `web/src/lib/errors.ts` 加文案“店铺地址由平台分配，不能修改”，字段超长、摊位号首尾有空格、货币不合法都是 `400 invalid_shop`（后端 B-3 确认不拆细；摊位号输入框失焦时可以提示“首尾不能有空格”，但不要替用户裁剪）。停用店铺时的确认框和保存后提示写“停用后，店里的账号**登录后进不了工作台**，需要平台管理员重新启用”，不要写“登录被拒绝”（后端 B-5：停用账号的 Login 仍会发 cookie）；“本场摊位号”的标签和说明按 K-Q1 推荐方案写成“店铺默认摊位号（新建活动时带入）”；前端对标签没有意见，等 K-Q1 确认后再定（F-8）。
- [ ] **步骤 4：** 通用验证（mock-api 需要补 `GET /api/admin/shop` 的假数据，只改测试脚本目录，不改后端）。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/app/admin/shop/page.tsx`、`web/src/lib/errors.ts`、`web/tests/shop-settings.test.cjs`

---

#### 任务 W15：操作记录“对象”列

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** S3；**K-Q5 = C（Kel 已定 2026-11-26 13:55）**：先显示实时名称，修复 plan 任务 7 之后后端补快照、快照优先，前端不用区分 ｜ **前端已改：** `audit/page.tsx:20` 用 `auditSubjectLabel(type, id)`；现在的回退是“订单 101”或“—”（`web/src/lib/audit.ts:16-19`）；`web/tests/audit.test.cjs` 已建。显示格式前端已同意（F-9）：“类型 · 名称”，没有名称时“类型 #id” ｜ **状态：** 部分完成 + 待后端

- [ ] 测试（追加到 `audit.test.cjs`）：`auditSubjectLabel({ subjectType: "good", subjectId: "12", subjectLabel: "亚克力立牌·海雾" })` 返回“商品 · 亚克力立牌·海雾”；`subjectLabel` 为空时退回“商品 #12”；类型和编号都没有时返回“—”。
- [ ] 实现：S3 上线后优先用 `subjectLabel`；`createdAt` 按 S4 改成 RFC3339 后用 `Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Kuala_Lumpur" })` 格式化。
- [ ] 只暂存，不 commit、不 push：`web/src/lib/audit.ts`、`web/src/app/admin/audit/page.tsx`、`web/tests/audit.test.cjs`

---

#### 任务 W18：顾客端摊位号显示规则；新建活动表单带店铺默认值（K-Q1 = A、K-Q2 = A）

> **端：** web ｜ **优先级：** P1 ｜ **依赖：** S9（`displayBoothNumber`、新建默认值）、S2（`GET /api/admin/shop`） ｜ **前端已改：** 无 ｜ **状态：** Kel 已定，待后端

**规则：** 顾客端显示摊位号时，优先用活动自己的摊位号，活动没填才用店铺的；两个都没有显示“待公布”（现有文案）。后端已经按这个规则算好 `displayBoothNumber`（S9），前端只用这一个字段，不在页面里自己拼规则。

**顾客端要改的显示点（13:22 只读核对）：** `web/src/app/c/[slug]/events/[eventId]/page.tsx:31`、`c/[slug]/listings/[listingId]/page.tsx:19`、`c/[slug]/bundles/[bundleListingId]/page.tsx:20`、`web/src/components/booth-view.tsx:27、69、129`、`web/src/components/goods-grid.tsx:121`（执行时确认它的数据来自店铺接口还是全站目录，见 S9）；类型 `web/src/lib/api.ts:217-227` `BoothEvent` 加 `displayBoothNumber?: string`。工作台页面（`admin/events`、`admin/goods`、`counter`）继续显示活动自己的 `boothNumber`，不改。

**新建活动表单：** `admin/events/page.tsx:9` 的 `empty` 和 `admin/goods/page.tsx:48` 的 `emptyEvent` 现在写死 `boothNumber: ""`、`currency: "MYR"`。改成打开“新建活动”时从 `GET /api/admin/shop` 取 `boothNumber`、`defaultCurrency` 预填（店铺没设默认货币时仍用 `MYR`）；店主可以改。编辑已有活动时不预填。

- [ ] **步骤 1：编写失败的测试**（新建 `web/tests/booth-display.test.cjs`）
  - 新建 `web/src/lib/booth.ts`：`customerBoothLabel(event)` 返回 `event.displayBoothNumber || event.boothNumber || "待公布"`。用例：`{ displayBoothNumber: "A-27", boothNumber: "" }` → `A-27`；`{ displayBoothNumber: "B-01", boothNumber: "B-01" }` → `B-01`；`{ displayBoothNumber: "", boothNumber: "" }` → `待公布`；只有旧字段 `{ boothNumber: "C-03" }`（后端还没上线时）→ `C-03`。
  - 源码断言：上面列的顾客端文件都用 `customerBoothLabel`，不再直接写 `event.boothNumber || "待公布"`。
  - 源码断言：`admin/events/page.tsx`、`admin/goods/page.tsx` 新建活动时读 `/api/admin/shop` 的 `boothNumber`、`defaultCurrency`。
- [ ] **步骤 2：** `node --test tests/booth-display.test.cjs`，确认失败。
- [ ] **步骤 3：实现**：按上面的显示点和表单改。
- [ ] **步骤 4：** 通用验证；mock-api 补 `displayBoothNumber` 和 `GET /api/admin/shop` 假数据（只改测试脚本目录）；手工看：活动没填摊位号时店铺页、活动页、商品页都显示店铺摊位号。
- [ ] **步骤 5：只暂存，不 commit、不 push**：`web/src/lib/booth.ts`、`web/src/lib/api.ts`、上面列的顾客端页面和组件、`web/src/app/admin/events/page.tsx`、`web/src/app/admin/goods/page.tsx`、`web/tests/booth-display.test.cjs`

**验收：** 三种情况（活动有、活动无店铺有、都没有）在店铺页、活动详情、商品详情、套餐详情显示正确；新建活动表单预填店铺摊位号和默认货币。


---

#### 任务 W19：修测试报告的 D9（店面窄屏侧栏「退出登录」错位）、D10（缺 favicon）（Kel 14:40：都让前端修）

> **端：** web ｜ **优先级：** P2（测试报告里两条都是“低”） ｜ **依赖：** 无；D9 和 W-S2（店面侧栏底部，S8–S12）改同一块 CSS，谁后做谁合并 ｜ **来源：** `docs/reports/stallink-test-report-2026-11-26.html` 13:25 轮“缺陷”表；截图 `docs/reports/stallink-test-2026-11-26/retest3/screens/sidebar-logout-shop-860.png`、`sidebar-logout-shop-390.png` ｜ **Kel 决定：** 2026-11-26 14:40“D9、D10 都让前端修” ｜ **状态：** Kel 已定，待开工；**D10 图案 K-Q20 = A（Kel 2026-11-26 15:11，Grok Bot 转达）：选项 A「S.」字标，炭黑圆底配青柠色「S.」，全站共用**；D9、D10 各列一条拟提交

**和 S11 的关系（Kel 15:01）：** 店面窄屏「退出登录」**保留**（S11 的“隐藏底部操作”作废，见 W-S2），D9 **只修白色底和文字错位**，不藏按钮、不挪到别处。

**D9 原文（测试报告）：** “shop ≤900 侧栏「退出登录」文字贴顶，白色圆角底错位到文字下方，导航卡片下方空出一块（860 和 390 都这样）。”前端 14:40 量过：按钮现在 54×44。

**D10 原文（测试报告）：** “没有 favicon：每次打开新浏览器的第一页，控制台都报 GET /favicon.ico 404。”

**前端 Mac 现状（14:42 只读核对）：**
- D9：`web/src/app/shop.css` 第 584 行（`.workspace-sidebar__brand, .workspace-sidebar__logout { display: flex }`）、第 658–674 行（`.workspace-sidebar__logout` 白底胶囊 `min-height: 40px; margin-top: 12px; box-shadow`）、`@media (max-width: 900px)` 里第 1274–1281 行（`.workspace-sidebar__foot { display: contents }`、`.workspace-sidebar__logout { margin: 4px 8px 6px auto; min-height: 44px }`）。未修。
- D9：15:07 看到 `web/tests/shop-sidebar-logout.test.cjs` 已出现（前端在做，未提交），以前端回报为准。
- D10：`web/src/app/` 下没有 `favicon.ico`、`icon.*`、`apple-icon.*`、`manifest.*`，`web/public/` 下也没有 `icons/`；`web/src/app/layout.tsx` 的 `metadata` 只有 `title`、`description`。未修。
- **图案（15:08 更新）：** UIUX 已交方案，`docs/design-v2/favicon/`：`option-{a,b,c}.svg` 及各自的 `-16.svg`、`-apple.svg`、`-maskable.svg`，说明 `favicon-设计说明.md`，预览 `favicon-预览.html`，截图 `docs/design-v2/screens/favicon-选项对比.png`、`favicon-小尺寸.png`。A「S.」字标、B 摊位图形、C 字母块（深青 `#215e58` 底，沿用工作台 logo，UIUX 推荐）。**Kel 15:11 定 K-Q20 = A。** 15:1x 只读核对，A 系列四个源文件都在：`docs/design-v2/favicon/option-a.svg`（1138 B）、`option-a-16.svg`（1018 B）、`option-a-apple.svg`（1019 B）、`option-a-maskable.svg`（1018 B）；`option-a.svg` 是炭黑 `#262820` 圆底、柠绿 `#e3f7a2` 细描边和「S.」。
- **导出工具：** 15:08 只读看了 Mac：没有 `magick`、`convert`、`rsvg-convert`，`web/package.json` 和 `node_modules` 里也没有 `sharp`、`resvg`。设计说明 §5 写的是“ImageMagick 7，也可用 `sharp`、`resvg`”。**建议**在 `/tmp` 下临时装一个导出工具来生成图片，**不加进 `web/package.json`**（导出的 PNG / ICO 是静态资产，运行时用不到）；装哪个由前端选，回报里写清楚用的是哪个、哪个版本。

**文件：**
- D9：`/Users/kelden/go/src/Stallink/web/src/app/shop.css`（≤900 块的 `.workspace-sidebar__foot`、`.workspace-sidebar__logout`）；测试 `/Users/kelden/go/src/Stallink/web/tests/shop-sidebar-logout.test.cjs`（新建）
- D10（K-Q20 = A，源文件全部用 `docs/design-v2/favicon/option-a*` 系列；依据：`docs/design-v2/favicon/favicon-设计说明.md` §5–§7，Next 文档 `web/node_modules/next/dist/docs/01-app/03-api-reference/03-file-conventions/01-metadata/app-icons.md`、`manifest.md`）：

  | 输出文件（完整路径） | 尺寸 / 格式 | 源文件 | 导出方法（设计说明 §5） |
  |---|---|---|---|
  | `/Users/kelden/go/src/Stallink/web/src/app/favicon.ico` | ICO，内含 16×16、32×32、48×48 | 16 档 `option-a-16.svg`；32、48 档 `option-a.svg` | 先栅格成三张 PNG（透明底），再打包成一个 ICO；`favicon.ico` 只能放在 `src/app` 根段 |
  | `/Users/kelden/go/src/Stallink/web/src/app/icon.svg` | SVG | `option-a.svg` | 复制，可去掉设计稿注释，保留 `<title>` |
  | `/Users/kelden/go/src/Stallink/web/src/app/icon.png`（可选） | 32×32 PNG | `option-a.svg` | 栅格成 32×32；ICO 已有 32 档时可以不做 |
  | `/Users/kelden/go/src/Stallink/web/src/app/apple-icon.png` | 180×180 PNG，**不透明实底**，直角、不预先画圆角 | `option-a-apple.svg` | 栅格时背景填炭黑 `#262820`（选项 A 的底色，不要写成深青 `#215e58`） |
  | `/Users/kelden/go/src/Stallink/web/public/icons/icon-192.png`、`icon-512.png` | 192×192、512×512 PNG（`purpose: "any"`） | `option-a-apple.svg` | 栅格成对应尺寸 |
  | `/Users/kelden/go/src/Stallink/web/public/icons/icon-192-maskable.png`、`icon-512-maskable.png` | 192×192、512×512 PNG（`purpose: "maskable"`），背景铺满四边 | `option-a-maskable.svg` | 栅格成对应尺寸 |
  | `/Users/kelden/go/src/Stallink/web/src/app/manifest.ts` | Web App Manifest（Next 输出 `/manifest.webmanifest`） | 设计说明 §5 示例 | 新建，见下 |

  - 不要把 `docs/design-v2/favicon/` 的设计稿直接拷进 `app` 或 `public`（设计稿注释是给评审看的），按上表导出。
  - 不要把 192 / 512 做成 `src/app/icon.png`（会和 `icon.svg` 抢标签栏图标）。
  - **不写 `metadata.icons`**：文件约定会自动写 `<link>`，两边都写会重复（设计说明 §6）。`layout.tsx` 不改。
  - **全站共用一个图标**，不随 classic / 展场 / 店面切换，也不随 classic 深浅色切换；SVG 里不用 `prefers-color-scheme`（设计说明 §7）。
  - **本期不加 theme-color**：根布局不加 `themeColor` / `<meta name="theme-color">`，`manifest.ts` 也**不写 `theme_color`**（设计说明 §5 示例里有 `theme_color: "#faf8f3"`，§6 写的是“若要写”，本期按父代理 15:08 的要求不写）。
  - `manifest.ts` 字段：`name: "Stallink"`、`short_name: "Stallink"`、`description: "以独立链接作为摊位入口"`（和 `layout.tsx` 的 description 一致）、`start_url: "/"`、`display: "standalone"`、`background_color: "#262820"`（选项 A）、`icons` 四项：`/icons/icon-192.png`（`192x192`、`image/png`、`any`）、`/icons/icon-512.png`（`512x512`、`any`）、`/icons/icon-192-maskable.png`（`192x192`、`maskable`）、`/icons/icon-512-maskable.png`（`512x512`、`maskable`）。
  - 测试 `/Users/kelden/go/src/Stallink/web/tests/favicon.test.cjs`（新建）。

- [ ] **步骤 1：编写失败的测试**
  - `shop-sidebar-logout.test.cjs`：读 `shop.css` 的 `@media (max-width: 900px)` 块，断言店面 `.workspace-sidebar__logout` **没有 `display: none`**（Kel 15:01 保留），并且有 `display: inline-flex` 或 `flex`、`align-items: center`、`justify-content: center`、`min-height: 44px`、`min-width: 44px`，且白底胶囊（`background`）和文字在同一个盒子上（同一条规则里不能再有把底和字拆开的 `margin-top`/`padding-top` 偏移）。按选择器定位，不按行号。
  - `favicon.test.cjs`：
    - 文件存在：`src/app/favicon.ico`、`src/app/icon.svg`、`src/app/apple-icon.png`、`public/icons/icon-192.png`、`icon-512.png`、`icon-192-maskable.png`、`icon-512-maskable.png`、`src/app/manifest.ts`。
    - `favicon.ico`：读文件头，`reserved = 0`、`type = 1`、图像数 `= 3`，三个目录项的宽高分别是 16、32、48（ICO 目录项里宽高 0 表示 256，这里不会出现）。
    - PNG：读 IHDR，`apple-icon.png` 180×180、`icon-192*.png` 192×192、`icon-512*.png` 512×512；`apple-icon.png` 的颜色类型不是带 alpha 的 4 / 6，或者带 alpha 但所有像素不透明（用导出工具或 `zlib` 解码检查都可以，前端选一种写进测试）。
    - `icon.svg`：以 `<svg` 开头，含 `<title>`，不含 `prefers-color-scheme`。
    - `manifest.ts`：源码断言含 `name: "Stallink"`、`short_name: "Stallink"`、`start_url: "/"`、`display: "standalone"`、`background_color: "#262820"`，`icons` 正好四项且 `src`、`sizes`、`purpose` 和上面一致；**不含 `theme_color`**。
    - `src/app/layout.tsx` 不含 `icons:`、`themeColor`。
- [ ] **步骤 2：运行测试，确认失败**：`cd /Users/kelden/go/src/Stallink/web && node --test tests/shop-sidebar-logout.test.cjs` `node --test tests/favicon.test.cjs`，两条都应失败。
- [ ] **步骤 3：实现**
  - D9：改 `shop.css` ≤900 块，按钮保留（不加 `display: none`），让「退出登录」文字在白底胶囊里垂直、水平居中，导航卡片下方不再空出一块；点击区 ≥ 44×44。只改店面主题选择器，classic、展场不受影响。
  - D10：按上面的表从 `docs/design-v2/favicon/option-a*.svg` 导出 7 个图片文件（`icon.png` 可选），新建 `manifest.ts`；导出用的临时目录和工具放 `/tmp`，不留在仓库里。
- [ ] **步骤 4：运行测试，确认通过**；通用验证；浏览器：店面登录态 860、390 宽打开 `/admin`，「退出登录」文字在胶囊正中，胶囊下方没有空块，点击区 ≥ 44×44，和测试报告截图对比；D10：`next build && next start` 在隔离端口上跑，新开无痕窗口打开首页，控制台没有 `favicon.ico 404`；`<head>` 里有 `rel="icon"`（`/favicon.ico`、`/icon.svg`）、`rel="apple-touch-icon"`、`rel="manifest"`，没有重复的 icon link，没有 `theme-color`；`/manifest.webmanifest` 返回的 JSON 字段和测试一致；浅色、深色浏览器标签栏里 16px 图标都能认出；三套外观下图标相同。
- [ ] **步骤 5：只暂存，不 commit、不 push**：D9 一条（`web/src/app/shop.css`、`web/tests/shop-sidebar-logout.test.cjs`），D10 一条（`web/src/app/favicon.ico`、`web/src/app/icon.svg`、可选 `web/src/app/icon.png`、`web/src/app/apple-icon.png`、`web/public/icons/` 下 4 个 PNG、`web/src/app/manifest.ts`、`web/tests/favicon.test.cjs`）。

**验收：** 测试报告 D9 复测关闭；D10 复测关闭（manifest 字段和上面 8 个文件按测试核对）；`node --test tests/*.cjs` 全绿；三套外观 × 390 / 860 / 1280 没有横向滚动。

---

### 8.8 测试

#### 任务 T1：补测与补截图

> **端：** 测试 ｜ **优先级：** P1 ｜ **依赖：** 上面各 web 任务 ｜ **来源：** UIUX P1 #6 ｜ **状态：** 待开始

- [ ] **真实登录截图（F-4）：** 用真实登录状态（本机 docker-compose 库 + 本机 API，**不连云端库**；或测试账号登录的预发环境，由 Kel 指定），classic、展场、店面三套风格各截一张 1280 宽的 `/admin/reports`，确认工具栏两组排在一行。前端目前只用 fixture 量过 1280 和 860 宽。
- [ ] 复测前端本轮已做完的项：D1～D5、商品名换行、店面侧栏、`.header-loading`、订单列表去掉实收和找零。
- [ ] 补截图：商品页（有数据）、收款方式、销售账号、店铺资料、操作记录（有数据），平台 3 页；售罄、空数据、出错状态。三套风格 × 390/820/1280。
- [ ] 复跑测试报告 D1–D5 的复现步骤，逐条写“已关闭 / 未关闭”。
- [ ] 按展场/店面 spec §9（10 条）、外观 spec §9（7 条）、日报设计说明 §5 状态一览逐条打勾。
- [ ] 产出放 `docs/reports/` 新目录（测试执行时建），截图原图放 `/tmp`，只把挑选后的放进报告。

---

## 9. 后端缺口和要问 Stallink-后端开发 的问题

### 9.1 已核实的后端缺口（2026-11-26 11:51–12:14 UTC+8，以代码为准）

| 缺口 | 证据 | 本计划任务 |
|---|---|---|
| 店主没有读取自家店铺资料的接口，前端只能拿 slug 去调公开接口 | 路由只有 `PATCH /api/admin/shop`（`server/internal/server/http.go:81`）；`admin/shop/page.tsx:78` 读 `/api/public/shops/{slug}` | S2 |
| `shops` 没有客服联系、店铺级摊位号、默认货币 | `schema.go:14-23` | S1 |
| 店主保存资料时必须带 slug 并且会写回 | `admin.go:18-38` `UpdateShopSettings` → `UpdateShop`（`catalog_store.go:155-170`） | S2 |
| 平台侧只能在开店时填 slug，之后没有修改入口 | `OpenShop`（`admin.go:385-410`）；`SetPlatformShop`（`admin.go:432-452`）只改 `isEnabled` | K-Q9 |
| 操作记录只有对象类型和内部 ID，没有名称 | `paged_lists.go:51-80`、`admin.go:258-282` | S3 |
| `audit_logs` 没有 `(shop_id, created_at, id)` 索引，分页靠全表排序 | 后端 B-6 确认 | S1（024 加索引） |
| 操作记录时间不带时区 | `to_char(... 'YYYY-MM-DD HH24:MI')` | S4 |
| 平台外观不记录发布人和时间 | `platform_appearance` 只有 `singleton`、`theme`（`012`、`013`）；`data/platform_appearance.go:8-25` | S1、S5 |
| 现金实收字段还在接收、校验、返回，日报还在算长短款 | 见 S6 核实表 | S6 |
| CSV 表头“订单数”下写了件数；CSV 不导出优惠、长短款、总览只有已付款订单 | `service/reports.go:51-115` | S7、K-Q12 |
| 报表“今天”只有服务端知道，前端用浏览器日期 | `service/reports.go:117-135`、`conf/conf.go:77-83`；`reports/page.tsx:39` | S8、W17 |

### 9.2 要问 Stallink-后端开发 的问题（12:25 已全部答复）

| 编号 | 问题 | 后端答复（12:25 UTC+8） | 状态 | 对任务的影响 |
|---|---|---|---|---|
| B-1 | 店铺资料新字段 JSON 名和空值 | 用 `contact`、`boothNumber`、`defaultCurrency`、`updatedAt`；`defaultCurrency` 未设置返回 `null`，库里 `text NULL`，CHECK 只允许 MYR、SGD、CNY；`contact`、`booth_number` 列 `NOT NULL DEFAULT ''`；`updatedAt` 复用 `shops.updated_at`，RFC3339 带时区；订单币种照旧跟活动走 | 已答复 | S1、S2 按原写法，无改动；S2 注明 `defaultCurrency` 不影响下单；W14 字段名定稿 |
| B-2 | slug 和当前值不同时怎么处理 | `422 slug_readonly`，整条请求都不写 | 已答复 | S2 维持（先比对 slug 再写库）；K-Q3 = B（Kel 13:22 已定） |
| B-3 | 摊位号格式和错误码 | 最多 40 字、无控制字符、首尾无空格，否则 `400 invalid_shop` | 已答复 | S2 改为**首尾有空格直接 400、不替用户裁剪**，测试加“booth padded”“booth control”两例；错误码不拆细；W14 沿用 `invalid_shop` 文案 |
| B-4（修改） | 在售数口径 | 只统计当前活动里同时满足 `is_listed`、审核已通过、共享池可售量（`total_stock` − Σ`reserved_qty`）> 0 的 listing；没有当前活动返回 0；不算套餐 | 已答复 | S2 的 `onSaleCount` SQL 改成这个口径（加 `el.is_listed`、`el.shop_id`、共享池可售量） |
| B-5（修改） | 停用店铺的会话和登录 | PATCH 把 `isEnabled` 改成 `false` 时，改旗标和删会话必须在同一个 data 方法、同一个事务里；停用账号的 Login 仍会发 cookie；前端文案写“登录后进不了工作台”，不写“登录被拒绝” | 已答复 | S2：`UpdateShopProfile` 在同一事务里删会话，service 不再调 `SetShopEnabled`；集成测试改成“停用后旧 cookie 401、重新登录拿到 cookie 仍 401、平台重新启用后恢复”；“停用后店主仍能读资料”的旧说法删除；W14 加文案 |
| B-6（修改） | 审计关联查询性能和索引 | 024 加 `CREATE INDEX IF NOT EXISTS audit_logs_shop_created_id_idx ON audit_logs (shop_id, created_at DESC, id DESC);`，不用 `CONCURRENTLY`，down 里 `DROP INDEX IF EXISTS`；listing 子查询加 `el.shop_id = a.shop_id`；分页默认 20、最多 50；“现有索引”其实不存在 | 已答复 | S1 的升级/回滚 SQL 和迁移测试加索引；S3 的 SQL 加 `el.shop_id`，删掉“超过 200ms 再讨论加索引”；plan 里“现有索引”的说法已删 |
| B-7 | `detail.subject` 快照 | 读取用 `COALESCE(detail->'subject'->>'label', 关联名称, '')`，不新增列；快照写入放在修复 plan 任务 7，本 plan 不改 `insertAudit` | 已答复 | S3 维持；K-Q5 推荐 C 与之一致 |
| B-8 | `createdAt` 改格式 | 改成 RFC3339，`order_store.go` 里审计查询的 `to_char` 一起改 | 已答复 | S4 写明一并改 `order_store.go:864` |
| B-9 | 实收找零清理方案 | 采用方案 B，但接口继续接收 `cashReceivedMinor`（S9b 要用）；幂等哈希按整个请求体算，省略该字段和传等于总额的值哈希不同，会返回 `idempotency_conflict`；vNext W5 作废还是推迟并进 K-Q10 | 已答复 | S6 加幂等警告；W10 删字段要一次切换。**Kel 13:22 定了 K-Q10 = A（前后端一起清理），和 B-9“接口继续接收”冲突**：在线收款按 A 不再接收；离线同步（S9b）是否还传列为 K-Q17。**13:55 K-Q17 = A：S9b 也不传，接口忽略；B-9 的“继续接收”作废** |
| B-10 | 外观元数据并进 024 | 024 只对 `platform_appearance` 做 `ADD COLUMN`，`updated_by REFERENCES users(id) ON DELETE SET NULL`；写入、审计、缓存失效同一事务；不改 012、013 | 已答复 | S1、S5 维持并注明不改 012/013 |
| B-11 | 外观元数据怎么返回 | 同一接口按会话返回，匿名只返回 `{ theme }`；不建 `/meta`，不走 `a.platform()` | 已答复 | S5 维持，删掉“是否改成 /meta”的选项；W-A3 读同一接口 |
| B-12 | CSV 6 列 | 同意；`report_csv_test.go` 里 `row[4]` 改 `row[5]` | 已答复；**作废：13:55 C-1 = A，按 R2 出 5 列** | S7 按 R2 改写，需通知后端 |
| B-13 | “报表今天”接口 | 新增 `GET /api/admin/daily-reports/today` 返回 `{ date, timezone }`；时区读 `reports.timezone`，默认 `Asia/Kuala_Lumpur` | 已答复 | S8 按此写（实现仍待 K-Q14）；W17 读这个接口 |
| B-14 | vNext 多时区怎么取 | 本 plan 不加时区字段；给 vNext 的后续建议：依次取主推活动、进行中的活动、`starts_on` 最近的活动，都没有用 `Asia/Kuala_Lumpur` | 已答复 | S8 记录这条建议；已写进 vNext server plan 第八节；细则 B14-1、B14-2 Kel 13:22 都选 A，已同步 |
| B-15 | 迁移编号 | 合入时按重新 `ls` 的结果确定；024 可以先落地；已落地的编号不再改。后端两个 worktree 最高都只到 013 | 已答复 | S1 编号说明更新 |

**后端额外要求（已写进 S1）：** ~~024 按主推活动或最近的活动回填 `default_currency`~~（13:22 改为：024 只改结构，回填移到手动脚本 `server/internal/data/manual/024a_*`，K-Q2、K-Q16），回填不改 `shops.updated_at`；回滚 SQL 放 `migrations/down/`；同一个 PR 里对齐 `schema.go` 的 shops DDL；API 启动会自动执行 024（含回填 UPDATE），**云端升级前必须先拿到 Kel 的确认 ⚠️**。

## 10. 要问 Stallink-前端开发 的问题（12:21 已回复）

| 编号 | 问题 | 前端回复（12:21 UTC+8） | 本计划怎么处理 |
|---|---|---|---|
| F-1 | 总览卡片和侧栏编号按 spec 改成 10 张、01–10 吗？`cashierGroup` 有没有别的调用方 | 和 Kel 决定 #2 冲突：前端按 #2 做成 8 张卡、01–08。`cashierGroup` 只在 `workspace-shell.tsx` 里用（第 44 行定义，第 65、78 行使用） | 新增 K-Q15；W-C2 标“待 K-Q15” |
| F-2 | 店面侧栏整栏 `overflow-y: auto` 是临时方案吗 | 是临时方案 | 正式方案（导航区单独滚动、底栏固定）放进 W-S2 |
| F-3 | 展场 `--muted` 按 spec 改成 `#4b6465` 吗 | 同意，改完复测；现在 `#4f696a` 在 `#d5e6df` 上 4.553:1 | W-E1 |
| F-4 | 日报工具栏 1280 宽是否一行 | fixture 量过：1280、860 宽一行，手机两行；没有真实登录截图 | T1 加“真实登录，三套风格各截一张 1280” |
| F-5 | 指标卡几张、3×2 做了没有 | 6 张（`page.tsx:351-376`）；桌面 4+2，3×2 没做；≤720 已 2×3；**D1 已修**（≤900 `--workspace-rail: 0`、`min-width: 0`、`clamp`，`nowrap` 有意保留），390 实测不溢出 | W2 改为“D1 已修，待复测；桌面 3×2 待做” |
| F-6 | `payCommandBody` 等后端方案 B 上线再删 | 同意 | W10、S6 |
| F-7 | `--shop-muted: #726f6e` 要回填 spec 吗 | 从 `#787574` 调来达 AA：`#f2f4f5` 上 4.52、白底 4.98；前端会请 UIUX 回填 | W-S1 标“待 UIUX 回填 spec” |
| F-8 | 摊位号标签按 K-Q1 推荐改名 | 没有意见，等 K-Q1 | W14：K-Q1 = A（Kel 13:22），改名“默认摊位号”并加说明 |
| F-9 | 操作记录显示格式 | 同意：“类型 · 名称”，没有名称时“类型 #id”；现在回退是“订单 101”或“—” | W15 |
| F-10（新） | 12:22 磁盘上侧栏是 `00 / 01–04 / POS / ORD / 06–09`（`workspace-shell.tsx:59-70`），`/admin/orders` 仍叫“订单”（`:62`），卡片是 01–08（收款方式起 05–08），侧栏和卡片没对齐 | 不需要再问：Kel 12:40 定了 K-Q15 = A | **按方案 A 修**：总览、现场开单、本场订单不编号；管理项 01–08；`/admin/orders` 改叫“订单记录”。**12:45 磁盘核对：侧栏已改好**（`workspace-shell.tsx:59-70`），W-C2 测试锁定 |

## 11. Kel 的决定记录（2026-11-26 13:55 已全部定；K-Q18、K-Q19 15:01 定，K-Q20 15:11 定；K-Q21 15:13 定，K-Q22 15:15 定；这一期没有待 Kel 定的项；下期待定 K-Q23 见 §13.1）

### 11.1 本计划提出的问题

**K-Q1　店铺级摊位号和活动摊位号是什么关系？**
- A 店铺摊位号只作为**新建活动的默认值**，顾客端显示活动自己的摊位号，活动没填时回退到店铺摊位号。
- B 店铺摊位号就是长期固定摊位，顾客端一律显示店铺的，活动摊位号只做备注。
- C 不加店铺级摊位号，店铺资料页直接显示“当前活动的摊位号”（只读）。
- **推荐 A。** 同一家店在不同展会摊位号不同，活动级要保留；店铺级省去每次重填。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：A。** 落点：S9（新建活动带默认值、公开接口返回 `displayBoothNumber`）、W18（顾客端显示规则、新建表单预填）、W14（标签改名）。`events.booth_number` 已存在（`schema.go:54`），不用加进 024。

**K-Q2　默认货币用来做什么？存量店铺要不要回填？**
- A 作为新建活动的默认货币；存量按“推荐活动 → 最近活动 → 不填”回填。
- B 只做展示，不影响任何业务；不回填。
- C 作为新建活动的默认货币，但不回填，店主自己填。
- **推荐 A。** 回填是对云端数据的 UPDATE，执行前 ⚠️ 需 Kel 手动确认。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：A。** Kel 特别要求云端回填必须等他手动确认后才执行，所以回填不放进 024（024 随 API 启动自动执行），改成手动脚本 `server/internal/data/manual/024a_backfill_shop_default_currency.sql` + 回滚脚本（S1 步骤 5b）。手动脚本单独成文件和“一个需求一份 SQL”是否冲突见 **K-Q16**；回填规则 **Kel 13:55 已定：严格按最近一场活动，去掉“主推活动优先”**。新建活动默认货币见 S9、W18。

**K-Q3　店主保存资料时请求里带了 slug 怎么办？**
- A 静默忽略。
- B 和当前值相同就忽略，不同返回 `422 slug_readonly`。
- C 只要带了就返回 `400`。
- **推荐 B。** 不破坏旧前端（旧前端会带当前 slug），又能挡住真正的修改。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：B。** 落点：S2（测试第 2、3 段已按 B 写）。

**K-Q4　客服联系的格式？**
- A 自由文本 ≤120 字（微信号、电话、Instagram 都行）。
- B 拆成“类型 + 值”两个字段。
- C 只允许电话号码。
- **推荐 A。**
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：A。** 落点：S1（`shops_contact_check` ≤120 字符）、S2（校验 120 字符）、W14（`maxLength={120}`）。

**K-Q5　操作记录的对象名称怎么取？**
- A 查询时关联实时名称（改名后显示新名）。
- B 写入时快照进 `audit_logs.detail`（依赖修复 plan 迁移 014、任务 7）。
- C A 先上，B 随任务 7 补，读取时快照优先、没有快照用实时名称。
- 不推荐再加 `subject_label` 列：和 014 的 `detail`、vNext S16 的 `after_json` 重复。
- **推荐 C。**
- **Kel 已定（2026-11-26 13:55）：C。** 阶段 1 先用实时名称；修复 plan 任务 7 完成后补写入快照，读取时快照优先。落点：S3、W15。

**K-Q6　迁移 024 在云端执行（含 `audit_logs` 新索引）** ⚠️ 需 Kel 手动确认。API 启动时会自动执行 024（后端确认）。13:22 起 024 只改结构，`default_currency` 回填已移到手动脚本（K-Q2、K-Q16），手动脚本无论选哪个都必须 Kel 手动确认后执行。
- A 跟随下次部署自动执行（API 启动会自动跑迁移）。
- B 部署前由 Kel 先在云端备份，再手动执行，确认后部署。
- **推荐 B。**
- **Kel 已定（2026-11-26 13:55）：B。** Kel 先备份云端库，再手动在云端执行 024（和 024a），确认没问题再部署。开发**云端库一律不动**，024、024a 只在本机 docker-compose 的 Postgres 上跑。落点：S1 步骤 6、§14。

**K-Q7　日报“现金长短款”指标怎么处理？**（S6 的一部分）
- A 继续显示（新订单恒为 0）。
- B 隐藏，历史数据只在 PDF 里保留。
- C 删除指标。
- **推荐 B。** 收银台已经不录实收，这个数字对新订单没有意义。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：B，隐藏。** 落点：W10（屏幕上去掉“现金长短款”，PDF 行保留）、S6（后端继续按历史现金单计算 `cashOverShortMinor`，所以库里两列保留）。**和日报设计说明 R2/R3（CSV 和分币种汇总保留“现金长短款”）相反，列为冲突 C-2（§11.4）。**

**K-Q8　classic 圆角要不要回到 design-v2 的直角？**（展场/店面 spec Q7，12:44 重新编号前是 Q8）
- 见 11.2；本计划不含这项，推荐 A 但单独开 classic 任务。
- **Kel 已定（2026-11-26 13:19，Grok Bot 转达）：B，classic 保持圆角。** 不回直角，**不再开 classic 修正任务**；W-C1 不动 classic 圆角值。spec 方案 B 原文是“保留柔化，并更新 design-v2 稿”，design-v2 稿改成圆角是 UIUX 的后续项（§13）。

**K-Q9　平台要不要能修改店铺 slug？**
- A 不做，slug 开店后永久不变。
- B 平台店铺管理里加“修改地址”，旧地址做 301 跳转。
- C 平台可以改，不做跳转。
- **推荐 A。** 顾客手里有二维码和链接，改地址会让它们失效；真有需要再做 B。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：A，不做。** 本计划没有对应任务；S2 里注明 `SetPlatformShop` 保持只改 `isEnabled`。

**K-Q10　“实收 · 找零”下线后的遗留怎么清理？**（详见 S6）
- A 前后端一起清理（数据库列保留或删除）；日报去掉长短款；vNext S9b、W5 要改。
- B 只清理前端，后端改成“现金码不带就按总额记”，接口兼容。
- C 暂时不动。
- **推荐 B。** 后端（B-9）、前端（F-6）都已同意 B；接口继续接收 `cashReceivedMinor`（vNext S9b 要用）。注意幂等哈希按整个请求体计算，前端删字段要一次切换，否则重试会返回 `idempotency_conflict`。
- **子问题：vNext 前端 W5 / F-07“现金实收输入与找零计算器”怎么办？**——**Kel 已定：作废 W5（2026-11-26 12:40，Grok Bot 转达）。** 理由：和这一轮去掉实收、找零 UI 的方向相反。已在 `docs/superpowers/plans/2026-11-26-stallink-vnext-web.md` 的 W5 标题、任务表和状态处标“已作废（Kel 2026-11-26）”，并注明依赖它的 W8（前置依赖）和 W10（断网验收场景 1 的“实收现金”输入步骤）。**后端继续接收 `cashReceivedMinor`**（S9b 离线同步要用），这一条不变。
- **Kel 已定（2026-11-26 13:22，Grok Bot 转达）：A，前后端一起清理**（W5 已作废；涉及库表时一个需求只出一个升级 SQL 和一个回滚 SQL）。落点：S6（后端清理清单；**不动库、不出 SQL**，理由是 K-Q7 = B 要保留历史长短款）、W10（前端请求清理）。
- **和后端 B-9 冲突：** 后端认为 vNext S9b 离线同步还要用 `cashReceivedMinor`，接口要继续接收。在线收款按 A 不再接收；S9b 怎么办列为 **K-Q17**。

**K-Q11　(a) 日报的币种下拉只影响导出，不改变屏幕报表**（设计说明 §10 问题 1，以下直接引用设计说明的方案）
- A 维持现状，把下拉明确标成“导出币种”（设计稿按此画）。
- B 屏幕报表和 PDF 用同一套过滤，非主币种时件数显示“—”（同时要遵守问题 11：非主币种不借用主币种的件数和状态笔数）。
- **推荐 A。** 不改屏幕上看到的数字，风险最小；B 需要先定问题 11 的口径。
- **Kel 已定（2026-11-26 13:19，设计说明 R1）：A。** 可见标签和无障碍名称都是“导出币种”。落点：W16 第 5 条。
- 相关：设计说明问题 2“全部币种”在 CSV 和 PDF 含义不同（A 保持提示、文件名写主币种 / B CSV 也导出全部币种）。12:25 版推荐 B。**Kel 已定（2026-11-26 13:32，Grok Bot 转达）：A，维持现状**——“全部币种”时 CSV 仍只导出主币种，页面给提示，文件名写明主币种。核对代码：文件名由**服务端** `Content-Disposition` 决定（`reports.go:223-228`，currency 为空时取主币种），前端 `page.tsx:180-181` 读响应头、兜底格式相同。落点：S7（加测试锁住文件名和筛选，不改行为）、W16 第 6 条（提示文案，`page.tsx:316` 已有，保留并加测试）。设计说明问题 2 标为已定。

**K-Q12　(b) CSV 表头“订单数”实际写的是件数**（设计说明 §10 问题 3；**这是缺陷，修复列为 S7，P1**，只有列结构要 Kel 选）
- A 增加“件数”列：订单类行填订单数，商品行填件数。
- B 表头改成中性的“数量”，在“项目”里写明“件”或“笔”。
- C 维持列结构，商品行在“项目”末尾注明件数，订单数列留空。
- **推荐 A。** 两列含义各自唯一，透视不会算错。
- 补充（设计说明问题 4–6，同一次可选做）：CSV 补优惠金额、现金长短款（若 K-Q7 不选 C）、总览补订单数/件数/待退金额。推荐同一次补优惠金额和总览，长短款跟随 K-Q7。
- **Kel 已定（2026-11-26 13:19，设计说明 R2，UIUX 13:30 写进 §8.3）：表头 5 列 `分组,币种,项目,数量,金额`（相当于方案 B），并按屏幕指标补齐（问题 4–6 一起关闭）。** 和本计划推荐的 A（6 列，后端 B-12 已同意）不一致，**列为冲突 C-1**；S7 已按 R2 改写，B-12 的 6 列作废。“现金长短款”行和 K-Q7 = B 冲突，**列为 C-2**（§11.4）。

**K-Q13　(c) 分币种汇总里没有优惠金额**（设计说明 §10 问题 7；接口 `currencies[].discountMinor` 已有，只是页面没显示）
- A 汇总行补“优惠”列，同时给汇总表和活动表单独定义列模板。
- B 不补优惠（优惠只在指标卡和活动行出现），但仍要修列模板，避免列对不齐和窄屏溢出。
- **推荐 A。** 纯前端改动，接口已有字段；放在 W16。
- **Kel 已定（2026-11-26 13:19，设计说明 R3）：A。** 列顺序：币种、订单（含已收款）、优惠、待退、长短款、收款，独立六列模板；“长短款”列受 C-2 影响。落点：W16 第 4 条。

**K-Q14　(d) 前端和服务端对“今天”的判断不一致**（设计说明 §10 问题 8、9）
- A 默认日期由服务端下发“报表今天”（S8 新接口 + W17）。
- B 前端按固定报表时区算今天（需要把时区配置暴露给前端）。
- C 维持现状，只在页面注明“按店铺报表时区，默认 Asia/Kuala_Lumpur”。
- **推荐 A。** 只有一个地方算“今天”。和 vNext 对齐：vNext Task S6（迁移 `016_event_timezone_and_presale`）把时区定义在活动上（`events.timezone`，IANA），Task S8 日报按 `e.timezone` 切日；本计划不新增时区字段，接口返回的 `timezone` 也用 IANA 名称，vNext 合入后由 vNext 决定“报表今天”取哪个时区（B-14）。问题 9（日期 `max`）随 A 一起解决。
- **Kel 已定（2026-11-26 13:19，设计说明 R4，§8.4）：A。** 接口字段名按 §8.4 用 `today`；前端请求失败时**不**退回浏览器日期。落点：S8、W17。

**K-Q15　总览卡片的数量和编号：跟 Kel 决定 #2 走，还是跟 spec C8/C11 走？**（F-1 提出）——**Kel 已定：A（2026-11-26 12:40，Grok Bot 转达）**
- 背景：Kel 决定 #2 是“总览卡片跟侧栏走，补‘销售日报’‘操作记录’两张卡，编号连续”。前端据此做成 8 张卡、01–08（`admin/page.tsx:4-11`）。展场/店面 spec §8.1 C8、C11 要求总览项不编号、侧栏 01–10、10 张卡分“经营 / 现场收银 / 店铺”三组（多出“现场开单”“本场订单”两张）。
- A 保持 8 张卡、01–08，请 UIUX 改 spec（C8、C11、§9 第 2 条，并撤掉展场/店面原 Q5——12:44 已完成）。侧栏的管理项也改成 01–08 和卡片一致；“现场开单”“本场订单”两项作为单独的现场收银组，不编号；总览项不编号。
- B 按 spec 改成 10 张卡、侧栏 01–10，总览卡片不编号；现场收银两项占 05、06，后面顺延到 10。
- **推荐 A。** 理由：
  1. Kel 的原话点名只补“销售日报”“操作记录”两张，6 + 2 = 8，没有提现场开单、本场订单的卡片。
  2. “跟侧栏走”的重点是卡片和侧栏同序、同号、同名，A 只要把侧栏的管理项也编成 01–08 就满足。
  3. 现场收银两项是摊主和销售员共用的入口，总览横幅上已经有“打开现场收银”按钮。spec 原 Q5（已删除）里也承认，再加卡片会重复入口。
  4. 前端已经按 A 做完并通过测试，改动最小。
- **不论选哪个，侧栏现在都要改：** 12:22 磁盘上侧栏是 `00 / 01–04 / POS / ORD / 06–09`、“订单”没改名（F-10），和卡片 01–08 对不上。选 A 时侧栏改为 01–08、现场收银两项不编号；选 B 时按 spec C8。
- **结果：** Kel 选 A。总览 8 张卡、01–08；侧栏管理项改成 01–08 和卡片对齐；“现场开单”“本场订单”单独作为收银组、不编号；总览本身不编号。W-C2 已改为只按方案 A 实施（方案 B 的测试集已删），F-10 按方案 A 修。
- **依赖 UIUX：已完成（12:44）。** UIUX 已按方案 A 改了 spec：§5.2 编号规则，C8、C9、C11、E11、S10，§9 第 2 项；卡片 01 活动、02 商品、03 订单记录、04 销售日报、05 收款方式、06 销售账号、07 店铺资料、08 操作记录，分组经营 01–04、店铺 05–08，收银组两项不编号、没有卡片；删除原 Q5，原 Q6、Q7、Q8 改为 Q5、Q6、Q7；原型和 12 张截图已重新渲染。

**K-Q16　存量回填的手动脚本单独成文件，和“一个需求只出一个升级 SQL 和一个回滚 SQL”冲突吗？**（K-Q2 引出，13:22 新增）
- 背景：Kel 要求云端回填必须等他手动确认后才执行；024 会在 API 启动时自动执行，所以回填 UPDATE 不能放进 024。
- A 回填写成单独的手动 SQL 和回滚 SQL，放在 `server/internal/data/manual/`（不在 `//go:embed migrations/*.sql` 路径下，API 启动不会执行）；本机用 docker Postgres 验证；云端执行 ⚠️ 必须 Kel 手动确认（S1 步骤 5b、6）。这样这个需求有两组 SQL：024（结构）+ 024a（手动回填）。
- B 024 不做回填，也不出手动脚本；存量店铺 `default_currency` 保持 `NULL`，店主在店铺资料页填写时再补（新建活动时没有默认货币就用 `MYR`）。
- **推荐 A。** 024 本身仍是“一份升级 + 一份回滚”；手动脚本只是数据修正，不随启动执行，满足“云端回填等 Kel 确认”。选 B 的话删掉 S1 步骤 5b。
- **Kel 已定（2026-11-26 13:55）：A。** 024a 是单独的手动脚本，自带回滚。落点：S1 步骤 5b。

**K-Q17　离线同步（vNext S9b）还传不传 `cashReceivedMinor`？**（K-Q10 = A 和后端 B-9 的冲突，13:22 新增）
- 背景：Kel 定 K-Q10 = A（前后端一起清理）；后端 B-9 认为 S9b 离线同步还要用这个字段，接口要继续接收。
- A S9b 也不再传；同步接口和在线收款一样直接忽略这个字段（宽松解析，不报错），离线现金单按总额入账，两列写 `NULL`。
- B 同步接口继续接收，只校验非负，但**不存储**。
- **推荐 A。** W5 已作废，离线收银也没有录实收的界面；接收一个不存的字段只会让接口含义更乱。选定后要改 vNext server plan 的 S1、S9b（S6 里写了具体改法）。
- **Kel 已定（2026-11-26 13:55）：A。** S9b 不再传 `cashReceivedMinor`，接口忽略这个字段。已同步改 vNext server plan（Task S4 测试、Task S9b）和 vNext web plan（W5 作废说明、W8）。

### 11.2 UIUX spec 里的问题（全部已由 Kel 定，按 13:30 设计文件的编号）

编号说明：13:30 UIUX 把 Kel 13:19 的决定写进了三份设计文件，下表以文件里的实际编号为准（展场/店面 Q1–Q6、外观 Q9–Q12、日报 R1–R4 和问题 2、11）。外观 spec 13:30 已把原 Q1–Q4 改名为 Q9–Q12（文件里写着“Q9（原 Q1）”），本计划以前写的“外观 Q1–Q4”就是它们。

| 问题 | Kel 的决定 | spec 条目 | 本计划落在 |
|---|---|---|---|
| 展场/店面 Q1 展场工作台主色 | **A**：保留深青，共用炭黑 `#262820`、柠绿、衬线标题，去掉海军蓝和钴蓝 | E2、E6、E7、E12、E16–E20、**E25–E28** | W-E1（token、E25–E28）、W-E3（E12）、W-E4（E16–E21） |
| 展场/店面 Q2 展场顾客端手机竖栏 | **不改横条，67px 缩到 52px**；退出固定在竖栏底部，52 × ≥44px | §5.3、E22、E23、**E29、E30** | **W-E5**（13:19 列的冲突已由 13:30 spec 解决） |
| 展场/店面 Q3 展场 152px 窄侧栏 | **A 保留** | — | W-E2、W-E3（不动） |
| 展场/店面 Q4 店面字体 | **A 只用系统字体，不自托管 Inter** | — | W-S1（已完成）、W-E1 |
| 展场/店面 Q5 收银台标题 | **A “现场开单”**（H1 和 `<title>`） | C15 | W-C2 |
| 展场/店面 Q6 classic 圆角 = K-Q8 | **B 保留圆角**；design-v2 直角和实现圆角的差异不算验收不通过 | §2 第 16 条 | W-C1（不动 classic 圆角值） |
| 输入框边框对比度（13:30 对比度复核） | ~~展场工作台 `#738b86`、顾客端 `#857e6e`~~ **更正（UIUX 14:25）：这两个值 Kel 没有定过，13:30 版 spec 误标为“已定”，现改为待 Kel 定，并入 K-Q19（§11.7）** | E28、E29、C18、S4 | W-C3（先 `var(--line-strong)`）；W-E1、W-E5、W-S1 不断言色值；**15:01 Kel 定 K-Q19 = A：顾客端 `#857e6e`、工作台 `#6f8681`（不是 `#738b86`）、店面 `#8f8c8b`** |
| 外观 Q9（原 Q1）记录时间和操作人 | **记录 `updatedAt`、`updatedBy`**；GET 只对平台管理员返回，其他人只有 `{ theme }`（13:32 再确认） | §8.1、**A19** | S1、**S5（补三种身份测试）**、W-A3 |
| 外观 Q10（原 Q2）实时预览 | **不做** | — | W-A2 |
| 外观 Q11（原 Q3）全站断网横幅 | **本期不做**，和收银台离线一起另外排期 | — | §13.1；W-A1 只做本页 |
| 外观 Q12（原 Q4）读取失败提醒 | **只提醒平台管理员**；新增“读取降级提醒”状态和截图 `平台外观-读取降级提醒-桌面.png` | §4.2、§6、**A20–A22** | **W-A4** |
| 日报 R1 = K-Q11 | 下拉只管导出，标题“导出币种” | §4.3、§10 问题 1 | W16 |
| 日报 R2 = K-Q12 + 问题 4–6 | CSV 5 列、“数量”、按屏幕指标补齐 | §8.3 | S7（冲突 C-1、C-2） |
| 日报 R3 = K-Q13 | 分币种汇总补优惠，六列模板 | §4.5 | W16（C-2） |
| 日报 R4 = K-Q14 | 服务端下发 `today` | §8.4 | S8、W17 |
| 日报问题 2 | **A 维持现状**（13:32）：“全部币种”CSV 只导主币种，页面提示，文件名写主币种 | §10 问题 2 | S7（测试）、W16 第 6 条 |
| 日报问题 11 | **维持现状，用文案说明**（13:32） | §10 问题 11 | W16 第 7 条（前端） |

**核对结论：** 13:32 Kel 的第 1、2 条和设计说明问题 2、问题 11 的方案原文逐字对得上（问题 2 方案 A“保持提示，文件名继续写入主币种代码”；问题 11“维持字段范围，只在文案上说明”），**两题标为已定**，不进待澄清。请 UIUX 在设计说明里把这两题改成“已关闭”。

### 11.3 外部依赖（已全部关闭）

- ~~UIUX 的 classic 边框对比度~~ —— **已关闭（Kel 2026-11-26 14:13 选 A 采纳）**：UIUX 建议稿 `docs/design-v2/classic-边框对比度-建议.md`。交互控件边框 `#c6bdaa` → `#8f8675`，深色 `#40444f` → `#676c78`，用新 token `--line-control`（`--line-strong` 本身不改）；附录 5 个用 `--line` 的控件一起改用 `--line-control`。落点：**W-C3**。建议稿顶部已标“已采纳”。

已关闭的待澄清项：日报 R1–R4 对应关系（13:30 已解决）、展场竖栏 52px（13:30 已解决，见 W-E5）、回填是否保留“主推活动优先”（13:55 Kel 定：去掉）。

### 11.4 冲突（13:55 已全部由 Kel 定）

- **C-1 日报 CSV 列结构 = A：** 按 R2 出 5 列 `分组,币种,项目,数量,金额`，B-12 的 6 列方案作废。落点：S7。
- **C-2 现金长短款 = A：** 指标卡、分币种汇总、CSV 三处隐藏；**PDF 保留**（K-Q7 = B；Kel 14:21 再次确认，见 §11.6）。落点：S7（CSV）、W16（分币种汇总五列）、W2 / W10（指标卡改 5 张）。设计说明 §4.4、§4.5、§8.3 和截图由 UIUX 同步（通知，不阻塞开工）。

### 11.5 classic 边框对比度（14:13 Kel 已定）

- **Kel 已定（2026-11-26 14:13，Grok Bot 转达）：A，采纳 UIUX 建议。** 新增 `--line-control`：classic 浅色 `#8f8675`、深色 `#676c78`；建议文档 §3.1 的 39 处、§3.2 的 4 处，以及**附录 5 个用 `--line` 的控件一起换**成 `var(--line-control)`。另外 grep 多出 1 处内联样式（`admin/goods/page.tsx:61`）一并改。落点：W-C3。

### 11.6 日报 PDF 保留现金长短款（14:21 Kel 已定）；K-Q18（15:01 Kel 已定：A）

- **Kel 已定（2026-11-26 14:21，Grok Bot 转达）：PDF 保留现金长短款，只在指标卡、分币种汇总、CSV 三处隐藏。** 本计划原来就是这样写的（K-Q7 = B“历史只在 PDF 保留”，C-2 = A 注明 PDF 保留），这次把措辞统一，并给 W10、W2、W16、S6 补了“PDF 仍显示长短款”的验收和测试。**销售日报设计说明 §1 C-2、§10 问题 5、C-2 小节写的是“PDF 也不展示”，和这条相反，由 UIUX 改**（本计划不动设计说明）。
- **PDF 怎么生成（14:2x 只读核对）：** 浏览器端用 pdfmake 0.3.15 生成（`web/src/lib/reports/pdf/export-report-pdf.ts`、`pdf-definition.ts`），内容来自 `web/src/lib/reports/document-model.ts`：指标“现金长短款（币种）”（第 84、97 行）和分币种汇总的长短款单元格（第 109 行），数值都是快照里的 `cashOverShortMinor`，经 `overShortLabel()` 显示（0 显示“{金额} 持平”）。服务端不生成 PDF；`@media print` 只是浏览器打印样式，不是导出路径。
- **数据从哪来：** 服务端 `server/internal/data/report_store.go:143-144、178-179`，`cashOverShortMinor = sum(cash_received_minor) − sum(total_minor)`，只算 `cash_received_minor IS NOT NULL` 的订单。

**K-Q18　新订单不再录实收，PDF 里的“现金长短款”以后恒为“持平”，怎么显示？**（14:21 核对时发现；**Kel 2026-11-26 15:01 定：A**，后端 S10、前端 W10，接口约定见 S10）
- **矛盾：** K-Q10 = A、K-Q17 = A 之后，在线收款和离线同步都不再传 `cashReceivedMinor`，新订单的 `cash_received_minor` 一律是 `NULL`（Mac 工作区的 `order_store.go:340-343` 已经这样实现）。所以只有改动前的历史现金单还能算出长短款；**改动之后的日子，PDF 会一直显示“MYR 0.00 持平”**，看起来像“点过钱箱、刚好对上”，实际是“没有数据”。
- **A（推荐）** 区分“持平”和“没有数据”：后端在快照根和 `currencies[]` 里加 `cashTenderedOrders`（当天 `cash_received_minor IS NOT NULL` 的订单数，一个 `count(*) FILTER` 就够，不动库表）；前端 `overShortLabel` 在这个数为 0 时显示“—（未录实收）”，PDF 注释改成“只统计录过实收的历史现金单”。历史日照常显示长款 / 短款 / 持平。改动：后端 S6 加字段和测试，前端 W10 改 PDF 标签和测试；旧快照没有这个字段时，非 0 照常显示，0 显示“—”。
- **B** 维持现状：PDF 照常显示计算值，新日子一律“持平”；只在 PDF 注释里加一句“2026-11-26 起不再录实收，之后的日子恒为 0”。改动最小，但读者仍可能误读。
- **C** 恢复现金实收录入（推翻 K-Q10 = A、K-Q17 = A，W5 复活）：数据真实，但和本轮“去掉实收、找零”的方向相反，改动最大。不推荐。
- **推荐 A。** 只加一个计数字段，PDF 就不会把“没数据”说成“持平”，也不用恢复实收界面。~~Kel 定之前，S6 / W10 按现状实施（PDF 显示计算值），不做 A 的额外字段。~~ **已定 A（15:01）**：历史日期照常显示，不动数据库表。

### 11.7 展场和店面的控件边框色（K-Q19；15:01 Kel 已定：A）

**K-Q19　展场顾客端、展场工作台、店面的 `--line-control` 用什么值？**（建议文档 §5；spec 14:25 的 C18、E28、E29、S4、§4.2.1、§7.3、§9 第 11 项）
- **背景：** 三个界面现在的 `--line-strong` 都不到 3:1——展场顾客端 `#999281`（canvas 2.72）、展场工作台 `#8ca29d`（白 2.70，渐变最深处 2.09）、店面 `#ebebeb`（白 1.19）。W-C3 只把它们接成 `--line-control: var(--line-strong)`，颜色不变，所以 W-C3 之后仍不达标。
- **Kel 2026-11-26 15:01 定：A，三处都加深**：展场顾客端 `#857e6e`、展场工作台 `#6f8681`、店面 `#8f8c8b`；分割线和装饰框保持原色（`--line-strong` 不动）；店面“浅灰填充加描边”的备选**不采用**。落点：W-C3（改 token 块、断言色值和 ≥3:1）、W-E1 / W-E5 / W-S1（测试恢复断言具体值）。
- **UIUX 建议（原文）：** 展场顾客端 `#857e6e`、展场工作台 `#6f8681`、店面 `#8f8c8b`（店面也可以改成浅灰填充加 `#8f8c8b` 描边，请 Kel 一并定）。`--line-strong` 本身不动，继续给分割线用。
- **合并说明：** 13:30 版 spec 把展场输入框边框 `#738b86`（E28）、`#857e6e`（E29）误标为“Kel 已定”，UIUX 14:25 更正为待 Kel 定。这是同一个问题：顾客端建议值仍是 `#857e6e`，工作台建议值由 `#738b86` 换成 `#6f8681`（`#738b86` 在渐变最深处只有 2.81）。所以合并成 K-Q19 一个待定项，不另开编号。
- ~~Kel 定之前开发怎么做：~~（已定，下面这条作废，留作记录） 按 W-C3 默认做法，几个块里都是 `--line-control: var(--line-strong)`，颜色和现在一样；W-E1、W-E5、W-S1 的测试只断言 `--line-control` 存在，不断言色值；展场工作台 3 处表单控件按 E28 先改读 `var(--line-control)`；卡片悬停边框（13:55 第 358 行）保持 `var(--line-strong)`（Kel 14:40）。
- **定了之后怎么做（15:0x 重新 grep 的行号）：** 只改 `shop.css` 第 14 行（块起始第 3 行）、`exhibition.css` 第 42、885 行（块起始第 32、875 行）、`exhibition-workspace.css` 第 36 行（块起始第 20–21 行）的 `--line-control` 值；`.product-page` 块已被 W-E4 删掉，不加回来；测试改成断言具体色值；不用改选择器。原来写的“shop 13、exhibition 32/930、workspace 35”是 `--line-strong` 的旧声明行。

### 11.8 Kel 14:40 的四项决定（Grok Bot 转达；问题由 Stallink-前端开发 14:33 提出）

| 项 | 问题（前端原话的选项） | Kel 选 | 写进 |
|---|---|---|---|
| 1 | `exhibition-workspace.css` 第 358 行（13:55 快照；14:38 磁盘是第 390 行）总览卡片悬停边框：A 保持 `--line-strong`，B 改 `--line-control` | **A 保持 `var(--line-strong)`**；父代理 14:28 让前端改成 `--line-control` 的指令作废。另外 3 处（407、427、470，现 439、459、502）是筛选按钮选中态、已加购商品按钮、审核开关轨道，确认是控件，照改 | W-C3 第 4 条 E28、E28 测试（给卡片悬停加例外）、任务总表 W-C3、§11.7 |
| 2 | 分币种汇总底部说明：A 保持短句“件数仅统计主币种。”，B 换设计长句 | **B**：用设计说明 §4.5 第 185 行原句「售出件数、待付款、已关闭和退款笔数只按主币种 MYR 统计，不按其他币种拆分。」（`MYR` 取 `report.currency`）；PDF 的“件数仅统计主币种”注释不变 | W16 第 7 条、W16 `issue 11` 测试 |
| 3 | W-E5 展场窄屏「退出」：A 保留 390 顶栏里的退出，B 按 plan 改竖栏 | **B**：≤620（含 390）52px 竖栏，退出钉在竖栏底部；删掉 390 顶栏模式；之前“顶栏退出不得回退”作废 | W-E5（重写）、任务总表、§14.2 |
| 4 | 测试报告 `docs/reports/stallink-test-report-2026-11-26.html`（13:25 轮）的 D9（店面 ≤900 侧栏「退出登录」错位）、D10（缺 favicon，`/favicon.ico` 404）修不修 | **都让前端修** | 新任务 W19、任务总表、§14.2；D10 的图案没有现成文件，待 UIUX / Kel 给（不自己画） |

### 11.9 Kel 15:01 的五项决定（Grok Bot 转达；1–3 是前端 14:5x 提的 Q1–Q3）

| 项 | 问题 | Kel 选 | 写进 |
|---|---|---|---|
| 1 | 店面侧栏吸顶：spec S11 写 `top: 56px`，但店面顶栏 `min-height: 72px`（`shop.css:86-89`），56px 会让侧栏钻到顶栏下面 | **A：保持现在的 `top: 96px`**（`shop.css:1367-1369`，代码不用改）；S11 的 56px 作废，spec 由 UIUX 改 | W-S2 头部和测试、任务总表 |
| 2 | 店面窄屏「退出登录」：S11 要求 ≤900 隐藏底部操作，但 D9 是把它修好 | **A：保留**；S11“隐藏底部操作”作废，品牌、身份、分组标签照旧隐藏；**D9 只修白底和文字错位** | W-S2、W19（D9 测试断言没有 `display: none`）、任务总表 |
| 3 | 外观页发布确认框焦点：外观 spec 要求在「取消」，共用 `ConfirmDialog` 非危险操作时把焦点给「确认」 | **A：只有外观页放「取消」**；`ConfirmDialog` 加可选 `initialFocus`，默认行为不变，其他页不变 | W-A1 头部、测试、文件和拟提交 |
| 4 | K-Q18：新订单不录实收后 PDF 长短款恒为“持平” | **A**：快照加 `cashTenderedOrders`（`count(*) FILTER (WHERE cash_received_minor IS NOT NULL)`，根和 `currencies[]`），为 0 时 PDF 指标卡和分币种汇总显示「—（未录实收）」；历史日期照常；不动表、不加迁移 | 新后端任务 S10、W10、§11.6 |
| 5 | K-Q19：展场、店面控件边框不到 3:1 | **A**：展场顾客端 `#857e6e`、展场工作台 `#6f8681`、店面 `#8f8c8b`；`--line-strong` 不动；店面浅灰填充方案不采用 | W-C3（改值 + 断言 ≥3:1）、W-E1 / W-E5 / W-S1 测试、§11.7 |

**依据和找不到的：**
- 1–3 的原问题来自前端 14:5x 给 Grok Bot 的报告（“【请 Kel 定】Q1–Q3”），选项 A 都是前端推荐项；spec 依据：展场/店面 spec S11（第 508 行）、§6.3 第 357 行、§6.4 第 381、383 行；外观 spec §4.2 第 58、70 行，§6 第 132 行，§9 第 5 条。
- 第 1 条的 `≤900` 档：spec §6.4 第 383 行写“≤900 顶栏 56px”，代码里店面顶栏没有按宽度改高度（`min-height: 72px` 不分宽度）。Kel 只说“保持 96px”，**≤900 时顶栏是否也要改成 56px 没有依据，本计划不改**，由 UIUX 改 spec 时一并写清楚。
- spec 和设计说明里要跟着改的（UIUX 负责，本计划不动）：S11 的 `top: 56px` 和“隐藏底部操作”、§6.3 第 357 行“品牌和退出在这个宽度下由顶栏承担”；C18、E28、E29、S4、§4.2.1、§7.3、§9 第 11 项的“待 Kel 定”改成已定值；日报设计说明里 PDF 长短款“—（未录实收）”的显示。

### 11.10 favicon 图案（K-Q20；15:11 Kel 已定：A）

**K-Q20　D10 的 favicon 用哪个图案？**（`docs/design-v2/favicon/favicon-设计说明.md` §4、§8）
- **A** 「S.」字标：圆形炭黑 `#262820` 底、柠绿细描边，延续顾客端窄屏侧栏的「S.」。
- **B** 摊位图形：炭黑底、柠绿篷顶、纸色立柱、赭石号牌，不靠字母。
- **C（UIUX 推荐）** 字母块：深青 `#215e58` 实底、纸色「S」、右上角赭石方块加纸色描边，沿用展场工作台 logo；16px 最清楚，但偏工作人员侧。
- **Kel 2026-11-26 15:11 定（Grok Bot 转达）：A，炭黑圆底配青柠色「S.」，全站共用。** W19 的 D10 源文件全部改用 `option-a.svg`、`option-a-16.svg`、`option-a-apple.svg`、`option-a-maskable.svg`（15:1x 核对四个文件都在 `docs/design-v2/favicon/`），`apple-icon.png` 底色和 `manifest.ts` 的 `background_color` 都是 `#262820`；“定之前 D10 不动”的限制取消。
- ~~Kel 定之前：W19 的 D10 不动（不建图标文件、不写 `favicon.test.cjs`），D9 照常做。~~
- **Kel 定之后：** 只影响 W19 D10 的源文件（`option-x*.svg`）、`apple-icon.png` 的底色和 `manifest.ts` 的 `background_color`（C 为 `#215e58`，A / B 为 `#262820`）；输出文件清单、全站共用一个图标、本期不加 theme-color 都不变。

### 11.11 提交状态（Kel 经 Grok Bot 批准；15:11 转达）

- `design/editorial-frontend` 分支上有 **7 个后端 commit**：`fe899df`（S1 迁移 024）、`ee103c1`（S6）、`0c8383c`（S7、S8）、`1ab02db`（S3 阶段 1、S4）、`ca6b0ef`（S2）、`8a6dc26`（S5）、`4c1d42a`（S9）。**Kel 经 Grok Bot 批准（后端 Q1 选 A，14:43）**，保留，**不要 reset**，不 push。
- **S10 单独提交一次**：Kel 同意。内容是 `cashTenderedOrders`，读取旧快照时后端重算（`fillMissingCashTenderedOrders`），不写库。15:1x 只读核对，已提交为 `9f7e7f4`“feat(reports): add cashTenderedOrders to daily report snapshot (K-Q18=A)”，不 push。
- **通用规则补一句**（§1 开工规则第 4 条、§8.0、§13 第 5 条、§14 通用规则同步）：**只有 Kel 经 Grok Bot 明确批准的提交可以做，其余一律只暂存**；push 一律不做。各任务步骤里的“只暂存，不 commit、不 push”照旧。
- 前端 `web/` 目前没有获批的提交，仍然只暂存。

### 11.12 店面窄屏吸顶空带（K-Q21；Kel 15:13 选 B）

**K-Q21　店面侧栏吸顶定为 `top: 96px`（15:01 第 1 条）后，窄屏吸顶栏上方会露出空带，怎么办？**
- **Kel 2026-11-26 15:13 选 B（Grok Bot 转达）：吸顶位置跟着页头高度走。**
  - 宽度 >900：`top: 96px`；
  - 宽度 641–900：`top: 72px`，页头高 72px（Kel 原话引 `shop.css` 第 87 行）；
  - 宽度 ≤640：`top: 64px`，页头高 64px（Kel 原话引 `shop.css` 第 735 行，在 `@media (max-width: 640px)` 里）。
  - **页头高度本身不改。** spec 原 S11、§6.4 的吸顶 / 页头 56px **作废**：吸顶不得用 56px（不得出现 `top: 56px`，页头 `min-height` 也不得是 56px）。注意不是“全文件不得出现 56px”——`shop.css` 里和吸顶无关的 56px 保留（15:25 grep 核对）：`.catalog__box` 的 `min-height: 56px`（第 251 行）和 `padding: 4px 56px 4px 20px`（第 252 行）、`.event-card` 的 `grid-template-columns: 56px minmax(0, 1fr) auto`（第 2260 行）、`.event-card__date` 的 `width: 56px`（第 2270 行）、约第 2351 行 `.event-card` 日期列 `grid-template-columns: 56px minmax(0, 1fr)`。
  - 变量名统一用 `--shop-header-height`（前端已经在用）；UIUX 原型 `docs/design-v2/展场与店面-工作台-原型.html` 第 276、299、308、349 行里的 `--shop-header-h` 是旧名，以代码为准。
- **原来的问题：** 吸顶 96px 而页头 72px / 64px，窄屏吸顶栏上方露出约 24–32px 空带。
- **磁盘现状（15:1x 只读）：** 前端 15:14 已改成变量（未提交）：`--shop-header-height: 72px`（第 51 行），`@media (max-width: 640px)`（第 729 行起）里 `--shop-header-height: 64px`（第 732 行），页头 `min-height: var(--shop-header-height)`（第 88、740 行），≤900 吸顶 `top: var(--shop-header-height)`（第 1246 行），>900 `top: 96px`（第 1376 行）。值和 Kel 的决定一致；Kel 原话里的第 87、735 行是改成变量之前的行号。
- **spec 交叉引用（15:15 版，逐行核对过）：** §2 决定 17 第 63 行；§6.3 第 360 行；§6.4 第 384–387 行（≥1100、901–1099、641–900、≤640 四档）；S11 第 512 行（已写 `top: 96px`）；§9 验收第 14 项第 539 行。spec 第 63、386、387 行仍引用 `shop.css` 第 87、735 行的旧写法，前端改成变量后行号已漂，建议 UIUX 改成按变量名引用（不影响开发）。
- **层叠 bug（UIUX 15:2x 发现；15:28 grep 仍在，15:29 前端已删掉重复声明，未提交）：** `shop.css` 第 1237 行在 `@media (max-width: 900px)` 里重复声明 `--shop-header-height: 72px`，排在 ≤640 块之后，所有宽度页头都成了 72px。W-S2 已写修法、源码顺序测试和 390 / 320 / 768 实测验收。
- **spec / 原型新版：** spec md5 `822af34f3bb2131eef7c30a752c43e99`，原型 md5 `2bea9105e36ef9836908ae32db213328`，变量名都已统一成 `--shop-header-height`。
- **落点：** W-S2（实现分三档、测试、六个宽度验收）、任务总表、§14.2。
- ~~Kel 定之前：W-S2 仍按 `top: 96px` 做~~（已定）。

### 11.13 多币种时「—（未录实收）」按币种还是按整天判断（K-Q22；Kel 15:15 选 A）

**K-Q22　快照有多种币种时，PDF 里的「—（未录实收）」按每个币种各自判断，还是按整天判断？**
- **Kel 2026-11-26 15:15 选 A（Grok Bot 转达）：每种币各自判断。** 分币种汇总每行用该行的 `cashTenderedOrders`；指标卡用根对象（主币种），筛非主币种时用该币种行。前端现有写法和已提交的 S10 都不用改。（对应销售日报设计说明 15:17 版 §10 第 13 条，**已关闭**，第 540–546 行；第 12 条 K-Q18 已关闭；§4.6 在第 198–210、220–221 行；设计说明的开放问题已全部关闭）
- **原因：** 根对象的 `cashTenderedOrders` 和其他根字段一样只算主币种（`server/internal/data/report_store.go` `applyHeadlineCurrency`，约第 313 行 `report.CashTenderedOrders = top.CashTenderedOrders`），不是整天所有币种的合计。某个币种自己没有录了实收的单、但当天别的币种有时，两种判断结果不同。
- **A（推荐）按币种各自判断：** 分币种汇总每行用 `currencies[]` 该行的计数；指标卡用根对象（主币种），筛非主币种时用该币种行。前端 `document-model.ts` 现在就是这样（第 87、100、112 行）。前后端都不用再改。
- **B 按整天判断，前端相加：** 前端把 `currencies[].cashTenderedOrders` 加起来，为 0 时指标卡和分币种汇总每一行都显示「—」。只改前端 W10（`document-model.ts`、`report-pdf.test.cjs`），后端不改。
- **C 按整天判断，后端给合计字段：** S10 追加一个整天合计字段（新提交，需 Kel 再批准），前端 W10 改读它。前后端都要改。
- ~~Kel 定之前：S10、W10 按 A 写~~（已定 A，S10、W10 里的“选 B / C 要改哪里”已收成一句备注）。
- **导出非主币种时（产品经理对齐，K-Q22 = A 的直接推论，不是 Kel 新定的）：** 指标卡长短款的金额取该币种行的 `cashOverShortMinor`，「—（未录实收）」也用同一行的 `cashTenderedOrders` 判断，两边口径一致；选「全部」或主币种时用根对象；分币种汇总每一行用自己那一行；字段缺失的旧快照不当成 0，仍显示长款、短款或「RM 0.00 持平」。前端 `document-model.ts` 第 53–59、87、100、112 行已经这样做，不用改；W10 补一条测试。
- **和设计说明的差异：** 设计说明 §4.6 第 205、218 行暂按“整天判断”写，但第 207 行写明第 13 条待确认、不要把任一种写成已定。UIUX 15:17 已按 A 改好设计说明（§4.6 第 198–210、220–221 行，第 13 条已关闭）。

## 12. 自检

- [x] 本计划文件是主要产出；没有改代码。按 Kel / 父代理要求另外给 vnext-web、vnext-server、feature-list、prd-vnext 四份文档加过标注（12:40、12:44、13:22）。没有 commit、push；没有连接任何数据库，没有启动 API（13:22 只读查看了 Mac 上的源码）。
- [x] 数据库改动只有一份升级 SQL `024_shop_profile_and_appearance_meta.sql` 和一份回滚 SQL `down/024_shop_profile_and_appearance_meta.down.sql`（shops 三列 + platform_appearance 两列 + `audit_logs_shop_created_id_idx`）；同一个 PR 对齐 `schema.go`；云端执行标了 ⚠️（S1、K-Q6）。
- [x] 13:22 写入 Kel 对本计划问题的决定：K-Q1 = A（S9、W18、W14）、K-Q2 = A（024 只改结构，回填改手动脚本 `manual/024a_*`，S1 步骤 5b）、K-Q3 = B、K-Q4 = A、K-Q7 = B（W10）、K-Q9 = A、K-Q10 = A（S6 重写、W10）、B14-1/B14-2 = A（vNext server plan 第八节已同步）；新增 K-Q16、K-Q17；计划状态改为“定稿（待 Kel 最终 review）”。只读核对：`events.booth_number` 已存在（`schema.go:54`）；实收找零相关只有 `orders.cash_received_minor`、`cash_change_minor` 两列；收款接口用宽松 `readJSON`。
- [x] 14:25 按 UIUX 14:25 的 spec §4.2（C17、C18、E28、E29、S4）和建议文档 §5 对齐：W-C3 的主题块按 Mac 磁盘重新 grep 核实（5 块，和 C18 一致，`.product-page` 块按“W-C3 先于 E16 就加”的条件处理），写明 UIUX“40 + 附录 5”和本任务“48 + 1 内联”的对应，补 E28 工作台 4 处控件；`#738b86`、`#857e6e` 由“已定”改为待 Kel 定，和建议文档 §5 合并成 K-Q19。只读发现：前端 14:24 已在 Mac 上做了 W-C3 的大部分（未提交）。
- [x] 14:41 写入 Kel 14:40 四项（§11.8）：358 行卡片悬停保持 `--line-strong`、W16 长句、W-E5 竖栏底部退出、W19（D9、D10）。
- [x] 15:01 写入 Kel 15:01 五项（§11.9）：W-S2（吸顶 96px、窄屏保留「退出登录」）、W19 D9 只修错位、W-A1 确认框焦点（`ConfirmDialog` 可选 `initialFocus`）、新后端任务 S10 + W10 PDF「—（未录实收）」（接口约定写在 S10 和 W10）、W-C3 / W-E1 / W-E5 / W-S1 断言 K-Q19 色值；主题块行号 15:0x 在 Mac 上重新 grep；vNext 两份 plan 里没有长短款、`--line-control`、`ConfirmDialog`、吸顶相关内容，不用改。
- [x] 15:08 按父代理补充：§11.10 新增 K-Q20（favicon 图案，待 Kel 定）；W19 的 D10 按 `favicon-设计说明.md` §5–§7 重写（8 个输出文件、源文件、导出方法、manifest 字段、不加 theme-color、全站一个图标），K-Q20 定之前 D10 不动、D9 先做。
- [x] 15:11 写入：K-Q20 = A（W19 D10 源文件改 `option-a*`，四个文件已核对存在，去掉“定之前不动”）；§11.11 提交状态（后端 7 个 commit 保留、S10 已单独提交 `9f7e7f4`、只有 Kel 批准的提交可以做）；S10 补旧快照重算；W-C3 表的展场两行按选择器标清楚，写明 `.product-page` 块不要加回来、加回来也不能写 `var(--line-strong)`；§11.12 新增 K-Q21（店面窄屏吸顶空带，待 Kel），W-S2 先按 96px。
- [x] 15:14 新增 §11.13 K-Q22（多币种「未录实收」判断口径，待 Kel），S10、W10 按 A 写，并写明选 B / C 要改哪里；销售日报设计说明的引用更新为“第 12 条已关闭、第 13 条 = K-Q22、§4.6 第 198–218 行”，并记下设计说明暂按整天判断的差异。待定项：K-Q21、K-Q22。
- [x] 15:15 K-Q22 记为已定（Kel 15:15 选 A，每种币各自判断）；S10、W10 的“选 B / C 要改哪里”收成一句备注；待定项只剩 K-Q21。
- [x] 15:17 K-Q21 记为已定（Kel 15:13 选 B，吸顶跟页头高度走，56px 作废）；W-S2 实现、测试、六个宽度验收按三档改写；spec 第 63、360、384–387、512、539 行逐行核对存在；`shop.css` 的 87 / 735 行已因前端 15:14 改成变量而漂移，记录了新行号；新增下期待定 K-Q23（两个退出入口，§13.1）；W10、§11.13 补“导出非主币种时口径一致”（PM 对齐）；销售日报设计说明引用更新为 §4.6 第 198–210、220–221 行，第 13 条第 540–546 行已关闭。这一期没有待 Kel 定的项。
- [x] 15:25 按前端指出更正：W-S2 测试、任务头部、§11.12、任务总表里“56px 一律作废 / 全文件没有 56px”统一改为“吸顶不得用 56px（不得出现 `top: 56px`，页头 `min-height` 也不得是 56px）”，并列出 `shop.css` 里 5 处和吸顶无关、要保留的 56px（第 251、252、2260、2270、2351 行，grep 核对过）；写明变量名统一用 `--shop-header-height`，原型里的 `--shop-header-h` 是旧名。
- [x] 15:28 W-S2 补 UIUX 发现的层叠 bug：`shop.css` 第 1235 行 `@media (max-width: 900px)` 里第 1237 行重复声明 `--shop-header-height: 72px`，排在 ≤640 块（第 732 行 64px）之后，把 64px 盖掉（15:28 grep 确认仍在；15:29 再看前端已删，现在只剩第 51、732 行两处声明）；实现写明变量只在根块和 ≤640 块各声明一次，测试按源码顺序检查 ≤640 最终值 64px，验收加 390 / 320 = 64px、768 = 72px、桌面 `top: 96px` 写死；记下 spec 新版 md5 `822af34f…`、原型新版 md5 `2bea9105…`，变量名已统一。
- [x] 14:21 写入 Kel 的决定“日报 PDF 保留现金长短款，只在指标卡、分币种汇总、CSV 隐藏”：统一 §11.4、W10、W2、W16、S6 和 §14 的措辞，补 PDF 回归测试和验收；只读核对 PDF 生成方式（浏览器 pdfmake）和数据来源（`report_store.go` 按 `cash_received_minor` 算），发现新订单没有实收，列为 K-Q18（推荐 A）。vNext 两份 plan 里没有 PDF / 长短款 / 打印相关内容，未改。
- [x] 14:13 写入 Kel 对 classic 边框对比度的决定（A，附录 5 个控件一起换）：新增任务 W-C3（先在 Mac 上 grep `web/src` 核对建议文档的行号和范围；测试在 box 上对 Mac 源码副本试跑：现状 1 过 6 失败，模拟改完后 7 条全过）；§11.3 外部依赖关闭；§14.2 加入 W-C3；建议文档顶部标“已采纳”。
- [x] 13:55 写入 Kel 的最后一批决定：C-1 = A、C-2 = A、K-Q5 = C、K-Q6 = B、K-Q16 = A、K-Q17 = A、回填严格按最近一场活动；计划状态改为“已批准开工”，头部写入开工规则（云端库不动、024/024a 只在本机 docker-compose Postgres 跑、不建 worktree、不 commit、不 push）；新增 §14 开工安排；同步改 vNext server plan（Task S4、S9b）和 vNext web plan（W5 说明、W8）。
- [x] 13:30–13:32 按 UIUX 13:30 设计文件和 Kel 13:32 决定更新：展场 Q1–Q6、E25–E30（W-E1、W-E3、W-E5、W-C1、W-C2）；外观 Q9–Q12、A19–A22（S5 改 `updatedBy` 形状并补匿名 / 店铺会话 / 平台管理员三种身份测试；W-A3、W-A4 重写）；日报 R1–R4 对上 K-Q11～K-Q14 并标已定，S7 按 §8.3 改 5 列、S8 字段改 `today`、W16/W17 按 R1、R3、R4 改；日报问题 2、11 按 Kel 13:32 标已定（文件名归服务端，已有逻辑满足）；新增冲突 C-1、C-2；classic 边框对比度列为 UIUX 外部依赖。
- [x] 13:19 写入 Kel 对 UIUX spec 问题的决定：展场/店面 Q1 = A、Q2 = C（待澄清）、Q3～Q5 = A、Q6 = B（= K-Q8）；外观 Q1 = B、Q2 = A、Q3 = B（全站横幅移到 §13 后续项）、Q4 = B（新任务 W-A4）；日报“R1–R4”找不到对应，列为待澄清。同步 spec 13:09 的 Kel 已定规则（W-S2、W-C2、W-E2、新任务 W-E5）。
- [x] 12:44 UIUX 按方案 A 改完 spec：展场/店面 spec 引用改成新编号（原 Q6→Q5、Q7→Q6、Q8→Q7，原 Q5 已删）；W-C2 测试按 spec 卡片名称、顺序、分组重写（已在 box 上对 12:45 磁盘源码的副本试跑：侧栏 3 条通过，总览标题/分组、C10、C15 共 4 条按预期失败）；UIUX 依赖标为已完成。注意：K-Q7 是“日报现金长短款”，从来不对应 spec 的问题；收银台标题是 spec Q6（原 Q7），没有对应的 K-Q 编号。
- [x] 12:40 写入 Kel 的两项决定：K-Q15 = A（W-C2 只按 A 实施，删掉方案 B 测试集）；vNext W5 作废（vNext web plan 已标注，vNext server plan 已加 B-14 后续项）。`--shop-muted` 标为已解决。
- [x] 12:25 合入后端对 B-1～B-15 的答复，每项标“已答复”并写明对任务的影响（§9.2）。
- [x] 编号 024 已核实：本机 `migrations/` 最大 013，没有 `down/`；014 归修复 plan，015–023 归 vNext。
- [x] 修复 plan 任务 7（原因、一步取消并退款、迁移 014）只引用，没有重复（S0）；vNext S16 只引用，没有重复（S3）；vNext S6/S8 时区定义只对齐，不新增字段（S8）。
- [x] spec 的改动清单按节和条目范围引用，没有逐条抄；每个 web 任务都写了“前端已改”。
- [x] 日报更新稿已交付，没有再标“待 UIUX 更新稿”。
- [x] 没写进任何密钥。
- [x] 12:21 按前端对 F-1～F-9 的回复重核了状态：D1 改为“已修，待复测”（更正早先“D1 未修”的说法）；D2～D5、商品名换行、店面侧栏（临时方案）、`.header-loading`、订单列表去掉实收和找零标为“前端已完成、自测通过，待复测”。
- **找不到的信息：** 侧栏、卡片、日报之外的工作台页面（商品、收款方式、销售账号、店铺资料、操作记录、平台店铺管理、商品审核）没有展场/店面目标截图，交给 T1；前端说“Kel 关于工作台的 6 条决定”已做完，12:45 磁盘上侧栏已对齐（F-10 已修），总览 01、02 标题和分组还没改（W-C2）；前端本轮的结论来自前端自测，还没有测试 bot 复测。

## 13. 交接

1. **计划状态：已批准开工（Kel 2026-11-26 13:55）。** ~~待 Kel：K-Q18、K-Q19~~ **15:01 都已定 A（§11.9）**，K-Q20 15:11 定 A（§11.10）；K-Q21 15:13 定 B（§11.12），K-Q22 15:15 定 A（§11.13）；**这一期没有待 Kel 定的项**；下期待定 K-Q23 见 §13.1；没有外部依赖（classic 边框对比度 14:13 Kel 已采纳，见 W-C3）。
2. **开工顺序：** 见 §14。
3. **云端：** 开发不碰。Kel 自己备份后手动执行 024、024a，确认后部署（S1 步骤 6）。
4. **要通知的人：** 后端、前端各一份消息（父代理转达）；UIUX：按 C-2 = A 同步设计说明 §4.4（5 张指标卡）、§4.5（分币种五列）、§8.3（无长短款行）。UIUX 13:47 已完成：日报问题 2、11 标为已关闭（文件名 `daily-report-{date}-{currency}.csv`，§8.3）；外观 §8.1 写明 `updatedAt`、`updatedBy` 只给平台管理员；展场/店面 spec 注明本期不补顾客端原型；classic 边框对比度建议稿已交。**15:01 之后 UIUX 还要改：**S11（吸顶 96px、窄屏保留「退出登录」）、§6.3 第 357 行、K-Q19 的已定值（C18、E28、E29、S4、§4.2.1、§7.3、§9 第 11 项）、日报设计说明 PDF“—（未录实收）”；D10 的 favicon 方案 15:0x 已交，Kel 15:11 选 A（`option-a*`）；favicon 设计说明 §8 可标为已定。K-Q21 的问题由 UIUX 问 Kel。销售日报设计说明 15:12 已关闭第 12 条（K-Q18）、新增第 13 条（= K-Q22）；15:17 版已按 K-Q22 = A 改好（§4.6 第 198–210、220–221 行，第 13 条第 540–546 行已关闭，开放问题全部关闭）。展场/店面 spec 15:15 版已写入 K-Q21 = B（第 63、360、384–387、512、539 行）；建议把引用 `shop.css` 第 87、735 行的地方改成按 `--shop-header-height` 变量引用。
5. 每个任务做完只留在工作区，**不 commit、不 push、不建 worktree**，由 Kel 统一处理；**只有 Kel 经 Grok Bot 明确批准的提交可以做**（已批准：后端 7 个 commit、S10 `9f7e7f4`，见 §11.11）。

### 13.1 后续项（不在本期）

| 后续项 | 来源 | 说明 |
|---|---|---|
| **K-Q23（下期待定，不阻塞本期）** 店面窄屏两个退出入口是否收成一个 | 15:17 父代理转达；Kel 15:01 第 2 条、15:13 K-Q21 = B | 窄屏上现在有两个退出：顶栏的 `.header-logout`（`web/src/components/app-nav.tsx` 第 183 行）和吸顶栏右端的「退出登录」胶囊（W19 D9 修的那个）。**这一期按 Kel 的决定两个都保留**；以后可以考虑收成一个，下期再问 Kel |
| 云端执行 024、024a | K-Q6 = B、K-Q16 = A（Kel 13:55） | Kel 本人：先备份，再手动执行，确认后部署；开发不做 |
| 全站断网横幅（放进 `SiteFrame`，收银台优先）、外观实时预览 | 外观 spec Q11、Q10（Kel 13:19：单独排期 / 不做） | 和收银台的离线草稿策略、vNext 离线收银（S9a/S9b、W8）一起设计；本期只做外观页本页的横幅（W-A1 的 A5） |
| design-v2 稿改成 classic 圆角 | 展场 spec Q6 = B = K-Q8（Kel 13:19） | UIUX 负责；classic 代码不动 |
| 删除 `orders.cash_received_minor`、`cash_change_minor` 两列 | K-Q10 = A 的“列保留或删除” | 本期保留（K-Q7 = B 要历史长短款）；真要删，另开一个迁移，一份升级 + 一份回滚 |
| vNext 多活动店铺的时区取值 | B-14、B14-1/B14-2 = A | 已写进 vNext server plan 第八节，vNext S6/S8 合入后做 |

## 14. 开工安排（Kel 2026-11-26 13:55 批准）

**通用规则：** 云端库一律不动；024、024a 只在本机 `docker-compose` 起的 Postgres 上跑（`docker compose up -d postgres`，`127.0.0.1:5432`，集成测试用隔离 schema）；不建 worktree；不 commit、不 push（**只有 Kel 经 Grok Bot 明确批准的提交可以做，其余一律只暂存**，§11.11）。每个任务按 TDD：先写失败测试 → 实现 → 测试通过 → 列出改动文件留在工作区。

**通用验收（每个任务都要满足）：**
- 后端：`cd server && go test ./...` 全绿；涉及数据库的跑 `go test -tags integration ./internal/data ./internal/service -run <本任务用例>`（只对本机库）；`gofmt`、`go vet ./...` 无新增问题。
- 前端：`cd web && node --test tests/*.cjs` 全绿；`npm run lint`、`npx tsc --noEmit` 无新增问题；涉及页面的对照对应截图，三套风格 × 390 / 768 / 1280 没有横向滚动。

### 14.1 后端（Stallink-后端开发）

| 顺序 | 任务 | 依赖 | 可并行 | 验收标准 |
|---|---|---|---|---|
| 1 | **S1** 迁移 024（结构）+ 手动脚本 024a（回填 + 回滚）+ `schema.go` 对齐 | 修复 plan 任务 3 的 `migrations/down/` 和“down 不被加载”测试（没有就先在本任务里建 `down/` 目录并补这条测试） | 和 S4、S6、S7、S8 并行 | `TestMigration024*`、`TestShopProfile*`、`TestManual024aBackfill`、`TestManual024aRollback`、`TestLoadMigrationsIgnoresManualDirectory` 通过；024 不含 `UPDATE shops`；024a 严格按最近一场活动回填（主推活动不影响结果），再跑一次结果不变；回滚只撤销没被店主改过的店；**全部只在本机 docker 库上验证**，不启动 API 连任何非本机库 |
| 2 | **S2** `GET /api/admin/shop`、`PATCH` 扩字段 | S1 | 和 S3、S5 并行 | `contact` ≤120 字、`boothNumber` ≤40 字且首尾无空格、`defaultCurrency` ∈ MYR/SGD/CNY/null；`slug` 和当前值相同忽略、不同返回 `422 slug_readonly`；平台侧仍只改 `isEnabled` |
| 2 | **S3** 阶段 1：操作记录 `subjectLabel`（实时名称） | S1 的 `audit_logs_shop_created_id_idx` | 和 S2、S5 并行 | 列表每条有 `subjectLabel`；对象被删时为空字符串；读取写成 `COALESCE(快照, 实时名称, '')`，阶段 2 只需补写入 |
| 2 | **S5** 平台外观 `updatedAt`、`updatedBy` | S1；修复 plan 任务 10（同一事务写审计；没合入就先不写审计，留接口） | 和 S2、S3 并行 | 平台管理员 GET 得到 `{ theme, updatedAt, updatedBy: { id, name } \| null }`；匿名和店铺会话只得到 `{ theme }`（两个键都不出现）；同风格 PUT 也刷新 `updatedAt`；三种身份的集成测试都在 |
| 3 | **S9** 新建活动带店铺默认摊位号和默认货币；公开接口返回 `displayBoothNumber` | S1、S2 | 和 S3 阶段 2 以外的任务并行 | 三条新建活动路径没传摊位号、货币时用店铺默认值，传了用传的；店铺没有默认货币时仍按原逻辑；公开接口 `displayBoothNumber` = 活动摊位号，空时取店铺摊位号 |
| 任意 | **S4** 操作记录 `createdAt` 改 RFC3339 带时区 | 无 | 可以随时做 | 审计列表和 `order_store.go:864` 的时间都带偏移，前端解析不报错 |
| 任意 | **S6** 实收找零清理（在线收款；K-Q17 = A 一并适用于将来的 S9b） | 无（和前端 W10 同批上线） | 可以随时做 | 收款请求带 `cashReceivedMinor` 时**忽略、不报错**；`orders.cash_received_minor`、`cash_change_minor` 新单写 `NULL`；**不动库、不出 SQL**；日报继续按历史现金单算 `cashOverShortMinor`（供 PDF）；不再返回 `invalid_cash_received` |
| 任意 | **S7** 日报 CSV 5 列 | 无 | 可以随时做 | 表头正好 `分组,币种,项目,数量,金额`；总览、按币种、按活动（含“{活动名} 优惠”行）、按商品、按收款方式按 §8.3；**没有“现金长短款”行**；不选币种时只导主币种、文件名 `daily-report-{date}-{主币种}.csv` |
| 任意 | **S8** `GET /api/admin/daily-reports/today` | 无 | 可以随时做 | 返回 **`{ today, timezone }`**（字段叫 `today`，不是 `date`）；未登录 401、无权限 403；`today` 等于报表时区的当天；读、生成、导出拒绝晚于 `today` 的日期 |
| 已提交 `9f7e7f4` | **S10** 快照加 `cashTenderedOrders`（K-Q18 = A，Kel 15:01） | S6（已提交 `ee103c1`） | 可以随时做；和前端 W10 的 PDF 部分对接 | 根和 `currencies[]` 都有 `cashTenderedOrders`，0 也输出；历史实收单 + 新单那天计数 1、只有新单那天计数 0；`cashOverShortMinor` 不变；不加迁移；集成测试只连本机 docker 库；多币种按币种各自判断（K-Q22 = A，Kel 15:15），不加整天合计字段 |
| 后续 | **S3 阶段 2** 写入快照 | 修复 plan 任务 7（迁移 014 的 `audit_logs.detail`） | — | 新审计写 `detail.subject.label`；读取时快照优先 |

**容易踩的坑：** 日报 PDF 保留长短款（Kel 14:21），快照的 `cashOverShortMinor` 继续算、继续返回，只有 CSV 不写；S10 的 `cashTenderedOrders` 不能加 `omitempty`（前端靠“0”和“字段不存在”区分新日子和旧快照）；S8 字段叫 `today`；`updatedBy` 是 `{ id, name }` 对象，`name` 取 `display_name`，空时取 `login_name`，而且只给平台管理员；CSV 是 5 列，B-12 的 6 列作废，测试里金额列仍是 `row[4]`；长短款在 CSV 里不出现，但快照字段 `cashOverShortMinor` 继续算（PDF 要用）；K-Q17 = A，vNext S9b 实现时 `OfflineSyncInput` 不要 `CashReceivedMinor`，传了也忽略；024 里不能有任何 `UPDATE`，回填只在 024a；`manual/` 目录不能被 `//go:embed migrations/*.sql` 带上。

### 14.2 前端（Stallink-前端开发）

| 顺序 | 任务 | 依赖 | 可并行 | 验收标准 |
|---|---|---|---|---|
| 1 | **W-C2** 侧栏编号、总览卡片、页面名称 | 无 | 和 W-C1、W2、W16、W-E*、W-S*、W-A1、W-A2、W-A4 并行 | `web/tests/workspace-nav.test.cjs` 全部通过：8 张卡 01–08 与侧栏同序同号同名，分组经营 01–04、店铺 05–08；总览、现场开单、本场订单不编号；收银台 H1 和 `<title>` 是“现场开单” |
| 1 | **W-C1** token、焦点环、减少动效 | 无 | 同上 | C1、C3–C7 测试通过；classic 圆角不动 |
| 1 | **W2** D1 复测 + 指标卡 5 张 | 无 | 同上（和 W10 的日报部分改同一处，谁先做谁改） | 页面无“长短款”，**PDF 仍有“现金长短款”**（`report-pdf.test.cjs` 通过）；桌面 2+1 / 3，≤720 收款总额整行 + 2×2；三档无横向滚动 |
| 1 | **W-C3** classic 控件边框 `--line-control` | 无（不依赖后端）；建议在 W-C1 之后（同改 `globals.css` `:root`） | 和 W2、W16、W-E*、W-S*、W-A* 并行；如果 W-E1、W-E5、W-S1 先合入，那几块已有自己的 `--line-control`，W-C3 就不再给它们加 | `tests/classic-line-control.test.cjs` 全过；classic 浅色 `#8f8675`、深色 `#676c78` 在所有相邻背景上 ≥3:1；48 处 CSS + `admin/goods/page.tsx:61` 内联样式改用 `--line-control`；分割线、hover、禁用态不变；shop、exhibition 控件边框计算值前后一致；展场工作台 4 处控件改读 `--line-control`（E28）；主题块的色值等 K-Q19，先用 `var(--line-strong)`；`exhibition-workspace.css` 只有 `.workspace-card:hover`（卡片悬停，Kel 14:40）可以留 `var(--line-strong)`；K-Q19 = A（15:01）：店面 `--line-control: #8f8c8b`、展场顾客端两块 `#857e6e`、展场工作台 `#6f8681`，各背景 ≥3:1，`--line-strong` 不变 |
| 1 | **W16** 日报 v2：标题、工具栏、热销四列、分币种五列、提示文案 | 无 | 同上 | 可见标签“导出币种”；屏幕分币种五列（币种、订单、优惠、待退、收款），**没有长短款**；PDF 分币种汇总**仍有长短款**；“全部币种”提示保留；非主币种说明“件数仅统计主币种”；1280 工具栏一行；分币种汇总下方用设计长句「售出件数、待付款、已关闭和退款笔数只按主币种 {主币种} 统计，不按其他币种拆分。」（Kel 14:40） |
| 1 | **W-E1 → W-E3**、**W-E2**、**W-E4**、**W-E5** | W-E3 依赖 W-E1；W-E2 依赖 W-C2 的 C10 | 展场内部按依赖，其他并行 | 按 E 条目断言通过；去掉 `#0e203a`、`#22467c`、`#1e36a8`；`--expo-charcoal`、`--line-control` 存在；≤620（含 390）竖栏 52px，退出在竖栏底部，点击区 ≥44px 高，390 不再有顶栏模式（Kel 14:40 选 B）；W-E5 排在 W-E1 之后 |
| 1 | **W-S1～W-S4** | W-S2 依赖 W-C2 的 C10 | 并行 | 按 S 条目断言通过；店面窄屏横滑；只用系统字体；W-S2：吸顶跟页头高度走（>900 `96px`、641–900 `72px`、≤640 `64px`，K-Q21 = B），1280 / 900 / 768 / 640 / 390 / 320 吸顶栏上方都不露空带；窄屏「退出登录」保留（Kel 15:01） |
| 1 | **W19** 测试报告 D9、D10（Kel 14:40） | 无；D9 和 W-S2 改同一块 `shop.css`，谁后做谁合并；D10 用 `option-a*`（K-Q20 = A，Kel 15:11） | 并行 | 店面 860 / 390「退出登录」保留、文字在胶囊正中、点击区 ≥44×44、导航下方无空块；D10：8 个文件齐、`manifest.webmanifest` 字段对、无 theme-color、无痕窗口首页控制台没有 `favicon.ico 404`；D9、D10 各一条拟提交 |
| 1 | **W-A1**、**W-A2**、**W-A4** | 无 | 并行 | 外观页状态流程按 A2–A11、A17、A18；缩略图固定示例色；读取降级提醒只对平台管理员显示，顾客端不渲染；外观页发布确认框打开时焦点在「取消」，其他页 `ConfirmDialog` 焦点不变（Kel 15:01） |
| 2 | **W10** 收款请求去掉 `cashReceivedMinor`；日报屏幕隐藏长短款 | S6（同批上线）；PDF“未录实收”依赖 S10 | 和其他任务并行，上线跟 S6 | `payCommandBody` 只返回 `{ payCodeId }`；订单页没有“实收”“找零”；日报屏幕没有长短款；**PDF 仍显示长短款**（`report-pdf.test.cjs` 通过）；`invalid_cash_received` 文案和 S6 同次删除；`cashTenderedOrders` 为 0 时 PDF 指标卡和分币种汇总显示「—（未录实收）」，字段不存在（旧快照）照常显示数值（K-Q18 = A）；多币种按币种各自判断（K-Q22 = A，Kel 15:15）；导出非主币种时金额和判断都用该币种行（PM 对齐） |
| 2 | **W14** 店铺资料页 | S2 | 后端 S2 好了就做 | 三个字段可编辑；`contact` `maxLength=120`；`slug` 只读；`422 slug_readonly` 有中文提示 |
| 2 | **W15** 操作记录对象列 | S3 阶段 1 | 后端 S3 好了就做 | 显示“类型 · 名称”，没有名称时“类型 #id”；不区分快照还是实时名称 |
| 2 | **W17** 日报默认日期 | S8 | 后端 S8 好了就做 | 读 `today`（不是 `date`）做默认值和 `max`；请求失败显示错误和重试，**不退回浏览器日期** |
| 2 | **W-A3** 外观状态条 | S5 | 后端 S5 好了就做 | “上次发布：{updatedBy.name} · {时间}”；没有记录写“尚无记录”；PUT 成功显示“你 · 刚刚” |
| 3 | **W18** 顾客端摊位号；新建活动表单预填 | S9、S2 | 后端 S9 好了就做 | 顾客端只用 `displayBoothNumber`；新建活动表单预填店铺默认摊位号和货币，可以改 |
| 最后 | **T1** 补测、补截图 | W 系列 | — | 测试报告 D1～D5、D9、D10 复测关闭；补齐 §1 列的页面截图 |

**容易踩的坑：** S8 返回字段是 `today` 不是 `date`；`updatedBy` 是 `{ id, name }` 对象，而且只有平台管理员能拿到，别的身份 GET 只有 `{ theme }`，状态条要按字段不存在处理；CSV 由后端生成、5 列，前端只下载，文件名读 `Content-Disposition`；长短款只在指标卡、分币种汇总、CSV 三处隐藏，**PDF 保留**（Kel 14:21）：屏幕隐藏只改 `admin/reports/page.tsx`，别动 `lib/reports/document-model.ts` 里的长短款，`report-pdf.test.cjs` 要继续断言 PDF 有“现金长短款”；W-C3 换成 `var(--line-control)` 时，shop、exhibition 每个重新定义 `--line-strong` 的块都要同时写 `--line-control`，否则它们会继承 classic 的 `#8f8675`；K-Q17 = A，vNext W8/W9 的离线单和同步请求都不带 `cashReceivedMinor`；W10 的请求清理要一次切换（幂等哈希按整个请求体算），和 S6 同批上线。
