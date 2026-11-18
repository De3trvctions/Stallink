"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Image from "next/image";
import { SafeImage } from "@/components/safe-image";
import { browserFormData, browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney, majorToMinor, type PriceTier } from "@/lib/pricing";
import ModalDialog from "@/components/modal-dialog";
import SelectMenu from "@/components/select-menu";
import { ConfirmDialog } from "@/components/confirm-dialog";

type EventRow = {
  id: string; name: string; currency: string; startsOn?: string; endsOn?: string;
  city?: string; venue?: string; boothNumber?: string; status?: string;
};
type FreshEvent = { name: string; startsOn: string; endsOn: string; city: string; venue: string; boothNumber: string; currency: string; status: string };
type TierDraft = { quantity: string; totalPrice: string };
type ProductImage = { id: string; url: string; sortOrder: number };
type Listing = {
  listingId: string; goodId: string; eventId: string; eventName: string; name: string; description: string; currency: string;
  unitPriceMinor: number; priceTiers: PriceTier[]; listedQty: number; soldQty: number; reservedQty: number;
  remaining: number; totalStock: number; isListed: boolean; reviewStatus: string; reviewNote: string;
  imageURL: string | null; images: ProductImage[];
};
type BundleComponent = { goodId: string; listingId: string; name: string; quantity: number; unitPriceMinor?: number };
type Bundle = {
  bundleListingId: string; bundleId: string; eventId: string; name: string; description: string;
  unitPriceMinor: number; priceTiers: PriceTier[]; remaining: number; isListed: boolean;
  reviewStatus: string; reviewNote: string; revision: number; components: BundleComponent[]; images: ProductImage[];
};
type ListingDraft = {
  name: string; description: string; totalStock: string; listedQty: string;
  price: string; tiers: TierDraft[]; isListed: boolean;
};
type BundleGoodDraftForm = {
  draftKey: string; name: string; description: string; totalStock: number; listedQty: number;
  quantity: number; price: string; tiers: TierDraft[]; imageFiles: File[];
};
type EventGoodsGroup = { event: EventRow; listings: Listing[]; bundles: Bundle[]; loading: boolean; error: string };

const emptyEvent: FreshEvent = { name: "", startsOn: "", endsOn: "", city: "", venue: "", boothNumber: "", currency: "MYR", status: "scheduled" };
const tierDraft = (tier?: PriceTier): TierDraft => ({ quantity: tier ? String(tier.quantity) : "", totalPrice: tier ? (tier.totalPriceMinor / 100).toFixed(2) : "" });
const minorInput = (minor: number) => (minor / 100).toFixed(2);
const formatUploadSize = (bytes: number) => `${(bytes / 1024 / 1024).toFixed(1)} MiB`;
const maxImageBytes = 6 * 1024 * 1024;
function imageSelectionError(files: File[], existing: number, limit: number): string {
  if (existing + files.length > limit) return limit === 1 ? "套餐总图只能选择 1 张。" : `最多可添加 ${limit} 张图片。`;
  const oversized = files.find((file) => file.size > maxImageBytes);
  if (oversized) return `图片“${oversized.name}”约 ${(Math.ceil(oversized.size / 1024 / 1024 * 100) / 100).toFixed(2)} MiB，超过单张 6 MiB 限制，请压缩后重新选择。`;
  const invalid = files.find((file) => !["image/jpeg", "image/png", "image/webp"].includes(file.type) || file.size <= 0);
  if (invalid) return `图片“${invalid.name}”格式不支持或文件为空，请选择 JPG、PNG 或 WebP 图片。`;
  return "";
}
const pickerStyle: React.CSSProperties = { position: "relative", display: "inline-flex", alignItems: "center", justifyContent: "center", width: "fit-content", minWidth: 116, minHeight: 44, padding: "0 1rem", border: "1px solid var(--line-strong)", borderRadius: 9, background: "var(--surface)", color: "var(--ink)", cursor: "pointer" };
const pickerInputStyle: React.CSSProperties = { position: "absolute", inset: 0, width: "100%", height: "100%", opacity: 0, cursor: "pointer" };

function parseTiers(drafts: TierDraft[], unitPriceMinor: number): PriceTier[] | null {
  const tiers: PriceTier[] = [];
  const seen = new Set<number>();
  for (const draft of drafts) {
    const quantity = Number(draft.quantity);
    const totalPriceMinor = majorToMinor(draft.totalPrice);
    if (!Number.isInteger(quantity) || quantity <= 1 || quantity > 100_000 || seen.has(quantity)
      || totalPriceMinor == null || totalPriceMinor <= 0 || totalPriceMinor >= unitPriceMinor * quantity) return null;
    seen.add(quantity);
    tiers.push({ quantity, totalPriceMinor });
  }
  return tiers;
}

function PriceTierEditor({ drafts, currency, basePrice, onChange }: {
  drafts: TierDraft[]; currency: string; basePrice: number; onChange: (drafts: TierDraft[]) => void;
}) {
  return (
    <div className="tier-editor">
      <div className="tier-editor__heading"><strong>组合优惠</strong><span>设置指定件数的套装总价</span></div>
      {drafts.map((draft, index) => {
        const quantity = Number(draft.quantity);
        const totalMinor = majorToMinor(draft.totalPrice);
        const savings = totalMinor != null && Number.isInteger(quantity) && quantity > 1
          ? basePrice * quantity - totalMinor : 0;
        return (
          <div className="tier-editor__row" key={index}>
            <label className="field"><span>件数</span><input type="number" min="2" step="1" inputMode="numeric" value={draft.quantity} onChange={(event) => onChange(drafts.map((item, at) => at === index ? { ...item, quantity: event.target.value } : item))} placeholder="例如 2" /></label>
            <label className="field"><span>套装总价 <small>{currency}</small></span><input type="number" min="0.01" step="0.01" inputMode="decimal" value={draft.totalPrice} onChange={(event) => onChange(drafts.map((item, at) => at === index ? { ...item, totalPrice: event.target.value } : item))} placeholder="例如 15.00" /></label>
            <button className="tier-editor__remove" type="button" aria-label={`删除第 ${index + 1} 个优惠包`} onClick={() => onChange(drafts.filter((_, at) => at !== index))}>删除</button>
            {savings > 0 ? <p className="tier-editor__saving">比按单价购买省 {formatMoney(currency, savings)}</p> : null}
          </div>
        );
      })}
      <button className="tier-editor__add" type="button" onClick={() => onChange([...drafts, tierDraft()])}>＋ 添加优惠包</button>
    </div>
  );
}

function DraftImagePreview({ file, alt }: { file: File; alt: string }) {
  const [url, setUrl] = useState("");
  useEffect(() => {
    const next = URL.createObjectURL(file);
    const timer = window.setTimeout(() => setUrl(next), 0);
    return () => { window.clearTimeout(timer); URL.revokeObjectURL(next); };
  }, [file]);
  return url ? <Image src={url} alt={alt} width={56} height={56} unoptimized /> : <span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />;
}

function statusLabel(status: string) {
  if (status === "approved") return "审核通过";
  if (status === "rejected") return "已拒绝";
  return "待平台审核";
}

function eventStatusLabel(status?: string) {
  if (status === "live") return "进行中";
  if (status === "ended") return "已结束";
  return "即将开始";
}

function normalizeGoodName(value: string) {
  return value.trim().toLocaleLowerCase().replace(/\s+/g, "");
}

function listingDraft(item: Listing): ListingDraft {
  return {
    name: item.name, description: item.description ?? "", totalStock: String(item.totalStock),
    listedQty: String(item.listedQty), price: minorInput(item.unitPriceMinor),
    tiers: (item.priceTiers ?? []).map(tierDraft), isListed: item.isListed,
  };
}

export default function GoodsPage() {
  const [events, setEvents] = useState<EventRow[]>([]);
  const [eventGroups, setEventGroups] = useState<EventGoodsGroup[]>([]);
  const [eventsLoading, setEventsLoading] = useState(true);
  const [eventsError, setEventsError] = useState(false);
  const [drafts, setDrafts] = useState<Record<string, ListingDraft>>({});
  const [createEvent, setCreateEvent] = useState(false);
  const [eventId, setEventId] = useState("");
  const [itemKind, setItemKind] = useState<"good" | "bundle">("good");
  const [editingBundle, setEditingBundle] = useState<Bundle | null>(null);
  const [replaceBundleImages, setReplaceBundleImages] = useState(false);
  const [componentQuantities, setComponentQuantities] = useState<Record<string, number>>({});
  const [componentSearch, setComponentSearch] = useState("");
  const [componentPickerOpen, setComponentPickerOpen] = useState(false);
  const [componentPickerSearch, setComponentPickerSearch] = useState("");
  const [newBundleGoods, setNewBundleGoods] = useState<BundleGoodDraftForm[]>([]);
  const [draftImageErrors, setDraftImageErrors] = useState<Record<string, string>>({});
  const [mainImageError, setMainImageError] = useState("");
  const [existingImageErrors, setExistingImageErrors] = useState<Record<string, string>>({});
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [price, setPrice] = useState("19.90");
  const [tiers, setTiers] = useState<TierDraft[]>([]);
  const [totalStock, setTotalStock] = useState(10);
  const [listedQty, setListedQty] = useState(4);
  const [imageFiles, setImageFiles] = useState<File[]>([]);
  const [imagePreviews, setImagePreviews] = useState<string[]>([]);
  const [message, setMessageText] = useState("");
  const [messageKind, setMessageKind] = useState<"success" | "error">("success");
  const setMessage = (text: string, kind: "success" | "error" = "success") => { setMessageText(text); setMessageKind(kind); };
  const [pending, setPending] = useState(false);
  const [imagePending, setImagePending] = useState("");
  const [imageDeleteTarget, setImageDeleteTarget] = useState<{ item: Listing; image: ProductImage } | null>(null);
  const [fresh, setFresh] = useState<FreshEvent>(emptyEvent);
  const [search, setSearch] = useState("");
  const [addOpen, setAddOpen] = useState(false);
  const [editingListing, setEditingListing] = useState<string | null>(null);
  const [copyEvent, setCopyEvent] = useState("");
  const [copyQty, setCopyQty] = useState("0");
  const [restockQty, setRestockQty] = useState("");
  const addDialogRef = useRef<HTMLDialogElement>(null);
  const listingDialogRef = useRef<HTMLDialogElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const dialog = listingDialogRef.current;
    if (!dialog) return;
    if (editingListing && !dialog.open) dialog.showModal();
    if (!editingListing && dialog.open) dialog.close();
  }, [editingListing]);
  const requestVersions = useRef(new Map<string, number>());

  const loadEventListings = useCallback(async (event: EventRow, isCurrent: () => boolean = () => true) => {
    const version = (requestVersions.current.get(event.id) ?? 0) + 1;
    requestVersions.current.set(event.id, version);
    setEventGroups((current) => current.some((group) => group.event.id === event.id)
      ? current.map((group) => group.event.id === event.id ? { ...group, event, loading: true, error: "" } : group)
      : [...current, { event, listings: [], bundles: [], loading: true, error: "" }]);
    try {
      const [res, bundleRes] = await Promise.all([
        browserJSON(`/api/staff/events/${event.id}/listings`),
        browserJSON(`/api/staff/events/${event.id}/bundles`),
      ]);
      if (!isCurrent() || requestVersions.current.get(event.id) !== version) return;
      if (!res.ok) throw new Error("无法读取该活动的商品。");
      const rows = Array.isArray(res.data) ? (res.data as Listing[]).map((item) => ({ ...item, images: item.images ?? [] })) : [];
      const bundleRows = bundleRes.ok && Array.isArray(bundleRes.data)
        ? (bundleRes.data as Bundle[]).map((bundle) => ({ ...bundle, images: bundle.images ?? [], components: bundle.components ?? [] })) : [];
      setEventGroups((current) => current.map((group) => group.event.id === event.id
        ? { ...group, event, listings: rows, bundles: bundleRows, loading: false, error: "" }
        : group));
      setDrafts((current) => ({ ...current, ...Object.fromEntries(rows.map((item) => [item.listingId, listingDraft(item)])) }));
    } catch {
      if (!isCurrent() || requestVersions.current.get(event.id) !== version) return;
      setEventGroups((current) => current.map((group) => group.event.id === event.id
        ? { ...group, loading: false, error: "该活动的商品暂时无法读取。" }
        : group));
    }
  }, []);

  const loadAllEvents = useCallback(async (isCurrent: () => boolean = () => true) => {
    setEventsLoading(true);
    setEventsError(false);
    try {
      const res = await browserJSON("/api/staff/events");
      if (!isCurrent()) return [];
      if (!res.ok) throw new Error("无法读取活动。");
      const rows = Array.isArray(res.data) ? res.data as EventRow[] : [];
      setEvents(rows);
      setEventGroups((current) => rows.map((event) => ({
        event,
        listings: current.find((group) => group.event.id === event.id)?.listings ?? [],
        bundles: current.find((group) => group.event.id === event.id)?.bundles ?? [],
        loading: true,
        error: "",
      })));
      await Promise.all(rows.map((event) => loadEventListings(event, isCurrent)));
      return rows;
    } catch {
      if (isCurrent()) setEventsError(true);
      return [];
    } finally {
      if (isCurrent()) setEventsLoading(false);
    }
  }, [loadEventListings]);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => { void loadAllEvents(() => active); }, 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [loadAllEvents]);

  useEffect(() => {
    const dialog = addDialogRef.current;
    if (!addOpen || !dialog || dialog.open) return;
    dialog.showModal();
    return () => { if (dialog.open) dialog.close(); };
  }, [addOpen]);

  useEffect(() => {
    const previews = imageFiles.map((file) => URL.createObjectURL(file));
    const timer = window.setTimeout(() => setImagePreviews(previews), 0);
    return () => { window.clearTimeout(timer); previews.forEach((url) => URL.revokeObjectURL(url)); };
  }, [imageFiles]);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    const unitPriceMinor = majorToMinor(price);
    const priceTiers = unitPriceMinor == null ? null : itemKind === "bundle" ? [] : parseTiers(tiers, unitPriceMinor);
    if (unitPriceMinor == null || unitPriceMinor <= 0 || priceTiers === null) {
      setMessage("请填写有效的基础价格和优惠包件数、套装总价。", "error");
      return;
    }
    if (itemKind === "bundle") {
      const components: Record<string, unknown>[] = Object.entries(componentQuantities)
        .filter(([, quantity]) => quantity > 0)
        .map(([goodId, quantity]) => ({ goodId, quantity }));
      const shopGoodNames = new Map<string, string>();
      eventGroups.forEach((group) => group.listings.forEach((listing) => {
        const key = normalizeGoodName(listing.name);
        if (!shopGoodNames.has(key)) shopGoodNames.set(key, listing.name);
      }));
      const draftNames = new Set<string>();
      for (const item of newBundleGoods) {
        const normalized = normalizeGoodName(item.name);
        if (normalized && shopGoodNames.has(normalized)) {
          setMessage(`店里已经有「${shopGoodNames.get(normalized)}」这件商品了，请从现有商品里加入套餐，不要重复新建。`, "error");
          return;
        }
        if (normalized && draftNames.has(normalized)) {
          setMessage(`套餐内有两个都叫「${item.name.trim()}」的新单品，请改名或合并成一件。`, "error");
          return;
        }
        if (normalized) draftNames.add(normalized);
      }
      for (const item of newBundleGoods) {
        const draftUnitMinor = majorToMinor(item.price);
        const draftTiers = draftUnitMinor == null ? null : parseTiers(item.tiers, draftUnitMinor);
        if (!item.name.trim() || draftUnitMinor == null || draftUnitMinor <= 0 || draftTiers === null
          || !Number.isInteger(item.totalStock) || item.totalStock < 1 || !Number.isInteger(item.listedQty)
          || item.listedQty < item.quantity || item.listedQty > item.totalStock || item.imageFiles.length > 8) {
          setMessage(`请完成套餐内商品“${item.name || "未命名"}”的名称、库存和价格设置。`, "error");
          return;
        }
        components.push({
          draftKey: item.draftKey,
          quantity: item.quantity,
          draft: {
            name: item.name.trim(), description: item.description, totalStock: item.totalStock,
            listedQty: item.listedQty, unitPriceMinor: draftUnitMinor, priceTiers: draftTiers, isListed: true,
          },
        });
      }
      if (createEvent || !eventId || components.length < 2 || components.length > 20) {
        setMessage("固定套餐需要在同一活动中选择 2 至 20 件不同商品。", "error");
        return;
      }
      if (bundleUploadBytes > 48 * 1024 * 1024) {
        setMessage("套餐与套餐内单品的图片合计不能超过 48 MiB，以给上传请求预留空间。", "error");
        return;
      }
      setPending(true);
      try {
        const form = new FormData();
        form.set("bundle", JSON.stringify({
          name, description, eventId, components, unitPriceMinor, priceTiers, isListed: true,
          ...(editingBundle ? { expectedRevision: editingBundle.revision, replaceImages: replaceBundleImages } : {}),
        }));
        imageFiles.forEach((file) => form.append("bundleImages", file));
        for (const draft of newBundleGoods) {
          draft.imageFiles.forEach((file) => form.append(`componentImages.${draft.draftKey}`, file));
        }
        const response = await browserFormData(
          editingBundle ? `/api/staff/bundles/${editingBundle.bundleId}` : "/api/staff/bundles",
          editingBundle ? "PATCH" : "POST",
          form,
        );
        if (!response.ok) {
          setMessage(describeRequest(response), "error");
          return;
        }
        setName("");
        setDescription("");
        setPrice("19.90");
        setTiers([]);
        setImageFiles([]);
        setEditingBundle(null);
        setReplaceBundleImages(false);
        setComponentQuantities({});
        setNewBundleGoods([]);
        setComponentSearch("");
        if (imageInputRef.current) imageInputRef.current.value = "";
        const selected = events.find((candidate) => candidate.id === eventId);
        if (selected) await loadEventListings(selected);
        setAddOpen(false);
        setMessage(editingBundle ? "套餐修改已保存，并重新提交平台审核。" : "固定套餐已提交平台审核。套餐商品仍可单独售卖，顾客选齐后收银台会自动凑套。");
      } catch {
        setMessage("无法连接服务，请检查网络后重试。", "error");
      } finally {
        setPending(false);
      }
      return;
    }

    const body: Record<string, unknown> = {
      name, description, totalStock, listedQty, isListed: true,
      unitPriceMinor, priceTiers,
    };
    if (createEvent) body.event = fresh;
    else body.eventId = eventId;
    setPending(true);
    try {
      const form = new FormData();
      form.set("product", JSON.stringify(body));
      imageFiles.forEach((file) => form.append("images", file));
      const response = await browserFormData("/api/staff/goods", "POST", form);
      const result = response.data;
      if (!response.ok) {
        setMessage(describeRequest(response), "error");
        return;
      }
      setName("");
      setDescription("");
      setPrice("19.90");
      setTiers([]);
      setImageFiles([]);
      if (imageInputRef.current) imageInputRef.current.value = "";
      setMessage("商品已提交平台审核，通过后会显示在店铺页面和收银台。");
      if (createEvent) {
        await loadAllEvents();
        const createdEvent = String(result.eventId ?? "");
        if (createdEvent) setEventId(createdEvent);
        setCreateEvent(false);
        setFresh(emptyEvent);
      } else {
        const selected = events.find((event) => event.id === eventId);
        if (selected) await loadEventListings(selected);
      }
      setAddOpen(false);
    } catch {
      setMessage("无法连接服务，请检查网络后重试。", "error");
    } finally {
      setPending(false);
    }
  }

  function selectNewImages(event: React.ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const files = Array.from(input.files ?? []);
    input.value = "";
    if (files.length === 0) return;
    const error = imageSelectionError(files, imageFiles.length, itemKind === "bundle" ? 1 : 8);
    setMainImageError(error);
    if (error) return;
    setMessage("");
    setImageFiles((current) => [...current, ...files]);
    if (editingBundle) setReplaceBundleImages(true);
  }

  function editBundle(bundle: Bundle) {
    setMainImageError("");
    setDraftImageErrors({});
    setEditingBundle(bundle);
    setReplaceBundleImages(false);
    setItemKind("bundle");
    setCreateEvent(false);
    setEventId(bundle.eventId);
    setName(bundle.name);
    setDescription(bundle.description);
    setPrice(minorInput(bundle.unitPriceMinor));
    setTiers([]);
    setComponentQuantities(Object.fromEntries(bundle.components.map((component) => [component.goodId, component.quantity])));
    setNewBundleGoods([]);
    setComponentSearch("");
    setImageFiles([]);
    if (imageInputRef.current) imageInputRef.current.value = "";
    setMessage("");
    setAddOpen(true);
  }

  function startNewItem() {
    setMainImageError("");
    setDraftImageErrors({});
    setEditingBundle(null);
    setReplaceBundleImages(false);
    setItemKind("good");
    setCreateEvent(false);
    setEventId("");
    setName("");
    setDescription("");
    setPrice("19.90");
    setTiers([]);
    setTotalStock(10);
    setListedQty(4);
    setComponentQuantities({});
    setNewBundleGoods([]);
    setComponentSearch("");
    setImageFiles([]);
    if (imageInputRef.current) imageInputRef.current.value = "";
    setMessage("");
    setAddOpen(true);
  }

  async function refreshGoodListings(goodId: string) {
    const linkedEvents = events.filter((event) => eventGroups.some((group) =>
      group.event.id === event.id && group.listings.some((item) => item.goodId === goodId)));
    await Promise.all(linkedEvents.map((event) => loadEventListings(event)));
  }

  async function addProductImages(item: Listing, files: File[]) {
    if (files.length === 0) return;
    const error = imageSelectionError(files, item.images.length, 8);
    setExistingImageErrors((current) => ({ ...current, [item.goodId]: error }));
    if (error) return;
    setImagePending(item.goodId);
    setMessage("");
    try {
      const form = new FormData();
      files.forEach((file) => form.append("images", file));
      const response = await browserFormData(`/api/staff/goods/${item.goodId}/images`, "POST", form);
      if (!response.ok) {
        setExistingImageErrors((current) => ({ ...current, [item.goodId]: describeRequest(response) }));
        return;
      }
      await refreshGoodListings(item.goodId);
      setMessage("商品相册已更新，正在等待平台审核。");
    } catch {
      setExistingImageErrors((current) => ({ ...current, [item.goodId]: "无法连接服务，图片没有上传。" }));
    } finally {
      setImagePending("");
    }
  }

  async function deleteProductImage(item: Listing, image: ProductImage) {
    setImagePending(item.goodId);
    setMessage("");
    try {
      const response = await browserJSON(`/api/staff/goods/${item.goodId}/images/${image.id}`, "DELETE");
      if (!response.ok) {
        setMessage(describeRequest(response), "error");
        return;
      }
      await refreshGoodListings(item.goodId);
      setMessage("商品相册已更新，正在等待平台审核。");
    } catch {
      setMessage("无法连接服务，图片没有删除。", "error");
    } finally {
      setImagePending("");
    }
  }

  async function moveProductImage(item: Listing, index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= item.images.length) return;
    const images = [...item.images];
    [images[index], images[target]] = [images[target], images[index]];
    setImagePending(item.goodId);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${item.goodId}/images`, "PATCH", { imageIds: images.map((image) => image.id) });
      if (!res.ok) {
        setMessage("图片顺序没有保存，请刷新后重试。", "error");
        return;
      }
      await refreshGoodListings(item.goodId);
      setMessage("封面顺序已更新，正在等待平台审核。");
    } catch {
      setMessage("无法连接服务，图片顺序没有保存。", "error");
    } finally {
      setImagePending("");
    }
  }

  function stockPool(goodId: string, totalStock: number) {
    let listed = 0;
    let sold = 0;
    eventGroups.forEach((group) => {
      group.listings.forEach((row) => {
        if (row.goodId !== goodId) return;
        listed += row.listedQty;
        sold += row.soldQty;
      });
    });
    return { total: totalStock, listed, sold, allocated: listed - sold, remain: Math.max(0, totalStock - (listed - sold)) };
  }

  async function copyListing(item: Listing) {
    const qty = Number(copyQty);
    if (!copyEvent || !Number.isInteger(qty) || qty < 0) { setMessage("请选择目标活动并填写有效的可售数量。", "error"); return; }
    setPending(true); setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${item.goodId}/listings`, "POST", { eventId: copyEvent, listedQty: qty, unitPriceMinor: item.unitPriceMinor, priceTiers: item.priceTiers ?? [], isListed: true });
      if (!res.ok) { setMessage(res.data.error === "stock_exceeded" ? "库存不足，请减少目标活动的可售数量。" : "复制失败：目标活动可能已有该商品，或库存不足。请检查后重试。", "error"); return; }
      setMessage("商品已复制到目标活动，原活动的商品仍保留。"); setCopyEvent(""); setCopyQty("0"); await loadAllEvents();
    } catch { setMessage("无法连接服务，请检查网络后重试。", "error"); } finally { setPending(false); }
  }

  async function restockItem(item: Listing) {
    const add = Number(restockQty);
    if (!Number.isInteger(add) || add <= 0) { setMessage("请输入要新增的库存件数。", "error"); return; }
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${item.goodId}`, "PATCH", {
        name: item.name, description: item.description, totalStock: item.totalStock + add,
      });
      if (!res.ok) { setMessage("库存没有增加，请稍后重试。", "error"); return; }
      setRestockQty("");
      const nextTotal = item.totalStock + add;
      setEventGroups((prev) => prev.map((group) => ({
        ...group,
        listings: group.listings.map((row) => row.goodId === item.goodId
          ? { ...row, totalStock: nextTotal, remaining: row.remaining + add }
          : row),
      })));
      setMessage(`已增加 ${add} 件库存。`);
    } catch {
      setMessage("无法连接服务，请稍后重试。", "error");
    } finally {
      setPending(false);
    }
  }

  async function saveAll(item: Listing) {
    const draft = drafts[item.listingId];
    if (!draft) return;
    const unitPriceMinor = majorToMinor(draft.price);
    const listed = Number(draft.listedQty);
    const priceTiers = unitPriceMinor == null ? null : parseTiers(draft.tiers, unitPriceMinor);
    if (unitPriceMinor == null || unitPriceMinor <= 0 || priceTiers === null) { setMessage("请修正优惠包数量和价格后保存。", "error"); return; }
    if (!Number.isInteger(listed) || listed < item.soldQty) { setMessage("本场可售数量不能低于已售数量。", "error"); return; }
    setPending(true);
    setMessage("");
    try {
      const goodRes = await browserJSON(`/api/staff/goods/${item.goodId}`, "PATCH", { name: draft.name, description: draft.description, totalStock: item.totalStock });
      if (!goodRes.ok) {
        setMessage(goodRes.data.error === "stock_exceeded" ? "商品总库存不能低于已分配库存。" : "商品资料没有保存，请检查输入。", "error");
        return;
      }
      const listingRes = await browserJSON(`/api/staff/listings/${item.listingId}`, "PATCH", { listedQty: listed, unitPriceMinor, priceTiers, isListed: draft.isListed });
      if (!listingRes.ok) {
        setMessage(listingRes.data.error === "stock_exceeded" ? "商品资料已保存，但本场数量超出库存；请调小数量后重新保存。"
          : listingRes.data.error === "invalid_price_tiers" ? "优惠包总价必须低于对应数量的单买价格。"
            : "商品资料已保存，但本场上架信息未保存；请检查价格和数量后重试。", "error");
        return;
      }
      setEditingListing(null);
      await loadAllEvents();
      setMessage("本场商品已保存。");
    } catch {
      setMessage("无法连接服务，请检查网络后重试。", "error");
    } finally {
      setPending(false);
    }
  }

  const editingItem = editingListing ? eventGroups.flatMap((group) => group.listings).find((row) => row.listingId === editingListing) ?? null : null;
  const editingDraft = editingItem ? drafts[editingItem.listingId] ?? null : null;
  const editingPriceMinor = editingDraft ? majorToMinor(editingDraft.price) ?? 0 : 0;
  const editingStock = editingItem ? stockPool(editingItem.goodId, editingItem.totalStock) : { total: 0, listed: 0, sold: 0, allocated: 0, remain: 0 };
  const editingLive = (() => {
    if (!editingItem || !editingDraft) return editingStock;
    const othersAllocated = Math.max(0, editingStock.allocated - (editingItem.listedQty - editingItem.soldQty));
    const draftListed = editingDraft.listedQty === "" ? editingItem.listedQty : Number(editingDraft.listedQty);
    const nextAllocated = othersAllocated + Math.max(0, draftListed - editingItem.soldQty);
    return {
      total: editingStock.total,
      sold: editingStock.sold,
      allocated: nextAllocated,
      remain: Math.max(0, editingStock.total - editingStock.sold - nextAllocated),
    };
  })();
  const editingCopyOver = Number(copyQty) > editingStock.remain;

  const selectedEvent = events.find((event) => event.id === eventId);
  const selectedGroup = eventGroups.find((group) => group.event.id === eventId);
  const currency = createEvent ? fresh.currency : selectedEvent?.currency ?? "MYR";
  const selectedComponentSubtotal = (selectedGroup?.listings ?? []).reduce((sum, item) =>
    sum + item.unitPriceMinor * (componentQuantities[item.goodId] ?? 0), 0)
    + newBundleGoods.reduce((sum, item) => sum + (majorToMinor(item.price) ?? 0) * item.quantity, 0);
  const bundleUploadBytes = imageFiles.reduce((sum, file) => sum + file.size, 0)
    + newBundleGoods.reduce((sum, item) => sum + item.imageFiles.reduce((total, file) => total + file.size, 0), 0);
  const normalizedSearch = search.trim().toLocaleLowerCase();
  const visibleGroups = eventGroups.map((group) => {
    const eventMatches = group.event.name.toLocaleLowerCase().includes(normalizedSearch);
    const listings = !normalizedSearch || eventMatches
      ? group.listings
      : group.listings.filter((item) => item.name.toLocaleLowerCase().includes(normalizedSearch));
    const bundles = !normalizedSearch || eventMatches
      ? group.bundles
      : group.bundles.filter((item) => item.name.toLocaleLowerCase().includes(normalizedSearch));
    return { ...group, listings, bundles, eventMatches };
  }).filter((group) => !normalizedSearch || group.eventMatches || group.listings.length > 0 || group.bundles.length > 0 || group.loading);

  const shopGoodCandidates = (() => {
    const map = new Map<string, { goodId: string; name: string; imageURL: string | null; current?: Listing; otherEvents: string[] }>();
    eventGroups.forEach((group) => {
      group.listings.forEach((listing) => {
        const existing = map.get(listing.goodId);
        if (!existing) {
          map.set(listing.goodId, {
            goodId: listing.goodId, name: listing.name, imageURL: listing.imageURL,
            current: group.event.id === eventId ? listing : undefined,
            otherEvents: group.event.id === eventId ? [] : [group.event.name],
          });
          return;
        }
        if (group.event.id === eventId) {
          existing.current = listing;
          existing.imageURL = listing.imageURL ?? existing.imageURL;
          existing.name = listing.name;
          return;
        }
        if (!existing.otherEvents.includes(group.event.name)) existing.otherEvents.push(group.event.name);
      });
    });
    return [...map.values()]
      .filter((candidate) => (componentQuantities[candidate.goodId] ?? 0) === 0)
      .filter((candidate) => !componentPickerSearch.trim() || candidate.name.toLocaleLowerCase().includes(componentPickerSearch.trim().toLocaleLowerCase()))
      .sort((a, b) => Number(Boolean(b.current?.isListed)) - Number(Boolean(a.current?.isListed)));
  })();

  const shopGoodNameIndex = (() => {
    const names = new Map<string, string>();
    eventGroups.forEach((group) => group.listings.forEach((listing) => {
      const key = normalizeGoodName(listing.name);
      if (!names.has(key)) names.set(key, listing.name);
    }));
    return names;
  })();

  function addExistingComponent(goodId: string) {
    setComponentQuantities((current) => {
      if ((current[goodId] ?? 0) > 0) return current;
      if (Object.values(current).filter((quantity) => quantity > 0).length + newBundleGoods.length >= 20) return current;
      return { ...current, [goodId]: 1 };
    });
  }

  return (
    <main>
      <p className="eyebrow">目录与库存</p>
      <h1>商品与优惠</h1>
      <p>商品内容和价格调整后需要平台审核；库存数量变化会即时保存，不会触发重新审核。</p>
      <button className="button button--dark goods-add-button" type="button" onClick={startNewItem}>＋ 添加商品</button>
      <ModalDialog
        ref={addDialogRef}
        kicker="商品目录"
        title={editingBundle ? "编辑固定套餐" : "添加商品或套餐"}
        description="单品可独立出售；固定套餐会和单品一起提交平台审核。"
        labelledById="goods-modal-title"
        onClose={() => setAddOpen(false)}
        closeDisabled={pending}
      >
          <form className="goods-form goods-form--modal" onSubmit={submit}>
        <label className="field" htmlFor="goods-item-kind"><span>{editingBundle ? "编辑类型" : "添加类型"}</span><SelectMenu id="goods-item-kind" ariaLabel={editingBundle ? "编辑类型" : "添加类型"} disabled={Boolean(editingBundle)} value={itemKind} options={[{ value: "good", label: "单个商品" }, { value: "bundle", label: "固定套餐（包含多个商品）" }]} onChange={(value) => {
          const kind = value as "good" | "bundle";
          setItemKind(kind);
          if (kind === "bundle") setCreateEvent(false);
        }} /></label>
        {itemKind === "good" ? <label className="check-field" htmlFor="goods-create-event"><input id="goods-create-event" type="checkbox" checked={createEvent} onChange={(event) => setCreateEvent(event.target.checked)} /><span>同时新建一场活动</span></label> : (
          <p className="form-hint">套餐先选择同一活动中已创建的商品。每件商品仍可单独出售，收银时服务端会自动找出最低组合价。</p>
        )}
        {createEvent && itemKind === "good" ? (
          <section className="goods-event-fields" aria-label="新活动信息">
            <div className="platform-form-divider"><span>活动信息</span><i /></div>
            <label className="field" htmlFor="goods-event-name"><span>活动名称</span><input id="goods-event-name" required value={fresh.name} onChange={(event) => setFresh({ ...fresh, name: event.target.value })} placeholder="漫展或活动名称" /></label>
            <div className="goods-form__dates">
              <label className="field" htmlFor="goods-event-start"><span>开始日期</span><input id="goods-event-start" type="date" required value={fresh.startsOn} onChange={(event) => setFresh({ ...fresh, startsOn: event.target.value })} /></label>
              <label className="field" htmlFor="goods-event-end"><span>结束日期</span><input id="goods-event-end" type="date" required value={fresh.endsOn} onChange={(event) => setFresh({ ...fresh, endsOn: event.target.value })} /></label>
            </div>
            <div className="goods-form__dates">
              <label className="field" htmlFor="goods-event-city"><span>城市</span><input id="goods-event-city" value={fresh.city} onChange={(event) => setFresh({ ...fresh, city: event.target.value })} placeholder="城市" /></label>
              <label className="field" htmlFor="goods-event-venue"><span>场馆</span><input id="goods-event-venue" value={fresh.venue} onChange={(event) => setFresh({ ...fresh, venue: event.target.value })} placeholder="场馆名称" /></label>
            </div>
            <div className="goods-form__dates">
              <label className="field" htmlFor="goods-event-booth"><span>摊位号</span><input id="goods-event-booth" value={fresh.boothNumber} onChange={(event) => setFresh({ ...fresh, boothNumber: event.target.value })} placeholder="例如：A-12" /></label>
              <label className="field" htmlFor="goods-event-currency"><span>币种</span><SelectMenu id="goods-event-currency" value={fresh.currency} options={[{ value: "MYR", label: "MYR" }, { value: "SGD", label: "SGD" }, { value: "CNY", label: "CNY" }]} onChange={(currency) => setFresh({ ...fresh, currency })} /></label>
            </div>
          </section>
        ) : (
          <div className="goods-event-picker">
            <label className="field" htmlFor="goods-event-select"><span>所属活动</span>
              <SelectMenu id="goods-event-select" value={eventId} required disabled={Boolean(editingBundle)} placeholder="选择活动" options={events.map((event) => ({ value: event.id, label: `${event.name} · ${eventStatusLabel(event.status)}` }))} onChange={(value) => { setEventId(value); if (!editingBundle) setComponentQuantities({}); }} />
            </label>
            {eventsLoading && events.length === 0 ? <p role="status">正在读取活动…</p> : null}
            {eventsError ? <p role="alert">活动列表暂时无法读取。<button type="button" onClick={() => void loadAllEvents()}>重试</button></p> : null}
            {!eventsLoading && !eventsError && events.length === 0 ? <p>当前店铺还没有活动，请先新建活动。</p> : null}
            {editingBundle ? <p>编辑套餐时使用原来的活动。</p> : null}
          </div>
        )}
        <div className="platform-form-divider"><span>商品信息</span><i /></div>
        <label className="field" htmlFor="goods-name"><span>商品名称</span><input id="goods-name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="作品或周边名称" /></label>
        <label className="field" htmlFor="goods-description"><span>商品说明 <small>选填</small></span><textarea id="goods-description" rows={3} value={description} onChange={(event) => setDescription(event.target.value)} placeholder="材质、尺寸或版本说明" /></label>
        <section className="goods-image-editor" aria-label="商品相册">
          <div className="goods-image-editor__heading"><strong>{itemKind === "bundle" ? "套餐总图" : "商品相册"}</strong><span>{imageFiles.length}/{itemKind === "bundle" ? 1 : 8} 张 · 第一张作为封面{itemKind === "bundle" && !editingBundle ? ` · 本次图片共 ${formatUploadSize(bundleUploadBytes)} / 48 MiB` : ""}</span></div>
          {editingBundle && !replaceBundleImages && editingBundle.images.length > 0 ? (
            <div className="goods-image-grid">
              {editingBundle.images.map((image, index) => <figure className="goods-image-item" key={image.id}>
                <div className="goods-image-frame"><SafeImage src={image.url} alt={`${editingBundle.name}现有图片 ${index + 1}`} fill unoptimized fallbackClassName="goods-image-frame--fallback" /></div>
                <figcaption><span>{index === 0 ? "封面" : `图片 ${index + 1}`}</span></figcaption>
              </figure>)}
            </div>
          ) : null}
          {editingBundle ? <label className="check-field"><input type="checkbox" checked={replaceBundleImages} onChange={(change) => {
            setReplaceBundleImages(change.target.checked);
            if (!change.target.checked) setImageFiles([]);
          }} /><span>替换现有相册{replaceBundleImages ? "（保存时会用新图片覆盖）" : ""}</span></label> : null}
          {imagePreviews.length > 0 ? (
            <div className="goods-image-grid">
              {imagePreviews.map((url, index) => (
                <figure className="goods-image-item" key={`${imageFiles[index]?.name}-${index}`}>
                  <div className="goods-image-frame"><Image src={url} alt={`商品图片预览 ${index + 1}`} fill unoptimized /></div>
                  <figcaption><span>{index === 0 ? "封面" : `图片 ${index + 1}`}</span><button type="button" aria-label={`移除商品图片 ${index + 1}`} onClick={() => setImageFiles((current) => current.filter((_, at) => at !== index))}>移除</button></figcaption>
                </figure>
              ))}
            </div>
          ) : <p className="goods-image-empty">{itemKind === "bundle" ? "还没有选择套餐总图。" : "还没有选择图片。你可以一次选择多张。"}</p>}
          <p className="field-hint">{imageFiles.length > 0 ? "图片已选择，提交时上传；关闭弹窗前请先保存。" : "选择后先预览，提交时上传。"}</p>
          <div className="image-picker-field"><span>选择图片 <small>JPG、PNG、WebP · 每张最大 6 MiB</small></span>
            <label className="image-picker" htmlFor="goods-images" style={pickerStyle}><span>选择图片</span><input style={pickerInputStyle} aria-label={itemKind === "bundle" ? "选择套餐总图" : "选择商品图片"} ref={imageInputRef} id="goods-images" type="file" accept="image/jpeg,image/png,image/webp" multiple={itemKind === "good"} onChange={selectNewImages} /></label>
          </div>
          {mainImageError ? <p className="form-message form-message--error" role="alert">{mainImageError}</p> : null}
        </section>
        {itemKind === "bundle" ? (
          <fieldset className="bundle-component-picker">
            <legend>套餐组成商品 <small>选择 2 至 20 种，设置每套数量</small></legend>
            {!eventId ? <p className="goods-image-empty">先选择活动。</p> : null}
            {eventId && (selectedGroup?.listings.length ?? 0) === 0 ? <p className="goods-image-empty">本场还没有已上架商品；你也可以直接在套餐中新增单品。</p> : null}
            {selectedGroup && selectedGroup.listings.length > 0 ? <label className="field"><span>搜索本场商品</span><input type="search" value={componentSearch} onChange={(change) => setComponentSearch(change.target.value)} placeholder="输入商品名称" /></label> : null}
            <p className="form-hint">已选 {Object.values(componentQuantities).filter((quantity) => quantity > 0).length + newBundleGoods.length} 种商品；已选商品会排在前面。</p>
            <div className="bundle-component-picker__results">
            {selectedGroup?.listings.filter((item) => item.isListed && (item.remaining > 0 || (componentQuantities[item.goodId] ?? 0) > 0))
              .filter((item) => (componentQuantities[item.goodId] ?? 0) > 0 || !componentSearch.trim() || item.name.toLocaleLowerCase().includes(componentSearch.trim().toLocaleLowerCase()))
              .filter((item, index, rows) => rows.findIndex((row) => row.goodId === item.goodId) === index)
              .sort((a, b) => Number((componentQuantities[b.goodId] ?? 0) > 0) - Number((componentQuantities[a.goodId] ?? 0) > 0))
              .map((item) => (
              <div className="bundle-component-picker__row" key={item.goodId}>
                <input type="checkbox" aria-label={`将 ${item.name} 加入套餐`} checked={(componentQuantities[item.goodId] ?? 0) > 0} onChange={(change) => setComponentQuantities((current) => {
                  if (!change.target.checked) { const next = { ...current }; delete next[item.goodId]; return next; }
                  return { ...current, [item.goodId]: 1 };
                })} />
                <span className="bundle-component-picker__product">
                  {item.imageURL ? <SafeImage src={item.imageURL} alt="" width={44} height={44} unoptimized fallback={<span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />} /> : <span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />}
                  <span><strong>{item.name}</strong><small>{item.reviewStatus === "approved" ? "审核通过" : statusLabel(item.reviewStatus)} · {formatMoney(currency, item.unitPriceMinor)} / 件 · 可售 {item.remaining}</small></span>
                </span>
                {(componentQuantities[item.goodId] ?? 0) > 0 ? <span className="quantity-control" onClick={(event) => event.stopPropagation()}>
                  <button type="button" aria-label={`减少套餐中的 ${item.name}`} onClick={() => setComponentQuantities((current) => ({ ...current, [item.goodId]: Math.max(1, (current[item.goodId] ?? 1) - 1) }))}>−</button>
                  <span>{componentQuantities[item.goodId]}</span>
                  <button type="button" aria-label={`增加套餐中的 ${item.name}`} onClick={() => setComponentQuantities((current) => ({ ...current, [item.goodId]: Math.min(100, (current[item.goodId] ?? 1) + 1) }))}>＋</button>
                </span> : null}
              </div>
            ))}
            </div>
            {<>
              <button className="tier-editor__add" type="button" aria-expanded={componentPickerOpen} onClick={() => setComponentPickerOpen((open) => !open)}>＋ 从店铺现有商品中选择</button>
              {componentPickerOpen ? (
                <div className="bundle-existing-picker" aria-label="店铺现有商品">
                  <div className="bundle-existing-picker__head">
                    <strong>现有商品</strong>
                    <span>{shopGoodCandidates.length} 件未加入套餐</span>
                    <button type="button" onClick={() => setComponentPickerOpen(false)}>收起</button>
                  </div>
                  <input type="search" value={componentPickerSearch} onChange={(change) => setComponentPickerSearch(change.target.value)} placeholder="搜索店铺里的商品" aria-label="搜索店铺现有商品" />
                  <ul className="bundle-existing-picker__list">
                    {shopGoodCandidates.length === 0 ? <li className="bundle-existing-picker__empty">{componentPickerSearch.trim() ? "没有匹配的商品。" : "店铺里的商品都已经在套餐里了。"}</li> : null}
                    {shopGoodCandidates.map((candidate) => {
                      const listing = candidate.current;
                      const blocked = !listing ? "本场未上架" : listing.reviewStatus !== "approved" ? statusLabel(listing.reviewStatus) : !listing.isListed ? "本场已下架" : listing.remaining <= 0 ? "本场已售罄" : "";
                      return (
                        <li key={candidate.goodId}>
                          <button type="button" disabled={Boolean(blocked) || (Object.values(componentQuantities).filter((qty) => qty > 0).length + newBundleGoods.length) >= 20} onClick={() => addExistingComponent(candidate.goodId)}>
                            {candidate.imageURL ? <SafeImage src={candidate.imageURL} alt="" width={40} height={40} unoptimized fallback={<span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />} /> : <span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />}
                            <span className="bundle-existing-picker__copy"><strong>{candidate.name}</strong><small>{listing ? `${formatMoney(listing.currency, listing.unitPriceMinor)} / 件 · 可售 ${listing.remaining}` : `本场未上架 · 见于 ${candidate.otherEvents.join("、") || "其他场次"}`}</small></span>
                            <span className="bundle-existing-picker__state">{blocked || "加入"}</span>
                          </button>
                        </li>
                      );
                    })}
                  </ul>
                  <p className="field-hint">本场没有上架的商品要先在活动里上架，才能放进套餐。</p>
                </div>
              ) : null}
              <button className="tier-editor__add tier-editor__add--quiet" type="button" disabled={Object.values(componentQuantities).filter((qty) => qty > 0).length + newBundleGoods.length >= 20} onClick={() => setNewBundleGoods((current) => [...current, {
                draftKey: `draft-${crypto.randomUUID()}`, name: "", description: "", totalStock: 1, listedQty: 1,
                quantity: 1, price: "10.00", tiers: [], imageFiles: [],
              }])}>没有的货？新增套餐内单品</button>
              {newBundleGoods.map((item, index) => (
                <section className="bundle-component-draft" key={item.draftKey} aria-label={`套餐内新单品 ${index + 1}`}>
                  <div className="bundle-component-draft__heading"><strong>套餐内新单品 {index + 1}</strong><button type="button" onClick={() => setNewBundleGoods((current) => current.filter((draft) => draft.draftKey !== item.draftKey))}>移除</button></div>
                  <label className="field"><span>商品名称</span><input required value={item.name} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, name: change.target.value } : draft))} placeholder="商品名称" /></label>
                  {normalizeGoodName(item.name) && shopGoodNameIndex.has(normalizeGoodName(item.name))
                    ? <p className="form-message form-message--error" role="alert">店里已经有「{shopGoodNameIndex.get(normalizeGoodName(item.name))}」了，请从上方现有商品里加入，不要重复新建。</p>
                    : null}
                  <label className="field"><span>商品说明 <small>选填</small></span><textarea rows={2} value={item.description} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, description: change.target.value } : draft))} /></label>
                  <div className="goods-form__dates">
                    <label className="field"><span>每套数量</span><input type="number" min="1" max="100" step="1" value={item.quantity} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, quantity: Math.max(1, Math.min(100, Number(change.target.value) || 1)) } : draft))} /></label>
                    <label className="field"><span>单品价格 <small>{currency}</small></span><input type="number" min="0.01" step="0.01" value={item.price} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, price: change.target.value } : draft))} /></label>
                  </div>
                  <div className="goods-form__dates">
                    <label className="field"><span>总库存</span><input type="number" min="1" step="1" value={item.totalStock} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, totalStock: Number(change.target.value) } : draft))} /></label>
                    <label className="field"><span>本场可售数</span><input type="number" min={item.quantity} step="1" value={item.listedQty} onChange={(change) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, listedQty: Number(change.target.value) } : draft))} /></label>
                  </div>
                  <PriceTierEditor drafts={item.tiers} currency={currency} basePrice={majorToMinor(item.price) ?? 0} onChange={(next) => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, tiers: next } : draft))} />
                  <div className="draft-gallery">
                    <div className="draft-gallery__heading"><strong>单品图片</strong><span>{item.imageFiles.length} / 8 张{item.imageFiles.length > 0 ? " · 待保存上传" : ""}</span></div>
                    <p>图片属于这个单品，第一张为封面。选择后先预览，保存套餐时上传。</p>
                    {item.imageFiles.length > 0 ? <div className="draft-gallery__images">
                      {item.imageFiles.map((file, imageIndex) => <figure className="draft-gallery__image" key={`${file.name}-${imageIndex}`}>
                        <DraftImagePreview file={file} alt={`${item.name || "新单品"}图片 ${imageIndex + 1}`} />
                        <figcaption><span title={file.name}>{imageIndex === 0 ? "封面 · " : ""}{file.name}</span><button type="button" aria-label={`移除${item.name || "新单品"}图片 ${imageIndex + 1}`} onClick={() => setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, imageFiles: draft.imageFiles.filter((_, at) => at !== imageIndex) } : draft))}>移除</button></figcaption>
                      </figure>)}
                    </div> : <p className="draft-gallery__empty">尚未添加图片</p>}
                    <label className="draft-gallery__upload" style={pickerStyle}><span>＋ 选择单品图片</span><input style={pickerInputStyle} aria-label={`选择套餐内新单品 ${index + 1} 的图片`} type="file" accept="image/jpeg,image/png,image/webp" multiple onChange={(change) => {
                      const input = change.currentTarget;
                      const files = Array.from(input.files ?? []);
                      input.value = "";
                      if (files.length === 0) return;
                      const error = imageSelectionError(files, item.imageFiles.length, 8);
                      setDraftImageErrors((current) => ({ ...current, [item.draftKey]: error }));
                      if (error) return;
                      setNewBundleGoods((current) => current.map((draft) => draft.draftKey === item.draftKey ? { ...draft, imageFiles: [...draft.imageFiles, ...files] } : draft));
                    }} /></label>
                    <small>JPG、PNG、WebP · 每张不超过 6 MiB · 最多 8 张</small>
                    {draftImageErrors[item.draftKey] ? <p className="form-message form-message--error" role="alert">{draftImageErrors[item.draftKey]}</p> : null}
                  </div>
                </section>
              ))}
            </>}
            <p className="bundle-component-picker__subtotal">单品合计：{formatMoney(currency, selectedComponentSubtotal)} / 套</p>
          </fieldset>
        ) : null}
        <label className="field" htmlFor="goods-price"><span>{itemKind === "bundle" ? "整套价格" : "单件价格"} <small>{currency}</small></span><input id="goods-price" type="number" inputMode="decimal" min="0.01" step="0.01" required value={price} onChange={(event) => setPrice(event.target.value)} /></label>
        <p className="field-hint">{itemKind === "bundle" ? "这是完整套餐价格。服务端只会在组件商品和库存足够时应用此优惠。" : "可添加同一商品的数量优惠，例如买 2 件 RM15。"}</p>
        {itemKind === "good" ? <PriceTierEditor drafts={tiers} currency={currency} basePrice={majorToMinor(price) ?? 0} onChange={setTiers} /> : null}
        {itemKind === "good" ? <div className="goods-form__dates">
          <label className="field" htmlFor="goods-total-stock"><span>商品总库存</span><input id="goods-total-stock" type="number" min="0" step="1" required value={totalStock} onChange={(event) => setTotalStock(Number(event.target.value))} /></label>
          <label className="field" htmlFor="goods-listed-qty"><span>本场可售数量</span><input id="goods-listed-qty" type="number" min="0" step="1" required value={listedQty} onChange={(event) => setListedQty(Number(event.target.value))} /></label>
        </div> : null}
        {message ? <p className={`form-message ${messageKind === "error" ? "form-message--error" : "form-message--success"}`} role="status">{message}</p> : null}
        <button className="button button--dark" type="submit" disabled={pending || (!createEvent && !eventId)}>{pending ? "正在提交…" : editingBundle ? "保存并重新提交审核" : itemKind === "bundle" ? "提交套餐审核" : "提交平台审核"}</button>
      </form>
      </ModalDialog>
      {!addOpen && message ? <p className={`form-message ${messageKind === "error" ? "form-message--error" : "form-message--success"}`} role="status">{message}</p> : null}

      <div className="workspace-list-heading">
        <h2>活动与商品</h2>
        <span>{visibleGroups.length} 场 · {visibleGroups.reduce((total, group) => total + group.listings.length + group.bundles.length, 0)} 件商品与套餐</span>
      </div>
      <div className="goods-search">
        <input type="search" aria-label="搜索活动或商品" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索活动名称或商品名称" />
        {search ? <button type="button" onClick={() => setSearch("")}>清除</button> : <span aria-hidden="true">⌕</span>}
      </div>
      {eventsError ? (
        <div className="event-goods-alert" role="alert">
          <span>活动列表暂时无法读取。</span>
          <button className="button button--outline" type="button" disabled={eventsLoading} onClick={() => void loadAllEvents()}>{eventsLoading ? "正在重试…" : "重试"}</button>
        </div>
      ) : null}
      {eventsLoading && eventGroups.length === 0 ? <p className="event-goods-loading" role="status">正在读取活动…</p> : null}
      {!eventsLoading && !eventsError && events.length === 0 ? (
        <div className="empty-state empty-state--compact"><p>先创建一场活动，再为它安排商品和优惠。</p></div>
      ) : null}
      {!eventsLoading && !eventsError && events.length > 0 && normalizedSearch && visibleGroups.length === 0 ? (
        <div className="empty-state empty-state--compact"><p>没有找到匹配的活动或商品，试试其他关键词。</p></div>
      ) : null}
      <div className="event-goods-groups">
        {visibleGroups.map((group) => {
          const { event } = group;
          const dateRange = [event.startsOn, event.endsOn].filter(Boolean).join(" - ");
          const location = [event.city, event.venue, event.boothNumber ? `摊位 ${event.boothNumber}` : ""].filter(Boolean).join(" · ");
          const details = [dateRange, location, event.currency].filter(Boolean).join(" · ");
          const eventStatus = event.status ?? "scheduled";
          return (
            <section className="event-goods-group" key={event.id} aria-labelledby={`event-goods-${event.id}`}>
              <header className="event-goods-group__heading">
                <div>
                  <p className="eyebrow">活动商品与套餐 · {group.listings.length} 件商品 · {group.bundles.length} 套套餐</p>
                  <h3 id={`event-goods-${event.id}`}>{event.name}</h3>
                  {details ? <p>{details}</p> : null}
                </div>
                <span className={`event-status event-status--${eventStatus}`}><i aria-hidden="true" />{eventStatusLabel(eventStatus)}</span>
              </header>
              {group.loading && group.listings.length === 0 ? <p className="event-goods-loading" role="status">正在读取本场商品…</p> : null}
              {group.loading && group.listings.length > 0 ? <p className="event-goods-loading" role="status">正在更新本场商品…</p> : null}
              {group.error ? (
                <div className="event-goods-alert" role="alert">
                  <span>{group.error}</span>
                  <button className="button button--outline" type="button" disabled={group.loading} onClick={() => void loadEventListings(event)}>{group.loading ? "正在重试…" : "重试本场"}</button>
                </div>
              ) : null}
              {!group.loading && !group.error && group.listings.length === 0 ? <p className="event-goods-empty">本场还没有上架商品。</p> : null}
              {group.bundles.length > 0 ? (
                <ul className="listing-admin-list listing-admin-list--bundles">
                  {group.bundles.map((bundle) => {
                    const soldOut = bundle.remaining <= 0;
                    return <li className={`listing-admin-card bundle-admin-card${soldOut ? " listing-admin-card--soldout" : ""}`} key={bundle.bundleListingId}>
                    <div className="listing-admin-card__heading">
                      <div><h3>{bundle.name}</h3><p>固定套餐 · {bundle.remaining} 套可售 · {formatMoney(event.currency, bundle.unitPriceMinor)} / 套</p></div>
                      <span className={`review-badge review-badge--${bundle.reviewStatus}`}>{statusLabel(bundle.reviewStatus)}</span>
                    </div>
                    {soldOut ? <p className="listing-admin-card__stockout" role="status">本场已售罄</p> : null}
                    {bundle.reviewNote ? <p className="review-note">平台意见：{bundle.reviewNote}</p> : null}
                    <p className="bundle-admin-card__components">包含：{bundle.components.map((component) => `${component.name} × ${component.quantity}`).join(" · ")}</p>

                    {bundle.images.length > 0 ? <div className="bundle-admin-card__images">{bundle.images.map((image, index) => <SafeImage key={image.id} src={image.url} alt={`${bundle.name}图片 ${index + 1}`} width={72} height={72} unoptimized fallbackClassName="bundle-admin-card__image--fallback" />)}</div> : null}
                    <button className="button button--outline" type="button" disabled={pending} onClick={() => editBundle(bundle)}>编辑套餐</button>
                  </li>;
                  })}
                </ul>
              ) : null}
              {group.listings.length > 0 ? (
                <ul className="listing-admin-list">
                  {group.listings.map((item) => {
                    const draft = drafts[item.listingId];
                    if (!draft) return null;
                    const soldOut = item.remaining <= 0;
                    return (
                      <li className={`listing-admin-card${soldOut ? " listing-admin-card--soldout" : ""}`} key={item.listingId}>
                        <div className="listing-admin-card__heading">
                          <div className="listing-admin-card__identity">
                            <span className="listing-admin-card__thumb" aria-hidden="true">
                              {item.imageURL ? <SafeImage src={item.imageURL} alt="" width={64} height={64} unoptimized fallback={<em>无图</em>} /> : <em>无图</em>}
                            </span>
                            <div><h3>{item.name}</h3><p>单品 · {formatMoney(item.currency, item.unitPriceMinor)} · {item.remaining} 件可售</p></div>
                          </div>
                          <span className={`review-badge review-badge--${item.reviewStatus}`}>{statusLabel(item.reviewStatus)}</span>
                        </div>
                        {soldOut ? <p className="listing-admin-card__stockout" role="status">本场已售罄</p> : null}
                        {item.reviewNote ? <p className="review-note">平台意见：{item.reviewNote}</p> : null}
                        <button type="button" className="button button--outline" onClick={() => { setCopyEvent(""); setCopyQty("0"); setRestockQty(""); setEditingListing(item.listingId); }}>编辑商品</button>
                                              </li>
                    );
                  })}
                </ul>
              ) : null}
            </section>
          );
        })}
      </div>

      <ModalDialog
        ref={listingDialogRef}
        className="goods-modal--listing"
        kicker="本场商品"
        title={editingItem ? editingItem.name : "编辑商品"}
        description="资料、价格与库存分配，一处改完。"
        labelledById="listing-modal-title"
        onClose={() => setEditingListing(null)}
        closeDisabled={pending}
      >
        {editingItem && editingDraft ? <div className="goods-form goods-form--modal listing-modal-body">

                        <section className="listing-admin-section listing-admin-section--stock">
                          <h4>商品资料与库存</h4>
                          <div className="goods-image-editor" aria-label={`${editingItem.name}商品相册`}>
                            <div className="goods-image-editor__heading"><strong>商品相册</strong><span>{editingItem.images.length}/8 张 · 第一张作为封面</span></div>
                            {editingItem.images.length > 0 ? (
                              <div className="goods-image-grid">
                                {editingItem.images.map((image, index) => (
                                  <figure className="goods-image-item" key={image.id}>
                                    <div className="goods-image-frame"><SafeImage src={image.url} alt={`${editingItem.name}，图片 ${index + 1}`} fill unoptimized fallbackClassName="goods-image-frame--fallback" /></div>
                                    <figcaption>
                                      <span>{index === 0 ? "封面" : `图片 ${index + 1}`}</span>
                                      <div>
                                        <button type="button" aria-label={`商品图片 ${index + 1} 上移`} disabled={imagePending === editingItem.goodId || index === 0} onClick={() => void moveProductImage(editingItem, index, -1)}>↑</button>
                                        <button type="button" aria-label={`商品图片 ${index + 1} 下移`} disabled={imagePending === editingItem.goodId || index === editingItem.images.length - 1} onClick={() => void moveProductImage(editingItem, index, 1)}>↓</button>
                                        <button type="button" aria-label={`删除商品图片 ${index + 1}`} disabled={imagePending === editingItem.goodId} onClick={() => setImageDeleteTarget({ item: editingItem, image })}>删除</button>
                                      </div>
                                    </figcaption>
                                  </figure>
                                ))}
                              </div>
                            ) : <p className="goods-image-empty">还没有商品图片。</p>}
                            <div className="image-picker-field"><span>{imagePending === editingItem.goodId ? "正在更新相册…" : "添加图片"} <small>JPG、PNG、WebP · 每张最大 6 MiB</small></span>
                              <label className="image-picker" style={pickerStyle} data-disabled={imagePending === editingItem.goodId}><span>选择图片</span><input style={pickerInputStyle} aria-label={`为${editingItem.name}添加图片`} type="file" accept="image/jpeg,image/png,image/webp" multiple disabled={imagePending === editingItem.goodId} onChange={(change) => {
                                const input = change.currentTarget;
                                const files = Array.from(input.files ?? []);
                                input.value = "";
                                void addProductImages(editingItem, files);
                              }} /></label>
                            </div>
                            {existingImageErrors[editingItem.goodId] ? <p className="form-message form-message--error" role="alert">{existingImageErrors[editingItem.goodId]}</p> : null}
                          </div>
                          <label className="field"><span>商品名称</span><input value={editingDraft.name} onChange={(change) => setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, name: change.target.value } }))} /></label>
                          <label className="field"><span>商品说明</span><textarea rows={2} value={editingDraft.description} onChange={(change) => setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, description: change.target.value } }))} /></label>
                          {(() => {
                            const total = Math.max(1, editingLive.total);
                            const soldPct = Math.min(100, (editingLive.sold / total) * 100);
                            const allocPct = Math.min(100 - soldPct, (editingLive.allocated / total) * 100);
                            return <div className="stock-pool">
                              <div className="stock-total"><span>商品总库存<em className="field-tag">全店</em></span><b>{editingStock.total}<i>件</i></b></div>
                              <div className="stock-restock">
                                <label className="field"><span className="field-label">新增库存<em className="field-limit">件</em></span><input type="number" min="1" step="1" inputMode="numeric" placeholder="0" value={restockQty} onChange={(change) => setRestockQty(change.target.value)} /></label>
                                <button type="button" className="button button--dark" disabled={pending || !(Number(restockQty) > 0)} onClick={() => void restockItem(editingItem)}>入库</button>
                              </div>
                              <div className="stock-meter" role="img" aria-label={`总库存 ${editingLive.total} 件，已售 ${editingLive.sold} 件，已分配 ${editingLive.allocated} 件，可分配 ${editingLive.remain} 件`}>
                                <div className="stock-meter__bar"><i className="is-sold" style={{ width: `${soldPct}%` }} /><i className="is-alloc" style={{ width: `${allocPct}%` }} /></div>
                                <div className="stock-meter__figures">
                                  <div><b>{editingLive.sold}</b><span>已售</span></div>
                                  <div><b>{editingLive.allocated}</b><span>已分配</span></div>
                                  <div className="is-free"><b>{editingLive.remain}</b><span>可分配</span></div>
                                </div>
                              </div>
                            </div>;
                          })()}
                        </section>
                        <section className="listing-admin-section">
                          <h4>活动上架与价格</h4>
                          <label className="check-field"><input type="checkbox" checked={editingDraft.isListed} onChange={(change) => setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, isListed: change.target.checked } }))} /><span>在本场公开售卖</span></label>
                          <label className="field"><span>基础单价 <small>{editingItem.currency}</small></span><input type="number" min="0.01" step="0.01" inputMode="decimal" value={editingDraft.price} onChange={(change) => setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, price: change.target.value } }))} /></label>
                          <PriceTierEditor drafts={editingDraft.tiers} currency={editingItem.currency} basePrice={editingPriceMinor} onChange={(next) => setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, tiers: next } }))} />
                          <label className="field"><span className="field-label">本场可售总数<em className="field-limit">上限 {editingItem.listedQty + editingStock.remain}</em></span><input type="number" min={editingItem.soldQty} max={editingItem.listedQty + editingStock.remain} step="1" value={editingDraft.listedQty} onChange={(change) => {
                            const raw = change.target.value;
                            const ceiling = editingItem.listedQty + editingStock.remain;
                            const next = raw === "" ? "" : String(Math.min(Math.max(Number(raw), editingItem.soldQty), ceiling));
                            setDrafts((prev) => ({ ...prev, [editingItem.listingId]: { ...editingDraft, listedQty: next } }));
                          }} /></label>
                        </section>

                        <section className="listing-admin-section listing-admin-section--alloc"><h4>复制到其他活动</h4>
                          <label className="field"><span>目标活动</span><SelectMenu value={copyEvent} placeholder="选择活动" options={events.filter((event) => event.id !== editingItem.eventId).map((event) => ({ value: event.id, label: event.name }))} onChange={setCopyEvent} /></label>
                          <label className="field"><span className="field-label">目标活动分配数量<em className={`field-limit${editingCopyOver ? " is-over" : ""}`}>{editingCopyOver ? "超出" : "上限"} {editingCopyOver ? Math.round(Number(copyQty)) - editingStock.remain : editingStock.remain}</em></span><input type="number" min="0" step="1" max={editingStock.remain} value={copyQty} onChange={(change) => {
                            const raw = change.target.value;
                            setCopyQty(raw === "" ? "" : String(Math.min(Number(raw), editingStock.remain)));
                          }} /></label>
                          <button type="button" className="button button--outline" disabled={pending || !copyEvent || editingCopyOver} onClick={() => void copyListing(editingItem)}>复制商品</button>
                          {message ? <p className="form-message" role="status">{message}</p> : null}
                        </section>
                          <footer className="goods-modal__footer">
                            <button type="button" className="button button--outline" disabled={pending} onClick={() => setEditingListing(null)}>取消</button>
                            <button type="button" className="button button--dark" disabled={pending} onClick={() => void saveAll(editingItem)}>{pending ? "正在保存…" : "保存修改"}</button>
                          </footer>
        </div> : null}
      </ModalDialog>
      <ConfirmDialog
        open={imageDeleteTarget !== null}
        title="删除商品图片"
        description="删除后该图片不再展示，商品资料会重新提交平台审核；此操作不可撤销。"
        confirmLabel="删除图片"
        danger
        pending={imagePending !== ""}
        onConfirm={() => imageDeleteTarget && void deleteProductImage(imageDeleteTarget.item, imageDeleteTarget.image).then(() => setImageDeleteTarget(null))}
        onCancel={() => setImageDeleteTarget(null)}
      />
    </main>
  );
}
