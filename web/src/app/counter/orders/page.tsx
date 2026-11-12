"use client";

import { useEffect, useRef, useState } from "react";
import { browserJSON } from "@/lib/api";
import { WorkspaceShell } from "@/components/workspace-shell";
import { pendingOrderCommand, sendOrderCommand } from "@/lib/order-command";
import { formatMoney } from "@/lib/pricing";
import { ListControls, usePagedList } from "@/components/paged-list";

type EventRow = { id: string; name: string };
type PayCode = { id: string; name: string; isDefault: boolean };
type Order = {
  id: string;
  number: string;
  status: string;
  boothName: string;
  currency: string;
  priceLabel: string;
  openedByLogin: string;
  openedByRole: string;
  reservationExpiresAt?: string | null;
  lines: { goodName: string; qty: number; lineTotalMinor: number; priceLabel: string; discountMinor: number }[];
  pricingGroups: { kind: string; name: string; applications: number; totalMinor: number }[];
};

const statusLabel: Record<string, string> = {
  pending: "待付款",
  paid: "已收款",
  fulfilled: "已交付",
  cancelled: "已取消",
  expired: "预留已过期",
  refund_required: "待退款处理",
};

export default function EventOrdersPage() {
  const [events, setEvents] = useState<EventRow[]>([]);
  const [eventId, setEventId] = useState("");
  const list = usePagedList<Order>(`/api/staff/orders?eventId=${encodeURIComponent(eventId)}`, Boolean(eventId));
  const orders = list.items;
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [detailError, setDetailError] = useState("");
  const [payCodes, setPayCodes] = useState<PayCode[]>([]);
  const [selectedPayCodeId, setSelectedPayCodeId] = useState("");
  const [actingOrder, setActingOrder] = useState("");
  const [message, setMessage] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const dialogRef = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (selectedId && !dialog.open) dialog.showModal();
    if (!selectedId && dialog.open) dialog.close();
  }, [selectedId]);

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
    Promise.all([browserJSON("/api/staff/events"), browserJSON("/api/staff/pay-codes")]).then(([res, codesRes]) => {
      const rows = Array.isArray(res.data) ? res.data as EventRow[] : [];
      const codes = Array.isArray(codesRes.data) ? codesRes.data as PayCode[] : [];
      setEvents(rows);
      setPayCodes(codes);
      setSelectedPayCodeId(codes.find((code) => code.isDefault)?.id ?? codes[0]?.id ?? "");
      if (rows[0]) {
        setEventId(rows[0].id);

      }
    });
  }, []);

  async function act(id: string, action: "pay" | "fulfill" | "cancel", payCodeId?: string) {
    const scope = `counter-order:${id}:${action}`;
    setActingOrder(id);
    setMessage("");
    try {
      let body: { payCodeId: string } | Record<string, never> = {};
      if (action === "pay") {
        const pending = pendingOrderCommand<{ payCodeId: string }>(scope);
        if (pending) body = pending.body;
        else if (payCodeId) body = { payCodeId };
        else {
          setMessage("没有可用的收款方式，请先在后台配置收款码。");
          return;
        }
      }
      const result = await sendOrderCommand(scope, `/api/staff/orders/${id}/${action}`, body);
      if (!result.ok) {
        setMessage(result.data.error === "command_payload_unresolved"
          ? "上次操作仍未确认，页面会保留原请求；请重试该操作后再继续。"
          : result.data.error === "reservation_expired" ? "库存预留已过期，请重新下单。"
            : "订单操作没有确认成功，请重试或刷新订单状态。");
        list.refresh();
        return;
      }
      list.refresh();
    } catch {
      setMessage("无法连接服务。操作请求已保留，重试时会使用同一个幂等请求。");
    } finally {
      setActingOrder("");
    }
  }

  async function reserve(id: string) {
    const scope = `counter-order:${id}:reserve`;
    setActingOrder(id);
    setMessage("");
    try {
      const result = await sendOrderCommand(scope, `/api/staff/orders/${id}/reserve`, {});
      if (!result.ok) setMessage("库存预留没有成功，请刷新订单后重试。");
      list.refresh();
    } catch {
      setMessage("无法连接服务。操作请求已保留，重试时会使用同一个幂等请求。");
    } finally {
      setActingOrder("");
    }
  }

  return (
    <WorkspaceShell kind="sales">
    <main>
      <p className="eyebrow">现场工作台</p>
      <h1>本场订单</h1>
      <p>这一场开出的单子都在这里；点开一行核对商品、金额与收款方式。</p>
      <label className="field event-selector" htmlFor="event-order-filter">
        <span>活动场次</span>
        <select id="event-order-filter" className="mt-2 w-full border border-stone-300 bg-white px-3 py-3" value={eventId} onChange={(e) => { setEventId(e.target.value); list.reset(); setSelectedId(null); }}>
          {events.map((event) => <option key={event.id} value={event.id}>{event.name}</option>)}
        </select>
      </label>
      <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="订单号、开单人或顾客称呼" />
      {message ? <p className="form-message form-message--error" role="status">{message}</p> : null}
      {list.error ? <p className="form-message form-message--error" role="alert">{list.error} <button type="button" onClick={list.refresh}>重试</button></p> : null}
      {list.loading ? <p className="order-empty" role="status">正在加载订单...</p> : list.error ? null : orders.length === 0 ? <p className="order-empty">{list.input ? "没有找到匹配的订单。" : "暂无订单。"}</p> : <ul className="order-list">{orders.map((order) => <li key={order.id}>
        <button type="button" className="order-list__row" onClick={() => { setSelectedOrder(null); setDetailError(""); setSelectedId(order.id); }}><span><strong>{order.number}</strong><small>{order.boothName || "未留称呼"} · {order.openedByLogin}</small></span><span className="order-list__meta"><strong>{order.priceLabel}</strong><small>{statusLabel[order.status] ?? order.status}　›</small></span></button>
      </li>)}</ul>}
      <dialog ref={dialogRef} className="order-dialog" style={{ position: "fixed", inset: 0, margin: "auto" }} aria-labelledby="counter-order-title" onClose={() => setSelectedId(null)} onClick={(event) => { if (event.target === event.currentTarget) setSelectedId(null); }}>
        {!selectedOrder ? <div className="order-dialog__content" role="status"><button type="button" onClick={() => setSelectedId(null)}>关闭</button><p>{detailError || "正在加载订单详情..."}</p></div> : <div className="order-dialog__content">
          <header className="order-dialog__header"><div><p className="eyebrow">订单详情</p><h2 id="counter-order-title">{selectedOrder.number}</h2></div><button type="button" aria-label="关闭订单详情" onClick={() => setSelectedId(null)}>×</button></header>
          <p>{statusLabel[selectedOrder.status] ?? selectedOrder.status} · {selectedOrder.boothName || "未留称呼"}</p>
          <p className="order-dialog__sub">开单人：{selectedOrder.openedByLogin}（{selectedOrder.openedByRole}）</p>
          {selectedOrder.reservationExpiresAt ? <p className="order-dialog__sub">库存预留至 {new Date(selectedOrder.reservationExpiresAt).toLocaleString("zh-CN")}</p> : null}
          <h3>商品明细</h3><ul className="order-dialog__items">{selectedOrder.lines?.map((line, index) => <li key={`${line.goodName}-${index}`}><span>{line.goodName} × {line.qty}</span><strong>{line.priceLabel}</strong></li>)}</ul>
          {selectedOrder.pricingGroups?.length ? <><h3>优惠明细</h3><ul className="order-dialog__items">{selectedOrder.pricingGroups.map((group, index) => <li key={`${group.kind}-${index}`}><span>{group.name}{group.applications > 1 ? ` × ${group.applications}` : ""}</span><strong>{formatMoney(selectedOrder.currency, group.totalMinor)}</strong></li>)}</ul></> : null}
          <div className="order-dialog__total"><span>订单总额</span><strong>{selectedOrder.priceLabel}</strong></div>
          {message ? <p className="form-message form-message--error" role="alert">{message}</p> : null}
            <div className="flex flex-wrap gap-2">
              {selectedOrder.status === "pending" && !selectedOrder.reservationExpiresAt ? <button className="h-10 border px-3" disabled={Boolean(actingOrder)} onClick={() => reserve(selectedOrder.id)}>{actingOrder === selectedOrder.id ? "处理中…" : "预留库存"}</button> : null}
              {selectedOrder.status === "pending" ? <>
                <select aria-label={`订单 ${selectedOrder.number} 收款方式`} value={pendingOrderCommand<{ payCodeId: string }>(`counter-order:${selectedOrder.id}:pay`)?.body.payCodeId ?? selectedPayCodeId} disabled={Boolean(actingOrder) || Boolean(pendingOrderCommand(`counter-order:${selectedOrder.id}:pay`))} onChange={(change) => setSelectedPayCodeId(change.target.value)}>
                  {payCodes.length === 0 ? <option value="">没有收款方式</option> : payCodes.map((code) => <option key={code.id} value={code.id}>{code.name}{code.isDefault ? " · 默认" : ""}</option>)}
                </select>
                <button className="h-10 border px-3" disabled={Boolean(actingOrder) || payCodes.length === 0} onClick={() => act(selectedOrder.id, "pay", pendingOrderCommand<{ payCodeId: string }>(`counter-order:${selectedOrder.id}:pay`)?.body.payCodeId ?? selectedPayCodeId)}>{actingOrder === selectedOrder.id ? "处理中…" : "确认收款"}</button>
              </> : null}
              {selectedOrder.status === "paid" ? <button className="h-10 border px-3" disabled={Boolean(actingOrder)} onClick={() => act(selectedOrder.id, "fulfill")}>{actingOrder === selectedOrder.id ? "处理中…" : "交付"}</button> : null}
              {!['cancelled', 'expired', 'refund_required'].includes(selectedOrder.status) ? <button className="h-10 border px-3" disabled={Boolean(actingOrder)} onClick={() => act(selectedOrder.id, "cancel")}>{selectedOrder.status === "pending" ? "取消订单" : "标记待退款"}</button> : null}
            </div>
        </div>}
      </dialog>
    </main>
    </WorkspaceShell>
  );
}
