# classic 风格控件边框对比度建议（`--line-strong #c6bdaa`）

> **已采纳（Kel 2026-11-26），附录 5 个控件一并改用 --line-control。** 实施任务见 `docs/superpowers/plans/2026-11-26-stallink-workspace-polish.md` 任务 W-C3。

- 日期：2026-11-26
- 状态：§1–§4 已采纳（Kel 2026-11-26 14:13，W-C3）；§5 展场和店面的取值已定（Kel 已定 2026-11-26 15:01，K-Q19=A）。本文只出建议，不改代码
- 依据：WCAG 2.2 SC 1.4.11 非文本对比度。识别交互控件所需的视觉信息（例如输入框、描边按钮的边框）跟相邻颜色的对比度要 **≥3:1**。禁用状态、纯装饰、不承载信息的分割线不受这条约束。
- 计算：WCAG 相对亮度公式，`(L1 + 0.05) / (L2 + 0.05)`。数值保留两位小数，不四舍五入到达标。
- 代码基线：2026-11-26 13:33（UTC+8）时 Mac 上 `web/src` 的快照（含未提交改动）。下面的行号都以这份快照为准。
- 相关文档：`docs/superpowers/specs/2026-11-26-stallink-theme-exhibition-shop-design.md` §4.2 和改动清单 C17、C18（2026-11-26 14:20 已同步：classic `--line-control: #8f8675`，深色 `#676c78`；展场和店面先写 `var(--line-strong)`）。

## 1. 结论

| | 颜色 | LCh（L / C / h） | canvas `#faf8f3` | surface `#fffdf8` | 白 `#ffffff` | 顶栏 `#fffefa` | surface-soft `#f0ece2` |
|---|---|---|---|---|---|---|---|
| 现在 `--line-strong` | `#c6bdaa` | 76.9 / 10.7 / 90.3 | **1.76** ✗ | **1.83** ✗ | **1.86** ✗ | **1.85** ✗ | **1.58** ✗ |
| **推荐 `--line-control`** | **`#8f8675`** | 56.3 / 10.4 / 87.8 | **3.39** ✓ | **3.54** ✓ | **3.60** ✓ | **3.57** ✓ | **3.05** ✓ |
| 备选（变化更小） | `#968d7b` | 58.9 / 10.8 / 88.9 | 3.10 ✓ | 3.23 ✓ | 3.28 ✓ | 3.26 ✓ | 2.78 ✗ |
| 参考：`--line` | `#dad3c3` | 84.7 / 8.8 / 92.1 | 1.40 ✗ | 1.47 ✗ | 1.49 ✗ | 1.48 ✗ | 1.26 ✗ |

**推荐 `#8f8675`**：
- **同色系**：色相（h 87.8°，原值 90.3°）和彩度（C 10.4，原值 10.7）基本不变，只降低明度，所以仍是 design-v2 的暖灰褐，不会偏黄或偏冷。
- **变化最小、所有背景都达标**：在同色相、同彩度的颜色里，这是在 canvas、surface、白和 surface-soft 上都 ≥3:1 的最浅一档（ΔE76 = 20.6）。
- surface-soft 必须算进去：文件选择框（`input[type="file"]`）、店铺链接前缀（`.slug-input > span`）、审核开关的轨道（`.review-switch__track`）的底色就是 `#f0ece2`。
- 备选 `#968d7b` 再浅一点（ΔE 17.9），但在 surface-soft 上只有 2.78。只有这三处改成跟外侧白底比较时才能用，所以不推荐。

**不直接改 `--line-strong`**：`--line-strong` 还画了约 30 处分割线、表格线和装饰框（见 §3.3）。直接加深会让编辑式版面的细线变重，偏离 design-v2。建议新增一个 token，只用在交互控件上：

```css
:root {
  --line-strong: #c6bdaa;   /* 不变：分割线、装饰框 */
  --line-control: #8f8675;  /* 新增：交互控件边框，≥3:1 */
}
```

## 2. 深色模式（classic 有深色模式，顺带算了）

| | 颜色 | 深色 canvas `#0e1014` | 深色 surface `#16181d` | 深色 surface-soft `#1e2129` |
|---|---|---|---|---|
| 现在 `--line-strong` | `#40444f` | 1.96 ✗ | 1.82 ✗ | 1.65 ✗ |
| 推荐 `--line-control` | `#676c78`（同色相，h 275.9°，原值 278.4°） | 3.62 ✓ | 3.38 ✓ | 3.06 ✓ |

改在 `globals.css` 第 1283 行（跟随系统的深色模式）和第 1366 行（手动选深色）这两个 token 块里，各加一行 `--line-control: #676c78`。

## 3. 用到 `#c6bdaa` 的地方和分类

`#c6bdaa` 在 `globals.css` 第 22 行定义为 `--line-strong`。classic 下所有 `var(--line-strong)` 都是这个颜色。另外有两处写死：`exhibition.css` 第 389 行 `.product-card__tag`（标签描边，装饰）和第 546 行 `.product-page` 的 classic token 覆盖（展场 spec 的 E16 已经要求删掉）。

### 3.1 需要达标：交互控件边框 → 改用 `var(--line-control)`

这些控件靠边框才能看出「这里能点、能输入」。

| 文件:行 | 选择器 | 控件 | 背景（控件内 / 外侧） |
|---|---|---|---|
| globals.css:190 | `.button--outline` | 描边次按钮 | 透明，落在 canvas 或 surface 上 |
| globals.css:422 | `.field input` | 输入框 | surface / canvas |
| globals.css:423 | `.field textarea` | 多行输入框 | surface / canvas |
| globals.css:1137 | `.field select` | 下拉框 | surface / canvas |
| globals.css:500 | `.workspace-toolbar button:not(.button)` | 工具栏按钮 | surface |
| globals.css:508 | `.workspace-page input:not([type="checkbox"]):not([type="file"]), .workspace-page select, .workspace-page textarea` | 工作台表单控件 | surface |
| globals.css:523 | `.workspace-page form button.button--outline, .workspace-page form .select-menu__trigger, .workspace-page form .counter-sheet__close, .workspace-page form .goods-modal__close` | 描边按钮、下拉触发器、关闭按钮 | surface |
| globals.css:527 | `.workspace-page li button` | 列表行内按钮 | surface |
| globals.css:581 | `.pay-code-form input[type="file"]`（虚线） | 收款码上传 | surface-soft / surface |
| globals.css:2198 | `.workspace-page input[type="file"]`（虚线） | 文件选择 | surface-soft / surface |
| globals.css:761 | `.tier-editor__remove, .tier-editor__add` | 阶梯价增删按钮 | surface |
| globals.css:1010 | `.workspace-page form button.tier-editor__remove, .workspace-page form button.tier-editor__add` | 同上（表单内） | surface |
| globals.css:798 | `.goods-search` | 商品搜索框 | surface |
| globals.css:903 | `.counter-catalog-search input` | 收银台搜索框 | surface |
| globals.css:1222 | `.workspace-page form .counter-catalog-filter button` | 收银台筛选按钮 | surface |
| globals.css:973 | `.quantity-control` | 数量步进器外框 | surface |
| globals.css:1122 | `.header-account` | 顶栏「工作台 / 创建账号」胶囊链接 | 顶栏 `#fffefa` |
| globals.css:1495 | `.draft-gallery__upload` | 图片上传按钮 | surface |
| globals.css:1502 | `.image-picker` | 图片选择器 | surface |
| globals.css:2054 | `.counter-page .counter-sheet__close`（≤1099px） | 结算抽屉关闭按钮 | surface |
| globals.css:2259 | `.pay-dialog__close` | 收款弹窗关闭按钮 | surface |
| globals.css:2431 | `.goods-view-tabs` | 商品浏览方式分段控件 | surface |
| globals.css:2497 | `.good-listing-editor__release` | 「下架」按钮 | surface |
| globals.css:2558 | `.bundle-existing-picker__head button` | 套餐选品按钮 | surface |
| globals.css:2574 | `.workspace-page form .bundle-existing-picker__head button` | 同上（表单内） | surface |
| globals.css:2559 | `.bundle-existing-picker input` | 套餐选品搜索框 | surface |
| globals.css:2592 | `.review-switch` | 审核开关外框 | surface |
| globals.css:2599 | `.review-switch__track` | 开关轨道 | surface-soft / surface |
| globals.css:2614 | `.report-date input` | 日报日期输入 | surface |
| globals.css:2776 | `.select-menu__trigger` | 自定义下拉触发器 | surface |
| globals.css:2838 | `.workspace-page .select-menu__trigger` | 同上（工作台） | surface |
| globals.css:2992 | `.slug-input > span` | 店铺链接输入框的前缀（属于输入框边界的一部分） | surface-soft / surface |
| globals.css:3022 | `.catalog__box` | 顾客端作品搜索框 | 白 / canvas |
| admin/goods/page.tsx:61 | `pickerStyle`（内联样式 `border: "1px solid var(--line-strong)"`） | 商品图片选择按钮（Kel 2026-11-26 14:13 并入 W-C3） | surface |
| globals.css:1307、1390 | 深色模式 `& .button--outline` | 同 :190 | 深色 surface |
| globals.css:1312、1395 | 深色模式 `& .workspace-page form button.button--outline, …select-menu__trigger, …counter-sheet__close, …goods-modal__close` | 同 :523 | 深色 surface |
| globals.css:1322、1405 | 深色模式 `& .workspace-page form .counter-catalog-filter button` | 同 :1222 | 深色 surface |

hover 和聚焦态已经换成 `--accent`（`#1e36a8`，在 surface 上 9.62），不用改。

### 3.2 建议同步（有别的提示，但改了更清楚）

| 文件:行 | 选择器 | 理由 |
|---|---|---|
| globals.css:565 | `.goods-event-option` | 选项框里有原生复选框，复选框自己就能看出是控件。外框是整块可点区域的边界，建议跟着改 |
| globals.css:913 | `.counter-product--bundle` | 收银台商品卡本身是按钮（`button.counter-product`），套餐变体用这条边框。普通商品卡用的是更浅的 `--line`（见附录） |
| globals.css:2312 | `.workspace-shell .stock-meter__bar` | 库存条是 `role="img"` 的图形，信息在填充段上；外框让人看出总长度，建议达标 |
| globals.css:192 | `.text-link`（`text-decoration-color`） | 链接下划线。链接文字颜色本身能区分时可以不改；如果文字跟正文同色，下划线就是唯一提示，建议改 |

### 3.3 可以保留 `#c6bdaa`：装饰、分割线、状态提示、hover、禁用

| 类别 | 文件:行 和选择器 |
|---|---|
| 版面分割线 | globals.css:267 `.landing-notes`（上下线）、:290 `.store-hero__event--empty`、:305 `.section-heading`、:381 `.event-head--detail`、:1875 `.workspace-shell .event-goods-group`、:1904 `.workspace-shell .review-shop`、:2619 `.report-sheet__head`、:2622 `.report-hero`、:2634 `.report-columns`、:2663 `.report-discount`、:2703 / :2747 `.report-discount__figures`（虚线） |
| 容器和弹层（靠阴影和内容区分，不是控件） | :601 `.pay-dialog__cash`（说明文字框）、:772 `.event-goods-alert`、:804 `.goods-modal`、:934 `.counter-toast`、:2799 `.select-menu__panel`、:2970 `.confirm-dialog` |
| 状态胶囊（文字已经表达状态） | :2935 `.order-status` |
| 图片框和占位 | :595 `.pay-code-row__thumb`、:787 `.goods-image-empty`、:828 `.bundle-component-draft`、:863 `.review-loading`、:1488 `.draft-gallery__empty`、:1870 `.workspace-shell .empty-state`、:2262 `.pay-dialog__qr`、:2964 `.pay-code-form__preview`、:3002 `.pay-dialog__qr--fallback` |
| 小图标、圆点、分隔符（旁边有文字） | :348 `.event-status i`、:552 `.account-status--disabled i`、:388 `.loading-mark`（转圈的底环，动的那段是强调色）、:457 `.gate-state__mark`、:965 / :1983 `.counter-product__price i`、:2469 `.goods-card__chips li i`、:1637 `.ed-crumb .ed-sep` |
| hover 态（不属于识别控件所需的信息） | :340 `.event-card:hover`、:619 `.workspace-card:hover`、:2459 `.goods-card:hover`、`admin/reports/reports.css`:78 `.report-range-day:hover` |
| 禁用态（WCAG 豁免） | :923 `.counter-product__add--off`、:976 `.quantity-control button:disabled, …[aria-disabled="true"]`（文字色） |
| 展场写死的值 | `exhibition.css`:389 `.product-card__tag`（装饰标签）、:546 `.product-page` 的 token 覆盖（E16 删除） |

## 4. 给前端的改动（采纳后再做）

1. `globals.css` 第 22 行后面加 `--line-control: #8f8675;`；第 1283 行、第 1366 行的深色 token 块各加 `--line-control: #676c78;`。
2. §3.1 表里的选择器：`var(--line-strong)` → `var(--line-control)`。CSS 里浅色 33 处、深色 6 处（深色是同 3 组选择器在两个深色块里各一次），再加 `admin/goods/page.tsx` 第 61 行 `pickerStyle` 的内联 `border`，共 40 处；附录里用 `--line` 的 5 个控件也一并改（Kel 2026-11-26 14:13）。§3.2 视情况一起改。
3. §3.3 不动。
4. 展场和店面目前**没有**定义 `--line-control`。如果只在 `:root` 加，这两个主题会继承 classic 的 `#8f8675`，控件边框会突然变色。所以 W-C3 要在展场和店面重新定义 `--line-strong` 的每个块里，紧跟着补一行 `--line-control: var(--line-strong)`，这样两个主题的显示不受影响：
   - `exhibition.css` 第 32 行（`[data-storefront-theme="exhibition"]`）、第 928 行（`:root:has(#storefront-shell[data-storefront-theme="exhibition"])`）
   - `exhibition-workspace.css` 第 35 行（工作台 token 块）
   - `shop.css` 第 13 行（`body:has(#storefront-shell[data-storefront-theme="shop"])`）
   - `exhibition.css` 第 546 行 `.product-page` 块（git HEAD 行号）。**更正（2026-11-26 15:06，产品经理转达，grep 核实）**：原来写的「不要加」和理由不对。顾客端块的选择器（HEAD 第 23 行）本身包含 `.product-page`，两块作用在同一个元素上，这块的 `--line-strong: #c6bdaa` 会覆盖前一块的值。所以在 `var(--line-strong)` 阶段，加不加商品页控件都是 `#c6bdaa`，外观不变：**加了也无影响，前端已经加了，随 E16 删整块时一起删**。15:04 磁盘上这块的 token 已经不在了。E16 之前如果这块还在，K-Q19 之后不能留 `var(--line-strong)`，否则会把 `#857e6e` 盖回 `#c6bdaa`（canvas 上 1.64）。

   展场和店面的 `--line-strong` 本身也不达标，所以控件边框另取定值，见 §5（Kel 已定 2026-11-26 15:01）。行号是 13:33 快照的；W-C3 落地后磁盘上分别是 `exhibition.css` 第 41–42 行、第 884–885 行，`exhibition-workspace.css` 第 35–36 行，`shop.css` 第 13–14 行。
5. 展场 spec §4.2 的 classic 一列已改成 `--line-control: #8f8675`（深色 `#676c78`），展场和店面写 `var(--line-strong)`；改动清单加了 C17、C18（2026-11-26 14:13 之后）。

验收：在 classic 浅色和深色下，各打开一个表单页（商品编辑）、收银台和销售日报，用对比度工具抽查输入框、描边按钮、下拉框和文件选择框的边框，都 ≥3:1；版面分割线的颜色不变。

## 5. 展场和店面的 `--line-strong` 实测和控件边框定值（Kel 已定 2026-11-26 15:01）

这一节是 2026-11-26 14:20 按 Kel 的要求加的，取值来自 Mac 上当前磁盘的 `web/src`。W-C3 只把这三处接成 `--line-control: var(--line-strong)`，颜色不变，所以下面这些不达标的情况在 W-C3 之后仍然存在。

### 5.1 现值和对比度（UI 组件标准 3:1）

| 主题 | `--line-strong` 定义位置 | 现值 | 主要背景上的对比度 | 结论 |
|---|---|---|---|---|
| 展场·顾客端 | `exhibition.css`:32、:928 | `#999281` | canvas `#f4f0e6` **2.72**；surface `#fcf9f0` **2.94**；surface-soft `#e8e2d5` **2.40**；白 3.10 | **不达标**（只有白底上过线） |
| 展场·工作台 | `exhibition-workspace.css`:35 | `#8ca29d` | 白（surface、面板）**2.70**；面板渐变 `#f4faf6` **2.55**；canvas `#edf3f1` **2.41**；surface-soft `#e2ece8` **2.24**；canvas 渐变最深处 `#d5e6df` **2.09** | **不达标** |
| 店面 | `shop.css`:13 | `#ebebeb`（14:20 快照）；15:03 磁盘上已是 `#dedfe0` | `#ebebeb`：白 **1.19**、canvas **1.08**；`#dedfe0`：白 **1.33**、canvas **1.21** | **严重不达标**；店面的 `--line` 是 `#ebebeb` |

在这些主题里，用到 `var(--line-strong)` 的控件包括：展场工作台 `exhibition-workspace.css` 第 357、404、424、467 行一带的表单控件和筛选按钮；`globals.css` 里没有被主题覆盖的 §3.1 规则（W-C3 之后改读 `--line-control`，值等于各主题的 `--line-strong`）。

### 5.2 跟 spec 里原来写的建议值是否一致

展场和店面 spec 先前给控件边框写过建议值：展场工作台 `#738b86`、展场顾客端 `#857e6e`、店面 `#8f8c8b`。**这三个值都不等于当前代码里的 `--line-strong`**，它们是给 `--line-control` 单独设的、更深的值，代码里还没有。14:20 时 spec §4.2 暂写为 `var(--line-strong)`；15:01 Kel 定了 K-Q19=A，取值见 §5.3。复核结果：

| 原建议 | 对比度 | 复核 |
|---|---|---|
| 展场·顾客端 `#857e6e` | canvas 3.54、surface 3.83、surface-soft 3.12、白 4.03 | 通过，可以沿用 |
| 展场·工作台 `#738b86` | 白 3.64、面板 `#f4faf6` 3.44、canvas 3.24、surface-soft 3.02、渐变最深处 `#d5e6df` **2.81** | 在渐变最深处不达标（输入框直接放在页面底色上时会碰到）。**已作废**，展场工作台统一用 `#6f8681`（Kel 已定 2026-11-26 15:01） |
| 店面 `#8f8c8b` | 白 3.34、canvas 3.03 | 通过，可以沿用 |

### 5.3 定值（Kel 已定 2026-11-26 15:01，K-Q19=A）

做法跟 classic 一样：保留 `--line-strong` 给分割线和装饰框，只把各主题的 `--line-control` 从 `var(--line-strong)` 改成下面的值。三个值都保持原色的色相和彩度，只降低明度。

| 主题 | 建议 `--line-control` | LCh（L / C / h） | 各背景对比度 | 说明 |
|---|---|---|---|---|
| 展场·顾客端 | **`#857e6e`** | 53.0 / 9.7 / 91.3（原 60.7 / 9.9 / 92.4） | canvas 3.54、surface 3.83、surface-soft 3.12、白 4.03 | 就是原建议值。刚好过线的最浅值是 `#888170`（surface-soft 正好 3.00），没有余量，所以不推荐 |
| 展场·工作台 | **`#6f8681`** | 54.0 / 9.3 / 180.5（原 64.8 / 8.7 / 180.0） | 白 3.89、面板 3.67、canvas 3.46、surface-soft 3.22、渐变最深处 3.00 | 替换原建议 `#738b86`（已作废）。这是所有工作台背景都 ≥3:1 的最浅值，展场工作台统一用它 |
| 店面 | **`#8f8c8b`** | 58.5 / 1.3 / 45.0（原 `#ebebeb` 93.0 / 0） | 白 3.34、canvas 3.03 | 就是原建议值，暖灰，跟店面的 `--shop-muted #726f6e` 同一个色调。中性灰的最浅过线值 `#8e8d8d` 在 canvas 上正好 3.00，没有余量。店面视觉改动最大（从几乎看不见变成清楚的细线），「浅灰填充加描边」方案**不采用**（Kel 已定 2026-11-26 15:01） |

**Kel 已定 2026-11-26 15:01（K-Q19=A）**：采用上表三个值。只加深控件边框，`--line-strong` 不变，分割线和装饰框保持原色。展场工作台统一用 `#6f8681`，不再用 `#738b86`。**店面不采用「浅灰填充加描边」方案**，控件只加深描边。

落到代码（W-C3 之后，按 2026-11-26 15:03 磁盘行号）：
- `exhibition.css` 第 42 行（顾客端页面主体块）和第 885 行（`:root:has(#storefront-shell[data-storefront-theme="exhibition"])`，展场根级 token，也是顾客端色板）：`--line-control: var(--line-strong)` → `#857e6e`
- `exhibition-workspace.css` 第 36 行（工作台块）：→ `#6f8681`
- `shop.css` 第 14 行：→ `#8f8c8b`（店面 `--line-strong` 现在是 `#dedfe0`，白底 1.33、canvas 1.21，保持不变）
- 同步：展场 spec §4.2、§4.2.1、§7.3、C18、E28、E29、S4、§9 第 11 项已改成定值。

## 附录：用更浅的 `--line #dad3c3` 画的控件（1.26–1.49，不在本次 `#c6bdaa` 范围内）

grep 时顺带发现，下面这些控件用的是 `--line`，比 `#c6bdaa` 还浅。建议一起换成 `var(--line-control)`：

| 文件:行 | 选择器 | 说明 |
|---|---|---|
| globals.css:790 | `.product-gallery__thumbs button` | 商品图缩略图按钮 |
| globals.css:834 | `.bundle-component-draft__images button` | 22px 圆形删除按钮 |
| globals.css:859 | `.review-filter button` | 审核筛选按钮 |
| globals.css:905 | `.counter-catalog-filter button` | 收银台筛选按钮（工作台表单内会被 :1222 覆盖成 `--line-strong`） |
| globals.css:1012 | `.workspace-page form button.counter-product` | 收银台商品卡按钮 |

`globals.css`:1015–1016 的 `.counter-product--unavailable` 是禁用态，豁免。:1448 / :1520 / :1521 的 `.order-date-filter .button` 只是分段按钮之间的内部分隔线，可以保留。
