"use client";

import { useEffect, useRef, type ReactNode } from "react";

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

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") onCancel(); };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <dialog ref={dialogRef} className="confirm-dialog" onCancel={onCancel}>
      <div className="confirm-dialog__panel">
        <h2>{title}</h2>
        <div className="confirm-dialog__description">{description}</div>
        <div className="confirm-dialog__actions">
          <button type="button" className="button button--outline" disabled={pending} onClick={onCancel}>{cancelLabel}</button>
          <button type="button" className={danger ? "button button--danger" : "button button--dark"} disabled={pending} onClick={onConfirm}>
            {pending ? "处理中…" : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  );
}
