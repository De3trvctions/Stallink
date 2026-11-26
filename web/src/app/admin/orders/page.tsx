"use client";

import { useRef, useState } from "react";
import type { StaffOrder } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney } from "@/lib/pricing";
import { orderStatusLabel } from "@/lib/orders";
import { messageClass, useMessage } from "@/lib/message";
import { clearPendingOrderCommand, pendingOrderCommand, sendOrderCommand } from "@/lib/order-command";
import { orderActionRequestBody, orderActionsFor, type OrderAction } from "@/lib/order-actions";
import { OrderActionDialog, type OrderActionTarget } from "@/components/order-action-dialog";
import { ListControls, usePagedList } from "@/components/paged-list";
import SelectMenu from "@/components/select-menu";

type StatusFilter = "all" | StaffOrder["status"];

const actionLabel: Record<OrderAction, string> = { fulfill: "标记交付", cancel: "取消订单", refund: "确认退款" };

export default function AdminOrdersPage() {
  const [filter, setFilter] = useState<StatusFilter>("all");
  const { message, setError, setSuccess } = useMessage();
  const [pendingId, setPendingId] = useState("");
  const [confirmTarget, setConfirmTarget] = useState<OrderActionTarget | null>(null);
  const inFlight = useRef(false);

  const list = usePagedList<StaffOrder>(`/api/admin/orders?status=${filter === "all" ? "" : encodeURIComponent(filter)}`);
  const visible = list.items;

  async function runAction(target: OrderActionTarget) {
    // 同步锁：React 状态更新前的连点也只发一次请求；幂等键另由 sendOrderCommand 保持。
    if (inFlight.current) return;
    inFlight.current = true;
    const { order, action } = target;
    setPendingId(order.id);
    setError(null);
    try {
      const scope = `${action}:admin:${order.id}`;
      // 原因已在弹窗内校验必填；后端暂不接收原因字段，见 orderActionRequestBody。
      const body = pendingOrderCommand(scope)?.body ?? orderActionRequestBody(action);
      const path = action === "refund" ? "refund-confirm" : action;
      const res = await sendOrderCommand(scope, `/api/staff/orders/${order.id}/${path}`, body);
      if (!res.ok) { setError(describeRequest(res)); return; }
      clearPendingOrderCommand(scope);
      setSuccess(action === "refund" ? "退款已确认，订单已取消并回补库存。" : action === "fulfill" ? "订单已标记为当场交付。" : order.status === "pending" ? "订单已取消。" : "订单已转为待退款，退款给顾客后请确认退款。");
      setConfirmTarget(null);
      list.refresh();
    } catch {
      setError("无法连接服务，请重试同一笔操作以安全恢复结果。");
    } finally {
      inFlight.current = false;
      setPendingId("");
    }
  }

  return (
    <main className="workspace-page orders-page">
      <header className="workspace-page__heading">
        <p className="eyebrow">订单记录</p>
        <h1>订单记录</h1>
        <p>查看商品明细、收款状态与经手人。取消已收款订单会转为待退款，需再确认退款。</p>
      </header>

      <div className="workspace-toolbar">
        <SelectMenu
          value={filter}
          onChange={(value) => { setFilter(value as StatusFilter); list.reset(); }}
          ariaLabel="按状态筛选订单"
          options={[
            { value: "all", label: "全部状态" },
            { value: "pending", label: "待付款" },
            { value: "paid", label: "已收款" },
            { value: "fulfilled", label: "已交付" },
            { value: "refund_required", label: "待退款处理" },
            { value: "cancelled", label: "已取消" },
            { value: "expired", label: "预留已过期" },
          ]}
        />
        <button type="button" className="button button--outline" onClick={list.refresh}>刷新</button>
      </div>

      <ListControls value={list.input} onChange={list.setInput} page={list.page} setPage={list.setPage} hasMore={list.hasMore} loading={list.loading} placeholder="订单号、活动、开单人或顾客称呼" />

      {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}

      {list.loading ? <p className="empty-state">正在读取订单…</p> : list.error ? (
        <p className="empty-state" role="alert">
          <span className="empty-state__mark" aria-hidden="true">!</span>
          <span>{list.error}</span>
          <button type="button" className="button button--outline" onClick={list.refresh}>重试</button>
        </p>
      ) : visible.length === 0 ? (
        <p className="empty-state">
          <span className="empty-state__mark" aria-hidden="true">○</span>
          <span>{filter === "all" ? "目前还没有订单。" : "这个状态下暂时没有订单。"}</span>
        </p>
      ) : (
        <ul className="order-list">
          {visible.map((order) => (
            <li className="order-row" key={order.id}>
              <div className="order-row__head">
                <div>
                  <strong>订单 {order.number}</strong>
                  <span className={`order-status order-status--${order.status}`}>{orderStatusLabel(order.status)}</span>
                  {order.status === "cancelled" && order.refundedAt ? <span className="order-status order-status--refunded">已退款</span> : null}
                </div>
                <div className="order-row__meta">
                  <span>{order.eventName || "活动商品"}</span>
                  <time dateTime={order.createdAt}>{new Date(order.createdAt).toLocaleString("zh-CN")}</time>
                </div>
              </div>
              {order.boothName ? <p className="order-row__booth">顾客称呼：{order.boothName}</p> : null}
              <ul className="order-lines">
                {order.lines.map((line, index) => (
                  <li key={`${order.id}-${index}-${line.goodName}`}>
                    <span>{line.goodName} × {line.qty}</span>
                    <span>{formatMoney(order.currency, line.lineTotalMinor)}</span>
                  </li>
                ))}
              </ul>
              <div className="order-row__footer">
                <div className="order-row__totals">
                  <span>合计 <strong>{formatMoney(order.currency, order.totalMinor)}</strong></span>
                  {order.payCodeName ? <span>收款方式：{order.payCodeName}</span> : null}
                  {order.openedByLogin ? <span>开单：{order.openedByLogin}</span> : null}
                  {order.paidByLogin ? <span>收款：{order.paidByLogin}</span> : null}
                </div>
                <div className="order-row__actions">
                  {orderActionsFor(order.status).map((action) => (
                    <button
                      key={action}
                      type="button"
                      className={action === "fulfill" ? "button button--outline" : "button button--danger"}
                      disabled={pendingId !== ""}
                      onClick={() => setConfirmTarget({ order, action })}
                    >{actionLabel[action]}</button>
                  ))}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}

      <OrderActionDialog
        key={confirmTarget ? `${confirmTarget.action}:${confirmTarget.order.id}` : "closed"}
        target={confirmTarget}
        pending={pendingId !== ""}
        onConfirm={(target) => void runAction(target)}
        onCancel={() => { if (pendingId === "") setConfirmTarget(null); }}
      />
    </main>
  );
}
