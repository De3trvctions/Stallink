# Stallink favicon 设计说明

> **已定：方案 A（Kel 已定 2026-11-26 15:07）**。「S.」字标，炭黑圆底配青柠色「S.」。
>
> D10，Kel 已定要修：现在 `/favicon.ico` 返回 404。本文只出设计，不改 `web/` 业务代码，也不把图标放进 `web/src/app` 或 `web/public`。不 commit、不 push。
>
> 配套文件在 `docs/design-v2/favicon/`，预览页为 `favicon-预览.html`，截图在 `docs/design-v2/screens/favicon-选项对比.png` 与 `favicon-小尺寸.png`。
>
> 开放问题已全部关闭。设计师原先推荐 C，记录见第 4 节；B、C 未采用，源文件保留备查。

---

## 1. 前端现状

查过 `web/src/app`、`web/public` 和根布局里的 metadata，结论如下。

| 位置 | 现状 |
| --- | --- |
| `web/src/app/favicon.ico` | 不存在 |
| `web/src/app/icon.svg` / `icon.png` | 不存在 |
| `web/src/app/apple-icon.png` | 不存在 |
| `web/src/app/manifest.ts` 或 `manifest.webmanifest` | 不存在 |
| `web/public/` | 没有 favicon、icon 或 manifest。现有内容是 `images/` 里三张摊位照片、`fonts/report/`，以及空的 `design-preview/` |
| `web/src/app/layout.tsx` 的 `metadata` | 只有 `title: "Stallink"` 和 `description: "以独立链接作为摊位入口"`，没有 `icons`、没有 `themeColor`、没有 `manifest` |
| 根布局 `<head>` | 有 `color-scheme` meta（展场 / 店面钉死 `light`，classic 为 `light dark`），没有 icon 链接 |

浏览器仍会按惯例请求 `/favicon.ico`。Next.js 16 在没有 `app/favicon.ico` 或 `public/favicon.ico` 时不会补一个默认文件，所以现在是 404。这与 Kel 要修的 D10 一致。

当前 App Router 目录是 `web/src/app`（不是仓库根下的 `app/`）。下文文件清单都按这个路径写。Next.js 版本为 16.4.6，图标文件约定见 `web/node_modules/next/dist/docs/01-app/03-api-reference/03-file-conventions/01-metadata/app-icons.md` 与 `manifest.md`。

---

## 2. 现有品牌标志（都是 CSS / 文字，没有图片资产）

仓库和 `docs/design-v2/` 里都没有品牌标志图片。现在能看到的标志都是页面里用 CSS 画的字标或圆点：

1. **顾客端展场侧栏「S.」**  
   `web/src/components/app-nav.tsx`：完整名「Stallink」，窄屏另有 `exhibition-nav__brand-short` 的「S」加上单独的「.」。  
   `exhibition.css`：宽屏侧栏 106px、衬线斜体 700、柠绿条纹 `::before`；`max-width: 620px` 时侧栏改为 52px，隐藏全称和条纹，只留「S.」。底为炭黑 `#262820`（`--expo-charcoal`），字为纸色 `#f4f0e6`（`--expo-rail-text`），柠绿为 `#e3f7a2`（`--expo-lime`）。

2. **展场工作台 logo**  
   `workspace-shell.tsx` 里是一个字母「S」。  
   `exhibition-workspace.css`：30×30 的圆角方块（`border-radius: var(--radius-control)`，工作台里是 6px），底为 `#49827a` → `#215e58` 的 150° 渐变，白字，右上角 8px 赭石方块（`--brand-mark: #8c3c2d`），外加 2px `#f4f0e6` 描边。spec 已定：赭红只出现在这个角标上，不用于按钮、文字和横幅。

3. **classic 顶栏圆标**  
   `app-nav.tsx` 的 `.site-brand__mark` 写的是「s.」。  
   `globals.css` 先把它画成 32px 钴蓝色描边圆，随后在画册样式里改成 24px 墨底纸字的直角块。

4. **店面**  
   工作台 logo 被 `shop.css` 收成 8px 胶囊圆点，颜色是紫色 `--accent: #5433eb`，字母设为透明。  
   顾客端主标题是「Stallink」加紫色句点（`.shop-hero__wordmark span`）。

已定方案 A **不用店面紫、不用 classic 钴蓝**。那两套是主题色，会随店面风格变。A 用的炭黑和青柠色就是现有 token，见第 3.1 节。

---

## 3. 设计原则

- 颜色只用现有 token。
- 字形画成 SVG path（A / C）或几何图形（B），不引用外部字体，也不依赖本机宋体 / Georgia 是否装了。
- **全站共用一个图标，不跟店面风格切换。** 标签栏、主屏、安装图标应指向同一套资产。顾客和工作人员打开的是同一个 Stallink 源。
- 16px 必须能辨认。浅色和深色浏览器标签栏都要看得见。做法是图标自带实底，而不是依赖 SVG 里的 `prefers-color-scheme`（各浏览器对 favicon 里媒体查询的支持不稳定）。对比度按 WCAG 1.4.11：图标与自身底色至少 3:1。
- `apple-icon` 必须是不透明实底、直角画布，由 iOS 裁成超椭圆。不要在素材里先画圆角。
- maskable 图标背景铺满四边，关键图形留在正中约 70%–80% 的安全区（圆形遮罩会裁掉四角）。
- `ui-ux-pro-max` 按「marketplace favicon」生成的设计系统是 Liquid Glass + Inter + 绿色市集色板。那和 Stallink 已定的纸色 / 炭黑 / 柠绿体系冲突，**不采用**。采纳的是：图标对比度、多尺寸检查、用 SVG 而不是 emoji、浅色和深色一起设计。

### 3.1 方案 A 的用色与对比度（已核对 token）

方案 A 只用两个色，都是展场 CSS 里已有的 token，不是另造的色值：

| 角色 | 色值 | Token | 出处 |
| --- | --- | --- | --- |
| 圆形底、apple / maskable 实底 | `#262820` | `--expo-charcoal` | `exhibition.css` 第 8 行；工作台 `exhibition-workspace.css` 第 47 行同样声明。侧栏背景也是 `background: var(--expo-charcoal)`（第 65 行） |
| 「S.」笔画、句点、圆环描边 | `#e3f7a2` | `--expo-lime` | `exhibition.css` 第 10 行。侧栏里用在柠绿条纹 `::before`（第 89 行）和选中态 |

需要分清：顾客端窄屏侧栏的「S.」**字本身是纸色** `#f4f0e6`（`--expo-rail-text`）叠在炭黑底上，柠绿用在条纹和选中态，并不给字上色。方案 A 把字也做成青柠色，是为了在深色标签栏上仍有足够对比；色值仍是同一个 `--expo-lime`。

按相对亮度公式实测（WCAG 2）：

| 组合 | 对比度 | 结论 |
| --- | --- | --- |
| 青柠色 `#e3f7a2` / 炭黑 `#262820` | **12.86** | 通过（非文本 3:1，按正文 4.5:1 也过） |
| 炭黑 `#262820` / 浅色标签栏约 `#dee1e6` | 11.39 | 圆底在浅色标签栏上分得清 |
| 青柠色 `#e3f7a2` / 浅色标签栏约 `#dee1e6` | 1.13 | **不通过**，所以不能做透明底只放柠绿字，必须自带炭黑圆底 |
| 炭黑 `#262820` / 深色标签栏约 `#202124` | 1.08 | 圆底外轮廓不够，所以在圆外加青柠色细描边 |
| 青柠色 `#e3f7a2` / 深色标签栏约 `#202124` | 13.87 | 描边和「S.」在深色标签栏上分得清 |
| 纸色 `#f4f0e6` / 炭黑 `#262820`（侧栏字色，备查） | 13.12 | 也通过；A 不用这个组合做字形 |

未采用的 B、C 相关对比（仅作记录）：纸色 / 深青 `#215e58` 为 6.57；赭石 / 柠绿为 6.46；赭石 / 深青为 1.00（所以 C 的角标必须留纸色描边）。

---

## 4. 三个选项

三个选项在 16px 上的外轮廓就不一样：圆标、摊位剪影、带角标的方块。请对照预览页和截图，不要只看大图。B、C 的源文件保留，便于对照，不进入前端导出清单。

### 选项 A：「S.」字标（已采用，Kel 已定 2026-11-26 15:07）

圆形炭黑底，青柠色细描边，略带斜体的「S」加句点。延续顾客端手机侧栏的「S.」，也呼应 classic 顶栏圆标和店面标题里那个句点。斜体是向展场侧栏 Georgia 斜体打招呼，但笔画是几何 path，不靠字体。

- **优点**：和顾客已经看到的字标最连续；「S.」比单独一个 S 更容易记住；圆形标签图标辨识度高。
- **缺点**：16px 上句点很小，要靠 `-16` 简化版加粗才站得住；炭黑圆在深色标签栏里几乎融进底，必须靠柠绿描边和字形本身；主屏 / maskable 若还用圆形套在超椭圆里会像一枚纽扣，所以 apple 与 maskable 改成直角实底上的「S.」，和标签栏里的圆形不完全一样。

源文件：`option-a.svg`（主文件，32 及 `icon.svg`）、`option-a-16.svg`、`option-a-apple.svg`、`option-a-maskable.svg`。

### 选项 B：摘要式摊位（未采用，留作记录）

圆角炭黑底加柠绿描边，上面是柠绿篷顶、纸色立柱和台面、篷顶上的赭石号牌。不靠拉丁字母，直接画出「摊位」。

- **优点**：一看就是摊位产品，不依赖会不会读 S；赭石号牌在柠绿篷顶上对比足够（约 6.46:1），不必另加描边；和 A、C 的字母路线差异最大。
- **缺点**：16px 上细节挤，要靠加粗立柱的简化版；抽象摊位也可能被看成售货亭或插座，辨识需要学一次；和现有页面里的「S.」/ 工作台 logo 没有直接继承关系。

源文件：`option-b.svg`、`option-b-16.svg`、`option-b-apple.svg`、`option-b-maskable.svg`。

### 选项 C：带赭石色角的字母块（未采用，留作记录）

圆角深青实底 `#215e58`（展场工作台 `--accent`），纸色「S」，右上角 8 单位左右的赭石方块，外加 2 单位纸色描边。这是把已定的工作台 logo 做成 favicon：spec 写明赭红只作 logo 角，角标在深青上只有约 1.00:1，所以描边不能省。16 / 32 用纯色底，不用工作台那个 150° 渐变，避免小尺寸发糊。apple / maskable 用直角实底，并把 S 与角标一起内收，以免超椭圆或圆形遮罩裁掉描边。

- **优点**：16px 上单字母比「S.」更清楚；深青实底在深色标签栏上比炭黑更容易看成一块（深青对 `#202124` 仍不到 3:1，但比炭黑有色差，加上纸色 S 和高对比角标，整体可辨）；直接沿用已定品牌零件；方块轮廓和 iOS / Android 应用图标的形状更合得上。
- **缺点**：更像工作人员侧的 logo，顾客端侧栏并不是这个青底方块；有人会觉得全站图标不该「偏工作台」。

源文件：`option-c.svg`、`option-c-16.svg`、`option-c-apple.svg`、`option-c-maskable.svg`。

### 决定

**Kel 已定采用 A（2026-11-26 15:07）。** 设计师原先推荐 C，理由是 16px 清晰度、深色标签栏上的块面，以及和已定工作台 logo / `--brand-mark` 的关系。Kel 选择了更贴近顾客端「S.」字标的 A。B、C 未采用，源文件和预览保留作记录。

---

## 5. 前端要输出的文件清单（只针对方案 A）

由前端把方案 A 的源文件导出到 `web/`，不要把本目录的设计稿直接拷进 `app` 或 `public`（设计稿注释是给评审看的）。B、C 的 SVG 不要导出。

按 Next.js 16 App Router 的文件约定（目录是 `web/src/app`）：

| 输出路径 | 尺寸 / 格式 | 源文件 | 怎么导出 | 框架行为 |
| --- | --- | --- | --- | --- |
| `web/src/app/favicon.ico` | ICO，内含 **16×16、32×32、48×48** 三档 | **16×16** 用 `option-a-16.svg`（16px 加粗简化版）；**32×32** 用 `option-a.svg`；**48×48** 也用 `option-a.svg`（由主文件放大，不要用 `-16` 版，否则 48px 会显得过粗） | 先用 sharp 栅格成三张 PNG（16 / 32 档保留圆形外的透明角；48 同），再打成 PNG-in-ICO。命令见下 | 自动写入 `<link rel="icon" href="/favicon.ico" sizes="any" />`。`favicon.ico` **只能放在** `src/app` 根段 |
| `web/src/app/icon.svg` | SVG，任意尺寸 | 复制 `option-a.svg`（可去掉设计稿注释，保留 `<title>`） | 直接放置，不必再导 PNG | 自动写入 `<link rel="icon" href="/icon.svg" sizes="any" type="image/svg+xml" />` |
| `web/src/app/icon.png`（可选） | 32×32 PNG | `option-a.svg` | 栅格成 32×32，保留透明角 | 作为不支持 SVG favicon 的补充。若已有多档 ICO，可以不做 |
| `web/src/app/apple-icon.png` | **180×180** 不透明 PNG | `option-a-apple.svg` | 栅格成 180×180，铺在 `#262820` 上做成不透明，不要预先圆角 | 自动写入 `<link rel="apple-touch-icon" ...>` |
| `web/public/icons/icon-192.png` | 192×192 PNG | `option-a-apple.svg`（与主屏同一构图，直角炭黑实底） | 栅格成 192×192，不透明 | 供 manifest `purpose: "any"` |
| `web/public/icons/icon-512.png` | 512×512 PNG | 同上 | 栅格成 512×512，不透明 | 供 manifest `purpose: "any"` |
| `web/public/icons/icon-192-maskable.png` | 192×192 PNG | `option-a-maskable.svg` | 栅格成 192×192，背景铺满，不透明 | 供 manifest `purpose: "maskable"` |
| `web/public/icons/icon-512-maskable.png` | 512×512 PNG | 同上 | 栅格成 512×512，背景铺满，不透明 | 供 manifest `purpose: "maskable"` |
| `web/src/app/manifest.ts` | Web App Manifest | 见下面示例 | 新建，不要和 `metadata.icons` 再写一套重复的 link | Next 会输出 `/manifest.webmanifest` |

不要把 192 / 512 做成 `src/app/icon.png`：那个文件约定是标签栏图标，会和 `icon.svg` 抢。这些大图放 `public/icons/`，由 manifest 引用。

### 5.1 导出命令示例

2026-11-26 在 Kel 的 Mac 上查过：没有 `rsvg-convert`，也没有 ImageMagick（`magick` / `convert`）。`web/node_modules/sharp@0.35.5` 已随 Next.js 安装，可以读 SVG、写 PNG，但不能写 ICO。下面脚本用 sharp 出 PNG，再用内置的 PNG-in-ICO 封装写出 `favicon.ico`，不另装依赖。

在仓库根目录、`web/` 依赖已安装的前提下运行。脚本只是示例，设计阶段不要真的往 `web/src/app` 或 `web/public` 写文件；等前端实施时再输出到这些路径。

```bash
cd web && node --input-type=module <<'JS'
import { mkdir, writeFile, readFile } from "node:fs/promises";
import path from "node:path";
import sharp from "sharp";

const root = path.resolve("..");
const src = path.join(root, "docs/design-v2/favicon");
const outApp = path.join(root, "web/src/app");
const outIcons = path.join(root, "web/public/icons");
const CHARCOAL = "#262820";

async function raster(file, size, { flatten = false } = {}) {
  const input = path.join(src, file);
  const meta = await sharp(input).metadata();
  const base = meta.width || 32;
  let img = sharp(input, { density: 72 * (size / base) }).resize(size, size);
  if (flatten) img = img.flatten({ background: CHARCOAL });
  return img.png().toBuffer();
}

function toIco(pngs) {
  const count = pngs.length;
  const headerSize = 6 + 16 * count;
  let offset = headerSize;
  const entries = pngs.map((p) => {
    const entry = { size: p.size, bytes: p.data.length, offset };
    offset += p.data.length;
    return entry;
  });
  const buf = Buffer.alloc(offset);
  buf.writeUInt16LE(0, 0);
  buf.writeUInt16LE(1, 2);
  buf.writeUInt16LE(count, 4);
  entries.forEach((e, i) => {
    const o = 6 + i * 16;
    buf.writeUInt8(e.size, o);
    buf.writeUInt8(e.size, o + 1);
    buf.writeUInt8(0, o + 2);
    buf.writeUInt8(0, o + 3);
    buf.writeUInt16LE(1, o + 4);
    buf.writeUInt16LE(32, o + 6);
    buf.writeUInt32LE(e.bytes, o + 8);
    buf.writeUInt32LE(e.offset, o + 12);
  });
  let pos = headerSize;
  for (const p of pngs) {
    p.data.copy(buf, pos);
    pos += p.data.length;
  }
  return buf;
}

const ico = toIco([
  { size: 16, data: await raster("option-a-16.svg", 16) },
  { size: 32, data: await raster("option-a.svg", 32) },
  { size: 48, data: await raster("option-a.svg", 48) },
]);

await mkdir(outApp, { recursive: true });
await mkdir(outIcons, { recursive: true });
await writeFile(path.join(outApp, "favicon.ico"), ico);
await writeFile(path.join(outApp, "icon.svg"), await readFile(path.join(src, "option-a.svg")));
await writeFile(path.join(outApp, "apple-icon.png"), await raster("option-a-apple.svg", 180, { flatten: true }));
await writeFile(path.join(outIcons, "icon-192.png"), await raster("option-a-apple.svg", 192, { flatten: true }));
await writeFile(path.join(outIcons, "icon-512.png"), await raster("option-a-apple.svg", 512, { flatten: true }));
await writeFile(path.join(outIcons, "icon-192-maskable.png"), await raster("option-a-maskable.svg", 192, { flatten: true }));
await writeFile(path.join(outIcons, "icon-512-maskable.png"), await raster("option-a-maskable.svg", 512, { flatten: true }));
console.log("exported favicon.ico, icon.svg, apple-icon.png, and public/icons/*");
JS
```

说明：

- 16 档必须用 `option-a-16.svg`，32 和 48 档必须用 `option-a.svg`。
- `favicon.ico` 的三档不要 `flatten`，圆外的透明角要保留，标签栏里才是圆形。
- `apple-icon` 和 manifest 的 PNG 要 `flatten` 到 `#262820`，保证不透明实底。
- `icon.svg` 直接复制主文件即可；若要去掉设计稿注释，保留 `<title>Stallink</title>`。
- 可选的 `icon.png`（32×32，源同样是 `option-a.svg`，不 flatten）需要时再加一行即可。

### 5.2 `manifest.ts` 示例

字段按已定方案写：`name`、`short_name`、`icons`、`background_color`、`display`。**不要加 `theme_color`**（也不要在根布局加 `metadata.themeColor` 或 `<meta name="theme-color">`）。`background_color` 可以保留：它是 iOS 和 Android 启动画面的底色，不是 theme-color；方案 A 用炭黑 `#262820`，与 apple-icon 实底一致。

`start_url: "/"` 不在上面那份字段名单里，但安装校验通常会要求它，所以示例里一并写出，避免 manifest 通不过校验。

```ts
import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Stallink",
    short_name: "Stallink",
    start_url: "/",
    display: "standalone",
    background_color: "#262820",
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
      { src: "/icons/icon-192-maskable.png", sizes: "192x192", type: "image/png", purpose: "maskable" },
      { src: "/icons/icon-512-maskable.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
  };
}
```

---

## 6. 和现有 metadata 的配合

根布局现在只有 title 和 description。按上一节放进文件约定后，Next.js 会自己往 `<head>` 加 favicon、icon、apple-icon 和 manifest 的 link，**不必再写 `metadata.icons`**。两边都写会造成重复的 `<link>`。

需要知道的一点：Next 给 `favicon.ico` 写的是 `sizes="any"`。部分 Chromium 会因此优先用 ICO，而不是 SVG。这可以接受——ICO 里已经有清晰的 32px。若以后一定要 Chrome 走 SVG，再单独讨论是不是不用文件约定、改手写 `metadata.icons` 并给 ICO 标准确尺寸；本期不必为此改约定。

**本期不加 theme-color**：根布局不要写 `metadata.themeColor` 或 `<meta name="theme-color">`，`manifest.ts` 也不要写 `theme_color`。展场和店面钉死浅色，classic 还可以切深色，单一的 theme-color 没法同时伺候三套主题和深浅模式。浏览器工具栏颜色仍跟页面走，现有的 `color-scheme` meta 已经在做这件事。manifest 的 `background_color` 只用于安装后的启动画面，与 theme-color 不是一回事，按方案 A 写成 `#262820` 即可。

---

## 7. 与店面风格切换的关系

全站只用方案 A 这一套图标，不随 classic / exhibition / shop 切换，也不随 classic 的深浅色切换。

理由：标签栏、书签、系统主屏会缓存图标；按主题换图标既不稳定，也会让「这是不是同一个站点」变模糊。三套主题已经靠页面色板区分，图标负责认出 Stallink 这个产品。

因此源文件不用 `prefers-color-scheme`，也不做三套主题变体。

---

## 8. 开放问题

开放问题已全部关闭。

1. **采用哪一个 favicon 方案？** — **已定 A**（Kel 已定 2026-11-26 15:07）。设计师原先推荐 C，见第 4 节。B、C 未采用。

前端按第 5 节导出时，仍先等实施结果确认再 commit。

---

## 9. 验收标准

前端落地后，按下面几条验收：

1. **`/favicon.ico` 返回 200**，响应体是 ICO，并且内含 16×16、32×32、48×48 三档（16 档来自 `option-a-16.svg`，32 / 48 档来自 `option-a.svg`）。
2. **浅色和深色浏览器标签栏上都能看清「S.」**：16px 下能认出字形和句点。对照 `favicon-预览.html` 里的模拟标签栏，以及 `docs/design-v2/screens/favicon-选项对比.png`、`favicon-小尺寸.png`。炭黑圆在深色标签栏上靠青柠色描边和字形辨认，不要做成透明底。
3. **iOS 添加到主屏幕后是实底**：使用 `apple-icon.png`（180×180，炭黑 `#262820` 铺满，无透明边、无预切圆角），由系统裁成超椭圆。对照预览页的 iOS 主屏一栏。
4. **manifest 能通过校验**：打开 `/manifest.webmanifest`（或 Chrome DevTools Application → Manifest，也可用 Lighthouse 的 PWA / installability 检查）。应含 `name`、`short_name`、`start_url`、`display`、`background_color: "#262820"`，以及 192 / 512 的 `any` 与 `maskable` 图标。清单里**不应出现** `theme_color`。图标文件都能打开，maskable 图是铺满的实底。

---

## 10. 交付文件

均未写入 `web/`，也未 commit。B、C 的源文件未删除，仅作未采用记录。

```
docs/design-v2/favicon/option-a.svg
docs/design-v2/favicon/option-a-16.svg
docs/design-v2/favicon/option-a-apple.svg
docs/design-v2/favicon/option-a-maskable.svg
docs/design-v2/favicon/option-b.svg
docs/design-v2/favicon/option-b-16.svg
docs/design-v2/favicon/option-b-apple.svg
docs/design-v2/favicon/option-b-maskable.svg
docs/design-v2/favicon/option-c.svg
docs/design-v2/favicon/option-c-16.svg
docs/design-v2/favicon/option-c-apple.svg
docs/design-v2/favicon/option-c-maskable.svg
docs/design-v2/favicon/favicon-预览.html
docs/design-v2/favicon/favicon-设计说明.md
docs/design-v2/screens/favicon-选项对比.png
docs/design-v2/screens/favicon-小尺寸.png
```

预览页用查询参数 `?shot=compare` / `?shot=pixels` 分别截对比图和 16px 放大图。直接打开 `favicon-预览.html` 会看到完整一页（含放大区）。方案 A 已定后，对比图已按「已定」徽标重截。
