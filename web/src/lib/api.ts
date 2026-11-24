import type { PriceTier } from "@/lib/pricing";
import { hasValidEntityIds, isEntityId } from "@/lib/entity-id";
import { appearancePath, isStorefrontTheme, type StorefrontTheme } from "@/lib/storefront-theme";

export function apiOrigin() {
  return process.env.API_ORIGIN ?? "http://127.0.0.1:8080";
}

export async function getStorefrontTheme(): Promise<StorefrontTheme> {
  try {
    const res = await fetch(`${apiOrigin()}${appearancePath}`, { cache: "no-store", signal: AbortSignal.timeout(2500) });
    if (res.ok) {
      const data = await res.json();
      if (isStorefrontTheme(data.theme)) return data.theme;
    }
  } catch {
    // Keep the site available if the appearance service is temporarily offline.
  }
  return "classic";
}

export type RequestCategory =
  | "ok"
  | "network"
  | "timeout"
  | "unauthorized"
  | "forbidden"
  | "client"
  | "server";

export type RequestResult = {
  ok: boolean;
  status: number;
  data: Record<string, unknown>;
  category: RequestCategory;
  errorCode: string;
};

const REQUEST_TIMEOUT_MS = 15_000;

function categoryForStatus(status: number): Exclude<RequestCategory, "ok" | "network" | "timeout"> {
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status >= 500) return "server";
  return "client";
}

function classifyError(error: unknown): Pick<RequestResult, "category" | "status" | "errorCode"> {
  if (error instanceof DOMException && error.name === "AbortError") {
    return { category: "timeout", status: 0, errorCode: "timeout" };
  }
  return { category: "network", status: 0, errorCode: "network" };
}

async function readResult(response: Response): Promise<Record<string, unknown>> {
  const text = await response.text();
  if (!text) return {};
  try {
    return JSON.parse(text) as Record<string, unknown>;
  } catch {
    // A gateway HTML error page or an empty body must not be swallowed into {}.
    return { error: response.ok ? "bad_response" : "internal" };
  }
}

async function requestJSON(
  path: string,
  method: string,
  body: unknown,
  extraHeaders?: HeadersInit,
): Promise<RequestResult> {
  const headers = new Headers(extraHeaders);
  if (body !== undefined) headers.set("Content-Type", "application/json");
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const res = await fetch(path, {
      method,
      credentials: "include",
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    });
    const data = await readResult(res);
    if (!hasValidEntityIds(data)) {
      return { ok: false, status: 502, data: { error: "invalid_entity_id" }, category: "server", errorCode: "invalid_entity_id" };
    }
    const errorCode = String(data.error ?? "");
    return {
      ok: res.ok,
      status: res.status,
      data,
      category: res.ok ? "ok" : categoryForStatus(res.status),
      errorCode,
    };
  } catch (error) {
    const classified = classifyError(error);
    return { ok: false, data: {}, ...classified };
  } finally {
    window.clearTimeout(timer);
  }
}

async function requestFormData(
  path: string,
  method: string,
  form: FormData,
  extraHeaders?: HeadersInit,
): Promise<RequestResult> {
  const headers = new Headers(extraHeaders);
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const res = await fetch(path, {
      method,
      credentials: "include",
      headers,
      body: form,
      signal: controller.signal,
    });
    const data = await readResult(res);
    const errorCode = String(data.error ?? "");
    return {
      ok: res.ok,
      status: res.status,
      data,
      category: res.ok ? "ok" : categoryForStatus(res.status),
      errorCode,
    };
  } catch (error) {
    const classified = classifyError(error);
    return { ok: false, data: {}, ...classified };
  } finally {
    window.clearTimeout(timer);
  }
}

export function browserJSON(path: string, method = "GET", body?: unknown, extraHeaders?: HeadersInit) {
  return requestJSON(path, method, body, extraHeaders);
}

export function browserFormData(path: string, method: string, form: FormData, extraHeaders?: HeadersInit) {
  return requestFormData(path, method, form, extraHeaders);
}

export type StaffOrderLine = { goodName: string; qty: number; unitPriceMinor: number; lineTotalMinor: number };

export type StaffOrder = {
  id: string;
  number: string;
  status: string;
  boothName: string;
  currency: string;
  priceLabel: string;
  totalMinor: number;
  payCodeName: string | null;
  stateVersion: number;
  createdAt: string;
  reservationExpiresAt: string | null;
  refundedAt: string | null;
  cashReceivedMinor: number | null;
  cashChangeMinor: number | null;
  openedByLogin: string | null;
  openedByRole: string | null;
  paidByLogin: string | null;
  paidByRole: string | null;
  eventId: string;
  eventName: string;
  lines: StaffOrderLine[];
};

export type Good = {
  listingId: string;
  goodId: string;
  eventId: string;
  eventName: string;
  name: string;
  description?: string;
  priceLabel: string;
  currency: string;
  unitPriceMinor: number;
  priceTiers: PriceTier[];
  remaining: number;
  soldOut: boolean;
  imageURL: string | null;
  images: GoodImage[];
  shopSlug?: string;
  shopName?: string;
};

export type GoodImage = { id: string; url: string; sortOrder: number };

export type BundleComponent = {
  name: string;
  quantity: number;
  unitPriceMinor: number;
  images: GoodImage[];
};

export type BundleOffer = {
  bundleListingId: string;
  shopSlug?: string;
  eventId: string;
  eventName: string;
  name: string;
  description: string;
  currency: string;
  unitPriceMinor: number;
  priceLabel: string;
  priceTiers: PriceTier[];
  remaining: number;
  imageURL: string | null;
  images: GoodImage[];
  components: BundleComponent[];
};

export type BoothEvent = {
  id: string;
  name: string;
  startsOn: string;
  endsOn: string;
  city: string;
  venue: string;
  boothNumber: string;
  currency: string;
  status: string;
};

export type Booth = {
  id: string;
  name: string;
  notice: string;
  slug: string;
  events: { scheduled: BoothEvent[]; live: BoothEvent[]; ended: BoothEvent[] };
  featuredEventId?: string;
  goods: Good[];
  bundles: BundleOffer[];
};

async function readBooth(path: string): Promise<Booth | null> {
  const res = await fetch(`${apiOrigin()}${path}`, { cache: "no-store" });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("booth request failed");
  }
  const data: unknown = await res.json();
  if (!hasValidEntityIds(data)) {
    throw new Error("booth response contains an invalid entity ID");
  }
  return data as Booth;
}

export function getLanding() {
  return readBooth("/api/public/landing");
}

export type CatalogResult = { goods: Good[]; total: number };

export async function getCatalog(query = "", sort = ""): Promise<CatalogResult> {
  const params = new URLSearchParams();
  if (query.trim()) params.set("q", query.trim());
  if (sort) params.set("sort", sort);
  const qs = params.toString();
  const res = await fetch(`${apiOrigin()}/api/public/catalog${qs ? `?${qs}` : ""}`, { cache: "no-store" });
  if (res.status === 404) return { goods: [], total: 0 };
  if (!res.ok) throw new Error("catalog request failed");
  const data: unknown = await res.json();
  if (!hasValidEntityIds(data)) throw new Error("catalog response contains an invalid entity ID");
  return data as CatalogResult;
}

export function getShop(slug: string) {
  return readBooth(`/api/public/shops/${encodeURIComponent(slug)}`);
}

export async function getEvent(slug: string, eventId: string) {
  if (!isEntityId(eventId)) return null;
  const res = await fetch(`${apiOrigin()}/api/public/shops/${encodeURIComponent(slug)}/events/${eventId}`, {
    cache: "no-store",
  });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("event request failed");
  }
  const data: unknown = await res.json();
  if (!hasValidEntityIds(data)) {
    throw new Error("event response contains an invalid entity ID");
  }
  return data as { event: BoothEvent; goods: Good[]; bundles: BundleOffer[] };
}

export async function getListing(slug: string, listingId: string) {
  if (!isEntityId(listingId)) return null;
  const res = await fetch(`${apiOrigin()}/api/public/shops/${encodeURIComponent(slug)}/listings/${listingId}`, {
    cache: "no-store",
  });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("listing request failed");
  }
  const data: unknown = await res.json();
  if (!hasValidEntityIds(data)) {
    throw new Error("listing response contains an invalid entity ID");
  }
  return data as Good;
}

export async function getBundle(slug: string, bundleListingId: string) {
  if (!isEntityId(bundleListingId)) return null;
  const res = await fetch(`${apiOrigin()}/api/public/shops/${encodeURIComponent(slug)}/bundles/${bundleListingId}`, {
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error("bundle request failed");
  const data: unknown = await res.json();
  if (!hasValidEntityIds(data)) {
    throw new Error("bundle response contains an invalid entity ID");
  }
  return data as BundleOffer;
}
