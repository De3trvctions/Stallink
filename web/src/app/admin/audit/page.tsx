"use client";

import { ListControls, usePagedList } from "@/components/paged-list";
import { auditSubjectLabel, formatAuditTime } from "@/lib/audit";
import { orderActionLabel } from "@/lib/orders";

type Audit = { action: string; actorLogin: string; subjectType: string; subjectId: string; subjectLabel?: string; createdAt: string };

export default function AuditPage() {
  const list = usePagedList<Audit>("/api/admin/audit");
  const entries = list.items;
  return <main className="workspace-subpage audit-page">
    <p className="eyebrow">店铺管理</p><h1>操作记录</h1><p>查看店铺的管理和订单操作。</p>
    <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="搜索操作、账号或对象编号" />
    {list.error ? <p className="form-message form-message--error" role="alert">{list.error} <button type="button" onClick={list.refresh}>重试</button></p> : null}
    {list.loading ? <p className="order-empty" role="status">正在加载操作记录…</p> : !list.error && entries.length === 0 ? <p className="order-empty">{list.input ? "没有找到匹配的记录。" : "暂无操作记录。"}</p> : null}
    {!list.loading && entries.length > 0 ? <ul className="order-list" aria-label="操作记录列表">{entries.map((entry) => (
      <li className="audit-row" key={`${entry.subjectId}-${entry.createdAt}-${entry.action}-${entry.actorLogin}`}>
        <strong>{orderActionLabel(entry.action)}</strong>
        <span className="audit-row__subject"><span className="audit-row__subject-label">对象</span>{auditSubjectLabel(entry)}</span>
        <div className="audit-row__meta">
          <span>{entry.actorLogin}</span>
          <time dateTime={entry.createdAt}>{formatAuditTime(entry.createdAt)}</time>
        </div>
      </li>
    ))}</ul> : null}
  </main>;
}
