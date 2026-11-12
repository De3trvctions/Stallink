"use client";

import { useEffect, useRef, useState } from "react";
import { browserJSON } from "@/lib/api";
import { sendOrderCommand } from "@/lib/order-command";
import { formatMoney } from "@/lib/pricing";
import { ListControls, usePagedList } from "@/components/paged-list";

type Order = {
  id: string;
  number: string;
  eventName: string;
  openedByLogin: string;
  openedByRole: string;
  status: string;
  currency: string;
  priceLabel: string;
  payCodeName?: string;
  reservationExpiresAt?: string | null;
  lines: { goodName: string; qty: number; unitPriceMinor: number; lineTotalMinor: number; discountMinor: number; priceLabel: string }[];
  pricingGroups: { kind: string; name: string; applications: number; totalMinor: number }[];
};

const statusLabel: Record<string, string> = { pending: "待付款", paid: "已收款", fulfilled: "已交付", cancelled: "已取消", expired: "预留已过期", refund_required: "待退款处理" };

export default function AdminOrdersPage() {
  const list = usePagedList<Order>("/api/admin/orders");
  const orders = list.items;
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [detailError, setDetailError] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const dialogRef = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    if (!selectedId) return;
    let active = true;
    browserJSON(`/api/staff/orders/${selectedId}`).then((res) => {
      if (!active) return;
      if (!res.ok) throw new Error("request failed");
      setSelectedOrder(res.data as Order); setDetailError("");
    }).catch(() => { if (active) setDetailError("订单详情加载失败，请关闭后重试。"); });
    return () => { active = false; };
  }, [selectedId, orders]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (selectedId && !dialog.open) dialog.showModal();
    if (!selectedId && dialog.open) dialog.close();
  }, [selectedId]);
  const [actingOrder, setActingOrder] = useState("");
  const [message, setMessage] = useState("");

  async function cancel(orderID: string) {
    setActingOrder(orderID);
    setMessage("");
    try {
      const res = await sendOrderCommand(`admin-order:${orderID}:cancel`, `/api/staff/orders/${orderID}/cancel`, {});
      if (!res.ok) setMessage("订单取消没有确认成功，请重试或刷新订单状态。");
      list.refresh();
    } catch {
      setMessage("无法连接服务。操作请求已保留，重试时会使用同一个幂等请求。");
    } finally {
      setActingOrder("");
    }
  }

  return (
    <main>
      <p className="eyebrow">现场流水</p>
      <h1>订单记录</h1>
      <p>按订单号、活动、开单人或顾客称呼查找；点开一行看商品、优惠与收款明细。</p>
      <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="订单号、活动、开单人或顾客称呼" />
      {list.error ? <p className="form-message form-message--error" role="alert">{list.error} <button type="button" onClick={list.refresh}>重试</button></p> : null}
      {message ? <p className="form-message form-message--error" role="status">{message}</p> : null}
      {list.loading ? <p className="order-empty" role="status">正在加载订单...</p> : list.error ? null : orders.length === 0 ? <p className="order-empty">{list.input ? "没有找到匹配的订单。" : "暂无订单。"}</p> : <ul className="order-list">
        {orders.map((order) => <li key={order.id}>
          <button type="button" className="order-list__row" onClick={() => { setSelectedOrder(null); setDetailError(""); setSelectedId(order.id); }}>
            <span><strong>{order.number}</strong><small>{order.eventName} · {order.openedByLogin}</small></span>
            <span className="order-list__meta"><strong>{order.priceLabel}</strong><small>{statusLabel[order.status] ?? order.status}　›</small></span>
          </button>
        </li>)}
      </ul>}
      <dialog ref={dialogRef} className="order-dialog" style={{ position: "fixed", inset: 0, margin: "auto" }} aria-labelledby="order-dialog-title" onClose={() => setSelectedId(null)} onClick={(event) => { if (event.target === event.currentTarget) setSelectedId(null); }}>
        {!selectedOrder ? <div className="order-dialog__content" role="status"><button type="button" onClick={() => setSelectedId(null)}>关闭</button><p>{detailError || "正在加载订单详情..."}</p></div> : <div className="order-dialog__content">
          <header className="order-dialog__header"><div><p className="eyebrow">订单详情</p><h2 id="order-dialog-title">{selectedOrder.number}</h2></div><button type="button" aria-label="关闭订单详情" onClick={() => setSelectedId(null)}>×</button></header>
          <p>{selectedOrder.eventName} · {statusLabel[selectedOrder.status] ?? selectedOrder.status}</p>
          <p className="order-dialog__sub">开单人：{selectedOrder.openedByLogin}（{selectedOrder.openedByRole}）</p>
          {selectedOrder.reservationExpiresAt ? <p className="order-dialog__sub">库存预留至 {new Date(selectedOrder.reservationExpiresAt).toLocaleString("zh-CN")}</p> : null}
          {selectedOrder.payCodeName ? <p className="order-dialog__sub">收款方式：{selectedOrder.payCodeName}</p> : null}
          <h3>商品明细</h3><ul className="order-dialog__items">{selectedOrder.lines?.map((line, index) => <li key={`${line.goodName}-${index}`}><span>{line.goodName} × {line.qty}{line.discountMinor > 0 ? `（优惠 ${formatMoney(selectedOrder.currency, line.discountMinor)}）` : ""}</span><strong>{line.priceLabel}</strong></li>)}</ul>
          {selectedOrder.pricingGroups?.length ? <><h3>优惠明细</h3><ul className="order-dialog__items">{selectedOrder.pricingGroups.map((group, index) => <li key={`${group.kind}-${index}`}><span>{group.name}{group.applications > 1 ? ` × ${group.applications}` : ""}</span><strong>{formatMoney(selectedOrder.currency, group.totalMinor)}</strong></li>)}</ul></> : null}
          <div className="order-dialog__total"><span>订单总额</span><strong>{selectedOrder.priceLabel}</strong></div>
          {message ? <p className="form-message form-message--error" role="alert">{message}</p> : null}
          {!['cancelled', 'expired', 'refund_required', 'paid', 'fulfilled'].includes(selectedOrder.status) ? <button className="button button--outline" disabled={Boolean(actingOrder)} onClick={() => cancel(selectedOrder.id)}>{actingOrder === selectedOrder.id ? "处理中…" : "取消订单"}</button> : null}
        </div>}
      </dialog>
    </main>
  );
}
