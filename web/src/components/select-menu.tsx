"use client";

import { useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";

export type SelectOption = { value: string; label: string; disabled?: boolean };

type SelectMenuProps = {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  id?: string;
  ariaLabel?: string;
  placeholder?: string;
  emptyLabel?: string;
  disabled?: boolean;
  required?: boolean;
  className?: string;
  compact?: boolean;
};

type MenuPosition = {
  left: number;
  width: number;
  maxHeight: number;
  top?: number;
  bottom?: number;
};

function nextEnabledIndex(options: SelectOption[], current: number, step: 1 | -1) {
  if (!options.length) return -1;
  let index = current;
  for (let count = 0; count < options.length; count += 1) {
    index = (index + step + options.length) % options.length;
    if (!options[index]?.disabled) return index;
  }
  return current;
}

const SELECT_OPEN_EVENT = "stallink:select-menu-open";

export default function SelectMenu({
  value,
  options,
  onChange,
  id,
  ariaLabel,
  placeholder = "请选择",
  emptyLabel = "没有可选项目",
  disabled = false,
  required = false,
  className = "",
  compact = false,
}: SelectMenuProps) {
  const generatedId = useId().replace(/:/g, "");
  const triggerId = id ?? `select-${generatedId}`;
  const listId = `${triggerId}-listbox`;
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);
  const [position, setPosition] = useState<MenuPosition | null>(null);

  const selectedIndex = options.findIndex((option) => option.value === value);
  const selected = selectedIndex >= 0 ? options[selectedIndex] : null;

  function updatePosition() {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const rect = trigger.getBoundingClientRect();
    const dialogRect = trigger.closest("dialog")?.getBoundingClientRect();
    const bounds = dialogRect
      ? { top: dialogRect.top, bottom: dialogRect.bottom, left: dialogRect.left, right: dialogRect.right }
      : { top: 0, bottom: window.innerHeight, left: 0, right: window.innerWidth };
    const gap = 6;
    const edge = 8;
    const targetWidth = compact ? 210 : Math.min(Math.max(rect.width, 220), 420);
    const width = Math.min(targetWidth, bounds.right - bounds.left - edge * 2);
    const left = Math.min(Math.max(bounds.left + edge, rect.left), bounds.right - width - edge);
    const below = bounds.bottom - rect.bottom - gap - edge;
    const above = rect.top - bounds.top - gap - edge;
    const placeAbove = below < 190 && above > below;
    const maxHeight = Math.max(120, Math.min(300, placeAbove ? above : below));
    setPosition(placeAbove
      ? { left, width, maxHeight, bottom: window.innerHeight - rect.top + gap }
      : { left, width, maxHeight, top: rect.bottom + gap });
  }

  function openMenu() {
    if (disabled) return;
    const trigger = triggerRef.current;
    setPortalTarget(trigger?.closest("dialog") ?? document.body);
    setActiveIndex(selectedIndex >= 0 ? selectedIndex : options.findIndex((option) => !option.disabled));
    document.dispatchEvent(new CustomEvent<string>(SELECT_OPEN_EVENT, { detail: listId }));
    updatePosition();
    setOpen(true);
  }

  function closeMenu() {
    setOpen(false);
    setActiveIndex(-1);
  }

  function choose(index: number) {
    const option = options[index];
    if (!option || option.disabled) return;
    onChange(option.value);
    closeMenu();
    triggerRef.current?.focus();
  }

  useLayoutEffect(() => {
    if (!open) return;
    updatePosition();
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const closeOther = (event: Event) => {
      if ((event as CustomEvent<string>).detail !== listId) closeMenu();
    };
    document.addEventListener(SELECT_OPEN_EVENT, closeOther);
    return () => document.removeEventListener(SELECT_OPEN_EVENT, closeOther);
  }, [listId]);

  useEffect(() => {
    if (!open || activeIndex < 0) return;
    const option = document.getElementById(`${listId}-option-${activeIndex}`);
    const panel = panelRef.current;
    if (!option || !panel) return;
    const top = option.offsetTop;
    const bottom = top + option.offsetHeight;
    if (top < panel.scrollTop) panel.scrollTop = top;
    else if (bottom > panel.scrollTop + panel.clientHeight) panel.scrollTop = bottom - panel.clientHeight;
  }, [activeIndex, listId, open]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node;
      if (triggerRef.current?.contains(target) || panelRef.current?.contains(target)) return;
      closeMenu();
    };
    const onScroll = (event: Event) => {
      if (panelRef.current?.contains(event.target as Node)) return;
      closeMenu();
    };
    const onResize = () => updatePosition();
    document.addEventListener("pointerdown", onPointerDown, true);
    window.addEventListener("scroll", onScroll, true);
    window.addEventListener("resize", onResize);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown, true);
      window.removeEventListener("scroll", onScroll, true);
      window.removeEventListener("resize", onResize);
    };
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  function handleKeyDown(event: React.KeyboardEvent<HTMLButtonElement>) {
    if (disabled) return;
    if (!open) {
      if (event.key === "ArrowDown" || event.key === "ArrowUp" || event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        openMenu();
      }
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      closeMenu();
      return;
    }
    if (event.key === "Tab") {
      closeMenu();
      return;
    }
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((current) => nextEnabledIndex(options, current < 0 ? (event.key === "ArrowDown" ? -1 : 0) : current, event.key === "ArrowDown" ? 1 : -1));
      return;
    }
    if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      const start = event.key === "Home" ? -1 : 0;
      setActiveIndex(nextEnabledIndex(options, start, event.key === "Home" ? 1 : -1));
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      choose(activeIndex);
    }
  }

  const menuStyle: CSSProperties | undefined = position ? {
    left: position.left,
    width: position.width,
    maxHeight: position.maxHeight,
    ...(position.top !== undefined ? { top: position.top } : { bottom: position.bottom }),
  } : undefined;

  return (
    <div className={`select-menu${compact ? " select-menu--compact" : ""}${className ? ` ${className}` : ""}`}>
      <button
        ref={triggerRef}
        id={triggerId}
        type="button"
        className="select-menu__trigger"
        role="combobox"
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open && activeIndex >= 0 ? `${listId}-option-${activeIndex}` : undefined}
        aria-required={required || undefined}
        aria-disabled={disabled || undefined}
        disabled={disabled}
        data-empty={!selected}
        onClick={() => { if (open) closeMenu(); else openMenu(); }}
        onKeyDown={handleKeyDown}
        onBlur={(event) => {
          if (!open) return;
          const next = event.relatedTarget as Node | null;
          if (next && panelRef.current?.contains(next)) return;
          closeMenu();
        }}
      >
        <span className="select-menu__value" title={selected?.label ?? placeholder}>{selected?.label ?? placeholder}</span>
        <svg className="select-menu__chevron" viewBox="0 0 16 16" aria-hidden="true">
          <path d="M3.5 6 8 10.5 12.5 6" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
        </svg>
      </button>
      {open && portalTarget ? createPortal(
        <div
          ref={panelRef}
          id={listId}
          className="select-menu__panel"
          role="listbox"
          aria-label={ariaLabel ?? "选项"}
          style={menuStyle}
        >
          {options.length === 0 ? <p className="select-menu__empty">{emptyLabel}</p> : options.map((option, index) => (
            <button
              key={option.value}
              id={`${listId}-option-${index}`}
              type="button"
              role="option"
              aria-selected={option.value === value}
              aria-disabled={option.disabled || undefined}
              data-active={index === activeIndex || undefined}
              tabIndex={-1}
              className="select-menu__option"
              onMouseEnter={() => { if (!option.disabled) setActiveIndex(index); }}
              onClick={() => choose(index)}
            >
              <span>{option.label}</span>
              {option.value === value ? (
                <svg className="select-menu__check" viewBox="0 0 16 16" aria-hidden="true">
                  <path d="M3.2 8.4 6.3 11.2 12.8 4.7" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
                </svg>
              ) : null}
            </button>
          ))}
        </div>,
        portalTarget,
      ) : null}
    </div>
  );
}
