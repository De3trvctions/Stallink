"use client";

import { ListControls, usePagedList } from "@/components/paged-list";

type Audit = { action: string; actorLogin: string; createdAt: string };
const actions: Record<string, string> = { "bundle.approve": "审核通过套餐", "bundle.create": "创建套餐", "bundle.update": "编辑套餐", "bundle.reject": "拒绝套餐", "good.approve": "审核通过商品", "good.reject": "拒绝商品", "good.image.add": "添加商品图片", "good.image.delete": "删除商品图片", "good.image.reorder": "调整商品图片顺序", "listing.create": "上架商品", "listing.update": "修改商品上架信息", "stock.update": "调整商品库存", "order.create": "创建订单", "order.pay": "确认收款", "order.cancel": "取消订单", "order.fulfill": "完成交付", "order.expire": "订单预留过期" };

export default function AuditPage() {
  const list = usePagedList<Audit>("/api/admin/audit");
  const entries = list.items;
  return <main>
    <p className="eyebrow">店铺管理</p><h1>操作记录</h1><p>查看店铺的管理和订单操作。</p>
    <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="搜索操作、账号或对象编号" />
    {list.error ? <p className="form-message form-message--error" role="alert">{list.error} <button type="button" onClick={list.refresh}>重试</button></p> : null}
    {list.loading ? <p className="order-empty" role="status">正在加载操作记录...</p> : !list.error && entries.length === 0 ? <p className="order-empty">{list.input ? "没有找到匹配的记录。" : "暂无操作记录。"}</p> : null}
    {!list.loading && entries.length > 0 ? <ul className="order-list">{entries.map((entry, index) => <li className="audit-row" key={`${entry.createdAt}-${index}`}><strong>{actions[entry.action] ?? entry.action}</strong><span>{entry.actorLogin} · {entry.createdAt}</span></li>)}</ul> : null}
  </main>;
}
