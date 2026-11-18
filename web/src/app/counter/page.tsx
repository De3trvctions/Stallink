"use client";

import { SafeImage } from "@/components/safe-image";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { browserJSON, type RequestResult } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney, majorToMinor, type PriceTier } from "@/lib/pricing";
import { clearCounterDraft, clearPendingCounterOrder, loadCounterDraft, loadPendingCounterOrder, saveCounterDraft, savePendingCounterOrder, type CounterDraftScope } from "@/lib/counter-draft";
import { clearPendingOrderCommand, newClientOrderId, pendingOrderCommand, sendOrderCommand } from "@/lib/order-command";
import { messageClass, useMessage } from "@/lib/message";
import { WorkspaceShell } from "@/components/workspace-shell";
import SelectMenu from "@/components/select-menu";

type EventRow = { id: string; name: string; boothNumber: string; currency: string; status?: string; endsOn?: string };
type Listing = {
  listingId: string; goodId: string; name: string; description: string; currency: string;
  unitPriceMinor: number; priceLabel: string; priceTiers: PriceTier[]; remaining: number;
  isListed: boolean; reviewStatus: string; reviewNote: string; imageURL: string | null;
};
type BundleComponent = { goodId: string; listingId: string; name: string; quantity: number };
type Bundle = {
  bundleListingId: string; bundleId: string; name: string; description: string; currency: string;
  unitPriceMinor: number; priceLabel: string; priceTiers: PriceTier[]; remaining: number;
  isListed: boolean; reviewStatus: string; reviewNote: string; components: BundleComponent[];
  imageURL: string | null;
};
type PayCode = { id: string; name: string; kind: "qr" | "cash"; isDefault: boolean; currency: string | null; imageURL: string };
type QuoteGroup = { kind: string; name: string; applications: number; totalMinor: number; lines: { name: string; quantity: number }[] };
type Quote = {
  id: string; eventId: string; cartVersion: number; catalogRevision: number; expiresAt: string;
  totalMinor: number; baseMinor: number; savingsMinor: number; itemCount: number; groups: QuoteGroup[];
};
type Me = { id: string; shopId: string };
type CreateOrderBody = { clientOrderId: string; quoteId: string; expectedTotalMinor: number; boothName: string };
type CartDraft = { qty: Record<string, number>; bundleQty: Record<string, number>; boothName: string };
type CatalogFilter = "all" | "good" | "bundle";
type PaidReceipt = { id: string; number: string; totalMinor: number; currency: string };

function eventIsClosed(event?: EventRow): boolean {
  if (!event) return false;
  if (event.status === "ended") return true;
  if (event.endsOn) {
    const end = new Date(`${event.endsOn}T23:59:59`);
    if (!Number.isNaN(end.getTime()) && end.getTime() < Date.now()) return true;
  }
  return false;
}

function availability(item: Listing, singleLimit = item.remaining, bundleUsed = 0) {
  if (item.reviewStatus === "pending") return "等待平台审核";
  if (item.reviewStatus === "rejected") return `审核未通过${item.reviewNote ? `：${item.reviewNote}` : ""}`;
  if (!item.isListed) return "本场已下架";
  if (item.remaining <= 0) return "缺货 · 本场已售完";
  if (singleLimit <= 0 && bundleUsed > 0) return "套餐已占满库存 · 暂不可单买";
  if (bundleUsed > 0) return `可售 · 单买最多 ${singleLimit} 件（套餐占 ${bundleUsed} 件）`;
  return `可售 · 本场剩 ${item.remaining} 件`;
}

function bundleAvailability(bundle: Bundle, limit = bundle.remaining) {
  if (bundle.reviewStatus === "pending") return "套餐等待平台审核";
  if (bundle.reviewStatus === "rejected") return `套餐审核未通过${bundle.reviewNote ? `：${bundle.reviewNote}` : ""}`;
  if (!bundle.isListed) return "本场已下架";
  if (bundle.remaining <= 0) return "套餐缺货 · 已售完";
  if (limit <= 0) return "套餐内商品已被单品占满";
  return `固定套餐 · 最多 ${limit} 套`;
}

function unavailableReason(item: Listing, closed: boolean) {
  if (closed) return "活动已结束，暂时不能加入订单。";
  if (item.reviewStatus === "pending") return `「${item.name}」正在等待平台审核，暂时不能加入订单。`;
  if (item.reviewStatus === "rejected") return `「${item.name}」审核未通过${item.reviewNote ? `：${item.reviewNote}` : ""}，暂时不能加入订单。`;
  if (!item.isListed) return `「${item.name}」本场已下架，暂时不能加入订单。`;
  if (item.remaining <= 0) return `「${item.name}」本场已售完，缺货，暂时不能加入订单。`;
  return "";
}

function bundleUnavailableReason(bundle: Bundle, closed: boolean) {
  if (closed) return "活动已结束，暂时不能加入订单。";
  if (bundle.reviewStatus === "pending") return `「${bundle.name}」套餐正在等待平台审核，暂时不能加入订单。`;
  if (bundle.reviewStatus === "rejected") return `「${bundle.name}」套餐审核未通过${bundle.reviewNote ? `：${bundle.reviewNote}` : ""}，暂时不能加入订单。`;
  if (!bundle.isListed) return `「${bundle.name}」本场已下架，暂时不能加入订单。`;
  if (bundle.remaining <= 0) return `「${bundle.name}」套餐已售完，套餐内商品缺货，暂时不能加入订单。`;
  return "";
}

function bundleGroup(kind: string) {
  if (kind === "bundle") return "套餐";
  if (kind === "bundle_tier") return "套餐阶梯";
  if (kind === "single_tier") return "数量价";
  return "单品";
}

function toMinor(amount: string): number | null {
  return majorToMinor(amount);
}

export default function CounterPage() {
  const router = useRouter();
  const [events, setEvents] = useState<EventRow[]>([]);
  const [eventId, setEventId] = useState("");
  const [listings, setListings] = useState<Listing[]>([]);
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [catalogSearch, setCatalogSearch] = useState("");
  const [catalogFilter, setCatalogFilter] = useState<CatalogFilter>("all");
  const [qty, setQty] = useState<Record<string, number>>({});
  const [bundleQty, setBundleQty] = useState<Record<string, number>>({});
  const [boothName, setBoothName] = useState("");
  const [me, setMe] = useState<Me | null>(null);
  const [orderId, setOrderId] = useState("");
  const [resumedOrder, setResumedOrder] = useState(false);
  const [orderNumber, setOrderNumber] = useState("");
  const [orderPriceLabel, setOrderPriceLabel] = useState("");
  const [orderTotalMinor, setOrderTotalMinor] = useState(0);
  const [orderCurrency, setOrderCurrency] = useState("MYR");
  const [reservationExpiresAt, setReservationExpiresAt] = useState("");
  const [codes, setCodes] = useState<PayCode[]>([]);
  const [payCodeId, setPayCodeId] = useState("");
  const [quote, setQuote] = useState<Quote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [quoteError, setQuoteError] = useState("");
  const { message, setMessage } = useMessage();
  const [catalogNotice, setCatalogNotice] = useState("");
  const [catalogNoticeVersion, setCatalogNoticeVersion] = useState(0);
  const [initError, setInitError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [cartVersion, setCartVersion] = useState(0);
  const [cartOpen, setCartOpen] = useState(false);
  const [cashReceived, setCashReceived] = useState("");
  const [paidReceipt, setPaidReceipt] = useState<PaidReceipt | null>(null);
  const [fulfilling, setFulfilling] = useState(false);

  useEffect(() => {
    if (!cartOpen) return;
    const previousBody = document.body.style.overflow;
    const previousRoot = document.documentElement.style.overflow;
    document.body.style.overflow = "hidden";
    document.documentElement.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousBody;
      document.documentElement.style.overflow = previousRoot;
    };
  }, [cartOpen]);

  useEffect(() => {
    if (!catalogNotice) return;
    const timer = window.setTimeout(() => setCatalogNotice(""), 3200);
    return () => window.clearTimeout(timer);
  }, [catalogNotice, catalogNoticeVersion]);

  const [payOpen, setPayOpen] = useState(false);
  const [loadedDraftKey, setLoadedDraftKey] = useState("");
  const quoteRequest = useRef(0);
  const catalogSignature = useRef("");
  const qtyRef = useRef(qty);
  useEffect(() => { qtyRef.current = qty; }, [qty]);
  const bundleQtyRef = useRef(bundleQty);
  useEffect(() => { bundleQtyRef.current = bundleQty; }, [bundleQty]);
  const refreshCatalogRef = useRef<() => void>(() => {});

  const loadInitial = useCallback(async () => {
    try {
      const [eventsRes, meRes, codesRes] = await Promise.all([
        browserJSON("/api/staff/events"),
        browserJSON("/api/auth/me"),
        browserJSON("/api/staff/pay-codes"),
      ]);
      if (eventsRes.category === "unauthorized") { router.replace("/login"); return; }
      const rows = Array.isArray(eventsRes.data) ? eventsRes.data as EventRow[] : [];
      const sorted = [...rows.filter((row) => row.status === "live"), ...rows.filter((row) => row.status !== "live")];
      setEvents(sorted);
      const preferred = sorted.find((row) => row.status === "live") ?? sorted[0];
      if (preferred) setEventId(preferred.id);
      if (meRes.ok) setMe({ id: String(meRes.data.id ?? ""), shopId: String(meRes.data.shopId ?? "") });
      const codeList = Array.isArray(codesRes.data) ? codesRes.data as PayCode[] : [];
      setCodes(codeList);
      setPayCodeId((current) => current || codeList.find((code) => code.isDefault)?.id || codeList[0]?.id || "");
    } catch {
      setInitError("目录暂时无法读取，请检查网络后重试。");
    }
  }, [router]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void loadInitial(); }, 0);
    return () => window.clearTimeout(timer);
  }, [loadInitial]);

  useEffect(() => {
    let active = true;
    if (!eventId) return () => { active = false; };
    void Promise.all([
      browserJSON(`/api/staff/events/${eventId}/listings`),
      browserJSON(`/api/staff/events/${eventId}/bundles`),
    ]).then(([listingRes, bundleRes]) => {
      if (!active) return;
      const nextListings = Array.isArray(listingRes.data) ? listingRes.data as Listing[] : [];
      const nextBundles = Array.isArray(bundleRes.data) ? bundleRes.data as Bundle[] : [];
      catalogSignature.current = JSON.stringify([nextListings, nextBundles]);
      setListings(nextListings);
      setBundles(nextBundles);
    });
    return () => { active = false; };
  }, [eventId]);

  useEffect(() => {
    if (!me?.id || !me.shopId || !eventId) return;
    const scope: CounterDraftScope = { accountId: me.id, shopId: me.shopId, eventId };
    const draftKey = JSON.stringify(scope);
    const timer = window.setTimeout(() => {
      const saved = loadCounterDraft<CartDraft>(scope);
      setQty(saved?.value.qty ?? {});
      setBundleQty(saved?.value.bundleQty ?? {});
      setBoothName(saved?.value.boothName ?? "");
      setLoadedDraftKey(draftKey);
      setCartVersion((version) => version + 1);
      refreshCatalogRef.current();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [eventId, me]);

  useEffect(() => {
    if (!me?.id || !me.shopId || events.length === 0) return;
    const resumeId = new URLSearchParams(window.location.search).get("resume");
    if (!resumeId || !/^[1-9][0-9]*$/.test(resumeId)) return;
    let active = true;
    void browserJSON(`/api/staff/orders/${resumeId}`).then((res) => {
      if (!active) return;
      if (!res.ok || res.data.status !== "pending") {
        setMessage({ kind: "error", text: res.ok ? "该订单已不是待付款状态，请刷新订单列表。" : describeRequest(res) });
        return;
      }
      const orderEventId = String(res.data.eventId ?? "");
      if (!events.some((event) => event.id === orderEventId)) {
        setMessage({ kind: "error", text: "未找到订单所属活动，请刷新后重试。" });
        return;
      }
      setEventId(orderEventId);
      setResumedOrder(true);
      setOrderId(resumeId);
      setOrderNumber(String(res.data.number));
      const total = Number(res.data.totalMinor ?? 0);
      const orderCurrency = String(res.data.currency ?? "MYR");
      setOrderCurrency(orderCurrency);
      setOrderTotalMinor(total);
      setOrderPriceLabel(String(res.data.priceLabel ?? formatMoney(orderCurrency, total)));
      setReservationExpiresAt(res.data.reservationExpiresAt ? String(res.data.reservationExpiresAt) : "");
      const pending = pendingOrderCommand<{ payCodeId: string; cashReceivedMinor?: number }>(`pay:${me.id}:${me.shopId}:${resumeId}`);
      if (pending) {
        setPayCodeId(pending.body.payCodeId);
        setCashReceived(pending.body.cashReceivedMinor !== undefined ? (pending.body.cashReceivedMinor / 100).toFixed(2) : "");
      }
      setPayOpen(true);
    });
    return () => { active = false; };
  }, [events, me, setMessage]);

  useEffect(() => {
    if (!me?.id || !me.shopId || !eventId || new URLSearchParams(window.location.search).has("resume")) return;
    const scope: CounterDraftScope = { accountId: me.id, shopId: me.shopId, eventId };
    const saved = loadPendingCounterOrder(scope);
    if (!saved) return;
    let active = true;
    void browserJSON(`/api/staff/orders/${saved.id}`).then(async (res) => {
      if (!active) return;
      if (!res.ok) {
        if (res.status === 404) clearPendingCounterOrder(scope);
        return;
      }
      if (String(res.data.status) !== "pending") {
        clearPendingCounterOrder(scope);
        return;
      }
      setOrderId(saved.id);
      setPayOpen(true);
      setOrderNumber(String(res.data.number ?? saved.number));
      setOrderPriceLabel(String(res.data.priceLabel ?? saved.priceLabel));
      setOrderTotalMinor(Number(res.data.totalMinor ?? 0));
      setOrderCurrency(String(res.data.currency ?? "MYR"));
      setReservationExpiresAt(res.data.reservationExpiresAt ? String(res.data.reservationExpiresAt) : "");
      const codesRes = await browserJSON("/api/staff/pay-codes");
      if (!active) return;
      const list = Array.isArray(codesRes.data) ? codesRes.data as PayCode[] : [];
      setCodes(list);
      const payScope = `pay:${me.id}:${me.shopId}:${saved.id}`;
      const savedPay = pendingOrderCommand<{ payCodeId: string; cashReceivedMinor?: number }>(payScope);
      setPayCodeId(savedPay?.body.payCodeId ?? list.find((code) => code.isDefault)?.id ?? list[0]?.id ?? "");
      if (savedPay?.body.cashReceivedMinor !== undefined) setCashReceived((savedPay.body.cashReceivedMinor / 100).toFixed(2));
      setMessage({ kind: "success", text: "已恢复上一笔待付款订单，可继续收款。" });
    }).catch(() => {
      // Keep the checkout marker and retry recovery on the next page load.
    });
    return () => { active = false; };
  }, [eventId, me, setMessage]);

  useEffect(() => {
    if (!me?.id || !me.shopId || !eventId) return;
    const scope: CounterDraftScope = { accountId: me.id, shopId: me.shopId, eventId };
    if (loadedDraftKey !== JSON.stringify(scope)) return;
    if (Object.values(qty).every((count) => count === 0) && Object.values(bundleQty).every((count) => count === 0) && !boothName) {
      clearCounterDraft(scope);
    } else {
      saveCounterDraft(scope, { qty, bundleQty, boothName });
    }
  }, [boothName, bundleQty, eventId, loadedDraftKey, me, qty]);

  function switchEvent(nextId: string) {
    if (new URLSearchParams(window.location.search).has("resume")) window.history.replaceState(null, "", "/counter");
    setResumedOrder(false);
    setEventId(nextId);
    setQty({});
    setBundleQty({});
    setBoothName("");
    setLoadedDraftKey("");
    setListings([]);
    setBundles([]);
    setQuote(null);
    setQuoteError("");
    setOrderId("");
    setPayOpen(false);
    setReservationExpiresAt("");
    setCartVersion((version) => version + 1);
  }

  const liveEvents = events.filter((event) => event.status === "live");
  const selectedEvent = events.find((event) => event.id === eventId);
  const eventClosed = eventIsClosed(selectedEvent);
  const currency = selectedEvent?.currency ?? "MYR";
  const normalizedSearch = catalogSearch.trim().toLocaleLowerCase();
  const visibleListings = useMemo(() => listings.filter((item) =>
    (catalogFilter !== "bundle") && (!normalizedSearch || item.name.toLocaleLowerCase().includes(normalizedSearch))), [catalogFilter, listings, normalizedSearch]);
  const visibleBundles = useMemo(() => bundles.filter((bundle) =>
    (catalogFilter !== "good") && (!normalizedSearch || bundle.name.toLocaleLowerCase().includes(normalizedSearch) || bundle.components.some((item) => item.name.toLocaleLowerCase().includes(normalizedSearch)))), [bundles, catalogFilter, normalizedSearch]);
  const bundleUsageByListing = useMemo(() => {
    const usage: Record<string, number> = {};
    bundles.forEach((bundle) => {
      const count = bundleQty[bundle.bundleListingId] ?? 0;
      if (count <= 0) return;
      bundle.components.forEach((component) => {
        usage[component.listingId] = (usage[component.listingId] ?? 0) + component.quantity * count;
      });
    });
    return usage;
  }, [bundles, bundleQty]);

  function showCatalogNotice(text: string) {
    setCatalogNotice(text);
    setCatalogNoticeVersion((version) => version + 1);
  }

  function maxSingleQty(item: Listing) {
    return Math.max(0, item.remaining - (bundleUsageByListing[item.listingId] ?? 0));
  }

  function maxBundleQty(bundle: Bundle, singles: Record<string, number> = qty) {
    let limit = bundle.remaining;
    const ownCount = bundleQty[bundle.bundleListingId] ?? 0;
    bundle.components.forEach((component) => {
      const listing = listings.find((item) => item.listingId === component.listingId);
      if (!listing) return;
      const singleUsed = singles[component.listingId] ?? 0;
      const otherBundleUsed = (bundleUsageByListing[component.listingId] ?? 0) - component.quantity * ownCount;
      const free = listing.remaining - singleUsed - otherBundleUsed;
      limit = Math.min(limit, Math.floor(free / component.quantity));
    });
    return Math.max(0, limit);
  }

  const refreshCatalog = useCallback(async () => {
    if (!eventId) return;
    const [listingRes, bundleRes] = await Promise.all([
      browserJSON(`/api/staff/events/${eventId}/listings`),
      browserJSON(`/api/staff/events/${eventId}/bundles`),
    ]);
    if (!Array.isArray(listingRes.data) || !Array.isArray(bundleRes.data)) return;
    const nextListings = listingRes.data as Listing[];
    const nextBundles = bundleRes.data as Bundle[];
    const signature = JSON.stringify([nextListings, nextBundles]);
    if (signature === catalogSignature.current) return;
    catalogSignature.current = signature;
    setListings(nextListings);
    setBundles(nextBundles);

    // Reconcile the cart against the fresh catalog. This previously lived in
    // effects watching listings/bundles/qty; keeping it in the catalog sync
    // callback means no setState runs synchronously inside an effect.
    const currentQty = qtyRef.current;
    const currentBundleQty = bundleQtyRef.current;
    const usage: Record<string, number> = {};
    nextBundles.forEach((bundle) => {
      const count = currentBundleQty[bundle.bundleListingId] ?? 0;
      if (count <= 0) return;
      bundle.components.forEach((component) => {
        usage[component.listingId] = (usage[component.listingId] ?? 0) + component.quantity * count;
      });
    });
    const singleOverflow = nextListings.filter((item) => {
      const limit = Math.max(0, item.remaining - (usage[item.listingId] ?? 0));
      return (currentQty[item.listingId] ?? 0) > limit;
    });
    const bundleOverflow = nextBundles.filter((bundle) =>
      (currentBundleQty[bundle.bundleListingId] ?? 0) > Math.max(0, bundle.remaining));
    if (singleOverflow.length) {
      setCatalogNotice(`库存已变化：「${singleOverflow.map((item) => item.name).join("、")}」已按本场剩余调整。`);
      setCatalogNoticeVersion((version) => version + 1);
    }
    if (bundleOverflow.length) {
      setCatalogNotice(`库存已变化：「${bundleOverflow.map((bundle) => bundle.name).join("、")}」已售完，已从购物车移除。`);
      setCatalogNoticeVersion((version) => version + 1);
    }
    if (singleOverflow.length || bundleOverflow.length) {
      setQty((previous) => {
        const next = { ...previous };
        nextListings.forEach((item) => {
          const limit = Math.max(0, item.remaining - (usage[item.listingId] ?? 0));
          if ((next[item.listingId] ?? 0) > limit) next[item.listingId] = limit;
        });
        return next;
      });
      setBundleQty((previous) => {
        const next = { ...previous };
        nextBundles.forEach((bundle) => {
          const limit = Math.max(0, bundle.remaining);
          if ((next[bundle.bundleListingId] ?? 0) > limit) next[bundle.bundleListingId] = limit;
        });
        return next;
      });
      setQuote(null);
      setQuoteError("");
      setCartVersion((version) => version + 1);
    }
  }, [eventId]);
  useEffect(() => { refreshCatalogRef.current = refreshCatalog; }, [refreshCatalog]);

  useEffect(() => {
    if (!eventId) return;
    const sync = () => {
      if (document.visibilityState !== "hidden") void refreshCatalog();
    };
    const timer = window.setInterval(sync, 20000);
    window.addEventListener("focus", sync);
    document.addEventListener("visibilitychange", sync);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("focus", sync);
      document.removeEventListener("visibilitychange", sync);
    };
  }, [eventId, refreshCatalog]);

  const cartLines = useMemo(() => {
    const lines = listings.flatMap((item) => {
      const count = qty[item.listingId] ?? 0;
      return count > 0 ? [{ kind: "single", listingId: item.listingId, qty: count }] : [];
    });
    const packageLines = bundles.flatMap((bundle) => {
      const count = bundleQty[bundle.bundleListingId] ?? 0;
      return count > 0 ? [{ kind: "bundle", bundleListingId: bundle.bundleListingId, qty: count }] : [];
    });
    return [...lines, ...packageLines];
  }, [bundleQty, bundles, listings, qty]);

  useEffect(() => {
    const request = ++quoteRequest.current;
    if (!eventId || cartLines.length === 0) {
      queueMicrotask(() => {
        if (request === quoteRequest.current) {
          setQuoteLoading(false);
          setQuoteError("");
        }
      });
      return;
    }
    const timeout = window.setTimeout(() => {
      setQuoteLoading(true);
      setQuoteError("");
      void browserJSON("/api/staff/order-quotes", "POST", { eventId, cartVersion, lines: cartLines })
        .then((res) => {
          if (request !== quoteRequest.current) return;
          if (!res.ok) {
            setQuote(null);
            if (res.data.error === "not_enough_stock" || res.data.error === "invalid_cart") void refreshCatalog();
            setQuoteError(res.data.error === "not_enough_stock" || res.data.error === "invalid_cart"
              ? "套餐与单品共用库存，请核对所选数量。"
              : res.data.error === "pricing_complexity_exceeded" ? "这组商品优惠规则过于复杂，暂时无法准确报价，请减少品项后重试。"
                : describeRequest(res));
            return;
          }
          const result = res.data as unknown as Quote;
          if (result.cartVersion !== cartVersion) return;
          setQuote(result);
        })
        .catch(() => {
          if (request === quoteRequest.current) {
            setQuote(null);
            setQuoteError("无法连接服务，价格预览暂不可用。");
          }
        })
        .finally(() => {
          if (request === quoteRequest.current) setQuoteLoading(false);
        });
    }, 150);
    return () => window.clearTimeout(timeout);
  }, [cartLines, cartVersion, eventId, refreshCatalog]);

  function reconcileSingleOverflow() {
    let changed = false;
    const next = { ...qty };
    listings.forEach((item) => {
      const limit = Math.max(0, item.remaining - (bundleUsageByListing[item.listingId] ?? 0));
      if ((next[item.listingId] ?? 0) > limit) {
        next[item.listingId] = limit;
        changed = true;
      }
    });
    if (!changed) return { changed: false, nextQty: qty };
    setQty(next);
    setQuote(null);
    setQuoteError("");
    setCartVersion((version) => version + 1);
    return { changed: true, nextQty: next };
  }

  function adjust(item: Listing, amount: number) {
    if (amount > 0) {
      const reason = unavailableReason(item, eventClosed);
      if (reason) { showCatalogNotice(reason); return; }
      const repaired = reconcileSingleOverflow();
      if (repaired.changed) {
        showCatalogNotice("套餐与单品共用库存，已把超出的单品数量调回可用数量。");
        return;
      }
    }
    const limit = maxSingleQty(item);
    const current = qty[item.listingId] ?? 0;
    if (amount > 0 && current >= limit) {
      const bundleUsed = bundleUsageByListing[item.listingId] ?? 0;
      showCatalogNotice(bundleUsed > 0
        ? `「${item.name}」有 ${bundleUsed} 件已被套餐占用，单品最多只能加到 ${limit} 件。`
        : `「${item.name}」本场只剩 ${item.remaining} 件，已经全部加入订单。`);
      return;
    }
    showCatalogNotice("");
    setQty((previous) => ({ ...previous, [item.listingId]: Math.max(0, Math.min(limit, (previous[item.listingId] ?? 0) + amount)) }));
    setQuote(null);
    setQuoteError("");
    setCartVersion((version) => version + 1);
  }

  function adjustBundle(bundle: Bundle, amount: number) {
    if (amount > 0) {
      const reason = bundleUnavailableReason(bundle, eventClosed);
      if (reason) { showCatalogNotice(reason); return; }
      const repaired = reconcileSingleOverflow();
      if (repaired.changed) {
        const limitAfterRepair = maxBundleQty(bundle, repaired.nextQty);
        showCatalogNotice(`套餐与单品共用库存，已把超出的单品调回可用数量；「${bundle.name}」最多 ${limitAfterRepair} 套。`);
        return;
      }
    }
    const limit = maxBundleQty(bundle);
    const current = bundleQty[bundle.bundleListingId] ?? 0;
    if (amount > 0 && current >= limit) {
      showCatalogNotice(limit <= 0
        ? `「${bundle.name}」内商品已被单品占满，暂时不能再加套餐。`
        : `「${bundle.name}」受套餐内商品库存限制，最多只能加到 ${limit} 套。`);
      return;
    }
    showCatalogNotice("");
    setBundleQty((previous) => ({ ...previous, [bundle.bundleListingId]: Math.max(0, Math.min(limit, (previous[bundle.bundleListingId] ?? 0) + amount)) }));
    setQuote(null);
    setQuoteError("");
    setCartVersion((version) => version + 1);
  }

  function clearPendingOrderMarker() {
    setOrderId("");
    setOrderNumber("");
    setOrderPriceLabel("");
    setReservationExpiresAt("");
    setPayOpen(false);
  }

  async function place(event: React.FormEvent) {
    event.preventDefault();
    if (cartLines.length === 0 || !me?.id || !me.shopId) return;
    if (!quote || quoteLoading || quote.cartVersion !== cartVersion) {
      setMessage({ kind: "error", text: "正在更新价格预览，请稍候再生成订单。" });
      return;
    }
    const scope = `create:${me.id}:${me.shopId}:${eventId}`;
    const saved = pendingOrderCommand<CreateOrderBody>(scope);
    const body = saved?.body ?? {
      clientOrderId: newClientOrderId(), quoteId: quote.id, expectedTotalMinor: quote.totalMinor, boothName,
    };
    setSubmitting(true);
    setMessage(null);
    try {
      const res = await sendOrderCommand(scope, "/api/staff/orders", body);
      if (!res.ok) {
        if (["quote_expired", "quote_stale", "quote_invalid", "invalid_cart", "pricing_complexity_exceeded"].includes(String(res.data.error))) {
          clearPendingOrderCommand(scope);
          setQuote(null);
          setCartVersion((version) => version + 1);
        }
        setMessage({ kind: "error", text: describeRequest(res) });
        return;
      }
      const result = res.data;
      const totalMinor = Number(result.totalMinor ?? quote.totalMinor);
      const orderCurrency = String(result.currency ?? currency);
      setOrderId(String(result.id));
      setPayOpen(true);
      setOrderNumber(String(result.number));
      setOrderPriceLabel(String(result.priceLabel ?? formatMoney(orderCurrency, totalMinor)));
      setOrderTotalMinor(totalMinor);
      setOrderCurrency(orderCurrency);
      setReservationExpiresAt(result.reservationExpiresAt ? String(result.reservationExpiresAt) : "");
      savePendingCounterOrder({ accountId: me.id, shopId: me.shopId, eventId }, {
        id: String(result.id), number: String(result.number),
        priceLabel: String(result.priceLabel ?? formatMoney(orderCurrency, totalMinor)),
        reservationExpiresAt: result.reservationExpiresAt ? String(result.reservationExpiresAt) : null,
      });
      // 开单成功但未收款：购物车与草稿保留，便于库存/价格变化后按原组合重开。
      const codesRes = await browserJSON("/api/staff/pay-codes");
      const list = Array.isArray(codesRes.data) ? codesRes.data as PayCode[] : [];
      setCodes(list);
      setPayCodeId(list.find((code) => code.isDefault)?.id ?? list[0]?.id ?? "");
      setCashReceived("");
      setCartOpen(false);
      void refreshCatalog();
      setMessage({ kind: "success", text: `订单 ${String(result.number)} 已生成，待付款。` });
    } catch {
      setMessage({ kind: "error", text: "无法连接服务，请重试同一笔开单以安全恢复结果。" });
    } finally {
      setSubmitting(false);
    }
  }

  function resetAfterFailedPayment(res: RequestResult) {
    // 预留过期 / 库存不足：关闭弹窗、清掉待付单标记，购物车保留并按最新目录重新计价。
    if (me?.id && me.shopId) {
      const scope = { accountId: me.id, shopId: me.shopId, eventId };
      if (loadPendingCounterOrder(scope)?.id === orderId) clearPendingCounterOrder(scope);
    }
    if (resumedOrder) window.history.replaceState(null, "", "/counter");
    setResumedOrder(false);
    clearPendingOrderMarker();
    setQuote(null);
    setCartVersion((version) => version + 1);
    void refreshCatalog();
    setMessage({ kind: "error", text: describeRequest(res) });
  }

  async function confirm() {
    if (!orderId || !me?.id || !me.shopId) return;
    const scope = `pay:${me.id}:${me.shopId}:${orderId}`;
    const saved = pendingOrderCommand<{ payCodeId: string; cashReceivedMinor?: number }>(scope);
    const activeCode = codes.find((code) => code.id === payCodeId);
    if (!saved && !activeCode) return;
    let cashReceivedMinor: number | undefined;
    if (!saved && activeCode?.kind === "cash") {
      const received = toMinor(cashReceived);
      if (received === null) {
        setMessage({ kind: "error", text: "请填写实收现金金额。" });
        return;
      }
      if (received < orderTotalMinor) {
        setMessage({ kind: "error", text: `实收金额不能少于订单总额 ${formatMoney(orderCurrency, orderTotalMinor)}。` });
        return;
      }
      cashReceivedMinor = received;
    }
    setSubmitting(true);
    try {
      const res = await sendOrderCommand(scope, `/api/staff/orders/${orderId}/pay`, {
        payCodeId,
        ...(cashReceivedMinor !== undefined ? { cashReceivedMinor } : {}),
        ...(saved?.body ?? {}),
      });
      if (res.status === 409 && res.data.error === "reservation_expired") {
        resetAfterFailedPayment(res);
        return;
      }
      if (res.status === 409 && res.data.error === "not_enough_stock") {
        resetAfterFailedPayment(res);
        return;
      }
      if (!res.ok) {
        setMessage({ kind: "error", text: describeRequest(res) });
        return;
      }
      // 收款成功：清掉草稿与待付单标记，弹出取货凭证。
      if (me?.id && me.shopId) {
        const scope = { accountId: me.id, shopId: me.shopId, eventId };
        if (loadPendingCounterOrder(scope)?.id === orderId) clearPendingCounterOrder(scope);
        if (!resumedOrder) clearCounterDraft(scope);
      }
      if (!resumedOrder) {
        setQty({});
        setBundleQty({});
        setBoothName("");
        setQuote(null);
      }
      if (resumedOrder) window.history.replaceState(null, "", "/counter");
      setResumedOrder(false);
      setPaidReceipt({
        id: orderId,
        number: orderNumber,
        totalMinor: orderTotalMinor,
        currency: orderCurrency,
      });
      clearPendingOrderMarker();
      void refreshCatalog();
    } catch {
      setMessage({ kind: "error", text: "无法连接服务，请重试同一笔收款操作以安全恢复结果。" });
    } finally {
      setSubmitting(false);
    }
  }

  async function fulfillReceipt() {
    if (!paidReceipt || !me?.id || !me.shopId) return;
    setFulfilling(true);
    try {
      const scope = `fulfill:${me.id}:${me.shopId}:${paidReceipt.id}`;
      const res = await sendOrderCommand(scope, `/api/staff/orders/${paidReceipt.id}/fulfill`, {});
      if (!res.ok) {
        setMessage({ kind: "error", text: describeRequest(res) });
        return;
      }
      setMessage({ kind: "success", text: `订单 ${paidReceipt.number} 已标记为当场交付。` });
      setPaidReceipt(null);
      void refreshCatalog();
    } catch {
      setMessage({ kind: "error", text: "无法连接服务，请重试交付操作。" });
    } finally {
      setFulfilling(false);
    }
  }

  async function unpaid() {
    try {
      const res = await browserJSON(`/api/staff/orders/${orderId}/unpaid`, "POST", {});
      if (!res.ok) {
        setMessage({ kind: "error", text: describeRequest(res) });
        return;
      }
      setMessage({ kind: "success", text: "顾客未付款，订单保留为待付款状态。" });
      if (me?.id && me.shopId) {
        const scope = { accountId: me.id, shopId: me.shopId, eventId };
        if (loadPendingCounterOrder(scope)?.id === orderId) clearPendingCounterOrder(scope);
      }
      if (resumedOrder) window.history.replaceState(null, "", "/counter");
      setResumedOrder(false);
      clearPendingOrderMarker();
      void refreshCatalog();
    } catch {
      setMessage({ kind: "error", text: "无法连接服务，请检查网络后重试。" });
    }
  }

  const activePayCode = codes.find((code) => code.id === payCodeId);
  const selectedCount = quote?.itemCount ?? cartLines.reduce((sum, line) => sum + line.qty, 0);
  const unresolvedPayCommand = orderId && me?.id && me.shopId
    ? pendingOrderCommand<{ payCodeId: string }>(`pay:${me.id}:${me.shopId}:${orderId}`)
    : null;
  const cashReceivedMinor = toMinor(cashReceived);
  const cashChangeMinor = activePayCode?.kind === "cash" && cashReceivedMinor !== null && orderTotalMinor > 0
    ? cashReceivedMinor - orderTotalMinor
    : null;
  const cashInvalid = activePayCode?.kind === "cash" && (cashReceivedMinor === null || cashReceivedMinor < orderTotalMinor);

  return (
    <WorkspaceShell kind="sales">
      <main className="counter-page">
        <p className="eyebrow">现场收银</p>
        <h1>创建订单</h1>
        <p>点选商品即可加一。服务器会实时计算数量优惠和固定套餐，自动选择顾客最省的组合。</p>
        {initError ? (
          <p className="counter-quote-error" role="alert">{initError} <button type="button" onClick={() => void loadInitial()}>重试</button></p>
        ) : null}
        <form className="counter-form" onSubmit={place}>
          <div className="counter-context">
            <label className="field"><span>活动场次</span>
              <SelectMenu value={eventId} placeholder="选择活动" ariaLabel="活动场次" options={events.map((event) => ({ value: event.id, label: `${event.name}${event.boothNumber ? ` · ${event.boothNumber}` : ""}${event.status === "live" ? "（进行中）" : event.status === "ended" ? "（已结束）" : ""}` }))} onChange={switchEvent} />
              {liveEvents.length > 1 ? <div className="counter-event-chips" role="group" aria-label="进行中的活动">
                {liveEvents.map((event) => <button type="button" key={event.id} data-active={event.id === eventId} aria-pressed={event.id === eventId} onClick={() => switchEvent(event.id)}>{event.name}{event.boothNumber ? ` · ${event.boothNumber}` : ""}</button>)}
              </div> : null}
            </label>
            <label className="field"><span>顾客称呼 <small>选填</small></span>
              <input value={boothName} onChange={(change) => setBoothName(change.target.value)} placeholder="方便现场叫号" />
            </label>
          </div>

          {eventClosed ? <p className="counter-quote-error" role="alert">这场活动已结束，暂时不能加入新订单。</p> : null}

          <div className="counter-layout">
            <section className="counter-catalog" aria-labelledby="counter-catalog-heading">
              <div className="counter-section-heading"><div><p className="eyebrow">活动商品</p><h2 id="counter-catalog-heading">点选加货</h2></div><span>{listings.length} 件</span></div>
              <div className="counter-catalog-tools">
                <label className="counter-catalog-search"><span className="sr-only">搜索商品或套餐</span><input type="search" value={catalogSearch} onChange={(change) => setCatalogSearch(change.target.value)} placeholder="搜索商品或套餐名称" /></label>
                <div className="counter-catalog-filter" role="group" aria-label="商品类型筛选">
                  {(["all", "good", "bundle"] as const).map((filter) => <button type="button" key={filter} data-active={catalogFilter === filter} aria-pressed={catalogFilter === filter} onClick={() => setCatalogFilter(filter)}>{filter === "all" ? "全部" : filter === "good" ? "单品" : "套餐"}</button>)}
                </div>
              </div>
              {visibleListings.length === 0 && visibleBundles.length === 0 ? <p className="counter-empty">{normalizedSearch ? "没有找到匹配的商品或套餐。" : "该活动暂时没有商品。"}</p> : null}
              {visibleListings.length > 0 ? (
                <ul className="counter-products">
                  {visibleListings.map((item) => {
                    const disabled = eventClosed || item.reviewStatus !== "approved" || !item.isListed || item.remaining <= 0;
                    const count = qty[item.listingId] ?? 0;
                    return (
                      <li key={item.listingId}>
                        <button type="button" className={`counter-product${disabled ? " counter-product--unavailable" : ""}${disabled && item.reviewStatus === "approved" && item.isListed && item.remaining <= 0 ? " counter-product--stockout" : ""}`} disabled={disabled} onClick={() => adjust(item, 1)}>
                          {item.imageURL
                            ? <SafeImage className="counter-product__thumb" src={item.imageURL} alt="" width={56} height={56} unoptimized fallback={<span className="counter-product__thumb counter-product__thumb--empty" aria-hidden="true">无图</span>} />
                            : <span className="counter-product__thumb counter-product__thumb--empty" aria-hidden="true">无图</span>}
                          <span className="counter-product__copy"><strong>{item.name}</strong><small className={disabled ? "counter-product__status counter-product__status--off" : "counter-product__status"}>{availability(item, maxSingleQty(item), bundleUsageByListing[item.listingId] ?? 0)}</small></span>
                          <span className="counter-product__price"><b>{item.priceLabel}</b>{disabled && item.isListed && item.reviewStatus === "approved" && item.remaining <= 0 ? null : item.priceTiers?.filter((tier) => tier.quantity <= item.remaining).map((tier) => <small key={tier.quantity}>{tier.quantity} 件 {formatMoney(currency, tier.totalPriceMinor)}</small>)}<i aria-hidden="true" className={disabled ? "counter-product__add--off" : undefined}>{disabled ? "原因" : "＋"}</i></span>
                          {count > 0 ? <span className="counter-product__count" aria-label={`已选 ${count} 件`}>{count}</span> : null}
                          {disabled && item.reviewStatus === "approved" && item.isListed && item.remaining <= 0 ? <span className="counter-product__badge">售罄</span> : null}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              ) : null}
              {visibleBundles.length > 0 ? (
                <section className="counter-bundles" aria-label="固定套餐">
                  <div className="counter-section-heading"><div><p className="eyebrow">固定搭配</p><h2>套餐</h2></div><span>{visibleBundles.length} 套</span></div>
                  <ul className="counter-products">
                    {visibleBundles.map((bundle) => {
                      const disabled = eventClosed || bundle.reviewStatus !== "approved" || !bundle.isListed || bundle.remaining <= 0;
                      const count = bundleQty[bundle.bundleListingId] ?? 0;
                      return <li key={bundle.bundleListingId}>
                        <button type="button" className={`counter-product counter-product--bundle${disabled ? " counter-product--unavailable" : ""}${disabled && bundle.reviewStatus === "approved" && bundle.isListed && bundle.remaining <= 0 ? " counter-product--stockout" : ""}`} disabled={disabled} onClick={() => adjustBundle(bundle, 1)}>
                          {bundle.imageURL
                            ? <SafeImage className="counter-product__thumb" src={bundle.imageURL} alt="" width={56} height={56} unoptimized fallback={<span className="counter-product__thumb counter-product__thumb--empty" aria-hidden="true">无图</span>} />
                            : <span className="counter-product__thumb counter-product__thumb--empty" aria-hidden="true">无图</span>}
                          <span className="counter-product__copy"><strong>{bundle.name}</strong><small className={disabled ? "counter-product__status counter-product__status--off" : "counter-product__status"}>{bundleAvailability(bundle, maxBundleQty(bundle))}</small><small>{bundle.components.map((component) => `${component.name} × ${component.quantity}`).join(" · ")}</small></span>
                          <span className="counter-product__price"><b>{bundle.priceLabel}</b><i aria-hidden="true" className={disabled ? "counter-product__add--off" : undefined}>{disabled ? "原因" : "＋"}</i></span>
                          {count > 0 ? <span className="counter-product__count" aria-label={`已选 ${count} 套`}>{count}</span> : null}
                          {disabled && bundle.reviewStatus === "approved" && bundle.isListed && bundle.remaining <= 0 ? <span className="counter-product__badge">售罄</span> : null}
                        </button>
                      </li>;
                    })}
                  </ul>
                </section>
              ) : null}
            </section>

            <aside className="counter-summary" aria-label="订单实时预览" aria-busy={quoteLoading} data-open={cartOpen}>
              <div className="counter-summary__heading">
                <div><p className="eyebrow">购物车</p><h2>订单预览</h2></div>
                <span>{selectedCount} 件</span>
                <button type="button" className="counter-sheet__close" onClick={() => setCartOpen(false)}>收起</button>
              </div>
              <div className="counter-summary__body">
              {cartLines.length === 0 ? <p className="counter-empty">点选上方商品，将商品加入订单。</p> : (
                <>
                  <ul className="counter-cart counter-cart--selection">
                    {listings.filter((item) => (qty[item.listingId] ?? 0) > 0).map((item) => <li className="counter-cart__line" key={item.listingId}>
                      {item.imageURL
                        ? <SafeImage className="counter-cart__thumb" src={item.imageURL} alt="" width={40} height={40} unoptimized fallback={<span className="counter-cart__thumb counter-cart__thumb--empty" aria-hidden="true">无图</span>} />
                        : <span className="counter-cart__thumb counter-cart__thumb--empty" aria-hidden="true">无图</span>}
                      <div className="counter-cart__identity">
                        <strong>{item.name}</strong>
                        <small>{formatMoney(currency, item.unitPriceMinor)} / 件</small>
                      </div>
                      <div className="quantity-control">
                        <button type="button" aria-label={`减少 ${item.name}`} onClick={() => adjust(item, -1)}>−</button><span aria-live="polite">{qty[item.listingId]}</span>
                        <button type="button" aria-label={`增加 ${item.name}`} disabled={qty[item.listingId] >= maxSingleQty(item)} onClick={() => adjust(item, 1)}>＋</button>
                      </div>
                      <strong className="counter-cart__amount">{formatMoney(currency, item.unitPriceMinor * qty[item.listingId])}</strong>
                    </li>)}
                    {bundles.filter((bundle) => (bundleQty[bundle.bundleListingId] ?? 0) > 0).map((bundle) => <li className="counter-cart__line" key={bundle.bundleListingId}>
                      {bundle.imageURL
                        ? <SafeImage className="counter-cart__thumb" src={bundle.imageURL} alt="" width={40} height={40} unoptimized fallback={<span className="counter-cart__thumb counter-cart__thumb--empty" aria-hidden="true">无图</span>} />
                        : <span className="counter-cart__thumb counter-cart__thumb--empty" aria-hidden="true">无图</span>}
                      <div className="counter-cart__identity">
                        <strong>{bundle.name}</strong>
                        <small>{formatMoney(currency, bundle.unitPriceMinor)} / 套</small>
                      </div>
                      <div className="quantity-control">
                        <button type="button" aria-label={`减少 ${bundle.name}`} onClick={() => adjustBundle(bundle, -1)}>−</button><span aria-live="polite">{bundleQty[bundle.bundleListingId]}</span>
                        <button type="button" aria-label={`增加 ${bundle.name}`} disabled={bundleQty[bundle.bundleListingId] >= maxBundleQty(bundle)} onClick={() => adjustBundle(bundle, 1)}>＋</button>
                      </div>
                      <strong className="counter-cart__amount">{formatMoney(currency, bundle.unitPriceMinor * bundleQty[bundle.bundleListingId])}</strong>
                    </li>)}
                  </ul>
                  <div className="counter-quote-breakdown" aria-label="服务器计算的优惠明细">
                    <p className="counter-quote-breakdown__caption">价格分摊 · 已含优惠</p>
                    {quote?.groups.map((group) => <div className="counter-quote-breakdown__row" key={`${group.kind}-${group.name}-${group.applications}`}>
                      <span>{bundleGroup(group.kind)}{group.applications > 1 ? ` × ${group.applications}` : ""}<small>{group.name}</small></span>
                      <strong>{formatMoney(currency, group.totalMinor)}</strong>
                    </div>)}
                  </div>
                </>
              )}
              </div>
              <div className="counter-total">
                {quote?.savingsMinor ? <p><span>商品原价</span><strong>{formatMoney(currency, quote.baseMinor)}</strong></p> : null}
                {quote?.savingsMinor ? <p><span>自动组合优惠</span><strong>− {formatMoney(currency, quote.savingsMinor)}</strong></p> : null}
                <p className="counter-total__amount"><span>应付总额（{currency}）</span><strong>{quoteLoading ? "正在重新计价…" : quote ? formatMoney(currency, quote.totalMinor) : "-"}</strong></p>
                {quoteError ? <p className="counter-quote-error" role="alert">{quoteError}</p> : null}
              </div>
              <button className="button button--dark counter-submit" type="submit" disabled={submitting || cartLines.length === 0 || !eventId || quoteLoading || !quote || eventClosed}>{submitting ? "正在生成…" : "生成订单"}</button>
            </aside>
          </div>
          <div className="counter-bar">
            <button type="button" className="counter-bar__summary" onClick={() => setCartOpen(true)}>
              <span className="counter-bar__count">已选 {selectedCount} 件 · 查看明细</span>
              <strong>{quoteLoading ? "正在计价…" : quote ? formatMoney(currency, quote.totalMinor) : "-"}</strong>
            </button>
            <button className="button button--dark counter-bar__submit" type="submit" disabled={submitting || cartLines.length === 0 || !eventId || quoteLoading || !quote || eventClosed}>{submitting ? "正在生成…" : "生成订单"}</button>
          </div>
        </form>

        {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
        {orderId && !payOpen && !paidReceipt ? (
          <section className="counter-pending" aria-label="待付款订单">
            <div><span className="eyebrow">待付款</span><strong>订单 {orderNumber}</strong><p>{orderPriceLabel}</p></div>
            <div className="counter-pending__actions">
              <button className="button button--dark" type="button" onClick={() => setPayOpen(true)}>收款</button>
              <button className="button button--outline" type="button" disabled={submitting} onClick={unpaid}>顾客未付</button>
            </div>
          </section>
        ) : null}

        {orderId && payOpen ? (
          <div className="pay-dialog" role="dialog" aria-modal="true" aria-labelledby="pay-dialog-title">
            <div className="pay-dialog__panel">
              <header className="pay-dialog__head">
                <div><p className="eyebrow">待付款 · {activePayCode?.kind === "cash" ? "收取现金" : "请顾客扫码"}</p><h2 id="pay-dialog-title">订单 {orderNumber}</h2></div>
                <button type="button" className="pay-dialog__close" aria-label="稍后处理这笔订单" disabled={submitting} onClick={() => setPayOpen(false)}>×</button>
              </header>
              <strong className="pay-dialog__amount">{orderPriceLabel}</strong>
              {resumedOrder ? <p className="counter-reservation-note">收款后当前购物车会保留，请检查其中商品，避免重复开单。</p> : null}
              {unresolvedPayCommand ? <p className="counter-reservation-note">上次收款结果尚未确认，重试将使用原收款方式和实收金额。</p> : null}
              {!activePayCode ? <p className="counter-empty">{unresolvedPayCommand ? "上次使用的收款方式已不可用，请重试以确认原收款结果。" : "本店还没有可用收款方式，请让店铺管理员先配置。"}</p> : activePayCode.kind === "cash" ? (
                <div className="pay-dialog__cash">
                  <span>现金收款</span>
                  <strong>{orderPriceLabel}</strong>
                  <label className="field pay-cash-received">
                    <span>实收金额（{orderCurrency}）</span>
                    <input inputMode="decimal" value={cashReceived} disabled={submitting || Boolean(unresolvedPayCommand)} onChange={(change) => setCashReceived(change.target.value)} placeholder="顾客交付的现金金额" />
                  </label>
                  {cashChangeMinor !== null && !cashInvalid ? <p className="pay-cash-change">找零：{formatMoney(orderCurrency, cashChangeMinor)}</p> : null}
                  {cashInvalid ? <p className="counter-quote-error" role="alert">实收金额不能少于订单总额。</p> : null}
                  <small>请点收现金，确认后扣减库存。</small>
                </div>
              ) : (
                <div className="pay-dialog__qr">
                  <SafeImage src={activePayCode.imageURL} alt={`${activePayCode.name} 收款码`} width={560} height={560} unoptimized fallbackClassName="pay-dialog__qr--fallback" />
                  <span>{activePayCode.name}{activePayCode.isDefault ? " · 默认收款方式" : ""}{activePayCode.currency ? ` · ${activePayCode.currency}` : ""}</span>
                </div>
              )}
              {codes.length > 1 ? <div className="pay-dialog__codes" role="group" aria-label="切换收款方式">
                {codes.map((code) => <button key={code.id} type="button" data-active={code.id === payCodeId} aria-pressed={code.id === payCodeId} disabled={Boolean(unresolvedPayCommand)} onClick={() => { setPayCodeId(code.id); setCashReceived(""); }}>{code.name}</button>)}
              </div> : null}
              {reservationExpiresAt ? <p className="counter-reservation-note">库存已预留至 {new Date(reservationExpiresAt).toLocaleString("zh-CN")}。请在此时间前确认收款。</p>
                : <p className="counter-reservation-note">此订单暂未设置库存预留期限。</p>}
              <div className="pay-dialog__actions">
                <button className="button button--dark" type="button" disabled={submitting || !payCodeId || cashInvalid} onClick={confirm}>{submitting ? "正在确认…" : "顾客已付款，扣减库存"}</button>
                <button className="button button--outline" type="button" disabled={submitting} onClick={() => setPayOpen(false)}>稍后处理</button>
                <button className="text-link" type="button" disabled={submitting} onClick={unpaid}>顾客未付，保留待付款</button>
              </div>
            </div>
          </div>
        ) : null}

        {paidReceipt ? (
          <div className="pay-dialog" role="dialog" aria-modal="true" aria-labelledby="receipt-title">
            <div className="pay-dialog__panel pay-receipt">
              <header className="pay-dialog__head">
                <div><p className="eyebrow">收款成功</p><h2 id="receipt-title">取货凭证</h2></div>
                <button type="button" className="pay-dialog__close" aria-label="关闭凭证" disabled={fulfilling} onClick={() => setPaidReceipt(null)}>×</button>
              </header>
              <p className="pay-receipt__hint">请把下面的订单号交给店员核对取货</p>
              <strong className="pay-receipt__number">{paidReceipt.number}</strong>
              <p className="pay-receipt__amount">{formatMoney(paidReceipt.currency, paidReceipt.totalMinor)}</p>
              <div className="pay-dialog__actions">
                <button className="button button--dark" type="button" disabled={fulfilling} onClick={() => void fulfillReceipt()}>{fulfilling ? "正在交付…" : "当场交付"}</button>
                <button className="button button--outline" type="button" disabled={fulfilling} onClick={() => setPaidReceipt(null)}>稍后交付</button>
              </div>
            </div>
          </div>
        ) : null}
        {catalogNotice ? <p className="counter-toast" role="status">{catalogNotice}</p> : null}
      </main>
    </WorkspaceShell>
  );
}
