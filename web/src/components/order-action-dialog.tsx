"use client";

import { useId, useState } from "react";
import type { StaffOrder } from "@/lib/api";
import { actionNeedsReason, REASON_MAX_LENGTH, REASON_MIN_LENGTH, validateActionReason, type OrderAction } from "@/lib/order-actions";
import { ConfirmDialog } from "@/components/confirm-dialog";

export type OrderActionTarget = { order: StaffOrder; action: OrderAction };

type Copy = { title: string; description: string; confirmLabel: string; danger: boolean };

function copyFor(target: OrderActionTarget): Copy {
  const { order, action } = target;
  if (action === "fulfill") {
    return { title: "标记当场交付", description: "交付后订单记为已履约，顾客凭订单号取货。", confirmLabel: "标记交付", danger: false };
  }
  if (action === "refund") {
    return { title: "确认退款", description: "确认款项已退给顾客。确认后订单转为已取消、库存回补、金额记为已退，不可撤销。", confirmLabel: "确认退款", danger: true };
  }
  if (order.status === "pending") {
    return { title: "取消待付款订单", description: "取消后订单作废，预留库存释放，未收款。", confirmLabel: "取消订单", danger: true };
  }
  return {
    title: "取消已收款订单",
    description: "取消后订单转为待退款，库存暂不回补；把钱退给顾客后，再点「确认退款」回补库存。此操作不可撤销。",
    confirmLabel: "取消订单",
    danger: true,
  };
}

// 取消与确认退款必须填写原因；交付无需原因。pending 期间按钮禁用，防止重复提交。
export function OrderActionDialog({ target, pending, onConfirm, onCancel }: {
  target: OrderActionTarget | null;
  pending: boolean;
  onConfirm: (target: OrderActionTarget, reason: string) => void;
  onCancel: () => void;
}) {
  const fieldId = useId();
  const [reason, setReason] = useState("");
  const [touched, setTouched] = useState(false);
  if (!target) return null;

  const copy = copyFor(target);
  const needsReason = actionNeedsReason(target.action);
  const validation = validateActionReason(reason);
  const showError = needsReason && touched && !validation.ok;

  function confirm() {
    if (!target || pending) return;
    if (needsReason) {
      setTouched(true);
      if (!validation.ok) return;
      onConfirm(target, validation.reason);
      return;
    }
    onConfirm(target, "");
  }

  return (
    <ConfirmDialog
      open
      title={copy.title}
      description={copy.description}
      confirmLabel={copy.confirmLabel}
      danger={copy.danger}
      pending={pending}
      confirmDisabled={needsReason && touched && !validation.ok}
      onConfirm={confirm}
      onCancel={onCancel}
    >
      {needsReason ? (
        <label className="field confirm-dialog__reason" htmlFor={fieldId}>
          <span>{target.action === "refund" ? "退款原因" : "取消原因"}（必填）</span>
          <textarea
            id={fieldId}
            rows={3}
            required
            maxLength={REASON_MAX_LENGTH * 2}
            value={reason}
            disabled={pending}
            aria-invalid={showError}
            aria-describedby={`${fieldId}-hint`}
            placeholder={target.action === "refund" ? "例如：顾客退货，已现金退回" : "例如：顾客改变主意"}
            onChange={(event) => setReason(event.target.value)}
            onBlur={() => setTouched(true)}
          />
          <small id={`${fieldId}-hint`} className={showError ? "confirm-dialog__error" : "field-hint"} role={showError ? "alert" : undefined}>
            {showError && !validation.ok ? validation.error : `请写明原因，便于核对。${REASON_MIN_LENGTH} 到 ${REASON_MAX_LENGTH} 字。`}
          </small>
        </label>
      ) : null}
    </ConfirmDialog>
  );
}
