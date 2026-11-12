# Stallink Web 自增 ID 适配计划

> **执行说明：** 本计划覆盖 Next.js 前端，依赖服务端已提供十进制字符串 ID。逐项勾选；不提交 Git，除非用户另行要求。

**目标：** 让店面、后台、收银、平台审核和搜索可靠处理数据库自增 ID，不把 ID 当普通 JS 数字。

**架构：** JSON 实体 ID 使用 decimal string；前端继续以 string 存储、传递和作为 React key。UUID checkout 和幂等键保持 UUID。内部数字 ID 与面向顾客的订单业务单号分开。

**技术栈：** Next.js 16 App Router、React 19、TypeScript、现有同源 API 代理。

**设计依据：** [数据库自增 ID 规范](../specs/2026-11-12-stallink-bigint-identity-spec.md)。

## 文件职责

| 文件 | 职责 |
|---|---|
| web/src/lib/api.ts | 统一实体 ID 类型和响应边界校验 |
| 新建 web/src/lib/entity-id.ts | 验证正十进制字符串 ID，不转成 number |
| web/src/lib/counter-draft.ts, order-command.ts | 保持草稿键、clientOrderId 和命令键的语义区分 |
| web/src/app/** | 更新所有返回类型、详情链接、筛选参数、图片路径和表单请求 |
| web/src/components/paged-list.tsx | 确保搜索的数字 ID 按字符串原样传递 |
| web/README.md | 写明 API ID 字符串契约及本地联调方式 |

## 任务 1：固定 Web ID 契约

- [ ] 新建 entity-id.ts：EntityId 保持 string 表示；parseEntityId(value) 只接受规范正十进制字符串且拒绝 0、负数和 UUID。
- [ ] 给解析器补充用例或可运行的 Node 内置测试；覆盖 "42"、"9007199254740993" 和非法边界，不引入重量级测试依赖。
- [ ] 在 api.ts 的响应入口验证实体 ID 字段；校验失败显示现有网络错误状态，不能把坏值送入动态路由。
- [ ] clientOrderId、Idempotency-Key 使用现有 UUID 生成/恢复逻辑，不经过实体 ID 解析器。

## 任务 2：更新页面和组件边界

- [ ] 审查 login, register, me, admin, counter, platform, c/[slug] 的 ID 类型与请求；所有实体 ID 保持字符串，不执行 Number(id)、parseInt(id) 或数值排序。
- [ ] 确保 dynamic route 参数 [eventId], [listingId], [bundleListingId] 原样传给 API；服务端仍负责最终授权和输入校验。
- [ ] 订单列表展示业务 number；只有在明确需要时才展示内部十进制 ID。操作记录对象编号和搜索结果按字符串显示。
- [ ] 检查 cart draft localStorage key、列表 React key、商品图片 URL、审核 PATCH、收款码请求和订单详情 modal 均能处理十进制字符串 ID。
- [ ] 对照请求契约验证 q=12345 搜索的是数字字符串，且当前页/活动筛选不会把 ID 转成数字。

## 任务 3：联调与响应式回归

- [ ] 在独立本地测试数据中创建至少两位数 ID 和大于 Number.MAX_SAFE_INTEGER 的模拟响应；确认前端保留字面字符串。
- [ ] 用真实浏览器检查店面活动/商品链接、套餐页、商品审核、图片、订单列表/详情弹窗、操作记录分页及搜索。
- [ ] 从 web/ 运行 npm run lint 和 npx tsc --noEmit；从仓库根运行 make web 做浏览器联调。
- [ ] 检查桌面、平板、手机布局；本任务仅改 ID 契约，不改已经确认的中性黑白灰与克制蓝色视觉方案。

## 执行顺序

服务端先确定并提供十进制字符串 API 契约；然后执行本计划。若仅靠现有 string 类型即可满足某项，不做无意义的全站重写。
