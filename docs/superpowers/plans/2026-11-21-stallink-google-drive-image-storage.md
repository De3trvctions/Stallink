# Stallink Google Drive 图片存储实现计划

> 后续执行使用 `superpowers:executing-plans`，逐项更新复选框。涉及 Go 开发时，先按根目录 `AGENTS.md` 加载 `samber/cc-skills-golang@golang-how-to`。

**状态**　本计划于 2026-11-21 在代码实现后补录。已勾选项有代码或测试结果对应。接入真实 Drive 所需的 Google 凭据尚未配置。

**目标**　将商品图、套餐图、套餐内单品图和收款码的持久存储切到指定的 Google Drive，并在后端压缩后按店铺归档。

**架构**　Go API 使用通过数据库鉴权取得的店铺 ID，在配置的根目录下直接写入以店铺 ID、图片类型、UUID 命名的文件。PostgreSQL 的现有图片引用字段保存 `drive:<fileId>`，前端通过原有图片路由读取。开发阶段的旧图片从页面重新上传。

**技术栈**　Go、PostgreSQL、Nacos、Google Drive API v3、OAuth 2.0、`golang.org/x/image`。

## 文件与职责

| 文件 | 职责 |
| --- | --- |
| `server/internal/imagestore/compress.go`、`server/internal/imagestore/orientation.go` | 解码并验证 JPEG、PNG、WebP，处理 JPEG 方向，压缩图像。 |
| `server/internal/imagestore/drive.go`、`server/internal/imagestore/google_auth.go` | 加载凭据，检查目录与共享权限，创建店铺目录，上传、读取和回收 Drive 文件。 |
| `server/internal/service/image_storage.go`、`server/internal/service/staff.go`、`server/internal/service/bundles.go`、`server/internal/service/admin.go`、`server/internal/service/handlers.go` | 接入商品、套餐、收款码上传和图片读取，保留店铺及平台角色鉴权。 |
| `server/internal/data/blob_cleanup.go`、`server/cmd/stallink/main.go` | 用 PostgreSQL 事务锁串行化同一店铺的目录创建，收集图片引用，安排后台清理。 |
| `server/internal/conf/conf.go`、`server/configs/nacos/stallink.yaml`、`server/Makefile`、`README.md`、`web/src/lib/errors.ts` | 配置根目录与启动方式，调整开发库重置流程和页面错误提示。 |
| `server/internal/imagestore/compress_test.go`、`server/internal/imagestore/drive_test.go`、`server/internal/service/image_storage_test.go`、`server/internal/service/bundle_images_integration_test.go` | 覆盖压缩、Drive 请求、错误分类和套餐图片上传后读取。 |

## 任务 1 — 按店铺存储与后端压缩

- [x] 通过已认证账号取得店铺 ID。文件直接写入根目录，以唯一文件名区分店铺和图片。Nacos 模板已填写根目录 ID `1VE__CW7uIEnn3lJmt6uKIz87if-h1Hd7`。
- [x] 商品图、套餐图和套餐内单品图限制最长边为 **1600 px**，压缩后每张最大 **1 MiB**。后端解码校验格式及尺寸，并处理手机 JPEG 的 EXIF 方向。
- [x] 收款码保持原尺寸和解码像素，选择较小的无损 PNG 或原文件，保存上限为 **4 MiB**。超限返回 `pay_code_image_limit`，上传失败不会写入图片引用。
- [x] 读取接口先执行现有可见性和店铺校验，再通过 Drive 文件 ID 读取图片。收款码响应使用 `Cache-Control: private, no-store`。Drive 根目录需要允许后端授权账号写入。
- [x] 后台每小时扫描由此根目录配置下的 Stallink 上传、创建超过 24 小时、且数据库已无引用的图片，将其移入 Drive 垃圾桶。开发与生产使用各自的根目录，避免跨环境清理。

## 任务 2 — 可复跑的本地验证

- [x] 从仓库根目录运行 `(cd server && go test -mod=readonly -count=1 ./... && go vet ./...)`。上一轮通过，Drive 模拟测试验证了店铺目录复用、上传入参、读取和只回收无引用图片。
- [x] 将 `STALLINK_TEST_DATABASE_URL` 指向隔离 PostgreSQL，从仓库根目录运行 `(cd server && go test -tags=integration -run '^TestBundleDraftImagesCreateAndEdit$' -count=1 -v ./internal/service)`。上一轮使用临时 PostgreSQL 容器通过，验证套餐内单品图片创建、编辑及从员工图片接口读回，图片后端使用测试替身。
- [x] 从仓库根目录运行 `(cd web && ./node_modules/.bin/tsc --noEmit --incremental false)`。上一轮通过。

## 任务 3 — 真实 Drive 启用与验收

- [ ] 确认指定目录的归属和写权限。个人「我的云端硬盘」使用有编辑权限的 `authorized_user` 凭据，Workspace「共享云端硬盘」使用具有上传和移入垃圾桶权限的 `service_account`。按 [README 的配置步骤](../../../README.md#google-drive-图片存储) 启用 Drive API 并准备凭据文件。
- [ ] 在运行 API 的环境设置 `STALLINK_DRIVE_CREDENTIALS_FILE`。容器内使用只读挂载后的路径。发布当前环境的 Nacos 配置并重启 API。本地使用 `make nacos-publish` 和 `make dev`。启动成功后，`GET /api/health` 应返回 200。若启动失败，查看 Drive 根目录检查对应的权限或凭据错误。
- [ ] 用测试店铺分别上传商品图、套餐图、套餐内单品图和收款码。检查 Drive 根目录只为该店铺创建一个子目录，四类图片均在目录内，再上传时仍复用该目录。用第二家测试店铺验证目录不同。检查商品和套餐图的实际尺寸与文件大小，确认收款码可扫码。
- [ ] 从商品页、后台预览和收银页读取对应测试图片。用其他店铺账号请求第一家店的收款码，预期 404。使用同一数据库和 Drive 配置重新创建 API 容器并再次读取图片，确认图片在新容器中仍可访问。

## 本机交互回归与演示数据

- [x] 本机浏览器使用现有数据验证了首页检索、商品详情和活动页。首页显示 13 件商品，搜索「拍立得」筛成 1 件，商品详情可进入 Comic Fiesta 2026 活动页。
- [x] 在本机 Nacos 的 `stallink / DEFAULT_GROUP / stallink.yaml` 发布并回读 `storage.drive_root_folder_id`。原先缺少该字段，回读值为 `1VE__CW7uIEnn3lJmt6uKIz87if-h1Hd7`。发布后当前 API 的健康检查为 HTTP 200。
- [ ] 核准本机开发库的清理范围并备份。现有 `reset-local` 会清空店铺、用户和收款方式，需另走保留商家数据的清理流程。应用现有的商品与活动路由未提供硬删除。
- [ ] 登录保留的商家账号，用 `POST /api/staff/events` 创建演示活动，再用返回的活动 ID 调用 `POST /api/staff/goods`。商品请求使用 multipart 的 `product` JSON 字段和 `images` 文件字段，`isListed=false`。通过商家后台预览、真实图片接口和 Drive 目录验证。
- [ ] 使用另一家店铺重复验证，确认它的图片保存到自己的子目录。

演示活动草案为「吉隆坡 Cosplay 摄影周末」，时间为 2026-12-12 至 2026-12-13，币种 MYR，状态为 `scheduled`。场地和摊位采用测试值。

| 商品草案 | 价格 | 库存 | 照片 |
| --- | ---: | ---: | --- |
| 蓝发人像明信片 | RM 12.00 | 30 | `IMG_0056.JPG` |
| 夜景金扇写真卡 | RM 18.00 | 20 | `IMG_0050_2.JPG` |
| 红白舞台写真 | RM 24.00 | 16 | `IMG_0067.JPG` |
| 格纹造型方形小卡 | RM 10.00 | 40 | `IMG_0072.JPG` |

测试照片来自用户提供的 2026-10-10 文件夹。三张超过现有上传大小限制的照片已有最长边 2400 px 的临时副本，保留了可验证后端二次缩放的尺寸。一张原始 JPEG 为 2731 × 2731、6,278,982 字节，可直接验证接近上传上限的后端压缩。

## 任务 4 — 接通 Nacos 中的 Web OAuth 配置

- [x] 在 `server/internal/conf/conf.go` 绑定 `storage.drive_oauth_web.client_id` 和 `storage.drive_oauth_web.client_secret`，按现有规则解析环境变量引用。使用 `server/internal/conf/conf_test.go` 加载仓库的 Nacos YAML，验证两个字段实际绑定。
- [x] 在 `server/internal/imagestore/google_auth.go` 复用 OAuth 刷新逻辑。显式设置 `STALLINK_DRIVE_CREDENTIALS_FILE` 时继续读取文件，兼容 `authorized_user` 与 `service_account`。其余情况下使用 Web OAuth 配置和与同一 Client ID 绑定的 `STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN`。缺少字段或文件不可读时启动报错。
- [x] 在 `server/cmd/stallink/main.go` 接通 Nacos 配置并更新 README。使用模拟 token endpoint 验证刷新请求携带的三项凭据，并运行 `(cd server && go test -mod=readonly -count=1 ./... && go vet ./...)`。
- [ ] 使用轮换后的 Web 客户端密钥和授权产生的 refresh token 启动 API，完成真实 Drive 验收。运行环境尚未注入这些凭据。
