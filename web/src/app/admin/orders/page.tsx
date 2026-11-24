"use client";

import { useState } from "react";
import type { StaffOrder } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney } from "@/lib/pricing";
import { orderStatusLabel } from "@/lib/orders";
import { messageClass, useMessage } from "@/lib/message";
import { sendOrderCommand, clearPendingOrderCommand } from "@/lib/order-command";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ListControls, usePagedList } from "@/components/paged-list";
import SelectMenu from "@/components/select-menu";

type StatusFilter = "all" | StaffOrder["status"];

type ConfirmTarget = {
  order: StaffOrder;
  action: "cancel" | "refund" | "fulfill";
};

export default function AdminOrdersPage() {
  const [filter, setFilter] = useState<StatusFilter>("all");
  const { message, setError, setSuccess } = useMessage();
  const [pendingId, setPendingId] = useState("");
  const [confirmTarget, setConfirmTarget] = useState<ConfirmTarget | null>(null);

  const list = usePagedList<StaffOrder>(`/api/admin/orders?status=${filter === "all" ? "" : encodeURIComponent(filter)}`);
  const visible = list.items;

  async function runAction(target: ConfirmTarget) {
    const { order, action } = target;
    setPendingId(order.id);
    setError(null);
    try {
      const scope = `${action}:admin:${order.id}`;
      const res = action === "refund"
        ? await sendOrderCommand(scope, `/api/staff/orders/${order.id}/refund-confirm`, {})
        : action === "fulfill"
          ? await sendOrderCommand(scope, `/api/staff/orders/${order.id}/fulfill`, {})
          : await sendOrderCommand(scope, `/api/staff/orders/${order.id}/cancel`, { reason: "后台取消订单" });
      if (!res.ok) {
        setError(describeRequest(res));
        return;
      }
      clearPendingOrderCommand(scope);
      setSuccess(action === "refund" ? "退款已确认，订单已取消并回补库存。" : action === "fulfill" ? "订单已标记为当场交付。" : "订单已取消。");
      setConfirmTarget(null);
      list.refresh();
    } catch {
      setError("无法连接服务，请重试同一笔操作以安全恢复结果。");
    } finally {
      setPendingId("");
    }
  }

  const pendingCancel = confirmTarget?.order.id === pendingId;

  return (
    <main className="workspace-page">
      <header className="workspace-page__heading">
        <p className="eyebrow">订单记录</p>
        <h1>核对现场订单</h1>
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
                  {order.cashReceivedMinor !== null ? <span>实收：{formatMoney(order.currency, order.cashReceivedMinor)}</span> : null}
                  {order.cashChangeMinor !== null ? <span>找零：{formatMoney(order.currency, order.cashChangeMinor)}</span> : null}
                  {order.openedByLogin ? <span>开单：{order.openedByLogin}</span> : null}
                  {order.paidByLogin ? <span>收款：{order.paidByLogin}</span> : null}
                </div>
                <div className="order-row__actions">
                  {order.status === "refund_required" ? (
                    <button
                      type="button"
                      className="button button--danger"
                      disabled={pendingId === order.id}
                      onClick={() => setConfirmTarget({ order, action: "refund" })}
                    >确认退款</button>
                  ) : null}
                  {order.status === "paid" || order.status === "fulfilled" ? (
                    <button
                      type="button"
                      className="button button--danger"
                      disabled={pendingId === order.id}
                      onClick={() => setConfirmTarget({ order, action: "cancel" })}
                    >取消订单</button>
                  ) : null}
                  {order.status === "pending" ? (
                    <button
                      type="button"
                      className="button button--danger"
                      disabled={pendingId === order.id}
                      onClick={() => setConfirmTarget({ order, action: "cancel" })}
                    >取消订单</button>
                  ) : null}
                  {order.status === "paid" ? (
                    <button
                      type="button"
                      className="button button--outline"
                      disabled={pendingId === order.id}
                      onClick={() => setConfirmTarget({ order, action: "fulfill" })}
                    >标记交付</button>
                  ) : null}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}

      <ConfirmDialog
        open={confirmTarget?.action === "refund"}
        title="确认退款"
        description="确认后：订单转为已取消、库存回补、金额记为已退，不可撤销。"
        confirmLabel="确认退款"
        danger
        pending={pendingCancel}
        onConfirm={() => confirmTarget && void runAction(confirmTarget)}
        onCancel={() => setConfirmTarget(null)}
      />
      <ConfirmDialog
        open={confirmTarget?.action === "cancel" && (confirmTarget.order.status === "paid" || confirmTarget.order.status === "fulfilled")}
        title="取消已收款订单"
        description="取消后订单转为待退款、库存不回补，需由管理员确认退款；此操作不可撤销。"
        confirmLabel="取消订单"
        danger
        pending={pendingCancel}
        onConfirm={() => confirmTarget && void runAction(confirmTarget)}
        onCancel={() => setConfirmTarget(null)}
      />
      <ConfirmDialog
        open={confirmTarget?.action === "cancel" && confirmTarget.order.status === "pending"}
        title="取消待付款订单"
        description="取消后订单作废，预留库存释放，未收款。"
        confirmLabel="取消订单"
        danger
        pending={pendingCancel}
        onConfirm={() => confirmTarget && void runAction(confirmTarget)}
        onCancel={() => setConfirmTarget(null)}
      />
      <ConfirmDialog
        open={confirmTarget?.action === "fulfill"}
        title="标记当场交付"
        description="交付后订单记为已履约，顾客凭订单号取货。"
        confirmLabel="标记交付"
        pending={pendingCancel}
        onConfirm={() => confirmTarget && void runAction(confirmTarget)}
        onCancel={() => setConfirmTarget(null)}
      />
    </main>
  );
}
