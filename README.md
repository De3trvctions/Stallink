# Stallink
以独立链接作为摊位入口，支持按漫展活动隔离库存、多摊位数据独立，现场快速开单、记录订单与销售台账，适配东南亚多地区漫展出摊场景，轻量化开箱即用，无需下载客户端。

## 项目结构

- `server/`：Go API、启动配置和运行数据。
- `web/`：Next.js 前端。
- `docs/`：产品与实现文档。

## 本地运行

`server/configs/local/stallink.yaml` 只写 Nacos 地址。运行参数的仓库副本是 `server/configs/nacos/stallink.yaml`，发布到 Nacos 的 namespace `stallink`、dataId `stallink.yaml`。换机器时先看这两份 YAML，再按下面的命令启动。环境变量写成 `$NAME`。不要写成 `${NAME}`，Kratos 会把花括号形式当成配置引用并清空。不要把真实密码写进仓库。

`docker-compose.yml` 在本机拉起 Postgres、Redis 和 Nacos，并把那份 yaml 上传进 Nacos。API 和网页仍在宿主机启动。Compose 里的数据库口令是 `postgres` / `stallink`，Nacos 口令是上游单机示例 `nacos` / `nacos`，只用于本机。

```bash
make compose
export STALLINK_NACOS_USERNAME=nacos
export STALLINK_NACOS_PASSWORD=nacos
export STALLINK_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable'
export STALLINK_PLATFORM_ADMIN_PASSWORD='至少 8 位的本地管理员密码'
export STALLINK_LOCAL_SEED_PASSWORD='至少 8 位的演示账号密码'
make seed-local
export STALLINK_DRIVE_CREDENTIALS_FILE=/path/to/google-drive-credentials.json
make dev
make web
```

`make compose` 会等待 Nacos 就绪并发布配置。改过 `server/configs/nacos/stallink.yaml` 之后，再执行一次 `make nacos-publish`。店面是 http://127.0.0.1:3000 ，接口是 http://127.0.0.1:8080 。

套餐、库存预留和强制幂等开关已在这份模板里启用。后端启动时会执行版本迁移，但不会自动创建管理员或演示数据。审核开关在 Redis 不可用时保持关闭。`secure_cookies` 在本地 HTTP 模板里是 `false`；站点改走 HTTPS 后再把它改成 `true`。

`seed-local` 用 `STALLINK_LOCAL_SEED_PASSWORD` 创建平台管理员、两家店、活动和商品。没有这个环境变量时命令会失败。如需重置现有本地开发数据，先确认已备份需要保留的 Drive 图片，再运行以下命令。

```bash
make reset-local
make seed-local
```

`reset-local` 仅允许连接回环地址上的 `stallink` 数据库，并要求确认词及非空的数据库备份文件。重置会清空业务表、重置自增序列并保留迁移记录。失去数据库引用的 Drive 图片之后会进入垃圾桶。不要对生产数据库运行此命令。启动 API 本身不会再次塞入演示记录。

生产部署时应将 Go API 放在私有网络，仅允许 Web 服务访问；公网入口必须对 `/api/auth/login` 和 `/api/auth/register` 按真实客户端来源限流。当前代码不信任客户端自行传入的转发 IP 头，也未验证具体反向代理是否配置了限流。

## Google Drive 图片存储

仓库的 Nacos 配置已填入 Stallink 的 Drive 根目录 ID。后端将压缩后的图片直接写入指定目录，文件名为 `shop-<店铺ID>-<图片类型>-<UUID>.<扩展名>`。商品数据库字段 `good_images.path` 保存 `drive:<文件ID>`。目录共享权限沿用 Google Drive 的设置。

1. 在 Google Cloud 项目启用 Google Drive API。
2. 按目录类型准备凭据。个人「我的云端硬盘」使用具备该目录权限的账号授权，并使用 `authorized_user` JSON。可创建桌面应用 OAuth 客户端，然后执行 `gcloud auth application-default login --client-id-file=/path/to/oauth-client.json --scopes="https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/drive"`，使用生成的 `application_default_credentials.json`。Workspace「共享云端硬盘」可使用 `service_account` JSON，并给服务账号能上传和移到垃圾桶的角色，例如内容管理员。
3. 将凭据文件保存在仓库以外，用 `STALLINK_DRIVE_CREDENTIALS_FILE` 指定绝对路径。以后用 Docker 运行 API 时，将凭据只读挂载到容器内，并把变量指向容器内路径。
4. 在 Nacos 发布配置后，运行中的 API 会自动接收并应用新版本。服务在启用 Drive 时检查文件夹权限。未提供上传凭据时，商品目录仍可访问，上传接口返回 503。

使用 Nacos 中的 Web OAuth 客户端时，在 `storage.drive_oauth_web` 配置 `client_id` 和 `client_secret`。模板的 `client_secret` 引用 `STALLINK_DRIVE_OAUTH_WEB_CLIENT_SECRET`，启动时从环境读取密钥。在同一个 Web OAuth 客户端完成 Google Drive 离线授权，把获得的 refresh token 配到同一节的 `refresh_token` 即可热更新。模板也支持在启动环境中提供 `STALLINK_DRIVE_OAUTH_WEB_REFRESH_TOKEN`。修改进程环境变量需要重启 API。已有 `STALLINK_DRIVE_CREDENTIALS_FILE` 的部署优先使用凭据文件。 [Google Web OAuth 离线授权](https://developers.google.com/identity/protocols/oauth2/web-server#offline)

商品图和套餐图最长边为 1600 像素，每张最多 1 MiB。收款码按原始像素无损优化，每张最多 4 MiB。接口接受 JPEG、PNG 和 WebP。旧的本地图片需要重新上传。后台每小时将已创建超过 24 小时且数据库未引用的由 Stallink 上传的图片移到 Drive 垃圾桶。
