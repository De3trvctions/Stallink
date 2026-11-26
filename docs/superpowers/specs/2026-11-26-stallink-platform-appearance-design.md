# Stallink 平台外观页设计规格（/platform/appearance）

- 日期：2026-11-26
- 状态：Kel 已审（2026-11-26 13:19），开放问题已全部关闭（本文只定设计，不改 `web/`、`server/` 代码）
- 姊妹文档：[`2026-11-26-stallink-theme-exhibition-shop-design.md`](./2026-11-26-stallink-theme-exhibition-shop-design.md)（三套风格的 token 表、导航编号规则）
- 可交互原型：`docs/design-v2/平台外观-原型.html`。用 `?state=` 切换十一种状态（第 11 种 `fallback` 是读取降级提醒，Kel 已定（2026-11-26））；底部有原型控制条，加 `&shot=1` 可以隐藏
- 截图（`docs/design-v2/screens/`）：
  - `平台外观-桌面.png`、`平台外观-手机.png`：已选未发布
  - `平台外观-读取中-桌面.png`、`平台外观-读取失败-桌面.png`
  - `平台外观-发布确认-桌面.png`、`平台外观-发布中-手机.png`、`平台外观-发布失败-手机.png`
  - `平台外观-无权限-桌面.png`、`平台外观-断网-手机.png`、`平台外观-已发布-手机.png`
  - `平台外观-读取降级提醒-桌面.png`（新增，Kel 已定（2026-11-26），Q12）
- 代码基线：2026-11-26 磁盘上的 `web/src/app/platform/appearance/{page.tsx, appearance.css}`、`web/src/lib/api.ts` 的 `getStorefrontTheme()`、`server/internal/service/platform_appearance.go`

---

## 1. 背景

平台管理员在这一页选择全站风格（classic / exhibition / shop）。选好之后，顾客端、摊主工作台、现场收银和平台工作台都会换掉。

**现状（读代码得出）**：
- `GET /api/platform/appearance` 返回 `{ theme }`，不需要登录。`PUT` 要求平台管理员，返回 `{ theme }`。没有 `updatedAt` 和 `updatedBy`，也不写操作记录，后写的覆盖先写的。
- 根布局 `layout.tsx` 每次请求都在服务端读取风格（`no-store`，2.5 秒超时）。**读取失败时静默退回 classic**，保证网站能打开。
- 页面现在有三张单选卡片和一个「发布所选风格」按钮。已经处理了「正在读取」「读取失败加重试」「发布中…」「已发布」四种文字状态。发布成功后调用 `router.refresh()`，**本页会立刻换成新风格**，其他已打开的页面要刷新后才生效。
- 发布时没有二次确认，但这个操作会影响所有正在用的人。
- 断网没有专门的提示，只会在操作失败后显示「无法连接服务」。
- 401 和 403 只显示一行文字，没有下一步该怎么做。
- 缩略图只有顾客端一面，看不出工作台换成什么样。店面缩略图写死了 `#5433eb` 和紫色阴影。

## 2. 目标和不做

**目标**：管理员知道现在用的是哪套风格、新风格会影响哪些页面、什么时候生效；每个异常都有明确的下一步；读取失败时，不会因为页面以为现在是 classic 而误覆盖别人刚发布的设置。

**不做**（YAGNI）：
- 不做「发布前在前台实时预览」（Kel 已定（2026-11-26），Q10）。靠双面缩略图，加上发布后随时可以改回
- 不做定时发布，不做按店铺设置不同风格
- 不做全站断网横幅（Kel 已定（2026-11-26），Q11）：断网横幅这期只在本页做；全站横幅跟收银台的离线策略一起单独排期
- 不做并发比对（`expectedTheme` / 409）和外观操作写入操作记录

**本期要做的接口改动（Kel 已定（2026-11-26），Q9）**：切换外观时记录 `updatedAt` 和 `updatedBy`。**需要后端提供 `updatedAt`/`updatedBy`**，只返回给平台管理员，其他人只拿到 `{ theme }`（Kel 已定（2026-11-26）），见 §8.1。

## 3. 方案比较

| 方案 | 做法 | 优点 | 缺点 |
|---|---|---|---|
| **A. 单选卡片 + 发布确认（采用）** | 先选卡片（只改本地状态），点「发布所选风格」后弹出确认框，确认后才提交 | 跟现在的交互一致，改动小。影响全站的操作有一次确认 | 多点一次 |
| B. 点卡片就立即生效 | 选中就 PUT | 最快 | 误点就会切换全站，现场收银的人会被换风格，风险高 |
| C. 先预览再发布 | 先在新标签页里用临时风格看前台，满意了再发布 | 最直观 | 需要后端和根布局支持临时风格。**这期不做（Kel 已定（2026-11-26），Q10）** |

## 4. 用户流程

### 4.1 主流程

1. 平台管理员在侧栏点「03 平台外观」。
2. 页面读取当前风格（读取中：骨架卡片，三张卡和按钮都不可用）。
3. 读取成功：状态条显示「当前使用：现有风格」；对应卡片有绿色「使用中」标签，单选已选中；「发布所选风格」不可用，旁边提示「选择与当前不同的风格后才能发布。」
4. 管理员点「展场风格」卡片：卡片出现强调色描边和「已选 · 未发布」标签；「发布所选风格」变成可用；出现文字按钮「还原为当前风格」。
5. 点「发布所选风格」→ 弹出确认框「把全平台换成『展场风格』？」，列出影响范围、生效时间和可以改回。焦点默认在「取消」上（Kel 已定 2026-11-26 15:01）。
6. 点「确认发布」→ 弹窗关闭，按钮变成「发布中…」加转圈，卡片锁住，`aria-busy="true"`。
7. 成功：本页随 `router.refresh()` 换成新风格；页面上出现绿色提示「已发布『展场风格』」，说明生效方式，并带「在新标签页查看前台 ↗」；「使用中」标签移到新卡片；焦点移到这条提示，读屏会播报。

### 4.2 状态表（每种状态都在原型里有对应画面）

| 状态 | 什么时候出现 | 界面 | 文案 | 能做什么 | 读屏 |
|---|---|---|---|---|---|
| **读取中** | 进入页面或重新读取 | 状态条换成转圈和文字；三张卡片换成骨架（减少动效时不闪）；`fieldset disabled` | 「正在读取当前风格…」 | 无 | `role="status"` |
| **读取失败** | GET 失败或超时；或返回了不认识的风格 | 不显示状态条和「使用中」标签，卡片都不选中、不可用；卡片下方是错误提示块；不显示发布按钮 | 标题「没能读取当前风格」；正文用 `describeRequest()` 的文案（例如「无法连接服务，请检查网络后重试。」），再加一句「读取成功前不能发布，以免覆盖别人刚发布的设置。」 | 「重新读取」 | `role="alert"` |
| **已读取** | GET 成功 | 状态条显示当前风格和最近一次发布记录（Kel 已定（2026-11-26），Q9）；对应卡片选中，带「使用中」 | 「当前使用：{风格名}」「上次发布：{updatedBy.name} · {updatedAt}」（从没发布过、两个字段为 `null` 时写「上次发布：尚无记录」）「发布后，顾客端和所有工作台在下次打开或刷新页面时切换。」 | 选卡片；查看前台 | — |
| **已改未发布** | 选中的跟当前不同 | 选中卡片带「已选 · 未发布」；发布按钮可用；出现「还原为当前风格」 | — | 发布、还原 | 卡片的 `aria-describedby` 指向它的说明 |
| **发布确认** | 点「发布所选风格」 | 复用 `ConfirmDialog`（原生 `<dialog>`，`danger=false`）；**焦点默认在「取消」上**（Kel 已定 2026-11-26 15:01：只在本页传 `initialFocus="cancel"`；`ConfirmDialog` 的默认规则不变，其他页面不受影响）；Esc 等于取消 | 标题「把全平台换成『{风格名}』？」；要点：①顾客端、摊主工作台、现场收银和平台工作台都会换成新风格 ②已打开的页面不会被打断，刷新或重新打开后生效 ③可以随时回到这里改回「{当前风格名}」 | 取消 / 确认发布 | 对话框 `aria-labelledby` 指向标题 |
| **发布中** | 已确认，PUT 请求还没返回 | 按钮显示转圈加「发布中…」并禁用；卡片锁住；确认框不能用 Esc 关（`pending`） | 按钮旁边提示「正在发布，请勿关闭页面。」 | 无 | 隐藏的 `aria-live` 播报「正在发布{风格名}」 |
| **发布失败** | PUT 返回 4xx、5xx 或网络错误 | 操作区上方显示错误提示块；保留管理员的选择 | 标题「发布没有成功，前台仍是『{当前风格名}』」；正文用 `describeRequest()` 的文案，再加「你的选择已保留。」 | 「重新发布」（直接重发，不再弹确认框）、还原 | `role="alert"`，焦点移到提示块 |
| **登录过期（401）** | GET 或 PUT 返回 401 | 跟无权限用同一个整页提示块 | 「登录已过期，请重新登录。」 | 「重新登录」（跳到 `/login?next=/platform/appearance`） | `role="alert"` |
| **无权限（403）** | PUT 返回 403；或会话被降权。正常情况下非平台管理员会被 `WorkspaceShell` 直接跳到自己的首页 | 整页提示块代替卡片区 | 「只有平台管理员可以修改平台外观」「你的账号没有这项权限，或者登录身份已变更。请换用平台管理员账号登录。」 | 「重新登录」「返回 Stallink」 | `role="alert"` |
| **断网** | `navigator.onLine === false`，或收到 `offline` 事件 | 页面顶部深色横幅；发布按钮禁用；保留选择；卡片区下方一块说明 | 横幅「网络已断开，恢复后会自动重新读取。」；说明「当前离线，暂时不能发布。你的选择已保留……」 | 还原、浏览 | `role="status"` |
| **恢复联网** | 收到 `online` 事件 | 横幅消失，自动重新读取；如果当前风格已经被别人改过，要更新「使用中」，并在状态条提示「当前风格刚刚被更新为『X』」 | — | — | `aria-live="polite"` |
| **已发布** | PUT 成功，返回值能被识别 | 本页已经换成新风格；绿色成功提示；状态条的「上次发布」用 PUT 返回的 `updatedBy`/`updatedAt` 更新成「你 · 刚刚」 | 标题「已发布『{风格名}』」；正文「本页已换成新风格。顾客和工作人员下次打开或刷新页面时生效，正在收银的页面不会被打断。」 | 「在新标签页查看前台 ↗」 | `role="status"`，焦点移到提示块 |
| **读取降级提醒**（Kel 已定（2026-11-26），Q12） | 根布局服务端读取外观失败或超时（2.5 秒），全站退回 classic；**只对平台管理员显示**，顾客、摊主、销售员都看不到 | 平台工作台所有页面（`/platform/*`）主区域顶部一条警示块：`--warning-soft` 底、`--warning` 标题、`--ink-soft` 正文，右侧「重新检查」按钮（≥44px）。外观页上同时照常读取，状态条显示保存的风格 | 标题「外观设置暂时读取失败，全站临时使用「现有风格」」；正文「保存的设置「{风格名}」没有丢，通常是外观服务短暂超时，刷新后会重新读取。这条提醒只有平台管理员能看到。」 | 「重新检查」（`router.refresh()`；读取成功后提醒消失） | `role="status"`，不抢焦点 |
| **发布成功但返回值认不出** | PUT 返回 200，但 `theme` 不认识（现有代码已经处理） | 当成失败显示 | 「未能确认发布结果，请刷新后检查。」 | 「重新读取」 | `role="alert"` |

### 4.3 前台生效的反馈

- 本页发布后会立刻换成新风格（根布局重新渲染），这本身就是最直接的反馈。成功提示要放在新风格下也看得清的位置，并且使用新风格的 token。
- 「查看前台 ↗」和「在新标签页查看前台 ↗」都用 `target="_blank" rel="noopener"`。现在的「查看前台」会在当前页跳走，管理员看不到发布结果，要改。
- 其他已打开的页面不会推送刷新，这是有意的，避免打断正在收银的人。文案里要明确写「刷新后生效」。

## 5. 信息架构

```
平台工作台
├─ 01 店铺管理    /platform
├─ 02 商品审核    /platform/goods
└─ 03 平台外观    /platform/appearance
   ├─ 页头：眉标「平台」· H1「平台外观」· 一句话说明
   ├─ （仅平台管理员，读取降级时）读取降级提醒，放在主区域最上方；平台工作台其他页面同样显示（Kel 已定（2026-11-26），Q12）
   ├─ 状态条：当前使用 + 上次发布（谁 · 什么时候，Kel 已定（2026-11-26），Q9）+ 生效方式（手机上多一个「查看前台 ↗」）
   ├─ 风格选择（radiogroup，三张卡片）
   │  └─ 卡片：双面缩略图（顾客端 | 工作台）· 单选 + 名称 + 标签 · 说明 · 色块
   ├─ 行内提示（错误 / 成功 / 离线）
   └─ 操作区：发布所选风格 · 还原为当前风格 · 查看前台 ↗ · 提示文字
```

## 6. 页面和组件

沿用 `appearance.css` 的 `.appearance-options`、`.appearance-option`、`.appearance-preview` 结构，以及全站的 `.button`、`.workspace-subpage`、`ConfirmDialog`、`gate-state` 样式。

| 组件 | 跟现有的区别 |
|---|---|
| **状态条** `.appearance-status`（新增） | 现在没有。在卡片上方用一行写清「当前使用」、「上次发布：{操作人} · {时间}」（Kel 已定（2026-11-26），Q9；`--ink-soft`，压 `--surface-soft` 9.18:1）和生效方式。时间按浏览器本地时区显示成「11月23日 18:40」，当年以外加年份；`<time datetime="{updatedAt}">`。底色 `--surface-soft`，前面一个 `--success` 圆点 |
| **风格卡片** `.appearance-option`（改） | 现在是缩略图、单选、名称、「使用中」小字、说明。改成：①缩略图换成**双面缩略图**；②自绘的单选圆点（原生 input 盖满整张卡片，透明，保留键盘和读屏行为）；③标签分两种：`使用中`（`--success` 描边胶囊）和 `已选 · 未发布`（`--accent` 描边胶囊）；④底部一排 4–5 个色块，加中文色名（色块 `aria-hidden`，色名可读） |
| **双面缩略图** `.appearance-preview`（改） | 现在只有顾客端一面，展场还要加载一张大图 `/images/artist-alley-booth.webp`。改成左右两半：左边 1.15fr 画顾客端，右边 1fr 画工作台，都用纯 CSS 画示意，不加载图片；每半左下角有「顾客端」「工作台」小标签。工作台一半要画出每套风格的侧栏（classic 纸白细线、展场深青、店面白色浮卡） |
| **示例色变量**（新增） | 缩略图必须显示**别的风格**的颜色，不能用当前风格的 `var(--accent)`。在 `appearance.css` 顶部的 `.appearance-options` 作用域里声明 `--sample-classic-*`、`--sample-expo-*`、`--sample-shop-*`，值跟姊妹文档 §4.2 一致。缩略图只引用这些变量，不再散写十六进制（修 `#5433eb` 写死的问题，以后改色只改一处） |
| **行内提示** `.appearance-message--error/ok/info`（新增） | 现在是一行裸文字。改成带图标的提示块：图标、标题、正文，加一个操作按钮，颜色取 `--warning` / `--success` / `--ink-soft` 和对应的 `-soft` 底色 |
| **操作区** `.appearance-actions`（改） | 新增「还原为当前风格」（文字按钮，只在已改未发布时出现）和右侧提示文字。「查看前台」改成新标签页打开。手机上变成底部固定栏 |
| **断网横幅**（新增，本页专用） | 页面顶部一条深色横幅加琥珀色圆点。只在本页做；全站横幅跟收银台离线策略一起单独排期，这期不做（Kel 已定（2026-11-26），Q11） |
| **读取降级提醒** `.platform-theme-fallback`（新增，Kel 已定（2026-11-26），Q12） | 跟行内错误提示 `.appearance-message--error` 同一套视觉（图标、标题、正文、按钮），但放在平台工作台外壳里，所有 `/platform/*` 页面共用；只在 `account.role === "platform_admin"` 且根布局标记了降级时渲染 |
| **确认弹窗** | 复用 `ConfirmDialog`，不新增组件。`confirmLabel="确认发布"`，`pending` 绑定发布状态，`description` 放三条要点 |
| **无权限 / 登录过期** | 复用 `gate-state` 的结构（圆标、标题、说明、按钮），放在主区域里，不占整个视口 |

**布局尺寸**：卡片区 `max-width: none`，现在是 52rem，改成跟主区域同宽，三列。缩略图高度桌面 150px、手机 120px。卡片内边距 14–16px。按钮最小 44px 高，手机底栏里的主按钮 48px 高。

**颜色和圆角**：全部取当前风格的 token（姊妹文档 §4），所以这一页在三种风格下各自好看。原型和截图按 classic 编辑式基线画。

## 7. 适配和无障碍

| 宽度 | 布局 |
|---|---|
| ≥1100 | 三列卡片；操作区排在卡片下方 |
| 768–1099 | 两列卡片；侧栏在 ≤900 宽时变成顶部横向导航（跟工作台规则一致） |
| ≤767 | 一列卡片；操作区变成底部固定栏，只放「发布所选风格」（铺满剩余宽度）和「还原」；「查看前台 ↗」移到状态条里；`main` 留出 `padding-bottom: 120px`；确认弹窗的按钮竖排，主按钮在下方，方便拇指点 |

- **键盘**：卡片组是原生 `radio`，方向键切换，Tab 进出整组；聚焦时卡片外面有 3px `--focus` 外框（现有写法）。发布确认框打开时焦点落在「取消」（Kel 已定 2026-11-26 15:01；只限本页，靠 `initialFocus="cancel"`，组件默认的「`danger` 时落在取消，否则落在确认」不改，其他页面的确认框行为不变），关闭后回到「发布所选风格」。成功和失败提示出现后，焦点移到提示块（`tabIndex=-1`）。
- **读屏**：`fieldset` 加隐藏的 `legend`「选择平台风格」；每张卡片的 `aria-describedby` 指向说明；标签文字（使用中、已选 · 未发布）是可读文本，不用伪元素画。
- **对比度**（新增部分按 WCAG 公式复核）：「上次发布」`#3c3d42` / `#f0ece2` 9.18；读取降级提醒标题 `#b23a18` / `#fbefe9` 5.30、正文 `#3c3d42` / `#fbefe9` 9.61、图标白字 / `#b23a18` 5.98，都通过；提醒块描边 `#e6b5a6` 只是装饰（1.62），信息靠图标和文字传达。原有：标签文字和描边都 ≥4.5:1（classic 下 `#315942` 配 `#edf5ed` 约 7:1，`#1e36a8` 配 `#e9edfb` 约 8:1）。缩略图是装饰，`aria-hidden`。
- **点击区域**：整张卡片都能点；所有按钮和链接 ≥44px。
- **减少动效**：骨架不闪，转圈停下，只留文字。卡片描边的过渡缩到 0.01ms。

## 8. 前端改动清单

| # | 文件 | 位置 | 现在 | 目标 |
|---|---|---|---|---|
| A1 | `web/src/app/platform/appearance/page.tsx` | `options` | 只有 `name` 和 `detail` | 再加 `swatches`（颜色数组）和 `swatchLabel`（「纸色 · 炭黑 · 赭红 · 柠绿 · 深青」这类）；`detail` 改用原型里的文案（说明顾客端和工作台两面） |
| A2 | 同上 | 状态 | `saved / selection / saving / error / notice / attempt` | 新增 `confirmOpen`、`offline`（由 `online`/`offline` 事件和 `navigator.onLine` 驱动）、`errorKind`（`load` / `save` / `auth` / `forbidden`）。`describeRequest` 的分类用来区分 401 和 403 |
| A3 | 同上 | 提交 | 点提交直接 PUT | 先 `setConfirmOpen(true)`；`ConfirmDialog` 的 `onConfirm` 里再 PUT；`initialFocus="cancel"`，打开时焦点在「取消」（Kel 已定 2026-11-26 15:01，见 A23）。「重新发布」跳过确认 |
| A4 | 同上 | 读取失败 | 卡片 `disabled`，显示「重试」 | 同时不显示「使用中」、不预选、隐藏发布按钮；错误块文案按 §4.2 |
| A5 | 同上 | 断网 | 无 | 监听 `online` 和 `offline`；离线时发布按钮禁用并显示横幅；恢复联网后 `setAttempt(a => a + 1)` 重新读取；如果读到的风格跟之前的 `saved` 不同，在状态条提示 |
| A6 | 同上 | 401 / 403 | 一行文字 | 401 显示「登录已过期」块，带「重新登录」（`/login?next=…`）；403 显示「只有平台管理员可以修改平台外观」块 |
| A7 | 同上 | 还原 | 无 | `selection !== saved` 时显示文字按钮「还原为当前风格」，点了 `setSelection(saved)` |
| A8 | 同上 | 「查看前台 ↗」 | `<Link href="/">`，在当前页跳走 | `<a href="/" target="_blank" rel="noopener">` |
| A9 | 同上 | 成功提示 | `<p role="status">已发布。打开或刷新平台页面即可看到新风格。</p>` | 改成成功提示块，标题「已发布『{name}』」，正文按 §4.2，加「在新标签页查看前台 ↗」；焦点移过去 |
| A10 | 同上 | 状态条 | 无 | 在 `fieldset` 前插入 `.appearance-status` |
| A11 | 同上 | 卡片标签 | `{saved === value && <small>使用中</small>}` | `使用中` 和 `已选 · 未发布` 两种胶囊标签 |
| A12 | `web/src/app/platform/appearance/appearance.css` | `.appearance-options` | `max-width: 52rem`；`repeat(auto-fit, minmax(min(100%, 15rem), 1fr))` | 去掉 `max-width`；改成 `repeat(3, minmax(0, 1fr))`；≤1099 宽两列，≤767 宽一列 |
| A13 | 同上 | `.appearance-shop i`（第 30 行） | `background: #5433eb; box-shadow: 0 4px 16px rgba(69, 36, 219, .34)` | `background: var(--sample-shop-accent); box-shadow: var(--sample-shop-glow)` |
| A14 | 同上 | `.appearance-classic`、`.appearance-exhibition`、`.appearance-shop` 的全部十六进制 | 散写 | 在 `.appearance-options` 上集中声明 `--sample-classic-{canvas,ink,accent,line}`、`--sample-expo-{paper,charcoal,red,lime,teal,teal-deep}`、`--sample-shop-{canvas,surface,ink,accent,glow}`，缩略图只引用这些变量 |
| A15 | 同上 | `.appearance-preview` | 单面，展场用背景图 `/images/artist-alley-booth.webp` | 双面网格 `grid-template-columns: 1.15fr 1fr`，高度 150px（≤767 宽时 120px）；左右各一个子元素，各带「顾客端」「工作台」小标签；展场顾客端改成 CSS 斜纹加炭黑竖栏，不再加载图片 |
| A16 | 同上 | `.appearance-label small` | `color: var(--accent)` | 删掉，换成 `.appearance-tag--live`（`--success`）和 `.appearance-tag--draft`（`--accent`）胶囊 |
| A17 | 同上 | 新增 | — | `.appearance-status`、`.appearance-message--{error,ok,info}`、`.appearance-offline`、`.appearance-actions` 在 ≤767 宽时 `position: fixed; inset: auto 0 0 0; padding-bottom: calc(10px + env(safe-area-inset-bottom))` |
| A18 | 同上 | 新增 | — | `@media (prefers-reduced-motion: reduce)` 下骨架和转圈不动 |
| A19 | `page.tsx` | 状态条 | 无 | **Kel 已定（2026-11-26），Q9**：读 GET 返回的 `updatedAt`、`updatedBy`，渲染「上次发布：{name} · {时间}」；为 `null` 时写「尚无记录」；PUT 成功后用返回值更新 |
| A20 | `web/src/lib/api.ts` `getStorefrontTheme()`、`web/src/app/layout.tsx` | 读取失败 | 静默返回 `classic` | **Kel 已定（2026-11-26），Q12**：返回 `{ theme, fallback: boolean }`；`layout.tsx` 把 `fallback` 传给 `SiteFrame`（例如 `data-theme-fallback`），顾客端不渲染任何东西 |
| A21 | `web/src/components/workspace-shell.tsx`（平台工作台） | 主区域顶部 | 无 | **Kel 已定（2026-11-26），Q12**：`fallback && account.role === "platform_admin"` 时渲染 `.platform-theme-fallback` 提醒（文案见 §4.2），「重新检查」调用 `router.refresh()` |
| A22 | `globals.css` 或 `appearance.css` | `.platform-theme-fallback` | 无 | 复用 `.appearance-message--error` 的样式：`--warning-soft` 底、`#e6b5a6` 描边、按钮 `min-height: 44px` |
| A23 | `web/src/app/platform/appearance/page.tsx` | `<ConfirmDialog>`（`initialFocus` 在磁盘第 273 行） | 已传 `initialFocus="cancel"`（2026-11-26 15:03 磁盘） | **Kel 已定 2026-11-26 15:01**：保持 `initialFocus="cancel"`。`web/src/components/confirm-dialog.tsx` 的默认值（`initialFocus ?? (danger ? "cancel" : "confirm")`）不改，其他页面不变 |

### 8.1 接口改动（需要后端提供 `updatedAt`/`updatedBy`，Kel 已定（2026-11-26），Q9）

本文不改 `server/`，下面是给后端的需求：
- 数据：外观设置存 `updated_at`（时间戳，带时区）和 `updated_by`（操作人的用户 ID）。每次 PUT 成功都覆盖；风格没有变化的 PUT 也算一次发布。
- `PUT /api/platform/appearance` 返回 `{ theme, updatedAt, updatedBy: { id, name } }`。
- `GET /api/platform/appearance`：平台管理员调用时返回 `{ theme, updatedAt, updatedBy: { id, name } | null }`；**Kel 已定（2026-11-26）：`updatedAt`/`updatedBy` 只返回给平台管理员；未登录或非平台管理员只拿到 `{ theme }`**，不对外公开管理员姓名（这个接口顾客端也在用）。
- `updatedAt` 用 ISO 8601 并带时区偏移，例如 `2026-11-23T18:40:00+08:00`；从没发布过时两个字段都是 `null`。
- 后端没上线前，前端拿不到这两个字段时按 `null` 处理，显示「尚无记录」，不报错。

接口改动见 §8.1。实现时注意：`PUT` 成功后先 `setSaved`，再 `router.refresh()`。刷新后本页会用新风格的 token 重新渲染，成功提示的 state 会保留，因为客户端组件不会被卸载。

## 9. 验收

1. 打开原型的十一种状态，对照截图，确认文案、可用的操作和焦点位置都跟 §4.2 一致。
2. 用 DevTools 把网络设成 Offline：出现横幅，发布被禁用；恢复后自动重新读取。
3. 读取接口返回 500：没有任何卡片显示「使用中」，也不能发布。
4. 发布后本页立刻换风格；另一个标签页刷新后才换；「查看前台 ↗」在新标签页打开。
5. 键盘：Tab 进入卡片组，方向键切换，Enter 提交，确认框打开时焦点在「取消」（Kel 已定 2026-11-26 15:01），Esc 关闭后焦点回到发布按钮；其他页面的确认框默认焦点跟改动前一样。
6. 三档宽度 390 / 820 / 1280 都没有横向滚动；手机上的发布按钮在底部固定栏里，≥48px 高。
7. 在三种风格下分别打开本页，缩略图颜色都是对应风格的固定示例色，不会跟着当前风格变。
8. 状态条显示「上次发布：{操作人} · {时间}」；发布后变成「你 · 刚刚」；未登录或非平台管理员调用 GET 只拿到 `{ theme }`，不返回 `updatedAt`/`updatedBy`（Kel 已定（2026-11-26），Q9）。
9. 让根布局读取外观超时：平台管理员在 `/platform/*` 看到读取降级提醒；顾客、摊主、销售员看不到任何提示（Kel 已定（2026-11-26），Q12）。
10. 本期没有「发布前预览」入口，也没有全站断网横幅（Kel 已定（2026-11-26），Q10、Q11）。

## 10. 待确认问题

**开放问题已全部关闭**（Kel 已定（2026-11-26），13:19 答复）。原 Q1–Q4 在汇总里编号为 Q9–Q12。

| 编号 | 问题 | Kel 的决定 | 落到哪里 |
|---|---|---|---|
| Q9（原 Q1） | 风格变更要不要记录时间和操作人 | 记录 `updatedAt` 和 `updatedBy`；**需要后端提供这两个字段**；这两个字段只返回给平台管理员，其他人只拿到 `{ theme }`（Kel 已定（2026-11-26），13:32） | §2、§4.2、§5、§6、A19、§8.1 |
| Q10（原 Q2） | 要不要支持发布前实时预览 | 这期不做 | §2、§3 |
| Q11（原 Q3） | 断网横幅要不要推广到全站 | 跟收银台离线策略一起单独排期，这期不做全站横幅 | §2、§6 |
| Q12（原 Q4） | 读取失败退回 classic 时要不要提醒 | 只提醒平台管理员 | §4.2、§5、§6、A20–A22、原型 `?state=fallback` |
