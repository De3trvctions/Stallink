# Stallink 数据库自增 ID 规范

**状态：** 待执行

## 目标

把数据库实体主键从 UUID 改为 PostgreSQL BIGINT GENERATED ALWAYS AS IDENTITY，API 返回简短、可读的十进制 ID。只重建本地开发库的数据；服务端迁移不能在启动时悄悄删除现有数据。

## 当前代码基线

- schema.go 和 001_catalog_orders.sql 为业务表的主键、外键使用 UUID。SQL 查询大多用 id::text 返回，Go 数据层把 ID 当字符串传递。
- 服务端校验活动、报价等 ID 时直接调用 UUID 解析。前端的大部分 ID 类型已经是 string。
- 迁移器先建立基础 schema，再按版本执行嵌入式 SQL；版本 001 已经是历史迁移。
- 当前 SeedDemo 假设 alpha/beta 店铺已存在，只补活动、商品和收款码；不会创建店铺或覆盖不完整的旧数据，且服务启动时会调用。
- clientOrderId、Idempotency-Key 是幂等契约，不是数据库行主键。

## 设计决策

### 主键与外键

- 以下表的 id 改为 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY：shops、users、sessions、events、goods、good_images、event_listings、pay_codes、orders、order_lines、audit_logs、bundles、bundle_images、event_bundle_listings、order_quotes、order_commands、inventory_movements、order_pricing_groups。
- 对应的 *_id 外键、审计 subject_id 和优惠组 rule_id 都改为 BIGINT，并保留现有唯一约束、复合外键、跨店约束、删除行为和索引。
- bundle_components 继续用 (bundle_id, good_id) 作为复合主键；inventory_reservations 继续用 (order_id, listing_id)；order_payments.order_id 继续同时作为主键和订单外键。schema_migrations.version 是迁移版本号，不改成 identity。
- 不保证序列无空洞；删除记录不会复用 ID，也不以 ID 推断业务创建顺序。

### API 与幂等 ID

- 数据库实体 ID 在 JSON 中统一编码为十进制字符串，例如 "id":"42"。不编码成 JSON number，避免 JavaScript 超过安全整数范围后发生精度损失。
- Go 内部用有符号 64 位整数，并由命名 ID 类型保证 JSON marshal/unmarshal 使用十进制字符串；HTTP 路径和请求体中的实体 ID 必须是大于零、规范格式的十进制数。无效 ID 返回现有风格的 400 错误，越权或跨店 ID 仍按现有规则返回 404。
- clientOrderId、Idempotency-Key、order_quotes.consumed_by_client_order_id 继续使用 UUID；图片文件随机名、request ID 及会话 token/hash 也不随实体主键改变。
- 订单业务单号 orders.number 保持原样。订单 UI 展示业务单号，不把内部主键当订单号。
- 数字 ID 可枚举。任何授权判断必须继续基于会话、角色、店铺和活动范围，不能把 ID 难猜当成安全措施。

### 本地数据重置与注入

- 只重置 configs/local/stallink.yaml 指向的本地开发数据库。重置前备份 PostgreSQL 数据和 storage.dir 图片目录；备份留存，直到新库验收结束。
- 重置命令必须显式确认目标数据库名，并检查配置环境为 local。普通 make dev、自动迁移和生产启动都不得触发清空。
- 新版本迁移只允许在应用表为空时转换 UUID 列；遇到任何旧业务记录都中止并显示 identity_reset_required，不自动删除、不自动重编号。生产数据保留路径不属于本计划。
- 本地种子创建 2 个演示店铺（alpha、beta），及 6 个账号：1 个 platform_admin、每店 1 个 shop_admin 和 1 个 sales、1 个 customer。platform_admin/customer 不绑定店铺，其他 4 个员工账号各绑定自己的店铺。每店创建 1 场活动、与现有 demo fixture 匹配的商品、1 个固定套餐和收款码；不注入伪造订单或操作记录。密码只从本地忽略文件/环境读取，并通过现有密码哈希流程保存。
- 图片只使用有来源、可恢复的本地素材；不得写入失效的外网 URL 或伪装为真实商品图的占位字节。重置前备份 blob；确认种子和审核流程正常后再清理孤儿文件。

## 验收条件

1. 全部业务实体主键和外键均为 BIGINT；有 id 的行插入后由 identity sequence 自动分配。
2. 同一数据库新建和空库升级后，所有外键、复合店铺约束、库存、订单、报价、幂等重放和审核规则保持正确。
3. 非空旧库迁移安全失败，不清数据；本地显式重置后迁移成功，第一批 ID 从正整数开始。
4. Go API 只接受规范正整数实体 ID，并将其序列化为十进制字符串；幂等 UUID 契约不变。
5. 两个本地店铺的用户角色绑定正确；店铺之间的数据不可互相读取或修改。
6. Web 能用大于 JavaScript 安全整数上限的十进制字符串 ID 导航、搜索和提交，不进行 Number(id) 转换。
7. 旧实现计划不再与新方案并列；历史功能规格保留为背景资料。

## 明确不做

- 不把客户端幂等 UUID 改成自增数。
- 不使用自增值作为公开订单业务编号。
- 不在本次计划内实现生产库无损 UUID 到 BIGINT 的在线映射迁移。
- 不在普通启动流程中清库、重置 sequence 或创建演示账号。
