"use client";

import { forwardRef, type ReactNode } from "react";

type ModalDialogProps = {
  kicker: string;
  title: string;
  description?: string;
  labelledById: string;
  onClose: () => void;
  closeDisabled?: boolean;
  className?: string;
  children: ReactNode;
};

const ModalDialog = forwardRef<HTMLDialogElement, ModalDialogProps>(function ModalDialog(
  { kicker, title, description, labelledById, onClose, closeDisabled = false, className = "", children },
  ref,
) {
  return (
    <dialog
      ref={ref}
      className={`goods-modal ${className}`.trim()}
      aria-labelledby={labelledById}
      onClose={onClose}
      onCancel={(event) => { if (closeDisabled) event.preventDefault(); }}
      onClick={(event) => { if (event.target === event.currentTarget && !closeDisabled) onClose(); }}
    >
      <div className="goods-modal__panel">
        <header className="goods-modal__heading">
          <div>
            <p className="eyebrow">{kicker}</p>
            <h2 id={labelledById}>{title}</h2>
            {description ? <p>{description}</p> : null}
          </div>
          <button className="goods-modal__close" type="button" aria-label="关闭弹窗" disabled={closeDisabled} onClick={onClose}>×</button>
        </header>
        {children}
      </div>
    </dialog>
  );
});

export default ModalDialog;
