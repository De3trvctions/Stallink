"use client";

import { forwardRef, useEffect, useRef, type ReactNode } from "react";

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
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const dialog = closeRef.current?.closest("dialog");
    if (!dialog || "closedBy" in HTMLDialogElement.prototype) return;
    const onClick = (event: MouseEvent) => {
      if (closeDisabled || event.target !== dialog) return;
      const rect = dialog.getBoundingClientRect();
      const inside = rect.top <= event.clientY && event.clientY <= rect.top + rect.height
        && rect.left <= event.clientX && event.clientX <= rect.left + rect.width;
      if (!inside) dialog.close();
    };
    dialog.addEventListener("click", onClick);
    return () => dialog.removeEventListener("click", onClick);
  }, [closeDisabled]);

  return (
    <dialog
      ref={ref}
      className={`goods-modal ${className}`.trim()}
      aria-labelledby={labelledById}
      {...({ closedby: closeDisabled ? "none" : "any" } as object)}
      onClose={onClose}
      onCancel={(event) => { if (closeDisabled) event.preventDefault(); }}
    >
      <div className="goods-modal__panel">
        <header className="goods-modal__heading">
          <div>
            <p className="eyebrow">{kicker}</p>
            <h1 id={labelledById}>{title}</h1>
            {description ? <p>{description}</p> : null}
          </div>
          <button ref={closeRef} className="goods-modal__close" type="button" aria-label="关闭弹窗" disabled={closeDisabled} onClick={() => closeRef.current?.closest("dialog")?.close()}>×</button>
        </header>
        {children}
      </div>
    </dialog>
  );
});

export default ModalDialog;
