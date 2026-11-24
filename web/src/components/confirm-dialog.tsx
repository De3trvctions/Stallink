"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  description: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  pending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

// 统一的危险操作确认弹窗：后果文案由调用方写明，避免二次点击才知道影响。
export function ConfirmDialog({ open, title, description, confirmLabel = "确认", cancelLabel = "取消", danger = false, pending = false, onConfirm, onCancel }: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      className="confirm-dialog"
      aria-labelledby={titleId}
      onCancel={(event) => { if (pending) event.preventDefault(); }}
      onClose={() => { if (open) onCancel(); }}
    >
      <div className="confirm-dialog__panel">
        <h1 id={titleId}>{title}</h1>
        <div className="confirm-dialog__description">{description}</div>
        <div className="confirm-dialog__actions">
          <button type="button" className="button button--outline" disabled={pending} onClick={onCancel} {...(danger ? { autofocus: "" } as object : {})}>{cancelLabel}</button>
          <button type="button" className={danger ? "button button--danger" : "button button--dark"} disabled={pending} onClick={onConfirm} {...(!danger ? { autofocus: "" } as object : {})}>
            {pending ? "处理中…" : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  );
}
