export type CounterDraftScope = {
  accountId: string;
  shopId: string;
  eventId: string;
};

export type CounterDraft<T> = {
  scope: CounterDraftScope;
  value: T;
  updatedAt: number;
};

const prefix = "stallink.counter-draft.v1.";
const orderPrefix = "stallink.pending-order.v1.";

export type PendingCounterOrder = {
  id: string;
  number: string;
  priceLabel: string;
  reservationExpiresAt: string | null;
};

function key(scope: CounterDraftScope) {
  return `${prefix}${encodeURIComponent(scope.accountId)}.${encodeURIComponent(scope.shopId)}.${encodeURIComponent(scope.eventId)}`;
}

function orderKey(scope: CounterDraftScope) {
  return `${orderPrefix}${encodeURIComponent(scope.accountId)}.${encodeURIComponent(scope.shopId)}.${encodeURIComponent(scope.eventId)}`;
}

export function loadPendingCounterOrder(scope: CounterDraftScope): PendingCounterOrder | null {
  try {
    const raw = localStorage.getItem(orderKey(scope));
    if (!raw) return null;
    const value = JSON.parse(raw) as PendingCounterOrder;
    return typeof value.id === "string" && typeof value.number === "string" ? value : null;
  } catch {
    return null;
  }
}

export function savePendingCounterOrder(scope: CounterDraftScope, value: PendingCounterOrder) {
  try {
    localStorage.setItem(orderKey(scope), JSON.stringify(value));
  } catch {
    // The server-side checkout ID and idempotency key remain authoritative.
  }
}

export function clearPendingCounterOrder(scope: CounterDraftScope) {
  try {
    localStorage.removeItem(orderKey(scope));
  } catch {
    // The current in-memory order view remains authoritative.
  }
}

export function loadCounterDraft<T>(scope: CounterDraftScope): CounterDraft<T> | null {
  try {
    const raw = localStorage.getItem(key(scope));
    return raw ? JSON.parse(raw) as CounterDraft<T> : null;
  } catch {
    return null;
  }
}

export function saveCounterDraft<T>(scope: CounterDraftScope, value: T) {
  try {
    localStorage.setItem(key(scope), JSON.stringify({ scope, value, updatedAt: Date.now() }));
  } catch {
    // A full or blocked local storage must not block in-memory checkout.
  }
}

export function clearCounterDraft(scope: CounterDraftScope) {
  try {
    localStorage.removeItem(key(scope));
  } catch {
    // The draft is already absent from the current in-memory view.
  }
}

export function clearCounterDraftsForAccount(accountId: string) {
  try {
    const accountPrefix = `${prefix}${encodeURIComponent(accountId)}.`;
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const entry = localStorage.key(i);
      if (entry?.startsWith(accountPrefix)) localStorage.removeItem(entry);
    }
  } catch {
    // Storage can be unavailable in private or restricted browser contexts.
  }
}
