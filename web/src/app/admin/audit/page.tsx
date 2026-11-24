"use client";

import { ListControls, usePagedList } from "@/components/paged-list";
import { orderActionLabel } from "@/lib/orders";

type Audit = { action: string; actorLogin: string; subjectType: string; subjectId: string; createdAt: string };

export default function AuditPage() {
  const list = usePagedList<Audit>("/api/admin/audit");
  const entries = list.items;
  return <main>
    <p className="eyebrow">店铺管理</p><h1>操作记录</h1><p>查看店铺的管理和订单操作。</p>
    <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="搜索操作、账号或对象编号" />
    {list.error ? <p className="form-message form-message--error" role="alert">{list.error} <button type="button" onClick={list.refresh}>重试</button></p> : null}
    {list.loading ? <p className="order-empty" role="status">正在加载操作记录…</p> : !list.error && entries.length === 0 ? <p className="order-empty">{list.input ? "没有找到匹配的记录。" : "暂无操作记录。"}</p> : null}
    {!list.loading && entries.length > 0 ? <ul className="order-list">{entries.map((entry) => (
      <li className="audit-row" key={`${entry.subjectId}-${entry.createdAt}-${entry.action}-${entry.actorLogin}`}>
        <strong>{orderActionLabel(entry.action)}</strong>
        <div className="audit-row__meta">
          <span>{entry.actorLogin}</span>
          <time dateTime={entry.createdAt}>{new Date(entry.createdAt).toLocaleString("zh-CN")}</time>
        </div>
      </li>
    ))}</ul> : null}
  </main>;
}
