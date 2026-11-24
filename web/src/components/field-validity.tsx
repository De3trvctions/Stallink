"use client";

import { useEffect } from "react";

function isField(value: EventTarget | null): value is HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement {
  return value instanceof HTMLInputElement || value instanceof HTMLTextAreaElement || value instanceof HTMLSelectElement;
}

// Keeps aria-invalid aligned with :user-invalid, including fields mounted after navigation.
export function FieldValidity() {
  useEffect(() => {
    const flagged = new WeakSet<Element>();
    const announced = new WeakSet<Element>();
    let announcedField: Element | null = null;
    let announceTimer = 0;

    const announcer = document.createElement("div");
    announcer.setAttribute("aria-live", "polite");
    announcer.className = "sr-only";
    document.body.append(announcer);

    const linkError = (el: HTMLElement, shown: boolean) => {
      const errorId = el.dataset.error;
      if (!errorId) return;
      const ids = (el.getAttribute("aria-describedby") || "").split(/\s+/).filter((id) => id && id !== errorId);
      if (shown) ids.push(errorId);
      if (ids.length) el.setAttribute("aria-describedby", ids.join(" "));
      else el.removeAttribute("aria-describedby");
    };

    const spokenText = (node: HTMLElement) => {
      const copy = node.cloneNode(true) as HTMLElement;
      copy.querySelectorAll("[aria-hidden='true'], .field__validity").forEach((item) => item.remove());
      copy.querySelectorAll("[role='img'][aria-label]").forEach((item) => {
        item.replaceWith(`${item.getAttribute("aria-label")}: `);
      });
      return (copy.textContent || "").replace(/\s+/g, " ").trim();
    };

    const announceError = (el: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement) => {
      const error = el.dataset.error ? document.getElementById(el.dataset.error) : null;
      if (!error) return;
      const label = el.labels?.[0] ? `${spokenText(el.labels[0])}: ` : "";
      window.clearTimeout(announceTimer);
      announcer.textContent = "";
      announcedField = el;
      announceTimer = window.setTimeout(() => {
        announcer.textContent = label + spokenText(error);
      }, 500);
    };

    const syncAria = (target: EventTarget | null, announce = false) => {
      if (!isField(target)) return;
      const invalid = target.matches(":user-invalid");
      if (invalid) {
        if (announce && !announced.has(target)) {
          announceError(target);
          announced.add(target);
        }
        target.setAttribute("aria-invalid", "true");
        linkError(target, true);
        flagged.add(target);
        return;
      }
      if (flagged.has(target) || target.getAttribute("aria-invalid") !== "true") {
        target.setAttribute("aria-invalid", "false");
        linkError(target, false);
        flagged.delete(target);
        announced.delete(target);
        if (announcedField === target) {
          window.clearTimeout(announceTimer);
          announcer.textContent = "";
        }
      }
    };

    const syncTree = (root: ParentNode) => {
      root.querySelectorAll("input, select, textarea").forEach((el) => syncAria(el));
    };

    syncTree(document);

    const onBlur = (event: Event) => {
      const target = event.target;
      syncAria(target, true);
      // :user-invalid can land after the blur event that committed the value.
      window.setTimeout(() => syncAria(target, true), 0);
    };
    const onInput = (event: Event) => {
      if (isField(event.target) && event.target.hasAttribute("aria-invalid")) syncAria(event.target);
    };
    const onChange = (event: Event) => syncAria(event.target, true);
    const onInvalid = (event: Event) => {
      if (event.target instanceof Element) announced.add(event.target);
      syncAria(event.target);
    };
    const onReset = (event: Event) => {
      if (!(event.target instanceof HTMLFormElement)) return;
      const form = event.target;
      window.setTimeout(() => [...form.elements].forEach((el) => syncAria(el)));
    };

    document.addEventListener("blur", onBlur, true);
    document.addEventListener("input", onInput);
    document.addEventListener("change", onChange);
    document.addEventListener("invalid", onInvalid, true);
    document.addEventListener("reset", onReset, true);

    const observer = new MutationObserver((records) => {
      for (const record of records) {
        record.addedNodes.forEach((node) => {
          if (!(node instanceof Element)) return;
          if (node.matches("input, select, textarea")) syncAria(node);
          syncTree(node);
        });
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });

    return () => {
      observer.disconnect();
      document.removeEventListener("blur", onBlur, true);
      document.removeEventListener("input", onInput);
      document.removeEventListener("change", onChange);
      document.removeEventListener("invalid", onInvalid, true);
      document.removeEventListener("reset", onReset, true);
      window.clearTimeout(announceTimer);
      announcer.remove();
    };
  }, []);

  return null;
}
