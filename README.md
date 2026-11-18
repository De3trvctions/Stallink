# Stallink
以独立链接作为摊位入口，支持按漫展活动隔离库存、多摊位数据独立，现场快速开单、记录订单与销售台账，适配东南亚多地区漫展出摊场景，轻量化开箱即用，无需下载客户端。

## 项目结构

- `server/`：Go API、启动配置和运行数据。
- `web/`：Next.js 前端。
- `docs/`：产品与实现文档。

## 本地运行

`server/configs/local/stallink.yaml` 只写 Nacos 地址。运行参数的仓库副本是 `server/configs/nacos/stallink.yaml`，发布到 Nacos 的 namespace `stallink`、dataId `stallink.yaml`。换机器时先看这两份 yaml，再按下面的命令启动。环境变量写成 `$NAME`。不要写成 `${NAME}`，Kratos 会把花括号形式当成配置引用并清空。不要把真实密码写进仓库。

`docker-compose.yml` 在本机拉起 Postgres、Redis 和 Nacos，并把那份 yaml 上传进 Nacos。API 和网页仍在宿主机启动。Compose 里的数据库口令是 `postgres` / `stallink`，Nacos 口令是上游单机示例 `nacos` / `nacos`，只用于本机。

```bash
make compose
export STALLINK_NACOS_USERNAME=nacos
export STALLINK_NACOS_PASSWORD=nacos
export STALLINK_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable'
export STALLINK_PLATFORM_ADMIN_PASSWORD='至少 8 位的本地管理员密码'
export STALLINK_LOCAL_SEED_PASSWORD='至少 8 位的演示账号密码'
make seed-local
make dev
make web
```

`make compose` 会等待 Nacos 就绪并发布配置。改过 `server/configs/nacos/stallink.yaml` 之后，再执行一次 `make nacos-publish`。店面是 http://127.0.0.1:3000 ，接口是 http://127.0.0.1:8080 。

套餐、库存预留和强制幂等开关已在这份模板里启用。后端启动时会执行版本迁移，但不会自动创建管理员或演示数据。审核开关在 Redis 不可用时保持关闭。`secure_cookies` 在本地 HTTP 模板里是 `false`；站点改走 HTTPS 后再把它改成 `true`。

`seed-local` 用 `STALLINK_LOCAL_SEED_PASSWORD` 创建平台管理员、两家店、活动和商品。没有这个环境变量时命令会失败。若要把现有本地数据清空并迁移到自增 ID，先确认数据库与图片备份存在，再运行：

```bash
make reset-local
make seed-local
```

`reset-local` 仅允许连接回环地址上的 `stallink` 数据库，并要求确认词以及非空的数据库/图片备份文件；它会清空业务表、重置自增序列并保留迁移记录与图片文件。不要对生产数据库运行此命令。启动 API 本身不会再次塞入演示记录。

生产部署时应将 Go API 放在私有网络，仅允许 Web 服务访问；公网入口必须对 `/api/auth/login` 和 `/api/auth/register` 按真实客户端来源限流。当前代码不信任客户端自行传入的转发 IP 头，也未验证具体反向代理是否配置了限流。
