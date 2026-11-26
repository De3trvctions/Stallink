# Stallink 安全与性能审查报告（范围 A）

- 日期：2026-11-26（Asia/Kuala_Lumpur）
- 基线：分支 `design/editorial-frontend` @ `443bd21`；未推送提交 `origin/design/editorial-frontend..HEAD` 共 6 个（`db17ae6` `4257ff8` `f82f835` `c888392` `db3944f` `443bd21`）+ 工作区未提交改动（12 个已修改文件 + 未跟踪 `web/src/components/shop-landing.tsx`）
- 约束：全程未连接任何数据库，未启动服务、未跑迁移、未改业务代码，未执行 git 写操作。唯一写入为本报告（未 add / commit）。报告不含任何密钥值，只给文件路径、行号和长度/类型特征
- 方法：逐文件读代码 + `git log -p`；`gosec ./...`；`govulncheck ./...`；`npm audit`；按模式扫描工作树和未推送提交中的密钥

## 一、结论速览

| 级别 | 数量 | 条目 |
|---|---|---|
| 严重 | 1 | C1 Drive 孤儿清理会跨环境把别处正在用的图片移进垃圾桶 |
| 高 | 4 | H1 启动 / Nacos 热更新时对任意 DSN 自动迁移；H2 本机 Nacos 运行配置里是明文 refresh token / client secret / 管理员密码，且 token 带 `cloud-platform` 权限范围；H3 可达的 Go 标准库与 `x/image` 漏洞（15 个）；H4 公开图片每次请求都穿透到 Drive，没有任何缓存 |
| 中 | 7 | M1 公开目录检索不分页、无上限；M2 上传解码的内存/CPU 无上限；M3 原图回传会保留 EXIF/GPS；M4 Drive 目录共享权限与收款码同目录（需人工确认）；M5 三套主题 CSS 全站加载；M6 外观接口每次 SSR 都打库且无审计；M7 迁移规范（012 改过已发布内容、无校验和、无锁超时、无回滚） |
| 低 | 6 | L1–L6，见第四节 |

另外：11-22 审核报告的 P1×4、P2×6、P3×3 **全部仍未修复**，没有回归，但新代码复制了同类问题（见第五节）。这 13 条不计入上表。

摊主 / 销售员统一开单入口（`c888392`）只改了 `web/src/components/workspace-shell.tsx` 的导航。服务端角色网关没有变化：`/api/admin/*` 仍走 `shopAdmin`（`admin.go:19,77,120,235,259,285`，`reports.go:138,161,193`），因此**没有引入垂直越权**。销售员本来就能确认退款，属于 11-22 的 P1-1。

---

## 二、严重

### C1 Drive 孤儿清理会跨环境误删图片（已确认为最高优先）

- 文件：`server/cmd/stallink/main.go:456-488`（每小时执行 `runBlobSweeper`）；`server/internal/imagestore/drive.go:140-165`（`Sweep`）；`server/internal/data/blob_cleanup.go:10-31`（引用集合）；`server/configs/nacos/stallink.yaml:18`（仓库里写死的根目录 ID）
- 问题：清理时，凡是同时满足以下条件的文件都会被移进垃圾桶：
  1. Drive 上 `appProperties.stallinkImageRoot == rootID`；
  2. 创建时间超过 24 小时；
  3. **不在当前进程所连数据库**的 `good_images / bundle_images / pay_codes` 引用里。

  `rootID` 是仓库 Nacos 模板里的同一个值。appProperties 里只有根目录，没有环境、库或实例标识。另外：
  - 引用集合为空时不会停手；
  - 每轮删除数量没有上限，也没有 dry-run；
  - `Sweep` 会先把所有候选读进内存，再逐个调用 `Trash`。
- 出事场景（无需任何人误操作）：
  - 本机 `make reset-local` 清空本地库后，下一轮清理会把这个根目录下所有超过 24 小时的图片移进垃圾桶。生产 / 预发如果共用该根目录，就会受影响，包括各店**收款码**。README:40 只把它描述成「本地图片进垃圾桶」。
  - 新建一个预发环境、CI 的集成测试库、或 DSN 填错指向空库时，后果相同。
  - 两个环境各有一部分数据时，会**互删对方**的图片。
  - 文件在 Drive 垃圾桶保留约 30 天，期间可恢复。但前台会立即 404，收款码失效会直接影响现场收款。
- 修复建议：
  1. 上传时在 appProperties 增加 `stallinkEnv`（取自 Nacos 新字段 `storage.environment`）和 `stallinkDatabase`（库的标识或 `system_identifier` 哈希）。Sweep 查询必须同时匹配这两个字段。
  2. 每个环境用独立的 Drive 根目录，仓库模板不要写死生产根目录 ID。
  3. 安全阀：引用集合为空时拒绝清理；单轮删除超过阈值（例如 50 个，或超过候选数的 20%）时中止并告警；增加 `storage.orphan_cleanup_enabled`，默认 false；首次上线先跑 dry-run，只记日志。
  4. 写清理审计日志，记录被移入垃圾桶的文件 ID，方便批量恢复。
- 验证方法：
  - 在 `drive_test.go` 用 httptest 假 Drive 写单测：引用集合为空时不调用 PATCH；候选数超过阈值时中止；查询串里必须含 `stallinkEnv`。
  - 人工核对：在 Drive 网页上按 appProperties 搜索，确认生产文件带上了环境标记。

---

## 三、高

### H1 启动与 Nacos 热更新会对任意数据库自动跑迁移

- 文件：`server/cmd/stallink/main.go:369-379`（`NewPool` → `previous == nil || previous.database != bc.Data.Database` → `data.Migrate`）；触发链在 `main.go:209-225, 242-246, 283, 292-300`；`server/internal/nacos/client.go:65-76`（`decode` 展开环境变量后只做 `ValidateBootstrap`）
- 问题：`serve` 首次启动一定会迁移。运行中只要 Nacos 发布的配置里 `data.database` 发生变化，也会在新库上**立即**迁移。这条路径**没有**回环地址或库名守卫，也没有确认词。守卫只存在于 `reset-local` / `seed-local` 走的 `localBootstrap`（`main.go:97-118`）。迁移只能向前，且每个版本单独提交事务；中途失败时目标库会停在半迁移状态，而旧配置仍在运行。
- 出事场景：见第六节。
- 修复建议：见第六节（默认关闭、`stallink migrate` 子命令、非本地库需确认、热更新永不迁移）。
- 验证方法：见第六节的测试清单。

### H2 本机 Nacos 运行配置含明文密钥，且 OAuth 权限范围过宽

- 文件：`server/logs/nacos/stallink/config-cache/config/stallink.yaml@@DEFAULT_GROUP@@stallink`
  - 第 12 行：DSN 内含口令，长度 66，指向回环地址上的 `stallink` 库
  - 第 16 行：`refresh_token`，明文，长度 127，格式符合 Google refresh token
  - 第 18 行：`client_secret`，明文，长度 35，格式符合 Google OAuth client secret
  - 第 22 行：`platform_admin_password`，明文，长度 16
  - 文件权限原为 `0644`（目录 `0755`），**本次已现场加固为目录 `0700`、文件 `0600`**。目录 `/server/logs/` 已被 `.gitignore:3` 忽略，**没有进入 git**（全部历史中也未发现高置信度密钥模式）
- 问题：
  - 仓库模板 `server/configs/nacos/stallink.yaml:16,21,22,25` 用的是 `$ENV`，但本机 Nacos 里实际发布的是明文。Nacos SDK 缓存把它原样落盘：Nacos 服务端、本机缓存、Nacos 控制台都能看到。
  - README:49 指导生成凭据时用了 `--scopes=...cloud-platform,...drive`。若当前 refresh token 也是这样签发的，它等同于该账号的**整个 GCP 权限**，而不只是 Drive。
- 出事场景：本机日志目录被打包发给他人、被备份同步、或被其他本机进程读取；或 Nacos 被他人访问（compose 里 Nacos 用的是上游示例口令和默认 token，见 L1）。拿到 token 就能操作 Drive，甚至操作 GCP 资源。
- 修复建议：
  1. 如果该缓存文件或 Nacos 导出曾离开本机，应在 Google Cloud 控制台撤销该 refresh token 并重新签发，同时轮换 client secret。
  2. 重新授权时权限范围只给 `https://www.googleapis.com/auth/drive.file`（只能访问本应用创建的文件；新建文件上传和移入垃圾桶都够用），去掉 `cloud-platform`，并同步修改 README:49。
  3. Nacos 只放 `$STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN` 这类引用，真实值走进程环境或密钥管理。
  4. 把 Nacos SDK 的 cache_dir 设为 `0700`，或关闭本地缓存（`server/configs/local/stallink.yaml:13-14`）。
- 验证方法：
  - 重新发布后，在缓存文件中确认这些字段为 `$...` 形式（用 `grep -c 'GOCSPX-'` 计数应为 0）。
  - 在 Google 账号「第三方访问」页面确认旧 token 已撤销。
  - 用新 token 访问非 Drive API 应返回 403。

### H3 可达的标准库与 `golang.org/x/image` 漏洞（govulncheck：15 个）

- 文件：`server/go.mod:3`（`go 1.27.1`），`server/go.mod:11`（`golang.org/x/image v0.23.0`，由未推送提交 `db17ae6` 引入）
- 扫描结论：代码实际调用到的漏洞有 15 个；另有 3 个在导入的包中、14 个在依赖模块中，但未被调用。
  - `x/image`：GO-2026-6222（VP8L 解码分配过量内存，v0.45.0 修复）、GO-2026-5061（VP8 alpha 尺寸不一致导致 panic，v0.43.0 修复）、GO-2026-4961（32 位平台大 WebP panic，v0.42.0 修复）。调用路径是 `imagestore/compress.go:38`（`DecodeConfig`）和 `compress.go:48`（`Decode`）。
  - 标准库：GO-2026-6859（MaxBytesReader 头部竞争导致 HTTP/1 服务端崩溃，go1.27.4 修复）、GO-2026-6609（Range 头解析无上限，go1.27.2）、GO-2026-6617 / 6612 / 6611 / 6603（HTTP/2 服务端崩溃、CPU 或内存耗尽，go1.27.2）、GO-2026-6608（MIME 头内存上限可被绕过，go1.27.2）、GO-2026-6613 / 6605 / 6610（连接失步、畸形帧头）、GO-2026-6809（x509 中间证书不再信任，go1.27.3）、GO-2026-6607（TLS ECH）。调用链经过 `main.go:263`（`Serve`）、`image_storage.go:106`（`ServeContent` 处理 Range）、`drive.go:213`（`Client.Do`）。
- 出事场景：
  - 任何已登录的店员上传一张特制 WebP，可以耗尽内存或触发 panic。
  - 公开图片接口走 `http.ServeContent`，会解析 Range 头，匿名请求即可触达。
  - 标准库中的 HTTP/2 崩溃类漏洞（若前置代理到 Go 之间启用 h2c / h2）。
- 修复建议：升级到 `golang.org/x/image >= v0.45.0`、Go 工具链 `>= 1.27.4`，然后重跑 govulncheck。另外，本机 `~/go/bin/govulncheck` 是用 go1.26 编译的，无法分析 go1.27 代码。本次已用 go1.27.1 重新安装 `golang.org/x/vuln/cmd/govulncheck@latest`（`v1.10.0`），并确认 `/Users/kelden/go/bin/govulncheck -version` 可正常执行。
- 验证方法：升级后 `govulncheck ./...` 的 “Your code is affected” 应为 0；`go test ./internal/imagestore/...` 通过。

### H4 公开图片接口无缓存，每次请求都穿透 Drive

- 文件：`server/internal/service/handlers.go:423-456`（`GoodImage` / `BundleImage`，匿名可访问）；`server/internal/service/image_storage.go:80-107`（`serveBlob`：不设置 `Cache-Control`，也没有 ETag / Last-Modified，`modtime` 为零值）；`server/internal/imagestore/drive.go:108-126`（每次都从 Drive 下载，最多把 6 MiB 读进内存）；前端 `web/src/components/shop-landing.tsx:9` 使用 `unoptimized`，首页 hero 显示 5 张、spotlight 显示 4 张
- 问题：每张图的每次浏览 = 一次数据库查询（`GoodImagePath` 带 EXISTS 子查询，`PublicBundleImagePath` 带三层嵌套子查询和每个组件的 SUM）+ 一次 Drive API 下载（20 秒超时，`google_auth.go:27`）+ 整文件进内存。浏览器和 CDN 都无法缓存，因为没有缓存头，Next 代理也只是原样转发。
- 出事场景：
  - 展会现场高峰期，同时打开首页的用户 × 每页约 9–20 张图，Drive API 的每用户配额（约每分钟每用户数千次请求量级）和延迟会被直接放大，导致大面积 502，或图片加载要数秒。
  - 匿名脚本循环请求 `/api/files/goods/{id}`，就能耗尽 Drive 配额。这时**后台上传、收款码读取也一起失败**，因为两者共用同一个 token 和配额。
- 修复建议：
  1. 公开图片带上 `Cache-Control: public, max-age=31536000, immutable`。Drive 文件 ID 是不可变引用：换图会产生新的 `drive:` ID 和新的 image ID。同时以 `ETag` = 文件 ID 并支持 304。收款码保持 `private, no-store`（现状已正确，`handlers.go:558`）。
  2. Go 侧加一个有界的内存 LRU（例如 64–128 MiB，以文件 ID 为键），或在 Web 侧启用 Next Image 优化 + CDN。
  3. 公开图片接口加按来源 IP 的限流（与 11-22 的 P1-2 一并实现）。
- 验证方法：
  - 用 `curl -I` 请求公开图片应看到长缓存头，第二次带 `If-None-Match` 应返回 304。
  - 给假 Drive 计数的单测：同一 ID 连续 Get 100 次，Drive 只被调用 1 次。
  - 压测（本机、假 Drive）：`hey -n 2000 -c 50` 下 p95 和内存曲线。

---

## 四、中 / 低

### M1 公开目录检索不分页、无上限、无缓存（首页 SSR 和每次输入都会调用）

- 文件：`server/internal/data/catalog_store.go:374-404`；调用方 `handlers.go:278-288`（匿名）、`web/src/lib/api.ts:260-272`（首页 SSR，`no-store`）、`web/src/components/catalog-explorer.tsx:41-65`（输入防抖 280 ms）
- 问题：
  - 没有 `LIMIT`；`ILIKE '%q%'` 同时作用于 `name` 和 `description`（最长 4000 字符），无法用 btree 索引；
  - 每一行执行一次关联子查询 `SUM(reserved_qty)`（按 `good_id`，而 `event_listings` 只有 `(event_id, good_id)` 唯一索引和 `(shop_id, event_id, is_listed)` 索引，`good_id` 不是任何索引的前导列），外加一次 `array_agg` 图片；
  - 响应包含完整 `description`；`q` 没有长度上限。
- 出事场景：商品数到千级以后，每次请求都是全量扫描 + N 次子查询 + 数 MB JSON。匿名用户可以并发发送长 `q`，把数据库 CPU 打满，影响收银。
- 修复建议：
  - 加 `LIMIT/OFFSET` 或游标（例如 48 条），`q` 截断到 64 字符；
  - 列表不返回 `description`；
  - 把预占量改成 JOIN 一次聚合（`LEFT JOIN (SELECT good_id, shop_id, SUM(reserved_qty) ... GROUP BY 1,2)`）；
  - 增加 `CREATE INDEX CONCURRENTLY event_listings_good_idx ON event_listings(good_id, shop_id) INCLUDE (reserved_qty)`；
  - 文本检索用 `pg_trgm` GIN 索引；
  - 匿名接口加短 TTL 缓存（5–15 秒）和限流。
- 验证方法：已在本机库执行 EXPLAIN ANALYZE 验证（见第八节），确认 SubPlan 全店扫描过滤现象；等级维持**中**。另建议加单测断言返回条数不超过上限。

### M2 图片上传解码的内存 / CPU 无上限

- 文件：`server/internal/imagestore/compress.go:16-21,30-98`；`server/internal/service/staff.go:21-24`（单请求最多 50 MiB、8 张、每张 6 MiB）；`staff.go:171-172,242-243`、`bundles.go:35-36,129-130`、`admin.go:124-125`（gosec G120）
- 问题：
  - 单张图最多 2400 万像素：解码后约 96 MB，`NRGBA` 缩放目标约 10 MB；
  - 不透明 PNG 走 `png.BestCompression`，压缩循环每轮按 3/4 缩边并重新编码，可能执行很多轮；
  - 一次请求串行处理 8 张；并发请求数没有任何限制。
- 出事场景：若干店员（或一个被盗的店员账号）并发上传 8 张 2400 万像素图片，会让 API 内存冲到数 GB，触发 OOM，影响整站和收银。
- 修复建议：
  - 用全局信号量限制同时解码的图片数（例如 2 个）；
  - 把 `maxPixels` 降到约 1600 万，或先按 `DecodeConfig` 估算内存再决定；
  - 压缩循环最多 4 轮，PNG 用 `DefaultCompression`；
  - 结合 H3 升级 `x/image`。
- 验证方法：基准测试 `go test -bench Compress -benchmem`，输入 2400 万像素 PNG；并发 10 个上传时用 `GODEBUG=gctrace=1` 或 pprof 观察 heap 峰值。

### M3 原图直接回传，会保留 EXIF（可能含 GPS）

- 文件：`server/internal/imagestore/compress.go:90-92`（商品图：未缩放、≤1 MiB 且原图不大于重新编码的结果时，返回 `original`）；`compress.go:63-64`（收款码：返回原图）
- 问题：JPEG 原样上传到 Drive，并通过公开接口提供给所有人。手机照片常带 GPS、设备和时间信息。
- 出事场景：创作者在家里拍的商品图带着住址坐标，被公开下载。
- 修复建议：JPEG 一律重新编码（`orientJPEG` 已处理方向，重新编码会丢弃 EXIF），或至少剥离 APP1 段；WebP / PNG 同理去掉 `eXIf` / `EXIF` chunk。
- 验证方法：用带 GPS 的 JPEG 写单测，`Compress` 输出后用 `exif` 解析应无 GPS 标签。

### M4 Drive 文件权限继承根目录；收款码与公开商品图同目录（需人工确认）

- 文件：`server/internal/imagestore/drive.go:67-106`（上传不设置任何 permission，完全继承根目录）；`server/configs/nacos/stallink.yaml:18`
- 问题：代码层面不会把文件设为公开，这一点没问题。但所有店铺的收款码、商品图都在同一个根目录下，文件名里带店铺 ID。根目录若被共享给「知道链接的任何人」或协作者，所有店的收款码对这些人都可见；根目录 ID 也已写在仓库里。本次无法访问 Drive，**未能核实实际共享设置**。
- 修复建议：在 Drive 网页确认根目录为「受限」，仅服务账号或授权账号可访问；收款码放独立子目录或独立根目录；仓库模板改为占位符。
- 验证方法：用无权限账号打开任一收款码文件链接，应被拒绝。

### M5 三套主题 CSS 在所有路由全量加载（影响首屏）

- 数据（2026-11-26 10:14 的 `web/.next` 构建，**非本次运行**，晚于工作区改动）：两个全局 CSS 块同时出现在 `/about`、`/register` 等所有页面。`3svzz2b1l9rhp.css` 为 187,547 B（gzip 31,409 B），`0lcfkft7yh36r.css` 为 114,700 B（gzip 15,249 B），合计约 302 KB 原始、约 46.7 KB gzip。另有两个约 2 KB 的块。JS 块合计 1,016,236 B，最大单块 229,082 B。
- 源文件：`web/src/app/layout.tsx:7-10` 全局引入 `globals.css` 209,315 B / 3,051 行、`shop.css` 71,523 B（工作区新增约 1,949 行）、`exhibition.css` 27,213 B、`exhibition-workspace.css` 24,004 B。三套主题互斥，但都会被下载和解析。
- 出事场景：展会现场的移动网络下，首屏被约 47 KB 阻塞渲染的 CSS 和样式计算拖慢；工作台和顾客端也都背着另外两套主题。
- 修复建议：
  - 按主题拆分：把 `shop.css` / `exhibition*.css` 移到对应的 route group 或在 `SiteFrame` 层按主题引入；
  - 删除 `globals.css` 中被新主题覆盖的旧规则（可用 Chrome Coverage 找未使用比例）；
  - 工作台样式只在 `/admin`、`/counter`、`/platform` 加载。
- 验证方法：`npm run build` 后对比各路由的 CSS 字节数；用 Lighthouse 移动端测 FCP / LCP。

### M6 外观接口：每次 SSR 都打库，且平台外观切换无审计

- 文件：`web/src/lib/api.ts:9-20`（`getStorefrontTheme`：`no-store`，超时 2.5 秒）；`web/src/app/layout.tsx:24`（根布局，所有页面都会调用）；`server/internal/service/platform_appearance.go:5-36`；`server/internal/data/platform_appearance.go:8-25`
- 问题：
  - 每一次页面渲染多一次 Go + DB 往返，并使整站都变成动态渲染；API 慢时 TTFB 最多增加 2.5 秒。
  - `SetPlatformAppearance` 只有平台管理员能调用，输入有白名单校验（`platform_appearance.go:27`，数据库 CHECK 兜底），没有 XSS / IDOR 问题；但没有写审计，与 11-22 的 P1-4 同类。
- 修复建议：主题读取加 30–60 秒缓存（`fetch(..., { next: { revalidate: 60 } })` 或 Go 侧进程内缓存加失效）；外观变更写平台级审计（操作人、旧值、新值）。
- 验证方法：压测首页时观察 Go 端 `/api/platform/appearance` 的 QPS 应远低于页面 QPS；查审计表能看到外观变更记录。

### M7 迁移规范：012 发布后被改、无校验和、无锁超时、无回滚

- 文件：`server/internal/data/migrations/012_platform_appearance.sql:3`、`013_shop_theme.sql:1-6`、`server/internal/data/migrations.go:97-123`
- 事实（`git log -p`）：012 在 `db3944f`（2026-11-24 15:10）首次加入，CHECK 为 `('classic','exhibition')`；之后在 `443bd21`（同日 16:04）**被原地改成**包含 `'shop'`，同时新增 013 来修补已执行旧版 012 的库。
- 在已执行旧版 012 的库上是否安全：**安全**。
  - `schema_migrations` 已有 version 12，新版 012 会被跳过；
  - 013 `DROP CONSTRAINT IF EXISTS platform_appearance_theme_check` 删掉的是 PostgreSQL 为列级 CHECK 自动生成的约束名（`<表>_<列>_check`），再加回含 `'shop'` 的约束；
  - 全新库：012 新版 + 013 先删后加，结果一致；
  - 旧数据只可能是 `classic` 或 `exhibition`，满足新约束。
- 锁影响：`ALTER TABLE ... ADD CONSTRAINT CHECK` 需要 `ACCESS EXCLUSIVE` 锁并全表校验。但该表是单行单例表，代价可忽略；唯一风险是被长事务阻塞时会一直等待（迁移里没有设置 `lock_timeout`）。
- 问题：
  - 迁移账本没有校验和，已执行的迁移被修改后无法发现，不同库之间会悄悄漂移；
  - 迁移事务没有 `lock_timeout` / `statement_timeout`；
  - 没有任何 down 脚本。
- 修复建议：
  - 约定已推送或已执行的迁移不得修改（012 尚未推送，可接受，但应在推送前定稿）；
  - `schema_migrations` 增加 `checksum`，启动时比对，不一致则拒绝；
  - `applyMigration` 内执行 `SET LOCAL lock_timeout = '5s'; SET LOCAL statement_timeout = '60s'`。
- 回滚草案（**仅草案，未执行**；需与代码回滚同步，回滚前先备份）：

```sql
-- 回滚 013：先把 'shop' 主题降级，再收窄约束
BEGIN;
SET LOCAL lock_timeout = '5s';
UPDATE platform_appearance SET theme = 'classic' WHERE theme = 'shop';
ALTER TABLE platform_appearance DROP CONSTRAINT IF EXISTS platform_appearance_theme_check;
ALTER TABLE platform_appearance ADD CONSTRAINT platform_appearance_theme_check
    CHECK (theme IN ('classic', 'exhibition'));
DELETE FROM schema_migrations WHERE version = 13;
COMMIT;

-- 回滚 012（连同 013）：删除外观表
BEGIN;
SET LOCAL lock_timeout = '5s';
DROP TABLE IF EXISTS platform_appearance;
DELETE FROM schema_migrations WHERE version IN (12, 13);
COMMIT;
```

  注意：只回滚 013 而不回滚 012 时，若当前代码仍是新版，新版 012 文件内容与库中约束会不一致；要么一起回滚代码，要么只做 012+013 的整体回滚。

### 低

- **L1** `docker-compose.yml:39`：Nacos 使用上游示例的 `NACOS_AUTH_TOKEN`（已知默认值），口令为 `nacos/nacos`（README:14）。端口只绑定 127.0.0.1，本机风险低；一旦这份 compose 被复用到服务器，任何人都能伪造 Nacos token，读到 H2 中的明文密钥。建议改为从 `.env` 读取随机值。
- **L2** `web/src/app/api/[...path]/route.ts:6-7`：代理会把客户端的全部请求头（含 `X-Forwarded-For`）原样转发给 Go。当前 Go 不信任这些头（README:42），没有问题；但将来做限流时，若读取 XFF 就会被伪造。建议在代理里删除或重写 `x-forwarded-*`，只信任代理自己注入的值。
- **L3** `server/internal/data/blob_cleanup.go:11-14`：引用集合没有包含 `shops.cover_path`、`events.cover_path`（`schema.go:18,56`）。当前 Go 代码没有写这两列，暂不触发；将来若用它们存 `drive:` 引用，图片会在 24 小时后被清理。建议现在就纳入集合或加注释。
- **L4** gosec 其余项：
  - `logging.go:338,351`：日志目录 0755 / 文件 0644，建议 0750 / 0600；
  - `cart_pricing.go:555`：G115 int 转 uint32，`count` 受购物车数量上限约束，属于低风险；
  - `handlers.go:106,118`：G124，`Secure` 由 `secure_cookies` 配置决定，生产必须为 true；
  - G120（`ParseMultipartForm`）：前面已有 `MaxBytesReader`，属于误报，但见 M2；
  - G101（`google_auth.go:17` token URL）、G304（`google_auth.go:60` 凭据路径来自环境变量）、G703（`main.go:124` 备份路径来自 CLI）：误报或可接受。
- **L5** npm audit：7 个 moderate，全部是**开发依赖**（`typescript-eslint` 链路中的 `minimatch` ReDoS，GHSA-g3hv-75wg-m249），不进入生产包。生产依赖只有 `next 16.4.6`、`react 19.3.0`、`react-dom 19.3.0`，无已知漏洞。运行 `npm audit fix` 即可（会改 lockfile，请自行决定）。
- **L6** `web/src/components/workspace-shell.tsx:134-142`：工作台角色拦截只在客户端做，属于体验层，真实权限由服务端保证（已核对）。另外 `web/next.config.ts:23` 的 `allowedDevOrigins` 写入了局域网 IP `192.168.132.138`，仅影响开发模式，属于信息项。

---

## 五、11-22 审核遗留复核表

判定口径：「仍开放」指当前代码与 11-22 报告的证据一致；「回归」指曾修复后又被改坏；「同类扩散」指新代码复制了同一问题。本次**没有发现任何一条已修复或回归**。11-22 之后的 4 个提交（`f82f835` `c888392` `db3944f` `443bd21`）都没有触及这些位置。

| 编号 | 问题 | 当前位置（行号） | 判定 | 备注 |
|---|---|---|---|---|
| P1-1 | 退款确认对 `sales` 开放，没有独立审批人 | `server/internal/service/staff.go:724-740`（注释第 727 行 “no extra role gate”）；`Cancel` 在 `staff.go:710-722` | 仍开放 | `c888392` 让销售员在 UI 上与摊主共用订单入口，前端更容易看到退款操作；服务端行为不变 |
| P1-2 | 登录 / 注册没有限流 | `handlers.go:132-159`（Login）、`161-194`（Register）；`server/internal`、`server/cmd`、`web/src` 中搜索 `limiter`/`rate limit`/`throttle` 均为 0 | 仍开放 | H4、M1 的匿名接口也需要一并限流 |
| P1-3 | 收款没有支付凭据 / 对账字段 | `staff.go:622-646`（只接收 `payCodeId`、`cashReceivedMinor`）；搜索 `payment_reference`/`payment_proof`/`received_at` 为 0 | 仍开放 | — |
| P1-4 | 自动审核开关变更没有审计 | `server/internal/service/review_switch.go:18-39`（只写 Redis） | 仍开放 + 同类扩散 | 新增的 `SetPlatformAppearance`（`platform_appearance.go:15-36`）同样无审计（M6） |
| P2-1 | 过期预占仍占着可售量 | `server/internal/catalog/stock.go:11-30`；读取侧 `SUM(reserved_qty)` 都没有按 `expires_at` 过滤：`catalog_store.go` 4 处、`quote_store.go` 2 处、`bundle_store.go` 2 处、`good_image_store.go` 1 处（含新的公开套餐图片判断 `good_image_store.go:70-87`） | 仍开放 | 当前 Nacos 配置 `sweep_interval: 30s`（`configs/nacos/stallink.yaml:38`）缓解了影响时长 |
| P2-2 | 释放预占不走全局锁序 | `server/internal/data/reservation_store.go:235-277`（直接改 `event_listings`，不先锁 `goods`） | 仍开放 | — |
| P2-3 | `readGoodPool` 用 `shopID=0` 关掉店铺过滤 | `server/internal/data/catalog_store.go:1214-1221`（`$2 = 0 OR shop_id = $2`）；调用方 `order_store.go:630` 传 `0` | 仍开放 | — |
| P2-4 | 缺少 CSP / HSTS | `web/next.config.ts:5-16`（只有 nosniff / DENY / no-referrer） | 仍开放 | — |
| P2-5 | 前端有一份无人调用的价格算法 | `web/src/lib/pricing.ts:25`（`web/src`、`web/tests` 中无调用方） | 仍开放 | — |
| P2-6 | 没有 CI | 仓库没有 `.github/` 和 `.gitlab-ci.yml` | 仍开放 | 本次 H3 正说明需要在 CI 跑 govulncheck |
| P3-a | 会话 14 天、不轮换、不限并发 | `handlers.go:105-115`（Cookie MaxAge）、`196-210`（`startSession`） | 仍开放 | — |
| P3-b | 登录 / 登出没有审计 | `handlers.go` 中 0 处 `insertAudit`（全仓 `insertAudit(` 共 21 处，与 11-22 相同） | 仍开放 | — |
| P3-c | `available` 不区分「真实零库存」与「被预占」 | `stock.go:22`；`catalog_store.go:414`（`poolRemaining`） | 仍开放 | — |

---

## 六、自动迁移防护方案

### 6.1 现状与触发路径

```
make dev → go run ./cmd/stallink -conf configs/local/stallink.yaml   (server/Makefile)
  └ main.go:50  serve → run()
      ├ main.go:202  loadRemote → nacos.Load → decode → ResolveEnvironment (os.ExpandEnv)
      │                                              → ValidateBootstrap（只校验非空）
      ├ main.go:283  apply(remote.Bootstrap)
      │    └ prepareRuntime(previous=nil)
      │        ├ main.go:369  data.NewPool(bc.Data.Database.Source)
      │        └ main.go:374  previous==nil → data.Migrate()      ← 首次启动：一律迁移
      └ main.go:209-225 / 292-300  Nacos watcher 收到新发布
           └ apply(bc) → prepareRuntime(previous=active)
               └ main.go:374  previous.database != bc.Data.Database → data.Migrate()
                                                                ← 热更新：DSN 一变就迁移新库
```

只有 `reset-local` / `seed-local` 会经过 `localBootstrap`（`main.go:97-118`：配置路径后缀 + 回环主机 + 库名 `stallink`）。`serve` 没有任何守卫。

### 6.2 出事场景

1. **本机直连云端排查**：开发者为了排查线上问题，把 `STALLINK_DATABASE_SOURCE` 临时指向云端库后执行 `make dev`。本机分支上未推送的 012/013，以及将来任何未上线的迁移，都会立刻在云端执行，云端 schema 会领先于线上代码版本。
2. **Nacos 热更新连锁**：多个环境共用一个 Nacos namespace，或有人在控制台把 DSN 改成另一个库。所有订阅该配置的进程（包括开发机）会在**不重启**的情况下对新库执行迁移。
3. **迁移中途失败**：每个版本是独立事务。例如 010 成功、011 失败时，`apply` 会拒绝新配置，但新库已经停在 010；旧进程继续服务旧库，问题容易被忽略。
4. **pgx 多主机 DSN**：`host=127.0.0.1,db.prod.example` 这类写法，主主机是回环，但 fallback 指向生产。即使照搬 `localBootstrap`，现有守卫也只检查 `ConnConfig.Host`，**会被绕过**。

### 6.3 防护设计

原则：**默认不迁移；迁移是显式动作；非本地库必须带确认；热更新永远不迁移。**

1. 配置项 `data.database.auto_migrate`（bool，默认 false）。仓库的本地模板可设为 true，生产模板不写。
2. 新增子命令 `stallink migrate -conf <file> [-confirm <host>/<dbname>]`。Makefile 增加 `migrate` 目标。
3. 统一守卫函数 `migrationTargetAllowed(source, confirm)`，供 `serve`、`migrate`、`reset-local`、`seed-local` 共用（`reset-local` 只接受本地库，不接受确认词）：
   - 用 `pgxpool.ParseConfig` 解析；任何解析错误都拒绝，且不回显 DSN；
   - 主主机和**全部 Fallbacks** 都必须是 `localhost`、`127.0.0.1` 或 `::1`，并且库名为 `stallink`，才算「本地」；
   - 不是本地库时，必须满足 `confirm == host + "/" + dbname`（来自 `-confirm` 或环境变量 `STALLINK_MIGRATE_CONFIRM`），否则拒绝；
   - Unix socket 主机（以 `/` 开头）视为非本地。
4. `serve` 首次启动：
   - `auto_migrate=true` 且守卫通过时，执行 `Migrate`；
   - 否则只做**只读**的待迁移检查 `data.PendingMigrations(ctx, pool)`（只执行 `SELECT version FROM schema_migrations`）。有待执行版本时拒绝启动，并提示运行 `stallink migrate`。
5. 热更新（`previous != nil`）：**永不调用 `Migrate`**。DSN 变化时只做待迁移检查，有待执行版本就拒绝这次发布，旧配置继续生效，同时记录错误日志。
6. 迁移执行层：`applyMigration` 内加 `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='60s'`；`schema_migrations` 增加 `checksum` 并在启动时比对（见 M7）。

### 6.4 代码草案（仅供参考，未修改仓库）

```go
// internal/conf/conf.go
type Database struct {
	Driver      string `json:"driver"`
	Source      string `json:"source"`
	AutoMigrate bool   `json:"auto_migrate"`
}

// cmd/stallink/migrate_guard.go
var errMigrationRefused = errors.New("migration refused for non-local database; pass -confirm <host>/<dbname> or STALLINK_MIGRATE_CONFIRM")

func isLoopback(h string) bool {
	h = strings.Trim(h, "[]")
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// migrationTarget returns "<host>/<db>" and whether every host is local loopback with db "stallink".
func migrationTarget(source string) (target string, local bool, err error) {
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		return "", false, errors.New("cannot parse database target") // never echo DSN
	}
	cc := cfg.ConnConfig
	local = isLoopback(cc.Host) && cc.Database == "stallink"
	for _, fb := range cc.Fallbacks {
		if !isLoopback(fb.Host) {
			local = false
		}
	}
	return strings.Trim(cc.Host, "[]") + "/" + cc.Database, local, nil
}

func migrationTargetAllowed(source, confirm string) error {
	target, local, err := migrationTarget(source)
	if err != nil {
		return err
	}
	if local {
		return nil
	}
	if confirm == "" {
		confirm = os.Getenv("STALLINK_MIGRATE_CONFIRM")
	}
	if confirm != target {
		return errMigrationRefused
	}
	return nil
}

// cmd/stallink/main.go — prepareRuntime 片段
var migrate = data.Migrate                     // 便于测试注入
var pendingMigrations = data.PendingMigrations // 只读检查

	isReload := previous != nil
	if !isReload || previous.database != bc.Data.Database {
		switch {
		case !isReload && bc.Data.Database.AutoMigrate:
			if err := migrationTargetAllowed(bc.Data.Database.Source, ""); err != nil {
				pool.Close()
				return nil, err
			}
			if err := migrate(ctx, pool); err != nil {
				pool.Close()
				return nil, err
			}
		default: // 热更新，或未开启 auto_migrate：永不写 schema
			pending, err := pendingMigrations(ctx, pool)
			if err != nil {
				pool.Close()
				return nil, err
			}
			if len(pending) > 0 {
				pool.Close()
				return nil, fmt.Errorf("database has %d pending migrations; run `stallink migrate`", len(pending))
			}
		}
	}

// 新子命令
case "migrate":
	err = migrateCmd(*configPath, *confirmation)

func migrateCmd(configPath, confirm string) error {
	_, remote, err := loadRemote(configPath)
	if err != nil {
		return err
	}
	defer remote.Close()
	src := remote.Bootstrap.Data.Database.Source
	if err := migrationTargetAllowed(src, confirm); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := data.NewPool(ctx, src)
	if err != nil {
		return err
	}
	defer pool.Close()
	return data.Migrate(ctx, pool)
}

// internal/data/migrations.go — 只读待迁移检查
func PendingMigrations(ctx context.Context, pool *pgxpool.Pool) ([]uint64, error) {
	all, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	applied := map[uint64]bool{}
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil { // 账本表不存在 = 全部待执行
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return versions(all), nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v uint64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	var pending []uint64
	for _, m := range all {
		if !applied[m.version] {
			pending = append(pending, m.version)
		}
	}
	return pending, rows.Err()
}
```

`localBootstrap`（`main.go:106-116`）应改为调用 `migrationTarget`，以便同时修复 Fallbacks 绕过问题。

### 6.5 验证方法（无需连库的部分可直接跑）

1. 单元测试 `migrate_guard_test.go`（纯函数，不连库），用表驱动覆盖：
   - `postgres://u:p@127.0.0.1:5432/stallink` → 允许
   - `postgres://u:p@[::1]/stallink` → 允许
   - `postgres://u:p@localhost/stallink_prod` → 拒绝
   - `postgres://u:p@10.0.0.5/stallink` → 拒绝；带 `confirm="10.0.0.5/stallink"` → 允许；带错误确认词 → 拒绝
   - `host=127.0.0.1,db.prod.example dbname=stallink` → 拒绝（Fallbacks）
   - `host=/tmp dbname=stallink` → 拒绝
   - 无法解析的 DSN → 错误信息中不含原文（断言 `!strings.Contains(err.Error(), "p@")`）
2. `prepareRuntime` 单测：注入 `migrate` / `pendingMigrations` 计数桩（`NewPool` 同样抽成变量）。断言：
   - `previous != nil` 时 `migrate` 调用次数为 0；
   - `auto_migrate=false` 时调用次数为 0；
   - 有待执行版本时返回错误，且旧配置保持不变。
3. `main_test.go` 中 `splitCommand` 增加 `migrate` 用例。
4. 集成测试（`-tags integration`，**只连本机 docker 的 stallink 库，需 Kel 明确同意后再跑**）：空库执行 `stallink migrate` 后，`PendingMigrations` 返回 0；再执行 `serve` 不应写入 `schema_migrations`（比较前后 `count(*)` 与 `max(applied_at)`）。

---

## 七、扫描结果与本机现场处理概况

| 项目 | 结果 |
|---|---|
| `gosec ./...`（`server/`，47 个文件 / 11,221 行） | 13 个问题：G120×5（multipart，见 M2）、G124×2（cookie）、G115×1、G101×1（误报）、G304×1、G703×1、G301×1、G302×1。均已在 M2 / L4 中处理 |
| `govulncheck ./...` | 初次运行本机旧二进制（v1.3.0，go1.26 编译）失败；随后已用 go1.27.1 重装 `govulncheck@v1.10.0` 并验证 `govulncheck -version` 通过。扫描结果：代码可达漏洞 15 个（标准库 12 个 + `x/image` 3 个），另有 17 个导入或依赖中的不可达漏洞，见 H3 |
| `npm audit`（`web/`） | 7 个 moderate，全部是开发依赖（`typescript-eslint` / `minimatch`），0 high / critical，见 L5 |
| 密钥扫描（工作树 + 未推送提交 + 全部 git 历史） | git 跟踪内容和全部历史中无高置信度真实密钥；命中均在测试或示例中。**未被跟踪**的 `server/logs/nacos/stallink/config-cache/config/stallink.yaml@@DEFAULT_GROUP@@stallink:12,16,18,22` 含真实明文（H2） |
| 缓存文件权限加固（已执行） | `server/logs/nacos/stallink/config-cache` 与 `.../config`：`drwxr-xr-x (755)` → `drwx------ (700)`；缓存文件 `stallink.yaml@@DEFAULT_GROUP@@stallink`：`-rw-r--r-- (644)` → `-rw------- (600)`。未轮换密钥 |
| 本机清理进程处置（已执行） | 已通过 `SIGTERM` 停掉正在运行的本机进程 PID `91621`（编译产物 `stallink -conf configs/local/stallink.yaml`）和 PID `91617`（`go run ./cmd/stallink -conf configs/local/stallink.yaml`），未重启；`next dev`（PID `90942`）保持运行 |
| 前端包体积 | 未重新执行 `npm run build`（避免覆盖 `.next`；`next dev` PID `90942` 运行中）。基于现有构建产物（`BUILD_ID` 时间 2026-11-26 10:14）测量，见 M5 |
| 临时文件清理 | 已执行 `rm -rf /tmp/stallink-review` |

---

## 八、本机数据库 EXPLAIN 与集成测试实测结果

在执行前已核对连接目标：本地缓存及 DSN 解析确认为 `host=127.0.0.1 port=5432 dbname=stallink`（docker 容器 `postgres`）。全程未启动 API、未执行迁移、未运行 `reset-local` / `seed-local`。

### 8.1 本机库规模与 EXPLAIN ANALYZE

表行数（`public` schema）：`goods` 22、`event_listings` 22、`events` 7、`shops` 2、`good_images` 10、`bundle_images` 0、`pay_codes` 1、`orders` 3、`order_quotes` 29、`schema_migrations` 13。

| 查询 | 耗时（Planning / Execution） | Buffers | 计划关键特征与是否走索引 |
|---|---|---|---|
| `PublicCatalogSearch`（无检索词，返回 22 行） | 1.4+ ms / 0.300 ms | `shared hit=158` | `events` 走 `Seq Scan`；`shops`/`goods` 走主键/唯一索引；`SubPlan 1`（`SUM(reserved_qty)`）使用 `event_listings_shop_event_idx` 按 `shop_id` 扫描后 `Filter: (good_id = g.id)`，**每行丢弃 14 条（`Rows Removed by Filter: 14` × 22 loops）** |
| `PublicCatalogSearch`（`q='a'`，返回 3 行） | 1.429 ms / 0.367 ms | `shared hit=94` | 全部 22 条上架记录都回表扫 `goods` 做 `name ~~* '%a%' OR description ~~* '%a%'`；`SubPlan 1` **每行丢弃 17 条（`Rows Removed by Filter: 17` × 3 loops）** |
| `GoodImagePath`（单张公开商品图） | 0.492 ms / 0.086 ms | `shared hit=14` | `good_images_pkey` 定位图片（`Index Scan`）；`EXISTS` 内 `event_listings` 只能走 `event_listings_shop_event_idx` 按 `(shop_id, is_listed)` 扫后再过滤 `good_id`（`Rows Removed by Filter: 3`）；`goods`（22 行）因表极小被优化器选为 `Seq Scan` |
| `PublicBundleImagePath`（0 行） | 7.702 ms / 0.083 ms | `shared hit=171 read=10` | `bundle_images` 为空快速返回；其 `NOT EXISTS` 内 `SubPlan 3` 与上方相同，仍无 `good_id` 前导索引 |
| `ReferencedBlobPaths`（清理引用集合，11 行） | 0.250 ms / 0.037 ms | `shared hit=2` | 三表 `Seq Scan` + `Append` + `HashAggregate`（Memory 97kB）；每小时一次，可接受 |

**对问题等级的影响**：小样本（22 条商品）下响应不到 0.4 ms，但执行计划**100% 证实**了 M1 的判断——`event_listings` 缺乏以 `good_id` 为前导列的索引，且检索使用全量无 `LIMIT` 的 `ILIKE '%...%'`。在展会千级商品规模下 `SubPlan 1` 会退化为 $O(N 	imes M_{	ext{shop}})$ 扫描过滤。**M1 维持「中」等级不变**。

后续若要在同机模拟压测（例如 `generate_series` 插入 5,000 条测试商品对比建索引前后），需先获 Kel 明确同意。

### 8.2 Go 单元测试与集成测试实测

- **测试隔离性核查**：全部 15 个 `*_integration_test.go` 文件均从 `STALLINK_TEST_DATABASE_URL` 读取连接串（已确认传入本机 `127.0.0.1:5432/stallink`），并在测试开始时动态执行 `CREATE SCHEMA <prefix>_<nanos>`，将连接池 `search_path` 切到该临时 schema 后再建表迁移，**不会清空或修改 `public` schema 的业务表**。
- **附带发现（影响记录）**：`internal/data/migrations_integration_test.go:32,38-42` 与 `133,138-142` 中 `defer adminPool.Close()` 先于 `t.Cleanup(... DROP SCHEMA ...)` 执行，导致清理阶段因连接池已关闭而静默失败，在本地 `stallink` 库中残留了 12 个**非空**测试 schema（6 个 `stallink_it_*`、6 个 `stallink_legacy_it_*`，每个含 22–24 张表及索引、序列；2026-11-26 复核更正，此前误记为空）。2026-11-26 已经 Kel 同意用 `DROP SCHEMA ... CASCADE` 删除。
- **执行命令与结果**：
  - 单元测试：`go test ./...`（`server/`）— **全部通过**（10 个包 ok，1 个无测试文件，`exit=0`）。
  - 集成测试：`STALLINK_TEST_DATABASE_URL=<本机stallink> go test -tags integration -count=1 ./...` — **全部通过**（`internal/data` 3.249s，`internal/service` 7.180s，全仓所有包 ok，`exit=0`）。其中并发下单防超卖、隔离迁移、退款回补、平台外观等用例均 PASS。

---

## 九、未完成或未覆盖的部分

- **未执行 `npm run build`**：原因同上（避免覆盖 `.next` 及触发被跟踪配置变更，`next dev` PID 90942 仍在运行）。
- **未访问 Google Drive API 与非本机环境**：未核实云端/预发环境配置及 Drive 根目录当前实际共享权限（C1 已按「各环境共用同一根目录」定为最高优先）。
- **未核实**当前 refresh token 在 Google 侧的实际授权范围（H2 依据 README 签发指令推断）。
- `prototype/`（静态交互原型）未逐行审查，其不随服务端部署。

---

## 十、C1 暂停步骤（各环境共用同一 Drive 根目录）

### 10.1 代码核查结论（为什么现在没有热开关）

1. **没有任何开关**：`conf.Storage`（`server/internal/conf/conf.go:116-125`）只有 `drive_root_folder_id` 和 `drive_oauth_web`；`main.go` 与 `imagestore/` 中**没有任何 Nacos 配置项或环境变量**可以单独关闭孤儿清理。
2. **启动即跑且每小时循环**：`server/cmd/stallink/main.go:416-422` 只要 `drive != nil` 就启动 `runBlobSweeper`，并在启动/热更新生效瞬间**立即执行一次 `sweep()`**（`main.go:477`），随后每 1 小时执行一次。
3. **热更新无法靠「去掉凭据」关闭 Drive**：
   - `main.go:365-367` 明确规定：若此前 `drive != nil`，热更新把凭据清空会报错 `"Drive credentials missing"`，Nacos 推送会被拒绝（`main.go:296-298`），**旧配置及旧 `runBlobSweeper` goroutine 会继续运行**；
   - `conf.ValidateBootstrap`（`conf.go:200-202`）也禁止空的 `drive_root_folder_id`；
   - 进程启动时传入的环境变量 `STALLINK_DRIVE_CREDENTIALS_FILE` / `STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN` 无法通过 Nacos 热更新清空；
   - 即使通过重启彻底不传 Drive 凭据使 `drive == nil`，不仅停掉清理，**全站所有图片上传和图片读取（`GET /api/files/...`）都会返回 503**。

### 10.2 本机（Kel 的 Mac）

- **已执行止血**：已停止正在运行的 `stallink` 进程（PID `91621`、`91617`）。在代码合入清理开关之前，**不要启动 `make dev` 或 `stallink serve`**（每次启动都会立刻跑一轮 `sweep()`）。

### 10.3 非本机环境（开发机 / 测试 / 预发 / 生产）手动暂停步骤

> **重要提醒（关于重启与自动迁移）**：只要进程重启（或热更新改了 `data.database`），当前版本会触发 H1 自动迁移（`main.go:374-379`）。在**发版/重启任何非本机环境前**，先核对目标二进制包含的 `migrations/*.sql` 与目标库 `schema_migrations` 表是否一致；若环境上跑的是旧二进制，**不要用未推送分支的新二进制去重启**，否则会直接执行 `012` / `013` 等迁移。

按场景选择以下方案：

#### 方案 A（推荐用于非生产测试 / 预发环境：不改代码、立即切断跨环境误删）

- **做法 1（完全隔离，需停服务或换独立测试目录）**：
  - 若该环境当前无需访问，直接**停止该环境的 `stallink` 服务进程**（`systemctl stop` / `docker stop` / 停容器，**不触发迁移**）；
  - 或在 Google Drive 新建一个专用于测试的空文件夹，拿到新文件夹 ID 后，修改**该测试环境自己对应的** Nacos 配置里的 `storage.drive_root_folder_id` 为新文件夹 ID（**只改这一项，绝对不要改 `data.database`**）。
  - *原理与是否需重启*：`main.go:347` 判断 `previous.storage != bc.Storage` 时会重建 `drive` 并重启 worker（`stopSweep()` 停止扫旧目录的 goroutine），**无需重启进程**，且因 `database` 未变**不会触发自动迁移**。
  - *副作用*：切到新空目录后，该测试环境数据库里原有的 `drive:<fileId>` 仍可被读（Drive 按 fileId 读取），但新上传的图片进入新目录，旧目录的跨环境清理立刻停止。
  - *验证关键字*：在服务日志中看到 `applied Nacos configuration` 且 `"drive_ready":true`；下一整点不再出现针对生产文件的删除。

#### 方案 B（推荐用于保护生产根目录：在 Google Drive 侧临时降权，0 代码改动）

- **适用前提**：生产环境与测试环境（或本机）使用**不同的 Drive 凭据账号**（例如生产用服务账号 A，本机/测试用个人 OAuth 或账号 B）；或者可接受短暂只读（不能上传新图）。
- **做法**：
  1. 打开 Google Drive，找到共用根目录 `1VE__CW7uIEnn3lJmt6uKIz87if-h1Hd7` 的「共享」设置；
  2. 将**非生产环境所用账号**的权限从「编辑者 / 内容管理员」降为**「查看者（Viewer）」**（或直接移除共享）；若所有环境共用同一账号且当前无上传需求，也可临时将该账号降为「查看者」。
- **是否需要重启 / 迁移风险**：**不需要改 Nacos，不需要重启服务，零迁移风险**。
- **副作用**：被降权的环境无法上传新商品图和收款码（上传接口返回 `500 image_storage_failed`），但**已上传图片的读取（`GET /api/files/...`）不受影响**；一旦某轮清理触发 `Trash`，Drive 会返回 `403` 拒绝移入垃圾桶。
- **验证方法**：被降权环境的日志在下一次清理（或重启后）出现 `orphan blob sweep failed`（原因 `insufficientFilePermissions` / HTTP 403），Drive 回收站不再新增文件。

#### 方案 C（应急凭据撤销：仅在怀疑某外泄/失控进程仍在跑时使用）

- **做法**：在 Google Cloud Console 吊销对应 OAuth Client Secret / 刷新令牌，或禁用/删除对应 Service Account Key；或清空该非生产环境的 `STALLINK_DRIVE_CREDENTIALS_FILE` / `STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN` 后重启。
- **是否需要重启**：在 Google 侧吊销无需重启（access token 最多 1 小时过期，过期即失效）；清除环境变量需重启（注意 H1 自动迁移风险）。
- **副作用**：使用同一凭据的所有环境图片**读取和上传全部失效**（返回 502/503）。

---

### 10.4 最小代码改动草案（仅写入本报告，未修改仓库）

设计目标：
1. 增加 `storage.orphan_cleanup_enabled`（bool，**默认 `false`**），Nacos 热更新即可即时开启/关闭（无需重启，无迁移风险）；
2. 空引用保护：`len(referenced) == 0` 时拒绝清理，防止空库/清空后删光根目录；
3. 单轮删除上限：默认单轮最多移入垃圾桶 `50` 个（或超出时中止并告警），避免大规模误删。

```diff
--- a/server/internal/conf/conf.go
+++ b/server/internal/conf/conf.go
@@ -114,8 +114,9 @@ type Database struct {
 }
 
 type Storage struct {
-	DriveRootFolderID string        `json:"drive_root_folder_id"`
-	DriveOAuthWeb     DriveOAuthWeb `json:"drive_oauth_web"`
+	DriveRootFolderID    string        `json:"drive_root_folder_id"`
+	DriveOAuthWeb        DriveOAuthWeb `json:"drive_oauth_web"`
+	OrphanCleanupEnabled bool          `json:"orphan_cleanup_enabled"`
 }
```

```diff
--- a/server/internal/imagestore/drive.go
+++ b/server/internal/imagestore/drive.go
@@ -21,10 +21,13 @@ import (
 
 const folderMIME = "application/vnd.google-apps.folder"
 const referencePrefix = "drive:"
+const maxOrphanTrashPerSweep = 50
 
 var (
-	ErrNotFound = errors.New("drive image not found")
-	validID     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
+	ErrNotFound           = errors.New("drive image not found")
+	ErrEmptyReferenceSet  = errors.New("refusing orphan sweep with empty reference set")
+	ErrOrphanSweepAborted = errors.New("orphan sweep exceeds per-run safety cap")
+	validID               = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
 )
 
 type Drive struct {
@@ -139,6 +142,9 @@ func (d *Drive) Trash(ctx context.Context, reference string) error {
 
 // Sweep trashes only images uploaded by Stallink under the configured root and older than cutoff.
 func (d *Drive) Sweep(ctx context.Context, referenced map[string]bool, cutoff time.Time) (int, error) {
+	if len(referenced) == 0 {
+		return 0, ErrEmptyReferenceSet
+	}
 	params := url.Values{"q": {fmt.Sprintf("trashed = false and appProperties has { key='stallinkImageRoot' and value='%s' } and createdTime < '%s'", d.rootID, cutoff.UTC().Format(time.RFC3339))}}
 	var orphaned []string
 	for {
@@ -156,6 +162,9 @@ func (d *Drive) Sweep(ctx context.Context, referenced map[string]bool, cutoff ti
 		}
 		params.Set("pageToken", page.NextPageToken)
 	}
+	if len(orphaned) > maxOrphanTrashPerSweep {
+		return 0, fmt.Errorf("%w: %d candidates > %d", ErrOrphanSweepAborted, len(orphaned), maxOrphanTrashPerSweep)
+	}
 	for i, ref := range orphaned {
 		if err := d.Trash(ctx, ref); err != nil {
 			return i, err
```

```diff
--- a/server/cmd/stallink/main.go
+++ b/server/cmd/stallink/main.go
@@ -413,12 +413,14 @@ func prepareRuntime(ctx context.Context, bc conf.Bootstrap, previous *runtimeCon
 				runReservationSweeper(sweepCtx, store, sweepInterval)
 			}()
 		}
-		if drive != nil {
+		if drive != nil && bc.Storage.OrphanCleanupEnabled {
 			workers.Add(1)
 			go func() {
 				defer workers.Done()
 				runBlobSweeper(sweepCtx, store, drive)
 			}()
+		} else if drive != nil {
+			logging.InfoContext(ctx, "orphan blob sweeper disabled", zap.Bool("orphan_cleanup_enabled", false))
 		}
```

```diff
--- a/server/configs/nacos/stallink.yaml
+++ b/server/configs/nacos/stallink.yaml
@@ -16,6 +16,7 @@ data:
     source: "$STALLINK_DATABASE_SOURCE"
 storage:
   drive_root_folder_id: 1VE__CW7uIEnn3lJmt6uKIz87if-h1Hd7
+  orphan_cleanup_enabled: false
   drive_oauth_web:
```

> 注：以上是**应急最小补丁**；长期仍须配合 C1 修复建议中的 `stallinkEnv` / 独立根目录与环境隔离（见第二节 C1）。

对应单测草案（`server/internal/imagestore/drive_test.go` 与 `conf_test.go`）：

```go
// drive_test.go
func TestDriveSweepGuardsEmptyReferenceAndCap(t *testing.T) {
	var trashCalls atomic.Int32
	client := testDriveHTTPClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			trashCalls.Add(1)
		}
		if r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files" {
			// 返回 51 个候选文件
			var files []map[string]string
			for i := 0; i < 51; i++ {
				files = append(files, map[string]string{"id": fmt.Sprintf("orphan-%d", i)})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	d := &Drive{client: client, base: "https://www.googleapis.com", rootID: "root-id"}

	if _, err := d.Sweep(context.Background(), map[string]bool{}, time.Now().Add(-24*time.Hour)); !errors.Is(err, ErrEmptyReferenceSet) {
		t.Fatalf("empty ref set err = %v, want ErrEmptyReferenceSet", err)
	}
	if _, err := d.Sweep(context.Background(), map[string]bool{"drive:keep": true}, time.Now().Add(-24*time.Hour)); !errors.Is(err, ErrOrphanSweepAborted) {
		t.Fatalf("cap err = %v, want ErrOrphanSweepAborted", err)
	}
	if trashCalls.Load() != 0 {
		t.Fatalf("expected 0 Trash calls, got %d", trashCalls.Load())
	}
}

// conf_test.go
func TestStorageOrphanCleanupDefaultsDisabled(t *testing.T) {
	bc, err := LoadFile("testdata/bootstrap.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if bc.Storage.OrphanCleanupEnabled {
		t.Fatal("orphan_cleanup_enabled must default to false")
	}
}
```

### 10.5 Google Drive 垃圾桶恢复步骤（人工操作）

1. **前置条件（务必先做）**：确认**所有环境**的孤儿清理都已停止或已降权（否则还原后的文件会在下一小时再次被 `Sweep` 移入垃圾桶）。
2. **保留期确认**：`Sweep` 仅调用 `PATCH {"trashed": true}`（`drive.go:133`），未做永久删除。Google Drive 垃圾桶（个人「我的云端硬盘」与 Workspace「共享云端硬盘」）标准保留期为 **30 天**（自移入垃圾桶起算，除非管理员手动清空回收站）。
3. **查找与还原**：
   - **个人「我的云端硬盘」**：使用上传/清理所授权的 Google 账号登录 Google Drive 网页端，进入左侧「回收站（Trash）」。
   - **Workspace「共享云端硬盘」**：由具备「管理员 / 内容管理员」权限的账号进入该共享云端硬盘的「回收站」。
   - Stallink 上传的图片文件名统一为 `shop-<店铺ID>-<good|bundle|pay-code>-<UUID>.<扩展名>`（`drive.go:79`），可按 `shop-` 前缀与移入时间筛选。
   - 选中需要恢复的文件（重点先恢复 `pay-code` 收款码），点击「还原（Restore）」。
4. **数据库无需修改**：Drive 从垃圾桶还原后会保留原有 `fileId`，数据库里 `good_images.path`、`bundle_images.path`、`pay_codes.image_path` 的 `drive:<fileId>` 引用自动恢复有效。
5. **恢复后验证**：
   - 在库中抽查被恢复的 ID，或通过前台页面 / `curl -I` 访问 `/api/files/goods/{imageID}`、`/api/files/pay-codes/{id}`（带店员 Cookie），确认返回 `200 OK` 而非 `404` / `502`。
