# Stallink 展场风格与店面风格设计规格

- 日期：2026-11-26
- 状态：待 Kel 审阅（本文只定设计，不改 `web/`、`server/` 代码）
- 适用范围：`data-storefront-theme="exhibition"`（展场风格）和 `data-storefront-theme="shop"`（店面风格）下的顾客端和三个工作台（摊主、现场、平台）
- 姊妹文档：[`2026-11-26-stallink-platform-appearance-design.md`](./2026-11-26-stallink-platform-appearance-design.md)（平台外观页 `/platform/appearance`）
- 代码基线：2026-11-26 11:45（UTC+8）时磁盘上的 `web/`，**包含未提交改动**。其中展场工作台 ≤900px 已改成横向导航，店面 `shop.css` 已删掉 Inter 声明。本文以这个状态为准，旧问题已在磁盘上修好的，会注明“已修”
- 视觉原型：`docs/design-v2/展场与店面-工作台-原型.html`（`?theme=exhibition|shop&page=overview|reports|counter`）
- 目标截图：`docs/design-v2/screens/展场-{总览,销售日报,收银台}-{桌面,手机}.png`、`店面-…`（桌面 1280 宽，手机 390 宽 @2x）。原型和截图只覆盖工作台；展场顾客端的 52px 竖栏和竖栏底部的退出入口（**本期不补顾客端原型，按文字说明实现**，Kel 已定（2026-11-26））

---

## 1. 背景和目标

`docs/design-v2` 只覆盖 classic（纸白、墨黑、钴蓝）。后来加上的展场风格和店面风格只有代码，没有设计依据，所以各自长出了互相矛盾的细节。2026-11-26 的测试报告和验收发现：

| # | 问题 | 位置 | 磁盘现状 |
|---|---|---|---|
| P1 | 展场顾客端用赭红 `#8c3c2d` 配柠绿 `#e3f7a2`，展场工作台用深青 `#215e58`，两边的柠绿也不一样（`#e3f7a2` 和 `#d6ef9f`） | `exhibition.css`、`exhibition-workspace.css` | 未修 |
| P2 | 展场总览的标题横幅是海军蓝渐变 `#0e203a → #22467c`，不在任何一套色里 | `exhibition-workspace.css` 第 695 行 | 未修 |
| P3 | 展场 `.product-page` 把整页 token 改回 classic，强调色变回钴蓝 `#1e36a8`；另有 5 处写死的 `#1e36a8` | `exhibition.css` 第 538–549、355、592、616、626、691 行 | 未修 |
| P4 | 展场风格固定浅色（`data-pinned="light"`），顶栏却还显示深浅色开关，点了没反应 | `app-nav.tsx` 第 171 行只对 `shop` 隐藏 | 未修 |
| P5 | 展场工作台 `--muted #526c6d` 在背景渐变最深处 `#d5e6df` 上对比度只有 4.35:1（<4.5）。现在只在日报页单独压暗 | `exhibition-workspace.css` 第 28、977–981 行 | 只修了日报页 |
| P6 | 店面 `--warning: #332f2d`，跟正文颜色几乎一样，起不到警示作用 | `shop.css` 第 18 行 | 未修（Kel 定：改为 `#b42318`） |
| P7 | 店面声明了 Inter 字体但没加载 | `shop.css` | **已修**：改成只用系统字体栈 |
| P8 | 店面 1280 宽时侧栏底部被裁，「返回 Stallink」「退出登录」看不到 | `shop.css` 第 1287–1295 行（`max-height: calc(100dvh - 120px)` 加整栏滚动） | 未修 |
| P9 | 店面 ≤860 宽时导航收成「店铺工作台 ▾」折叠按钮，跟验收要求的横向滑动不符 | `shop.css` 第 1175–1253 行 | 未修 |
| P10 | 展场工作台 390 宽时是 74px 竖向窄栏 | `exhibition-workspace.css` | **已修**：≤900 改成横向导航；但退出按钮只有 36px 高 |
| P11 | 圆角没有 token：classic 有 0/8/10/14/16/18/20px，展场工作台有 .38/.4/.55/.6/.65/.8/.85rem，店面有 12/14/16/20/28/9999px | 三个 CSS | 未修 |
| P12 | 遗留的旧绿色 `rgba(49, 89, 66, …)` 还用在加载骨架和首页装饰上；展场工作台的阴影偏海军蓝（`#0f1f33`、`#0b1628`） | `globals.css` 第 133、233、239 行；`exhibition-workspace.css` 第 45、82 行 | 未修 |
| P13 | `appearance.css` 写死店面紫 `#5433eb` 和紫色阴影 | 第 30 行 | 未修（见姊妹文档） |
| P14 | 店面工作台的页面标题只有 20px/500，还加了 `letter-spacing: -0.05em`，中文挤在一起，层级弱 | `shop.css` 第 177–188、996–1001 行 | 未修 |
| P15 | 店面收银台在 1280 宽时商品卡两列，商品名每行只排 2–3 个字；筛选按钮是方的，跟胶囊风格不一致 | `shop.css` 收银区 | 未修 |
| P16 | 店面「生成订单」禁用态是白字压在 `#c0b5f3` 上，对比度 1.88:1 | `shop.css` | 未修 |

**目标**：给这两套风格各写一份可以验收的设计依据，让前端不用猜就能改。具体是：同一个风格内，顾客端和工作台用同一套品牌规则；三个风格共用一套 token 名字；收银台单手好用；达到 WCAG 2.2 AA。

**不做**（YAGNI）：
- 不给展场和店面做深色模式。两者都固定浅色，跟现状一致
- 不加载任何网络字体
- 不新增第四种风格
- 不改 classic 的视觉。classic 只作为 token 对照的基线，只有 §8.1 列出的共用 token 新增会碰到 `globals.css`
- 不改接口和数据结构

## 2. 已定规则（Kel 2026-11-26）

1. 验收以 `docs/design-v2/screens/` 的编辑式图为准。展场和店面跟 classic 共用**信息架构、组件结构和交互**，只换**视觉 token**。
2. 店面 `--warning` 改成 `#b42318`。
3. 总览卡片跟着侧栏走，编号连续。具体做法 Kel 已定 A（K-Q15）：总览 8 张卡，编号 01–08；侧栏的管理项也是 01–08；「现场开单」「本场订单」单独成收银组，不编号；总览本身不编号。
4. 页面名称统一叫：**销售日报**（不再叫「经营日报」）、**订单记录**（摊主的 `/admin/orders`）、**本场订单**（现场的 `/counter/orders`）。
5. 展场 390 宽的导航改成跟 classic 一样横向滑动，验收要求横滑。
6. 日报保持现有实现：日期范围、分币种、更多指标。设计稿跟着实现走。
7. **Kel 已定（2026-11-26）**：店面风格 ≤900px 时侧栏导航也改成横向滑动，去掉折叠按钮（原开放问题「店面窄屏导航要不要改成横滑」已关闭）。
8. **Kel 已定（2026-11-26）**：侧栏编号只在 classic 显示；展场和店面的侧栏不显示编号。01–08 跟卡片的对应关系不变：01 活动、02 商品、03 订单记录、04 销售日报、05 收款方式、06 销售账号、07 店铺资料、08 操作记录。总览卡片照常显示编号。
9. **Kel 已定（2026-11-26）**：总览卡片 01、02 用短名「活动」「商品」，跟侧栏一致（不再叫「活动与场次」「商品与套餐」）。
10. **Kel 已定（2026-11-26）**：展场风格 390 宽时，页眉要有「退出」入口，点击区域 ≥44px，对比度达到 AA（具体见 §5.3「390 宽的退出入口」）。
11. **Kel 已定（2026-11-26）（原 Q1，选 A）**：展场工作台保留深青（`#215e58` 系），跟顾客端共用炭黑 `#262820`、柠绿 `#e3f7a2` 和衬线标题 `--expo-serif`；去掉海军蓝横幅和钴蓝商品页。见 §3.1、§4.2、§4.2.1。
12. **Kel 已定（2026-11-26）（原 Q2）**：展场顾客端手机（≤620px）上的竖栏从 67px 缩到 **52px**，不改成横条；退出入口仍固定在竖栏底部，点击区域 52px 宽 × 至少 44px 高。见 §5.3、E22–E23、E29–E30。
13. **Kel 已定（2026-11-26）（原 Q3）**：保留 901–1080px 时 152px 的窄侧栏。
14. **Kel 已定（2026-11-26）（原 Q4）**：店面不自托管 Inter，继续只用系统字体栈。
15. **Kel 已定（2026-11-26）（原 Q5）**：收银台标题叫「现场开单」，跟导航名一致。见 C15。
16. **Kel 已定（2026-11-26）（原 Q6）**：classic 保持磁盘上现有的 10–20px 圆角，不改回直角。design-v2 稿里 classic 的直角跟实现不同，这一处圆角差异不算验收不通过。
17. **Kel 已定 2026-11-26 15:01；Kel 已定 2026-11-26 15:13 补充**：店面侧栏吸顶位置分宽度：>900 宽 `top: 96px`；≤900 宽跟页头高度走，`top: var(--shop-header-height)`（Kel 已定 2026-11-26 15:13，选项 B）。页头高度 = `--shop-header-height`：大于 640 宽时 72px，≤640 时 64px（shop.css 已有这个变量，定义在 `body:has(#storefront-shell[data-storefront-theme="shop"])` token 块，≤640 的媒体查询里改成 64px；页头 `.site-header__inner` 的 `min-height` 引用它）。页头高度不改；原来写的 56px 作废。这样窄屏的吸顶条紧贴页头，上方不露空带（§6.3、§6.4、S11、§9 第 14 项）。
18. **Kel 已定 2026-11-26 15:01**：店面 ≤900 宽时保留侧栏里的「退出登录」，这次只修 D9 的底色错位（S11）。
19. **Kel 已定 2026-11-26 15:01（K-Q19=A）**：控件边框 `--line-control` 展场顾客端 `#857e6e`、展场工作台 `#6f8681`、店面 `#8f8c8b`；只加深控件边框，分割线和装饰框保持原色（`--line-strong` 不变）；店面不采用「浅灰填充加描边」方案（§4.2、§4.2.1、§7.3、C18、E28、E29、S4）。

## 3. 方案比较

### 3.1 展场两面不同色怎么处理（P1、P2）

| 方案 | 做法 | 优点 | 代价 |
|---|---|---|---|
| **A. 一个品牌，两种场景色（Kel 已定（2026-11-26））** | 顾客端保留赭红和柠绿的海报感；工作台保留深青侧栏和浅青底。两边共用炭黑 `#262820`、柠绿 `#e3f7a2`（工作台也从 `#d6ef9f` 改成这个）、衬线标题，赭红只在工作台的品牌标识上出现。去掉海军蓝横幅和钴蓝商品页 | 改动最小。符合 `appearance` 页现有文案「工作人员使用深青色固定侧栏」，说明深青是有意设计。主操作色不会跟赭红的警示色撞上 | 两面还是能看出不同色，要靠共用元素把它们连起来 |
| B. 工作台也改成纸色加炭黑 | 工作台侧栏换成炭黑 `#262820`，底色换成纸色 `#f4f0e6`，主按钮用炭黑，选中态用柠绿 | 两面一眼就是同一套 | 赭红跟 `--warning #982f20` 色相几乎一样，工作台的主操作和报错会混在一起。改动面大（约 60 条规则）。要重做全部截图 |
| C. 顾客端也改成深青 | 把海报的赭红换成深青 | 一套色 | 丢掉展场海报的辨识度，顾客端要重新验收 |

**Kel 已定（2026-11-26）：选 A**。原型和目标截图都按 A 画。共用元素具体落到这三处：
- **炭黑**：工作台新增 `--expo-charcoal: #262820`，柠绿底上的文字一律用炭黑（收银台分段筛选的选中段、已选数量徽标、柠绿标签），跟顾客端「柠绿配炭黑」的组合一致。正文仍用 `--ink #142e32`。
- **柠绿**：工作台 `--accent-light` 从 `#d6ef9f` 改成 `#e3f7a2`，侧栏选中竖线 `#d8f69a` 也改成它（E2、E6、E7）。
- **衬线标题**：工作台 `--font-display` 改成跟顾客端同一个 `--expo-serif`（Songti SC → Noto Serif SC → STSong → Georgia），中文标题两面同一字形。侧栏 logo 的「S」也改用它。
- **赭红**只出现在侧栏 logo 右上角的 8px 角标上（装饰，不承载信息），不用于按钮、文字和横幅。
- **去掉**：总览横幅的海军蓝渐变（E12）、商品详情页的钴蓝（E16–E20）。

### 3.2 窄屏导航（P9、P10）

| 方案 | 做法 | 结论 |
|---|---|---|
| **横向滑动标签（Kel 已定（2026-11-26），展场和店面都用）** | ≤900px 时侧栏变成顶部吸顶条：第一行是品牌和退出，第二行是可以横向滑动的导航标签，当前项自动滚到中间（`workspace-shell.tsx` 第 145–152 行已经有这段逻辑） | 符合验收口径，跟 classic 和展场现状一致，当前位置一眼能看到 |
| 折叠菜单（店面现状） | 一个按钮展开整组导航 | 不采用（Kel 已定（2026-11-26））：多一次点击，看不到当前位置以外的入口，不符合验收口径 |
| 底部标签栏 | 底部固定 5 个入口 | 摊主有 11 个导航项（总览、8 个管理项、2 个收银项），超过 5 个；而且会跟收银台的底部结算栏抢位置 |

### 3.3 字体（P7）

| 方案 | 结论 |
|---|---|
| **系统字体栈（Kel 已定（2026-11-26），磁盘现状）** | 展会现场网络不稳定，构建也不应该依赖外网。中文字形占绝大多数，Inter 只影响拉丁字母和数字，收益很小。数字统一用 `font-variant-numeric: tabular-nums` 保证对齐 |
| 自托管 Inter 子集（woff2，放在 `web/public/fonts/`） | 拉丁字形更统一。但要加约 30–60 KB 的资源和 `font-display: swap`，还要处理字体闪烁。不采用（Kel 已定（2026-11-26）） |

## 4. Token 体系（三风格并排）

### 4.1 架构

- token 名字三套风格共用。默认值放在 `globals.css` 的 `:root`，各风格在自己的作用域里覆盖：
  - 展场顾客端：`[data-storefront-theme="exhibition"]`
  - 展场工作台：`[data-storefront-theme="exhibition"][data-surface="workspace"]`
  - 店面：`body:has(#storefront-shell[data-storefront-theme="shop"])`
- **新增**四组 token：圆角 `--radius-*`、阴影 `--shadow-1/2/rail/brand`、动效 `--dur-*`、`--ease-*`、间距 `--space-*`，再加三个语义色：`--success`、`--focus`、`--on-accent`。组件只引用 token，**不再写死十六进制颜色**。例外：`appearance.css` 的风格缩略图要展示别的风格的颜色，见姊妹文档。
- `⚠` 表示要改的值，`＋` 表示新增的 token。

### 4.2 颜色

| token | classic（基线） | 展场·顾客端 | 展场·工作台 | 店面 |
|---|---|---|---|---|
| `--canvas` | `#faf8f3` | `#f4f0e6` | `#edf3f1`（另叠一层 `#d5e6df` 径向渐变） | `#f2f4f5` |
| `--surface` | `#fffdf8` | `#fcf9f0` | `#ffffff` | `#ffffff` |
| `--surface-soft` | `#f0ece2` | `#e8e2d5` | `#e2ece8` | `#f2f4f5` |
| `--ink` | `#16171a` | `#262820` | `#142e32` | `#000000` |
| `--ink-soft` | `#3c3d42` | `#49463b` | `#345253` | `#332f2d` |
| `--muted` | `#6b685f`（5.25） | `#605c50`（5.87） | ⚠ `#526c6d` → **`#4b6465`**（在 `#d5e6df` 上从 4.35 提到 4.90，在 canvas 上 5.65） | ⚠ `#787574` → **`--shop-muted: #726f6e`**，`--muted: var(--shop-muted)`（在 canvas `#f2f4f5` 上 4.52，在白底上 4.98；原值在 canvas 上只有 4.14）；小于 14px 的辅助文字改用 `--shop-muted-strong #625f5e`（5.74） |
| `--line` | `#dad3c3` | `#c9c2b3` | `#d0dfda` | `#ebebeb` |
| `--line-strong` | `#c6bdaa` | `#999281` | `#8ca29d` | ⚠ `#ebebeb` → **`#dedfe0`**（复用 `--shop-line-soft`，跟 `--line` 拉开层级） |
| ＋`--line-control`（交互控件边框，要求 ≥3:1） | ⚠ **`#8f8675`**；深色模式 **`#676c78`**（**Kel 已定（2026-11-26 14:13）**，W-C3；canvas 3.39、白 3.60、surface-soft 3.05；深色 3.06–3.62。`--line-strong #c6bdaa` 保留给分割线） | ⚠ **`#857e6e`**（**Kel 已定 2026-11-26 15:01**，K-Q19=A；canvas `#f4f0e6` 3.54、surface `#fcf9f0` 3.83、surface-soft `#e8e2d5` 3.12、白 4.03；`--line-strong #999281` 不变，只给分割线和装饰框） | ⚠ **`#6f8681`**（**Kel 已定 2026-11-26 15:01**，K-Q19=A；白 3.89、面板 `#f4faf6` 3.67、canvas 3.46、surface-soft 3.22、渐变最深处 `#d5e6df` 3.00；`--line-strong #8ca29d` 不变） | ⚠ **`#8f8c8b`**（**Kel 已定 2026-11-26 15:01**，K-Q19=A；白 3.34、canvas `#f2f4f5` 3.03；`--line-strong #dedfe0` 不变。**不采用「浅灰填充加描边」方案**，控件只加深描边） |
| `--accent` | `#1e36a8` | `#8c3c2d` | `#215e58` | `#5433eb` |
| `--accent-dark`（主按钮底色） | `#16171a` | `#262820` | `#123b40` | ⚠ `#5433eb` → **`#4524db`**（hover 加深；这个色已经用在紫色阴影里） |
| `--accent-soft`（选中底） | `#e9edfb` | `#eee5d4` | `#dcebe3` | `#f2f4f5`（紫色只用于标识和主操作，选中态用灰） |
| `--accent-light`（高亮） | `#c9d2f2` | `#e3f7a2` | ⚠ `#d6ef9f` → **`#e3f7a2`**（跟顾客端统一；侧栏选中竖线 `#d8f69a` 也改成它） | `#c0b5f3` |
| ＋`--brand-mark`（品牌标识） | `var(--accent)` | `#8c3c2d` | **`#8c3c2d`**（Kel 已定（2026-11-26）：只用在侧栏 logo 右上角的 8px 角标上，外加 2px `#f4f0e6` 描边才看得清；不用于文字、按钮和横幅。赭红压深青横幅只有 1.70，所以横幅眉标改用柠绿） | `#5433eb` |
| ＋`--expo-charcoal`（柠绿底上的文字） | — | `#262820`（已有） | ＋**`#262820`**（Kel 已定（2026-11-26），跟顾客端共用；在 `#e3f7a2` 上 12.86） | — |
| `--warning` | `#b23a18` | `#982f20` | `#a03a2b` | ⚠ `#332f2d` → **`#b42318`**（6.57:1，Kel 定） |
| ＋`--warning-soft` | `#fbefe9` | `#f3e1d8` | `#f6e4e0` | `#fef3f2`（就是现有的 `--shop-danger-soft`） |
| ＋`--success` | `#315942` | `#3d5a2a` | `#215e58` | `#067647` |
| ＋`--focus`（焦点环） | `var(--accent)` | `#262820`（在柠绿上也看得清） | `var(--accent)`；在侧栏里用 `var(--accent-light)` | `#5433eb` |
| ＋`--on-accent`（主按钮上的文字） | `#fffdf8` | `#fcf9f0` | `#ffffff` | `#ffffff` |
| 禁用主按钮 | 不变 | 底 `--surface-soft`，字 `--ink-soft` | 底 `--surface-soft`，字 `--ink-soft` | ⚠ 白字压 `#c0b5f3`（1.88）→ **底 `#ebebeb`，字 `#625f5e`（5.31）**，去掉紫色阴影 |
| 总览标题横幅 | 无底色 | — | ⚠ `#0e203a → #22467c`（海军蓝）→ **`#123740 → #215e58`** | 无底色 |

> 状态色不能只靠颜色区分：库存紧张、短款、报错都要同时带文字（例如「仅剩 2 件」「短款」），或者带图标。

#### 4.2.1 展场工作台对比度复核（Q1 选 A 之后，Kel 已定（2026-11-26））

按 WCAG 2.x 相对亮度公式算。文字要求 ≥4.5（大字 ≥3），控件边框和焦点环要求 ≥3。

| 组合 | 前景 / 背景 | 比值 | 结论 |
|---|---|---|---|
| 正文 | `#142e32` / canvas `#edf3f1` | 12.74 | 通过 |
| 正文压渐变最深处 | `#142e32` / `#d5e6df` | 11.06 | 通过 |
| 次要文字 | `#345253` / 白 | 8.48 | 通过 |
| 辅助文字 `--muted` | `#4b6465` / `#d5e6df` | 4.90 | 通过（原值 4.35） |
| 辅助文字 `--muted` | `#4b6465` / `#e2ece8` | 5.26 | 通过 |
| 强调色文字（优惠金额等） | `#215e58` / 白 | 7.48 | 通过 |
| 主按钮 | 白 / `#123b40` | 12.17 | 通过 |
| 柠绿底上的炭黑字（分段选中、数量徽标） | `#262820` / `#e3f7a2` | 12.86 | 通过 |
| 横幅标题（渐变两端） | 白 / `#123740`、`#215e58` | 12.76、7.48 | 通过 |
| 横幅眉标（柠绿，渐变浅端） | `#e3f7a2` / `#215e58` | 6.45 | 通过 |
| 横幅正文（渐变浅端） | `#dcebe3` / `#215e58` | 6.07 | 通过 |
| 侧栏链接 | `#d7e5e0` / `#143e46`、`#0e3036` | 8.95、10.82 | 通过 |
| 侧栏小字 | `#b9cdc8` / `#143e46` | 6.99 | 通过 |
| 侧栏选中竖线（非文字） | `#e3f7a2` / `#143e46` | 10.01 | 通过 |
| 警告文字 | `#a03a2b` / 白、`#edf3f1`、`#f6e4e0` | 6.70、5.96、5.45 | 通过 |
| 禁用主按钮 | `#345253` / `#e2ece8` | 7.02 | 通过 |
| 焦点环 | `#215e58` / 白 | 7.48 | 通过 |
| 控件边框 `--line-control` | `#6f8681` / 白、`#f4faf6`、`#edf3f1`、`#e2ece8`、`#d5e6df` | 3.89、3.67、3.46、3.22、3.00 | 通过（Kel 已定 2026-11-26 15:01，K-Q19=A）。原值 `#8ca29d`（`--line-strong`，现在只给分割线）在白底上只有 2.70 |
| 赭红眉标（不采用） | `#8c3c2d` / `#123740` | 1.70 | 不通过，所以赭红不用于文字 |
| 赭红 logo 角标（装饰） | `#8c3c2d` / `#143e46` | 1.55 | 装饰元素不要求对比度；加 2px `#f4f0e6` 描边让它看得清 |


### 4.3 字体

| token | classic | 展场·顾客端 | 展场·工作台 | 店面 |
|---|---|---|---|---|
| `--font-ui` | PingFang SC → Hiragino Sans GB → Noto Sans SC → system-ui | 同左 | 同左 | 同左，再补 -apple-system、Segoe UI |
| `--font-display` | Songti SC → Noto Serif SC → Source Han Serif SC → STSong | `--expo-serif`（Songti SC … Georgia） | ⚠ Georgia → Songti SC → Noto Serif SC → **`var(--expo-serif)`**（Kel 已定（2026-11-26），跟顾客端共用） | `= --font-ui` |
| `--font-mono` | SF Mono → Menlo → IBM Plex Mono | 同左 | 同左 | `= --font-ui`，数字加 `tabular-nums` |
| 加载方式 | 系统字体 | 系统字体 | 系统字体 | 系统字体（P7 已修，不再声明 Inter；不自托管 Inter，Kel 已定（2026-11-26）） |
| 工作台 H1 | 衬线 500，`clamp(2rem, 4vw, 3rem)` | — | 衬线 650，`clamp(2.05rem, 4vw, 3.4rem)`，行高 1.15，字距 −0.03em | ⚠ 20px/500/−0.05em → **28px/600/字距 0**；手机 24px |
| 正文 | 15px/1.55 | 15px/1.55 | 15px/1.55 | 15px/1.55；输入框 16px，避免 iOS 聚焦时自动放大 |
| 金额 | mono + `tabular-nums` | 同左 | 同左 | sans + `tabular-nums`（＋） |

### 4.4 间距（＋，4px 基准）

| token | 值 | 用途 |
|---|---|---|
| `--space-1` | 4px | 图标和文字的间距 |
| `--space-2` | 8px | 触控目标之间的最小间距 |
| `--space-3` | 12px | 卡片网格间距（展场） |
| `--space-4` | 16px | 卡片内边距、网格间距（店面） |
| `--space-5` | 24px | 区块间距 |
| `--space-6` | 32px | 页面分组 |
| `--space-7` | 48px | 大区块 |
| `--workspace-gutter`（已有） | 展场 `clamp(1rem, 3.4vw, 3rem)`，≤760 时 `clamp(.55rem, 2.5vw, .95rem)`；店面 0（外层容器 1200px 居中） | 主区域左右留白 |

已有的 rem 写法不用一次全换。改到哪条规则，就顺手换成 token。

### 4.5 圆角（＋，修 P11）

| token | classic（磁盘上的柔化层） | 展场·顾客端 | 展场·工作台 | 店面 |
|---|---|---|---|---|
| `--radius-control`（按钮、输入框、分段控件） | 10px | 0 | **6px**（现有 .38/.4rem 统一成这个） | **14px**；按钮用 `--radius-pill` |
| `--radius-card`（商品卡、列表行、指标块） | 16px | 0 | **10px**（.55/.6/.65rem 统一） | **20px** |
| `--radius-panel`（面板、侧栏、弹窗、横幅） | 18px | 0 | **13px**（.8/.85rem 统一） | **28px** |
| `--radius-pill` | 999px | 999px | 999px | 9999px |

> classic 的 design-v2 稿是直角，磁盘上的 `globals.css` 第 1106–1205 行加了 10–20px 的柔化圆角。**Kel 已定（2026-11-26）：保持现在的 10–20px 圆角，不改直角**；上表 classic 一列就是目标值。

### 4.6 阴影（＋，修 P12）

| token | classic | 展场·顾客端 | 展场·工作台 | 店面 |
|---|---|---|---|---|
| `--shadow-1`（控件、标签） | none | none | `0 1px 2px #142e3210` | `rgba(0,0,0,.06) 0 2px 8px`（就是 `--shop-shadow-chip`） |
| `--shadow-2`（卡片、面板） | `0 18px 40px rgba(22,23,26,.06)`（就是 `--shadow-soft`） | none（海报是平的） | ⚠ `#0f1f33…` → `0 1px 2px #142e3210, 0 10px 26px #142e320d` | `--shop-shadow` |
| `--shadow-rail` | — | — | ⚠ `6px 0 24px #0b16281a` → `6px 0 24px #0e30361a` | — |
| `--shadow-brand` | — | — | — | `rgba(69,36,219,.34) 0 4px 24px`（**只**给主按钮和品牌圆点用） |
| 遗留旧绿色 | ⚠ `globals.css` 第 133、233、239 行的 `rgba(49,89,66,α)` → `rgba(22,23,26,α)`，α 保持不变 | | | |

### 4.7 动效（＋）

| token | 值 | 用途 |
|---|---|---|
| `--dur-1` | 120ms | 按下缩放到 0.98 |
| `--dur-2` | 180ms | hover、颜色、边框 |
| `--dur-3` | 240ms | 底部抽屉、弹窗进出（退出动画取 0.7 倍） |
| `--ease-out` | `cubic-bezier(.2,.7,.2,1)` | 进入（design-v2 模板里用的就是这条） |
| `--ease-in` | `cubic-bezier(.4,0,1,1)` | 退出 |
| 减少动效 | `@media (prefers-reduced-motion: reduce)` 下，所有 transition 和 animation 缩到 0.01ms，`html` 取消平滑滚动，卡片 hover 不位移。加载圈改成静态标记加文字「正在…」 | 现在 shop.css 第 3690 行只是把加载圈放慢，要补齐 |

## 5. 展场风格

### 5.1 用户流程

主流程跟 classic 一致，只换视觉。下面只写跟主题有关的反馈。

**摊主开摊（主流程）**：登录 → `/admin` 总览，横幅写「开摊之前，把清单过一遍。」→ 点「打开现场收银」→ `/counter` 现场开单：选场次 → 点商品加一 → 右侧订单预览实时报价 → 「生成订单」→ 收款弹窗（现金或扫码）→ 取货凭证 → 回到空白开单页。
**收摊对账（主流程）**：侧栏 04「销售日报」→ 选日期范围和币种 →「查询日报」→ 看指标、分币种汇总、热销商品、收款方式 → 导出 PDF 或 CSV。

| 异常 | 界面反馈（展场） |
|---|---|
| 权限检查中 | 整页 `gate-state`：深青圆形加载标记加「正在检查工作台权限」。减少动效时不转圈 |
| 权限检查失败 | 同一位置换成 `--warning` 圆标「!」和「无法验证账号权限」，下面是「重新检查」按钮（44px） |
| 没有进行中的场次 | 开单页的场次下拉显示「暂无进行中的活动」，商品区换成空状态卡片，带「去活动页」链接；「生成订单」禁用 |
| 商品售完 | 商品行整行用 `--surface-soft` 底色，加「本场已售完」标签，「+」按钮禁用。不能只靠变灰表达 |
| 库存紧张（≤3 件） | 元信息改成「仅剩 N 件」，用 `--warning` 字色加粗 |
| 报价失败或断网 | 订单预览上方出现 `--warning-soft` 底的提示条：「无法连接服务，请检查网络后重试」和「重试」按钮。「生成订单」禁用。已选商品保留（本地草稿已有） |
| 生成订单中 | 按钮文字改成「正在生成…」并禁用，`aria-busy="true"` |
| 日报还没生成 | 报表区显示空状态「这一天还没有生成日报」，带主按钮「生成日报」 |
| 日期范围超过 31 天 | 结束日期下方行内报错，用 `--warning` 字色，文案「每次最多查 31 天」，焦点回到结束日期 |
| 导出失败 | 工具栏下方的行内提示条，带「重试」 |
| 退出失败 | 侧栏底部（手机上是顶栏右侧）出现 `role="alert"` 文字「退出失败，请重试」 |

### 5.2 信息架构

```
展场风格
├─ 顾客端（exhibition-nav 炭黑竖栏；≤620px 时 52px 宽，Kel 已定（2026-11-26））
│  ├─ 首页 / 展场海报
│  ├─ 摊位 /c/[slug] → 商品详情（.product-page）/ 套餐 / 活动
│  └─ 关于、登录、注册、我的
└─ 工作台（深青侧栏 224px；901–1080px 时 152px（保留，Kel 已定（2026-11-26））；≤900px 时顶部横向导航）
   ├─ 摊主工作台（管理项编号 01–08；Kel 已定 A（K-Q15）；展场侧栏不显示编号，只在卡片上显示，Kel 已定（2026-11-26））
   │  ├─ 经营：总览（不编号）· 01 活动 · 02 商品 · 03 订单记录 · 04 销售日报
   │  ├─ 现场收银：现场开单 · 本场订单（不编号）
   │  └─ 店铺：05 收款方式 · 06 销售账号 · 07 店铺资料 · 08 操作记录
   ├─ 现场工作台（销售员）：现场开单 · 本场订单（不编号）
   └─ 平台工作台：01 店铺管理 · 02 商品审核 · 03 平台外观
```

**编号规则（Kel 已定 A（K-Q15））**：
- 只给管理项编号，摊主工作台从 01 连续排到 08：01 活动、02 商品、03 订单记录、04 销售日报、05 收款方式、06 销售账号、07 店铺资料、08 操作记录。
- 总览是卡片的容器，本身不编号（去掉现在的 `00`）。
- 「现场开单」「本场订单」单独作为「现场收银」组，**不编号**（去掉现在的 `POS`/`ORD`），摊主和销售员工作台都一样。
- 总览卡片 = 8 个管理项，**同序、同号、同名、同分组**（经营 01–04、店铺 05–08）。收银组不出卡片，入口由横幅下的「打开现场收银」主按钮承担。卡片标题就是导航名，描述放第二行。现在缺的「销售日报」「操作记录」两张要补上（6 + 2 = 8）。
- 平台工作台照旧：01 店铺管理、02 商品审核、03 平台外观。
- **Kel 已定（2026-11-26）**：侧栏编号**只在 classic 显示**（`.workspace-nav__index`，收银组两项为空不渲染）。展场和店面的侧栏不显示编号，`index` 数据照样保留，对应关系不变：01 活动、02 商品、03 订单记录、04 销售日报、05 收款方式、06 销售账号、07 店铺资料、08 操作记录。总览卡片在三套风格里都照常显示编号。

### 5.3 页面和组件

**工作台外壳**（沿用 `exhibition-workspace.css` 的 `.workspace-sidebar` 写法）：
- 侧栏背景保留 `linear-gradient(180deg, #143e46, #123740 48%, #0e3036)`。选中项改成 `inset 3px 0 var(--accent-light)`，原来是 `#d8f69a`。
- **侧栏改成三段**：品牌和身份 / 导航（`flex: 1 1 auto; min-height: 0; overflow-y: auto`）/ 底部「返回 Stallink」和「退出登录」（`flex: 0 0 auto`）。这样不管侧栏多高，底部操作都看得见，同一个结构也用来修店面的 P8。
- 「退出登录」⚠ `min-height: 36px` → `var(--touch)`（44px），桌面和窄屏都一样。
- **导航编号**（Kel 已定（2026-11-26））：展场侧栏**不显示**编号，`.workspace-nav__index` 保持 `display: none`；侧栏只显示图标加名称。编号只出现在总览卡片上。

**总览**（`admin/page.tsx` 的 `.workspace-overview`）：
- 横幅：⚠ 海军蓝改成深青渐变 `linear-gradient(115deg, #123740, #215e58)`。眉标颜色用 `--accent-light`，正文用 `#dcebe3`。
- 状态条和「打开现场收银」主按钮保持现有的独立面板。
- **新增按组分区**：两组卡片各有一个小标题（经营 01–04 / 店铺 05–08；收银组不出卡片，Kel 已定 A（K-Q15）），网格是 `repeat(auto-fill, minmax(min(100%, 13.5rem), 1fr))`，1280 宽时一行 4 张，正好每组一行。跟现有 `.workspace-card-grid` 的区别只是多了分组标题，卡片组件本身不变。
- 卡片：左上角等宽编号（照常显示），标题就是导航短名（01「活动」、02「商品」，Kel 已定（2026-11-26）），用衬线 600，描述用 `--muted`，右上角是箭头。整张卡是一个 `<a>`，hover 时上移 2px（减少动效时不移）。
- 「开摊前检查」提示块保持不变。

**销售日报**（`admin/reports/page.tsx` 和 `reports.css`）：
- H1 ⚠「经营日报」→「销售日报」，上方眉标「经营」。
- **工具栏改成两组**：筛选组（起始日期、结束日期、币种、主按钮「查询日报」）和操作组（「重新生成」「导出 CSV」「导出 PDF」，都是描边按钮）。外层 `flex-wrap: wrap; justify-content: space-between`，每组内部 `flex-wrap: nowrap`。1280 宽时排一行；放不下就把操作组整组换到第二行，**最多两行**（修验收里 1280 宽工具栏断成三行的问题）。
- 工具栏下方一行说明：「每次最多查 31 天。选择『全部币种』时 PDF 包含全部币种，CSV 只导出主币种 MYR。」
- 报表单：
  - 头部是眉标（日期 · 币种）、标题「当日订单与收款」、右边「生成于 …」。
  - 指标网格 4 列。主指标「实收金额」占两行，用 `--accent-soft` 底，金额 `white-space: nowrap`。「现金长短款」是负数时用 `--warning` 字色，并且写出「短款」两个字。
  - 分币种汇总表；热销商品和收款方式两列并排。
- **表格防竖排**：商品名列 `min-width: 8em; word-break: keep-all; overflow-wrap: anywhere`，数字列 `white-space: nowrap; text-align: right; tabular-nums`。修验收里商品名一字一行的问题。
- 日报页单独的 `--muted: #4f696a` 覆盖可以删掉，因为全局 `--muted` 已经改成 `#4b6465`。

**收银台**（`counter/page.tsx`）：
- H1 现在是「创建订单」，导航名是「现场开单」。**Kel 已定（2026-11-26）**：H1 改成「现场开单」，跟导航名一致。
- 场次和顾客称呼放在一个面板里，两列。
- 商品区：搜索框加分段筛选（全部 / 单品 / 套餐）。选中段用 `--accent-light` 柠绿底配 `--expo-charcoal` 炭黑字（Kel 已定（2026-11-26），Q1=A）。
- **商品行**统一成一列的「行卡」：48px 缩略图、名称（最多两行，超出用省略号）、元信息、右侧金额、44×44 的「+」按钮。已选数量用柠绿圆角徽标跟在名称后面。现在 1100–1330 宽时已经是单列，本文把单列推广到所有宽度，因为两列卡片在中文商品名下会挤成竖排。
- 订单预览：每一行是名称、金额，加一个步进器（− 数量 +，每个按钮 44×40）；下面是小计、优惠（用 `--accent` 色）、应付总额（衬线 1.6rem），最后是 52px 高的「生成订单」主按钮。
- ≤1099px：右侧预览收起，底部出现固定结算栏 `.counter-bar`：左边「已选 N 件 · 查看明细」加总额（点了打开底部抽屉），右边「生成订单」（最小 52px 高、9.5rem 宽），放在右手拇指好按的区域。

**顾客端修正**（`exhibition.css`）：
- ⚠ 删掉 `[data-storefront-theme="exhibition"] .product-page` 里整组 classic token 的覆盖（第 538–549 行），商品详情页跟其他顾客页一样用展场 token。
- ⚠ 5 处写死的 `#1e36a8` 改成 `var(--accent)`。商品占位底色改成 `var(--expo-charcoal)`，眉标横线改成 `var(--accent)`，购买按钮 hover 改成 `var(--accent)`。
- **顾客端手机竖栏（Kel 已定（2026-11-26））**（**本期不补顾客端原型，按文字说明实现**，Kel 已定（2026-11-26））：≤620px 时竖栏从 67px 缩到 **52px**，不改成横条（保留海报风格的标志），390 宽屏上内容区多出 15px（竖栏占比从约 17% 降到约 13%）。具体：
  - `--expo-rail: 52px`；竖栏左右内边距改为 0，竖栏里的每一项都占满 52px 宽。
  - 品牌：52px 放不下「Stallink.」（15px 字约 60px 宽），≤620 时只显示衬线「S.」（1.15rem），链接的 `aria-label="Stallink 首页"` 不变。
  - 导航项：18px 图标加 12px 文字（最长「工作台」三个字约 36px，放得下），每项 52×≥44px。
  - 图标和文字的颜色、hover、选中态（柠绿底炭黑字）都不变。

**390 宽的退出入口**（Kel 已定（2026-11-26））：
- **顾客端页眉**（`exhibition-nav`，已登录时）（**本期不补顾客端原型，按文字说明实现**，Kel 已定（2026-11-26））：
  - 位置：竖栏最底部，`.exhibition-nav__settings` 组的最后一项，在「我的 / 工作台」下面，上方有 1px `#f4f0e63d` 分隔线。
  - 必须不滚动就能看到：`.exhibition-nav__settings` 改成 `position: sticky; bottom: 0; background: var(--expo-charcoal)`。竖栏内容超出屏幕高度时，只滚导航链接，退出始终钉在底部。
  - 样式：跟其他竖栏项一致，上面一个 18px「退出」图标（门加箭头），下面文字「退出」。字号从 ≤620 宽时的 .7rem 提到 .75rem（12px）。
  - 点击区域：整格可点，**52px 宽（占满竖栏）× 最小 44px 高**（Kel 已定（2026-11-26））。
  - 颜色和对比度：文字和图标 `#f4f0e6`，压在炭黑 `#262820` 上是 13.12:1（AA 要求 4.5）。hover 底色 `#44463a`。焦点环 2px `var(--expo-lime)`（柠绿压炭黑 12.86:1）。
  - 退出中显示「退出中」并禁用；失败时在按钮下方显示 `role="alert"`「退出失败，请重试」。
  - 竖栏不改成横条（Kel 已定（2026-11-26）），所以退出始终在竖栏底部。
- **工作台页眉**（≤900px 的吸顶条）：「退出登录」在第一行最右端，跟品牌同一行，不跟导航挤在一起；描边按钮，最小 44px 高、左右内边距 14px；文字 `#ebf3ef` 压在 `#143e46` 上是 10.29:1；边框 `#ffffff33`；焦点环用 `--accent-light`。见截图 `展场-*-手机.png`。

**深浅色开关**（P4）：⚠ `app-nav.tsx` 第 171 行的 `storefrontTheme === "shop" ? null : …` 改成 `storefrontTheme !== "classic" ? null : …`。展场和店面都固定浅色，开关就不该出现。

### 5.4 适配（展场）

| 宽度 | 工作台布局 | 收银台 |
|---|---|---|
| ≥1100（桌面，1280 是验收档） | 侧栏 224px（1081 及以上） | 商品列表加右侧预览 `minmax(300px, 360px)`，预览吸顶 |
| 901–1099（平板横屏） | 侧栏 152px（磁盘现状 ≤1080；保留，Kel 已定（2026-11-26）） | 预览收起，改成底部结算栏加抽屉 |
| 768–900（平板竖屏） | 顶部吸顶条：第一行 logo 加退出，第二行横向导航 | 同上 |
| ≤767（手机，390 是验收档） | 同上；总览卡片两列，不显示描述 | 同上；商品行变成 44 / 名称和金额 / 44 三列；分段筛选铺满一行 |

## 6. 店面风格

### 6.1 用户流程

主流程和异常流程的触发条件跟 §5.1 一致，下面只写店面的表现：
- 提示条和报错用 `--warning #b42318` 字色、`--warning-soft #fef3f2` 底、`--shop-danger-line #f4b4ad` 边框，圆角 `--radius-card`，前面带「!」图标。现在的 `form-message--error` 已经接近这个样子，只是把 `--warning` 统一过去。
- 库存紧张「仅剩 N 件」以前跟正文同色，换成新的 `--warning` 后才真正有警示作用（P6）。
- 加载、空状态沿用 `gate-state`、`empty-state--compact` 的店面样式。

### 6.2 信息架构

跟 §5.2 相同：同样的入口、编号和名称（侧栏同样不显示编号，Kel 已定（2026-11-26））。差别只在外壳：
- 店面有白色顶栏（`site-header`）：品牌圆点、「现场订单 · 库存管理」，右侧是「关于」「查看店铺」、紫色胶囊「店铺工作台 ↗」、「退出」。
- 工作台侧栏是一张浮起的白卡（240px），放在 1200px 居中的容器里。

### 6.3 页面和组件

**外壳**（沿用 `shop.css` 的 `.workspace-sidebar` 白卡写法）：
- ⚠ **侧栏三段式**（修 P8）：`.workspace-sidebar` 改成 `display: flex; flex-direction: column; overflow: hidden`，保留 `max-height: calc(100dvh - 112px)`；`.workspace-nav` 改成 `flex: 1 1 auto; min-height: 0; overflow-y: auto`；「返回 Stallink」和「退出登录」放进底部，`flex: 0 0 auto`，上方加 `border-top: 1px solid var(--line)`。导航分组间距从 24px 缩到 16px，标签上边距从 24px 缩到 8px。这样 900px 高的屏幕可以完整放下 10 个入口加底部操作。
- 导航链接是 44px 胶囊。选中态用 `--surface-soft` 灰底加 600 字重，紫色不用于导航。**不显示编号**（Kel 已定（2026-11-26）），`.workspace-nav__index` 保持 `display: none`。
- 退出登录改成描边胶囊按钮，44px。
- ⚠ **≤900px 改成横向导航**（修 P9，Kel 已定（2026-11-26））：去掉 `.workspace-nav-toggle` 的折叠交互（这个按钮只在店面用，其他风格本来就 `display: none`）。侧栏变成吸顶条，紧贴页头下沿：≤900 宽 `top: 72px`，≤640 宽 `top: 64px`，跟页头高度一致；>900 宽仍是 `top: 96px`（Kel 已定 2026-11-26 15:13；原来写的 56px 作废）。里面是可以横向滑动的胶囊：未选中是 `--surface-soft` 灰底，选中是黑底白字。**品牌由顶栏承担**，吸顶条里不再显示；**「退出登录」保留在吸顶条右端**（Kel 已定 2026-11-26 15:01），只修正 D9 的底色错位，见 S11。顶栏本身也有「退出」入口（`components/app-nav.tsx` 第 183 行 `.header-logout`，所有宽度都显示），所以窄屏上有两个退出入口：顶栏右上角的「退出」（灰色文字链接）和吸顶条右端的「退出登录」（白色胶囊），两个都保留。顶栏在 ≤900 宽时隐藏「关于」「查看店铺」，现在 390 宽时顶栏会挤成竖排（见测试截图 `shop-390-counter.png`），「店铺工作台 ↗」加 `white-space: nowrap`。
- 手机上滑动条右边缘加渐隐遮罩（`mask-image`），提示还能往右滑。

**总览**：
- ⚠ H1 改成 28px/600、字距 0；描述 16px/1.6，颜色 `--shop-muted-strong`。
- 「打开现场收银」是紫色胶囊主按钮，带 `--shadow-brand`。
- 卡片按组分区（经营 01–04 / 店铺 05–08，共 8 张，Kel 已定 A（K-Q15）），网格是 `repeat(auto-fill, minmax(min(100%, 20rem), 1fr))`，1280 宽时两列（跟现在一样）。卡片是白底、`--radius-panel` 圆角、`--shop-shadow` 阴影，左上角写编号（现在写的是「开摊准备」这类标签，改成编号；侧栏虽然不显示编号，对应关系不变），标题用短名（01「活动」、02「商品」，Kel 已定（2026-11-26）），右上角是 36px 灰圆箭头。卡片标题就是导航名。
- 手机上卡片变成单列横排：编号、标题、描述。

**销售日报**：
- 工具栏跟展场一样分两组，最多两行；放在白色面板里，圆角 `--radius-panel`。输入框 44px 高、`--radius-control` 圆角、`--line-control` 边框、16px 字。按钮都是胶囊。
- 主指标用很浅的紫底 `#f5f3fe`（只在主指标上用）。其他指标是白底，用 1px `--line` 分隔。
- 表格防竖排的规则跟展场一样。

**收银台**：
- ⚠ 商品区单列行卡（修 P15）：`--surface-soft` 灰底、无边框、`--radius-card` 圆角，金额用 `--accent` 紫色，「+」是 44px 圆形描边按钮。
- ⚠ 分段筛选改成胶囊组：外层是 `--surface-soft` 灰底胶囊，选中段黑底白字，跟窄屏导航的选中态一致。
- 订单预览白卡用 `--radius-panel`。步进器是胶囊形。「生成订单」是紫色胶囊，52px 高。⚠ 禁用态见 §4.2（修 P16）。
- ≤1099px 的底部结算栏：白底，上方投影 `0 -6px 20px #0000000f`，紫色胶囊「生成订单」。

### 6.4 适配（店面）

| 宽度 | 外壳 | 收银台 |
|---|---|---|
| ≥1100 | 顶栏高度 `--shop-header-height`（72px），1200px 居中容器，侧栏白卡 240px 吸顶（`top: 96px`） | 商品列表加右侧预览 |
| 901–1099 | 同上，侧栏照常显示 | 底部结算栏 |
| 641–900 | 顶栏高度不改，`--shop-header-height` 为 72px。侧栏变成吸顶条，`top: var(--shop-header-height)`（72px），紧贴页头（Kel 已定 2026-11-26 15:13；原写的 56px 作废）。导航变成横滑胶囊条，右端保留「退出登录」 | 底部结算栏 |
| ≤640 | `--shop-header-height` 为 64px（shop.css `@media (max-width: 640px)` 里的 `body:has(#storefront-shell[data-storefront-theme="shop"])` 块），顶栏 64px；吸顶条 `top: var(--shop-header-height)`（64px，Kel 已定 2026-11-26 15:13），其余同上 | 底部结算栏 |
| ≤767 | 同上；总览卡片单列；内边距 16px | 商品行改成三列网格 |

## 7. 适配和无障碍（两套风格共用）

### 7.1 断点

统一三档验收：**390**（手机）、**820 / 1024**（平板竖屏和横屏）、**1280**（桌面）。CSS 断点用这几个：767 / 900 / 1099。展场多一个 1080 档，给 152px 侧栏用。

### 7.2 收银台单手操作

- 主操作「生成订单」在 ≤1099px 时固定在底部右侧，最小 52×152px，留出 `env(safe-area-inset-bottom)`。
- 加货按钮和步进器都 ≥44×44，相邻触控目标间距 ≥8px。
- 列表只上下滚动，不做左滑删除，避免和系统返回手势冲突。删除用步进器减到 0，或者用明确的「移除」按钮。
- 底部结算栏会盖住内容，`main` 要留出 `padding-bottom: 110px`。
- 订单预览抽屉从底部滑入（`--dur-3`、`--ease-out`），有关闭按钮，支持 Esc 关闭。打开时焦点移到抽屉标题，关闭后回到「查看明细」。

### 7.3 对比度（目标值，按 WCAG 相对亮度计算）

| 组合 | 比值 | 结论 |
|---|---|---|
| 展场工作台 `--muted #4b6465` / `#d5e6df` | 4.90 | 通过（原值 4.35） |
| 展场工作台 `--muted #4b6465` / `--surface-soft #e2ece8` | 5.3 | 通过 |
| 展场工作台侧栏 `#b9cdc8` / `#143e46` | 6.99 | 通过 |
| 展场顾客端 `--ink` / `--expo-lime` | 12.86 | 通过 |
| 店面 `--muted #726f6e` / `#f2f4f5` | 4.52 | 勉强通过。小于 14px 的文字改用 `#625f5e`（5.74） |
| 店面 `--warning #b42318` / 白 | 6.57 | 通过 |
| 店面 白 / `--accent #5433eb` | 6.88 | 通过 |
| 店面禁用按钮 `#625f5e` / `#ebebeb` | 5.31 | 通过（原来白字压 `#c0b5f3`，1.88） |
| 展场工作台控件边框 `--line-control #6f8681` / 白、`#edf3f1`、`#e2ece8`、`#d5e6df` | 3.89、3.46、3.22、3.00 | 通过（Kel 已定 2026-11-26 15:01，K-Q19=A；原 `#8ca29d` 为 2.70、2.41、2.24、2.09） |
| 展场工作台柠绿底炭黑字 `#262820` / `#e3f7a2` | 12.86 | 通过 |
| 展场顾客端退出 `#f4f0e6` / `#262820`（52px 竖栏） | 13.12 | 通过 |
| 展场顾客端控件边框 `--line-control #857e6e` / `#fcf9f0`、`#f4f0e6`、`#e8e2d5` | 3.83、3.54、3.12 | 通过（Kel 已定 2026-11-26 15:01，K-Q19=A；原 `#999281` 为 2.94、2.72、2.40） |
| 店面控件边框 `--line-control #8f8c8b` / 白、`#f2f4f5` | 3.34、3.03 | 通过（Kel 已定 2026-11-26 15:01，K-Q19=A；原 `--line-strong #dedfe0` 为 1.33、1.21；不采用浅灰填充方案） |
| classic 控件边框 `#8f8675` / `#faf8f3`、白、`#f0ece2`；深色 `#676c78` / `#0e1014`、`#16181d`、`#1e2129` | 3.39、3.60、3.05；3.62、3.38、3.06 | 通过（Kel 已定（2026-11-26 14:13），W-C3） |

### 7.4 键盘和读屏

- 每页的 `skip-link`「跳到主要内容」保留。切换路由后焦点落到 `#main-content`。
- 横向导航可以用 Tab 逐个聚焦，聚焦的项自动滚进可视区域（`scrollIntoView({ inline: "nearest" })`）。当前项带 `aria-current="page"`。
- 焦点环用 `--focus`，2px，偏移 2–3px。深青侧栏里改用 `--accent-light`。
- 编号是 `aria-hidden="true"`，读屏只读导航名。
- 分段筛选用 `role="group"` 加 `aria-pressed`。商品「+」按钮的 `aria-label` 写成「加一件{商品名}」。已售完的商品用 `disabled`，同时写出文字原因。
- 订单预览用 `aria-live="polite"` 播报总额变化。报错用 `role="alert"`。

### 7.5 减少动效

见 §4.7。另外：卡片 hover 不位移；骨架屏不闪；底部抽屉直接出现，不滑入。

## 8. 前端改动清单

> 「现值」以 2026-11-26 磁盘为准，行号是近似值，按选择器查找更可靠。

### 8.1 共用

| # | 文件 | 选择器 / 位置 | 现值 | 目标值 |
|---|---|---|---|---|
| C1 | `web/src/app/globals.css` | `:root` | 没有圆角、阴影、动效、间距 token | 新增 `--radius-control: 10px; --radius-card: 16px; --radius-panel: 18px; --radius-pill: 999px; --shadow-1: none; --shadow-2: var(--shadow-soft); --dur-1: 120ms; --dur-2: 180ms; --dur-3: 240ms; --ease-out: cubic-bezier(.2,.7,.2,1); --ease-in: cubic-bezier(.4,0,1,1); --space-1…7; --success: #315942; --focus: var(--accent); --on-accent: #fffdf8; --line-control: #8f8675（Kel 已定（2026-11-26 14:13）；深色两个 token 块里是 `#676c78`）; --warning-soft: #fbefe9; --brand-mark: var(--accent)` |
| C2 | `globals.css` | `.header-loading`（第 133 行） | `rgba(49, 89, 66, 0.06)` | `rgba(22, 23, 26, 0.06)` |
| C3 | `globals.css` | 第 233 行 | `border: 1px solid rgba(49, 89, 66, 0.16)` | `rgba(22, 23, 26, 0.16)` |
| C4 | `globals.css` | `.landing-orbit`（第 239 行） | `rgba(49, 89, 66, 0.3)` | `rgba(22, 23, 26, 0.3)` |
| C5 | `globals.css` | `:focus-visible` | `outline: 2px solid var(--accent)` | `outline: 2px solid var(--focus)` |
| C6 | `globals.css` | 文件末尾 | 减少动效规则分散在各处 | 加一条全局规则：`@media (prefers-reduced-motion: reduce) { *, *::before, *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; } html { scroll-behavior: auto; } }` |
| C7 | `web/src/components/app-nav.tsx` | 第 171 行 | `storefrontTheme === "shop" ? null : <button className="theme-toggle">` | `storefrontTheme !== "classic" ? null : …` |
| C8 | `web/src/components/workspace-shell.tsx` | `workspace.shop.groups` | 2026-11-26 12:45 磁盘：总览 `00`；管理项 `01–04`、`06–09`（跳过 05）；收银组 `POS/ORD`；文案已是「订单记录」 | **Kel 已定 A（K-Q15）**：总览不编号（`index: ""`）；管理项 `01 活动, 02 商品, 03 订单记录, 04 销售日报, 05 收款方式, 06 销售账号, 07 店铺资料, 08 操作记录`；收银组不编号（见 C9）；`index` 数据保留，但只有 classic 的侧栏渲染编号，展场和店面不渲染（Kel 已定（2026-11-26）） |
| C9 | `workspace-shell.tsx` | `cashierGroup` | 磁盘：文案已改为「本场订单」，编号仍是 `POS`/`ORD` | **Kel 已定 A（K-Q15）**：文案「本场订单」；两项 `index: ""`，不编号，`.workspace-nav__index` 为空时不渲染；摊主和销售员工作台共用同一个 `cashierGroup`，不需要改成工厂函数 |
| C10 | `workspace-shell.tsx` | 侧栏结构 | 「返回」和「退出」直接挂在 `aside` 下 | 用 `<div className="workspace-sidebar__foot">` 包住「返回 Stallink」「退出登录」和 `logoutError`，供三栏布局使用 |
| C11 | `web/src/app/admin/page.tsx` | `sections` | 磁盘：已是 8 张卡、编号 01–08（前端已做）；未分组；标题仍是「活动与场次」「商品与套餐」，另有「开摊准备」这类小标签 | **Kel 已定 A（K-Q15）**：保持 8 张、01–08；分「经营」（01 活动、02 商品、03 订单记录、04 销售日报）和「店铺」（05 收款方式、06 销售账号、07 店铺资料、08 操作记录）两组；`index` 跟 C8 一致，`title` 就是导航短名：01「活动」、02「商品」，替换现在的「活动与场次」「商品与套餐」（Kel 已定（2026-11-26）），其余标题已经跟导航一致；描述见原型；不做「现场开单」「本场订单」卡片 |
| C12 | `web/src/app/admin/reports/page.tsx` | 第 228 行 | `<h1>经营日报</h1>` | `<h1>销售日报</h1>`，上方加 `<p className="eyebrow">经营</p>` |
| C13 | `reports/page.tsx` 和 `reports.css` | `.report-toolbar` | 按钮在两个区域间换行，1280 宽时断成三行 | DOM 分成 `.report-toolbar__filters`（日期 ×2、币种、查询）和 `.report-toolbar__actions`（重新生成、导出 CSV、导出 PDF）；外层 `display: flex; flex-wrap: wrap; justify-content: space-between; gap: 12px 24px`，每组内部 `flex-wrap: nowrap`；≤767 宽时改成两列网格，操作组三等分 |
| C14 | `reports.css` | 热销商品和活动表的商品名单元格 | 窄列下一字一行 | `min-width: 8em; word-break: keep-all; overflow-wrap: anywhere`；数字列 `white-space: nowrap; font-variant-numeric: tabular-nums; text-align: right` |
| C15 | `web/src/app/counter/page.tsx` | 第 780 行 | `<h1>创建订单</h1>` | **Kel 已定（2026-11-26）**：`<h1>现场开单</h1>`；页面 `<title>` 同步 |
| C16 | `web/src/app/counter/orders/page.tsx`、`admin/orders/page.tsx` | H1 | 按现状 | 分别是「本场订单」「订单记录」，跟导航一致 |
| C17 | `web/src/app/globals.css`、`web/src/app/admin/goods/page.tsx` | `:root`（第 22 行）、深色 token 块（第 1283、1366 行）；交互控件边框 | 控件边框用 `--line-strong #c6bdaa`（1.58–1.86）；深色 `#40444f`（1.65–1.96） | **Kel 已定（2026-11-26 14:13）（W-C3）**：新增 `--line-control: #8f8675`，深色 `#676c78`。建议文档 §3.1 的控件规则（浅色 33 处、深色 6 处）、附录里用 `--line` 的 5 个控件，以及 `admin/goods/page.tsx` 第 61 行 `pickerStyle` 的 `border`，都改用 `var(--line-control)`。分割线和装饰框继续用 `--line-strong`。清单见 `docs/design-v2/classic-边框对比度-建议.md` |
| C18 | `exhibition.css`、`exhibition-workspace.css`、`shop.css` 里重新定义 `--line-strong` 的 token 块 | 各块的 `--line-control` | 2026-11-26 15:04 磁盘（grep 核对）：前端已按 K-Q19=A 写入定值，`exhibition.css` 第 42、885 行 `#857e6e`，`exhibition-workspace.css` 第 36 行 `#6f8681`，`shop.css` 第 14 行 `#8f8c8b` | **Kel 已定 2026-11-26 15:01（K-Q19=A）**：只改 `--line-control`，`--line-strong` 不变（分割线、装饰框保持原色）。行号以 2026-11-26 15:04 磁盘上的 grep 结果为准：<br>• `exhibition.css` 第 31–32 行起的块（`--line-strong` 第 41 行、`--line-control` 第 42 行；在 git HEAD 里是第 23 行起、第 32 行）。选择器 `[data-storefront-theme="exhibition"]:has(> .exhibition-nav) > #main-content, [data-storefront-theme="exhibition"] :is(.exhibition-home, .ed-about, .ed-booth, .exhibition-event, .product-page, .auth-page)` 是**顾客端**页面主体 → `--line-control: #857e6e`<br>• `exhibition.css` 的 `:root:has(#storefront-shell[data-storefront-theme="exhibition"])` 块（磁盘现在第 875 行起，第 884–885 行；在 git HEAD 里是第 849 行起、第 858 行）。这是展场主题的根级 token，用的是顾客端纸色色板（`--canvas #f4f0e6`），**属于顾客端** → `--line-control: #857e6e`。工作台在自己的块里覆盖。挂在 `body` 下、不在工作台容器里的弹层会用到这个值。产品经理转达的「原 :928 现在是 :930」跟 15:04 磁盘对不上，这里写的是 grep 出来的行号；前端再插行后，以选择器为准<br>• `exhibition-workspace.css` 第 35–36 行（`[data-storefront-theme="exhibition"][data-surface="workspace"]` 和挂在 body 下的 `.select-menu__panel`）：**工作台** → `--line-control: #6f8681`<br>• `shop.css` 第 13–14 行（`body:has(#storefront-shell[data-storefront-theme="shop"])`）→ `--line-control: #8f8c8b`<br>• **`.product-page` 块**（git HEAD 第 539 行起，里面第 546 行 `--line-strong: #c6bdaa`）。更正：原来写的「不要加」和理由不对。顾客端块的选择器本身包含 `.product-page`（grep 核对过，HEAD 第 23 行、磁盘第 32 行），两块作用在同一个元素上，后面这块的 `--line-strong` 会覆盖前面那块。所以在 `--line-control: var(--line-strong)` 阶段，这块加不加都得到 `#c6bdaa`，商品页外观不变：**加了也无影响，前端已经加了，随 E16 删整块时一起删**。15:04 磁盘上，`exhibition.css` 只在第 41 行和第 884 行定义 `--line-strong`，第 553 行的 `.product-page` 块只剩布局属性，没有 token，说明这块的 token 已经随 E16 删掉。E16 之前如果这块还在，里面的 `--line-control` 应该删掉或者写成 `#857e6e`；不能留 `var(--line-strong)`，否则会把商品页控件边框盖回 `#c6bdaa`（canvas `#f4f0e6` 上 1.64，不达标） |

### 8.2 展场

| # | 文件 | 选择器 | 现值 | 目标值 |
|---|---|---|---|---|
| E1 | `web/src/app/exhibition-workspace.css` | `[data-storefront-theme="exhibition"][data-surface="workspace"]` token 块 | `--muted: #526c6d` | `#4b6465` |
| E2 | 同上 | 同上 | `--accent-light: #d6ef9f` | `#e3f7a2`（Kel 已定（2026-11-26），Q1=A） |
| E3 | 同上 | 同上 | 没有圆角和阴影 token | `--radius-control: 6px; --radius-card: 10px; --radius-panel: 13px; --shadow-1: 0 1px 2px #142e3210; --shadow-2: 0 1px 2px #142e3210, 0 10px 26px #142e320d; --shadow-rail: 6px 0 24px #0e30361a; --brand-mark: #8c3c2d; --warning-soft: #f6e4e0; --success: #215e58` |
| E4 | 同上 | `--workspace-panel-shadow` | `0 1px 2px #0f1f3310, 0 10px 26px #0f1f330d` | `var(--shadow-2)` |
| E5 | 同上 | `.workspace-sidebar` 的 `box-shadow` | `6px 0 24px #0b16281a` | `var(--shadow-rail)` |
| E6 | 同上 | `.workspace-nav__link[aria-current="page"]` | `box-shadow: inset 3px 0 #d8f69a` | `inset 3px 0 var(--accent-light)`（Kel 已定（2026-11-26），Q1=A） |
| E7 | 同上 | 窄屏选中态（第 674 行附近） | `inset 0 -2px 0 #d8f69a` | `inset 0 -3px 0 var(--accent-light)`（Kel 已定（2026-11-26），Q1=A） |
| E8 | 同上 | `.workspace-sidebar__logout` | `min-height: 36px; border-radius: .38rem` | `min-height: var(--touch); border-radius: var(--radius-control)` |
| E9 | 同上 | `@media (max-width: 900px) .workspace-sidebar__logout` | `min-height: 36px` | `min-height: var(--touch)` |
| E10 | 同上 | `.workspace-sidebar` | 整栏 `overflow: hidden`，「返回」和「退出」跟着内容一起往下排 | `.workspace-nav { flex: 1 1 auto; min-height: 0; overflow-y: auto }`；`.workspace-sidebar__foot { flex: 0 0 auto; display: grid; gap: 6px; border-top: 1px solid #ffffff1f; padding-top: 8px }` |
| E11 | 同上 | `.workspace-nav__index` | `display: none` | **Kel 已定（2026-11-26）**：保持 `display: none`，展场侧栏不显示编号（覆盖原来「宽屏显示编号」的建议）；编号只在总览卡片上显示 |
| E12 | 同上 | `.workspace-overview__heading`（第 695 行） | `linear-gradient(115deg, #0e203a, #22467c)`；眉标和正文 `#d7e4fb` | `linear-gradient(115deg, #123740, #215e58)`；眉标 `var(--accent-light)`；正文 `#dcebe3`（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E13 | 同上 | `.report-page` 的 `--muted` 覆盖（第 977–981 行） | `#4f696a` | 删掉这段覆盖（E1 已经覆盖到） |
| E14 | 同上 | `.workspace-shell :is(.workspace-card, .report-sheet, …)` | `border-radius: .8rem` | `var(--radius-panel)`；其他 `.38/.4/.55/.6/.65/.85rem` 按 §4.5 归到对应 token |
| E15 | 同上 | `@media (min-width: 1100px) and (max-width: 1330px) .counter-products` | 只在这个区间是单列 | 所有宽度都单列：`grid-template-columns: minmax(0, 1fr)` |
| E16 | `web/src/app/exhibition.css` | `[data-storefront-theme="exhibition"] .product-page`（第 538–549 行） | 整组 classic token 加 `background: #faf8f3; color: #16171a` | 删掉这段 token 覆盖，让商品页继承展场 token（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E17 | `exhibition.css` | `.product-placeholder` 一带（第 355 行） | `background: #1e36a8` | `var(--expo-charcoal)`（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E18 | `exhibition.css` | `.product-detail__placeholder`（第 592 行） | `background: #1e36a8` | `var(--expo-charcoal)`（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E19 | `exhibition.css` | `.product-kicker`、`.product-kicker::before`（第 616、626 行） | `#1e36a8` | `var(--accent)`（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E20 | `exhibition.css` | `.product-page .product-buy:hover`（第 691–692 行） | `border-color` 和 `background` 都是 `#1e36a8` | `var(--accent)`（Kel 已定（2026-11-26），Q1=A：去掉海军蓝和钴蓝） |
| E21 | `exhibition.css` | `[data-storefront-theme="exhibition"]` token 块 | 没有圆角和阴影 token | `--radius-control: 0; --radius-card: 0; --radius-panel: 0; --shadow-1: none; --shadow-2: none; --focus: #262820; --brand-mark: #8c3c2d; --warning-soft: #f3e1d8; --success: #3d5a2a` |
| E22 | `exhibition.css` | `@media (max-width: 620px) .exhibition-nav :is(nav, .exhibition-nav__settings) :is(a, button, select)`（第 821 行附近） | `font-size: .7rem`；宽度跟着 67px 竖栏减去内边距（约 60px） | **Kel 已定（2026-11-26）**：`font-size: .75rem; width: 52px; min-height: 44px; padding-inline: 0; border-radius: 0`；退出按钮同样 52×≥44px，文字和图标 `#f4f0e6`（压 `#262820` 13.12:1），`:focus-visible { outline: 2px solid var(--expo-lime); outline-offset: -2px }`（12.86:1） |
| E23 | `exhibition.css` | `.exhibition-nav__settings` | `margin-top: auto`，竖栏内容超高时会随竖栏一起滚走 | **Kel 已定（2026-11-26）**：≤620 宽时加 `position: sticky; bottom: 0; background: var(--expo-charcoal); padding-bottom: max(.5rem, env(safe-area-inset-bottom))`，退出始终钉在 52px 竖栏底部、不用滚动就能看到 |
| E24 | `web/src/components/app-nav.tsx` | `exhibition-nav__settings` 里的退出 `<button>`（第 148 行） | 只有文字「退出」，没有图标 | **Kel 已定（2026-11-26）**：加一个 18px 的 `RailIcon name="logout"`（要在 `railIcons` 里新增门加箭头的路径），样式跟其他竖栏项一致；文字保留 |
| E25 | `exhibition-workspace.css` | 工作台 token 块（第 42 行） | `--font-display: Georgia, "Songti SC", "Noto Serif SC", "STSong", serif` | **Kel 已定（2026-11-26），Q1=A**：`--font-display: "Songti SC", "Noto Serif SC", "STSong", Georgia, serif`（跟 `--expo-serif` 相同）；新增 `--expo-charcoal: #262820` |
| E26 | `exhibition-workspace.css` | `.counter-catalog-filter button[data-active="true"], .review-filter button[data-active="true"]`（第 402–406 行）及数量徽标 | `color: var(--ink)` | **Kel 已定（2026-11-26），Q1=A**：`color: var(--expo-charcoal)`（12.86:1） |
| E27 | `exhibition-workspace.css` | `.workspace-brand__mark`（第 100–106 行） | `font: 800 15px/1 Georgia, serif`；没有赭红角标 | **Kel 已定（2026-11-26），Q1=A**：`font: 800 15px/1 var(--font-display); position: relative`；`::after { content: ""; position: absolute; top: -3px; right: -3px; width: 8px; height: 8px; border-radius: 2px; background: var(--brand-mark); box-shadow: 0 0 0 2px #f4f0e6 }`（装饰，`aria-hidden`） |
| E28 | `exhibition-workspace.css` | 输入框、下拉框边框（`border-color: var(--line-strong)`，第 356、403、423、465 行一带的表单控件） | `#8ca29d`，白底上 2.70 | **Kel 已定 2026-11-26 15:01（K-Q19=A）**：token 块里 `--line-control: #6f8681`（白 3.89、canvas 3.46、surface-soft 3.22、渐变 `#d5e6df` 3.00），表单控件边框用 `var(--line-control)`；`--line-strong #8ca29d` 不变，卡片 hover 边框这类非控件继续用它 |
| E29 | `exhibition.css` | `@media (max-width: 620px)`（第 803–823 行） | `--expo-rail: 67px`；`.exhibition-nav { padding: .5rem .22rem }` | **Kel 已定（2026-11-26）（原 Q2）**：`--expo-rail: 52px`；`.exhibition-nav { padding: .5rem 0 }`。顾客端控件边框 `--line-control: #857e6e`（Kel 已定 2026-11-26 15:01，K-Q19=A，见 C18） |
| E30 | `app-nav.tsx` 第 137 行 + `exhibition.css` | `.exhibition-nav__brand` | 全称「Stallink.」，≤620 时 .95rem | **Kel 已定（2026-11-26）（原 Q2）**：品牌拆成 `<span className="exhibition-nav__brand-full">Stallink</span><span className="exhibition-nav__brand-short" aria-hidden="true">S</span><span aria-hidden="true">.</span>`；≤620 宽时隐藏 full、显示 short，字号 1.15rem；`aria-label` 不变 |

### 8.3 店面

| # | 文件 | 选择器 | 现值 | 目标值 |
|---|---|---|---|---|
| S1 | `web/src/app/shop.css` | `body:has(…shop)` token 块（第 3–39 行） | `--warning: #332f2d` | `#b42318`（之后 `--shop-danger` 可以改成 `var(--warning)`，不再单独维护） |
| S2 | 同上 | 同上 | `--line-strong: #ebebeb` | `#dedfe0` |
| S3 | 同上 | 同上 | `--accent-dark: #5433eb` | `#4524db` |
| S4 | 同上 | 同上 | 没有这些 token | `--line-control: #8f8c8b`（Kel 已定 2026-11-26 15:01，K-Q19=A；只加深描边，不采用浅灰填充方案；`--line-strong` 不变）; `--radius-control: 14px; --radius-card: 20px; --radius-panel: 28px; --radius-pill: 9999px; --shadow-1: var(--shop-shadow-chip); --shadow-2: var(--shop-shadow); --shadow-brand: var(--shop-shadow-violet); --warning-soft: #fef3f2; --success: #067647; --focus: #5433eb` |
| S5 | `shop.css` | 字体说明（第 32–35 行） | 已修：只用系统字体 | 保持；另加 `.price, .counter-product__price, .report-metric strong, table td { font-variant-numeric: tabular-nums }` |
| S6 | `shop.css` | `.workspace-page > main > h1, .workspace-page__heading h1, .workspace-overview__heading h1`（第 177–188 行） | `font-size: 20px; font-weight: 500; letter-spacing: -0.05em` | `28px; 600; letter-spacing: 0`；≤767 宽时 24px |
| S7 | `shop.css` | `.admin-overview .workspace-overview__heading h1`（第 996–1001 行） | `20px / 500` | 跟 S6 一样 |
| S8 | `shop.css` | `.workspace-sidebar`（第 1287–1295 行） | `max-height: calc(100dvh - 120px); overflow-y: auto`，整栏一起滚 | `display: flex; flex-direction: column; max-height: calc(100dvh - 112px); overflow: hidden`；`.workspace-nav { flex: 1 1 auto; min-height: 0; overflow-y: auto; margin-top: 8px }`；`.workspace-sidebar__foot { flex: 0 0 auto; border-top: 1px solid var(--line); padding-top: 8px }` |
| S9 | `shop.css` | `.workspace-nav__group`（第 1155 行附近） | `margin-bottom: 24px` | `16px` |
| S10 | `shop.css` | `.workspace-nav__index`（第 1160 行附近） | `display: none` | **Kel 已定（2026-11-26）**：保持 `display: none`，店面侧栏不显示编号（覆盖原来「宽屏显示编号」的建议）；编号只在总览卡片上显示 |
| S11 | `shop.css` | `@media (max-width: 860px)` 整段 `.workspace-nav-toggle…`（第 1175–1253 行） | 折叠按钮加展开网格 | **Kel 已定（2026-11-26）**：去掉折叠按钮，换成 `@media (max-width: 900px)`：`.workspace-nav-toggle { display: none }`；`.workspace-sidebar { position: sticky; top: 96px; z-index: 19; border-radius: 0; box-shadow: 0 1px 0 var(--line); padding: 8px 16px }`（Kel 已定 2026-11-26 15:13：≤900 宽 `top: var(--shop-header-height)`。页头高度 = `--shop-header-height`：大于 640 宽时 72px，≤640 时 64px。**复用已有的 `--shop-header-height`**：它定义在 shop.css 的 `body:has(#storefront-shell[data-storefront-theme="shop"])` token 块里，`@media (max-width: 640px)` 里改成 64px，页头 `.site-header__inner` 的 `min-height` 也引用它，所以页头和吸顶条共用同一个值。宽屏（>900）`.workspace-sidebar` 规则里的 `top: 96px` 不变，是写死的数值，不引用变量。原写的 56px 作废。**注意，代码现状有层叠顺序问题，需要前端修**：shop.css 的 `@media (max-width: 900px)` 块开头又写了一次 `body:has(#storefront-shell[data-storefront-theme="shop"])` `{ --shop-header-height: 72px }`。这一段在文件里排在 ≤640 那条 64px 声明之后，选择器优先级相同，所以 ≤640 时变量被盖回 72px。2026-11-26 15:2x 在 box 上用磁盘上的 shop.css 实测：640 和 390 宽时变量都是 72px，页头高 72px。修法：删掉 ≤900 块里重复的 72px（token 块里本来就是 72px），或者把 ≤640 的 64px 声明移到 ≤900 块之后）；隐藏品牌、身份、「返回 Stallink」和分组标签。**保留「退出登录」，只修正底色错位**（Kel 已定 2026-11-26 15:01，测试报告 D9 / plan W19）：`.workspace-sidebar__foot { display: contents }`；`.workspace-sidebar__logout` 放在吸顶条右端，跟横滑导航在同一行（导航 `flex: 1 1 0; min-width: 0` 横向滑动，退出按钮 `flex: 0 0 auto` 不被挤压；`margin: 0 0 0 8px`，覆盖宽屏规则里的 `margin-top: 12px`），`display: inline-flex; align-items: center; justify-content: center; min-height: 44px; min-width: 44px; padding: 0 12px; white-space: nowrap`。**底色就是按钮自己这个盒子的底**：`background: var(--surface)`（`#ffffff`）、`border-radius: var(--radius-pill)`、`box-shadow: var(--shop-shadow-chip)`，文字 `var(--ink)`（`#000`，21:1），跟宽屏侧栏底部的退出按钮同一套样式。白底和文字必须在同一个盒子上，不能再用 `margin-top` / `padding-top` 把底色和文字错开；胶囊下方不能空出一块；`.workspace-nav { display: flex; gap: 4px; overflow-x: auto; scroll-snap-type: x proximity; mask-image: linear-gradient(90deg, #000 88%, transparent) }`；`.workspace-nav__link { flex: 0 0 auto; background: var(--surface-soft) }`；`[aria-current="page"] { background: #000; color: #fff }` |
| S12 | `shop.css` | `@media (max-width: 860px) .workspace-sidebar`（第 1532–1539 行） | `position: static` | 合并进 S11（Kel 已定（2026-11-26）） |
| S13 | `shop.css` | 顶栏 `.site-header__actions`，在 ≤900 宽时 | 「关于」「查看店铺」挤成竖排 | 隐藏 `.header-link[href="/about"]` 和「查看店铺」；`.header-account { white-space: nowrap }` |
| S14 | `shop.css` | 收银台商品网格 | 两列卡片，名称竖排 | `.counter-products { grid-template-columns: minmax(0, 1fr) }`；`.counter-product { grid-template-columns: 48px minmax(0, 1fr) auto 44px; border: 0; background: var(--surface-soft); border-radius: var(--radius-card) }`；名称 `-webkit-line-clamp: 2` |
| S15 | `shop.css` | 收银台筛选按钮 | 方形描边按钮 | 胶囊分段：外层 `background: var(--surface-soft); border-radius: var(--radius-pill); padding: 3px`；按钮 `border-radius: var(--radius-pill); min-height: 38px`；`[aria-pressed="true"] { background: #000; color: #fff }` |
| S16 | `shop.css` | `:is(.button--dark, .ed-btn):disabled` | 白字压 `--accent-light` | `background: #ebebeb; color: var(--shop-muted-strong); box-shadow: none` |
| S17 | `shop.css` | `@media (prefers-reduced-motion: reduce)`（第 3690 行） | 只把加载圈放慢 | 改由 C6 统一处理；加载圈停转，旁边写「正在…」 |
| S18 | `shop.css` | 各处写死的 `border-radius: 28px / 20px / 14px / 9999px` | 字面值 | 按 §4.5 换成 token。可以分批做，但新代码不要再写字面值 |
| S19 | `shop.css` | `body:has(…shop)` token 块（第 9–11 行） | 原 `#787574`；前端已改为 `--shop-muted: #726f6e; --muted: var(--shop-muted)` | 已修：保持。新代码的辅助灰一律引用 `--shop-muted` 或 `--muted`，不要写字面值 |

## 9. 验收清单

按 390 / 820 / 1280 三档，在展场和店面下各走一遍：
1. 工作台导航：展场和店面 ≤900 宽时都能横向滑动，店面没有折叠按钮（Kel 已定（2026-11-26）），当前项在可视区内；1280 宽时侧栏底部「返回 Stallink」「退出登录」都看得见，不需要滚动。
2. 总览 8 张卡片和侧栏管理项同序、同号、同名，编号 01–08 连续；「现场开单」「本场订单」在侧栏里不编号、总览里没有卡片；总览本身不编号（Kel 已定 A（K-Q15））。展场和店面的侧栏不显示编号，只有 classic 显示；卡片 01、02 叫「活动」「商品」（Kel 已定（2026-11-26））。
   - 展场 390 宽：顾客端竖栏底部的「退出」不滚动就能看到，点击区域 ≥44px，焦点环可见；工作台吸顶条右上角有「退出登录」（Kel 已定（2026-11-26））。
3. 页面名称：销售日报 / 订单记录 / 本场订单，H1 跟导航名一致。
4. 销售日报工具栏在 1280 宽时 ≤2 行；390 宽时不出现横向滚动；商品名不会一字一行。
5. 收银台：商品名不竖排；≤1099 宽时底部结算栏可见，「生成订单」≥52px 高；所有加减按钮 ≥44px。
6. 展场：工作台里没有海军蓝和钴蓝；商品详情页的强调色是赭红；顶栏没有深浅色开关。
7. 店面：「仅剩 N 件」和报错显示为 `#b42318`；禁用的主按钮字色对比度 ≥4.5。
8. 对比度抽查 §7.3 里的组合都 ≥4.5（文字）或 ≥3（边框）。
9. 打开系统「减少动效」后，没有位移动画和转圈。
10. 键盘：Tab 能走完导航、筛选、加货、步进器、生成订单，焦点环清晰可见。
11. 展场 Q1=A：工作台没有海军蓝和钴蓝；柠绿底上是炭黑字；标题是 `--expo-serif`（Kel 已定（2026-11-26））；表单控件边框 ≥3:1：classic `#8f8675`、展场顾客端 `#857e6e`、展场工作台 `#6f8681`、店面 `#8f8c8b`，分割线颜色不变（Kel 已定 2026-11-26 15:01）。
12. 展场顾客端 390 宽：竖栏 52px，退出在竖栏底部、52×≥44px、不用滚动就能看到（Kel 已定（2026-11-26））。
13. 收银台 H1 是「现场开单」；classic 圆角保持 10–20px（Kel 已定（2026-11-26））。
14. 店面吸顶（Kel 已定 2026-11-26 15:13）：1280 宽滚动后，侧栏吸顶 `top: 96px`。860 宽滚动后，吸顶条上沿贴住 72px 页头的下沿；390 宽贴住 64px 页头。上方都不露出灰色空带。吸顶条右端的「退出登录」是白色胶囊，文字在胶囊正中，至少 44×44（D9）。顶栏的「退出」也还在。

对照截图：`docs/design-v2/screens/展场-*.png`、`店面-*.png`（只有工作台；展场顾客端（**本期不补顾客端原型，按文字说明实现**，Kel 已定（2026-11-26）））。原型是示意图，像素级以实现为准，结构和 token 以本文为准。

## 10. 待确认问题

**开放问题已全部关闭**（Kel 已定（2026-11-26），13:09 和 13:19 两次答复）。

| 原编号 | 问题 | Kel 的决定 | 落到哪里 |
|---|---|---|---|
| 原 Q2（13:09） | 店面窄屏导航要不要改成横滑 | 改成横滑，去掉折叠按钮 | §2 第 7 条、§6.3、S11–S12 |
| Q1 | 展场工作台主色要不要跟顾客端统一 | A：保留深青，共用炭黑、柠绿、衬线标题，去掉海军蓝和钴蓝 | §2 第 11 条、§3.1、§4.2.1、E2、E6、E7、E12、E16–E20、E25–E28 |
| Q2 | 展场顾客端手机竖栏要不要改横条 | 不改横条，67px 缩到 52px；退出固定在竖栏底部，52×≥44px | §2 第 12 条、§5.3、E22、E23、E29、E30 |
| Q3 | 901–1080 宽的 152px 窄侧栏 | 保留 | §2 第 13 条、§5.2、§5.4 |
| Q4 | 店面要不要自托管 Inter | 不自托管，用系统字体 | §2 第 14 条、§3.3、§4.3 |
| Q5 | 收银台标题 | 「现场开单」 | §2 第 15 条、§5.3、C15 |
| Q6 | classic 圆角要不要回直角 | 保持 10–20px 圆角 | §2 第 16 条、§4.5 |
