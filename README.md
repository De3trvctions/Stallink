# Stallink
以独立链接作为摊位入口，支持按漫展活动隔离库存、多摊位数据独立，现场快速开单、记录订单与销售台账，适配东南亚多地区漫展出摊场景，轻量化开箱即用，无需下载客户端。

## 项目结构

- `server/`：Go API、启动配置和运行数据。
- `web/`：Next.js 前端。
- `docs/`：产品与实现文档。

## 本地运行

`server/configs/local/stallink.yaml` 只写 Nacos 地址。运行参数在 Nacos 的 namespace `stallink`、dataId `stallink.yaml`。

凭据通过环境变量提供，不要写入仓库或提交到 Git：`STALLINK_NACOS_USERNAME`、`STALLINK_NACOS_PASSWORD` 用于连接本地 Nacos；`STALLINK_DATABASE_SOURCE` 和 `STALLINK_PLATFORM_ADMIN_PASSWORD` 在运行 API 的终端环境中提供。数据库连接串应指向本地 `stallink` 数据库，管理员密码至少 8 个字符。

套餐、库存预留和强制幂等开关已在本地 Nacos 模板中启用。后端启动时会执行版本迁移，但不再自动创建管理员或演示数据。

本地账号、两家店铺、活动、商品和套餐由 `seed-local` 显式创建。若要把现有本地数据清空并迁移到自增 ID，先确认数据库与图片备份存在，再运行：

```bash
make reset-local
make seed-local
```

`reset-local` 仅允许连接回环地址上的 `stallink` 数据库，并要求确认词以及非空的数据库/图片备份文件；它会清空业务表、重置自增序列并保留迁移记录与图片文件。不要对生产数据库运行此命令。演示账号统一使用密码 `12345678`；启动 API 本身不会再次塞入演示记录。

生产部署时应将 Go API 放在私有网络，仅允许 Web 服务访问；公网入口必须对 `/api/auth/login` 和 `/api/auth/register` 按真实客户端来源限流。当前代码不信任客户端自行传入的转发 IP 头，也未验证具体反向代理是否配置了限流。

```bash
export STALLINK_NACOS_USERNAME='你的本地 Nacos 用户名'
export STALLINK_NACOS_PASSWORD='你的本地 Nacos 密码'
export STALLINK_DATABASE_SOURCE='postgres://用户:密码@127.0.0.1:5432/stallink?sslmode=disable'
export STALLINK_PLATFORM_ADMIN_PASSWORD='至少 8 位的本地管理员密码'
make nacos-publish
make dev
make web
```

店面是 http://127.0.0.1:3000 ，接口是 http://127.0.0.1:8080 。
