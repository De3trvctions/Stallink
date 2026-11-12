import type { PriceTier } from "@/lib/pricing";
import { hasValidEntityIds, isEntityId } from "@/lib/entity-id";

export function apiOrigin() {
  return process.env.API_ORIGIN ?? "http://127.0.0.1:8080";
}

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
  imageURL: string | null;
  images: GoodImage[];
};

export type GoodImage = { id: string; url: string; sortOrder: number };

export type BundleComponent = {
  goodId: string;
  listingId: string;
  name: string;
  quantity: number;
  unitPriceMinor: number;
  images: GoodImage[];
};

export type BundleOffer = {
  bundleListingId: string;
  bundleId: string;
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

export async function browserJSON(path: string, method = "GET", body?: unknown, extraHeaders?: HeadersInit) {
  const headers = new Headers(extraHeaders);
  if (body !== undefined) headers.set("Content-Type", "application/json");
  const res = await fetch(path, {
    method,
    credentials: "include",
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data: unknown = await res.json().catch(() => ({}));
  if (!hasValidEntityIds(data)) {
    return { ok: false, status: 502, data: { error: "invalid_entity_id" } };
  }
  return { ok: res.ok, status: res.status, data: data as Record<string, unknown> };
}
