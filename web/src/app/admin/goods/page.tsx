"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
type GoodListing = {
  listingId: string; eventId: string; eventName: string; currency: string;
  unitPriceMinor: number; priceTiers: PriceTier[]; soldQty: number; reservedQty: number;
  remaining: number; isListed: boolean;
};
type ShopGood = {
  id: string; name: string; description: string; reviewStatus: string; reviewNote: string; revision: number;
  totalStock: number; soldQty: number; reservedQty: number; availableQty: number;
  listings: GoodListing[]; images: ProductImage[]; imageURL: string | null;
};
type Listing = GoodListing & {
  goodId: string; name: string; description: string; totalStock: number;
  reviewStatus: string; reviewNote: string; imageURL: string | null; images: ProductImage[];
};
type BundleComponent = { goodId: string; listingId: string; name: string; quantity: number; unitPriceMinor?: number };
type Bundle = {
  bundleListingId: string; bundleId: string; eventId: string; name: string; description: string;
  unitPriceMinor: number; priceTiers: PriceTier[]; remaining: number; isListed: boolean;
  reviewStatus: string; reviewNote: string; revision: number; components: BundleComponent[]; images: ProductImage[];
};
type BundleGoodDraftForm = {
  draftKey: string; name: string; description: string; totalStock: number;
  quantity: number; price: string; tiers: TierDraft[]; imageFiles: File[];
};
type ListingDraft = { price: string; tiers: TierDraft[]; isListed: boolean };
type GoodDraft = { name: string; description: string; listings: Record<string, ListingDraft> };
type EventGoodsGroup = { event: EventRow; listings: Listing[]; bundles: Bundle[] };

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

function goodStockPool(good: ShopGood) {
  return { total: good.totalStock, sold: good.soldQty, reserved: good.reservedQty, free: Math.max(0, good.totalStock - good.reservedQty) };
}

function StockMeter({ total, sold, reserved, free }: { total: number; sold: number; reserved: number; free: number }) {
  const base = Math.max(1, total);
  const soldPct = Math.min(100, (sold / base) * 100);
  const reservedPct = Math.min(100 - soldPct, (reserved / base) * 100);
  return (
    <div className="stock-meter" role="img" aria-label={`总库存 ${total} 件，已售 ${sold} 件，预占 ${reserved} 件，可售 ${free} 件`}>
      <div className="stock-meter__bar"><i className="is-sold" style={{ width: `${soldPct}%` }} /><i className="is-alloc" style={{ width: `${reservedPct}%` }} /></div>
      <div className="stock-meter__figures">
        <div><b>{sold}</b><span>已售</span></div>
        <div><b>{reserved}</b><span>预占</span></div>
        <div className="is-free"><b>{free}</b><span>可售</span></div>
      </div>
    </div>
  );
}

export default function GoodsPage() {
  const [events, setEvents] = useState<EventRow[]>([]);
  const [goods, setGoods] = useState<ShopGood[]>([]);
  const [bundlesByEvent, setBundlesByEvent] = useState<Record<string, Bundle[]>>({});
  const [catalogLoading, setCatalogLoading] = useState(true);
  const [catalogError, setCatalogError] = useState(false);
  const [view, setView] = useState<"goods" | "events">("goods");
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
  const [imageFiles, setImageFiles] = useState<File[]>([]);
  const [imagePreviews, setImagePreviews] = useState<string[]>([]);
  const [message, setMessageText] = useState("");
  const [messageKind, setMessageKind] = useState<"success" | "error">("success");
  const setMessage = (text: string, kind: "success" | "error" = "success") => { setMessageText(text); setMessageKind(kind); };
  const [pending, setPending] = useState(false);
  const [imagePending, setImagePending] = useState("");
  const [imageDeleteTarget, setImageDeleteTarget] = useState<{ good: ShopGood; image: ProductImage } | null>(null);
  const [fresh, setFresh] = useState<FreshEvent>(emptyEvent);
  const [search, setSearch] = useState("");
  const [addOpen, setAddOpen] = useState(false);
  const [editingGoodId, setEditingGoodId] = useState<string | null>(null);
  const [goodDraft, setGoodDraft] = useState<GoodDraft | null>(null);
  const [restockQty, setRestockQty] = useState("");
  const [addEventId, setAddEventId] = useState("");
  const [addEventPrice, setAddEventPrice] = useState("");
  const addDialogRef = useRef<HTMLDialogElement>(null);
  const goodDialogRef = useRef<HTMLDialogElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);
  const catalogVersion = useRef(0);

  useEffect(() => {
    const dialog = goodDialogRef.current;
    if (!dialog) return;
    if (editingGoodId && !dialog.open) dialog.showModal();
    if (!editingGoodId && dialog.open) dialog.close();
  }, [editingGoodId]);

  const loadCatalog = useCallback(async (isCurrent: () => boolean = () => true) => {
    const version = ++catalogVersion.current;
    const current = () => isCurrent() && catalogVersion.current === version;
    setCatalogLoading(true);
    setCatalogError(false);
    try {
      const [eventRes, goodRes] = await Promise.all([
        browserJSON("/api/staff/events"),
        browserJSON("/api/staff/goods"),
      ]);
      if (!current()) return [];
      if (!eventRes.ok || !goodRes.ok) throw new Error("catalog unavailable");
      const eventRows = Array.isArray(eventRes.data) ? (eventRes.data as unknown as EventRow[]) : [];
      const goodRows = Array.isArray(goodRes.data)
        ? (goodRes.data as unknown as ShopGood[]).map((good) => ({ ...good, images: good.images ?? [], listings: good.listings ?? [] }))
        : [];
      setEvents(eventRows);
      setGoods(goodRows);
      const bundleEntries = await Promise.all(eventRows.map(async (event) => {
        const res = await browserJSON(`/api/staff/events/${event.id}/bundles`);
        const rows = res.ok && Array.isArray(res.data)
          ? (res.data as unknown as Bundle[]).map((bundle) => ({ ...bundle, images: bundle.images ?? [], components: bundle.components ?? [] }))
          : [];
        return [event.id, rows] as const;
      }));
      if (!current()) return [];
      setBundlesByEvent(Object.fromEntries(bundleEntries));
      return eventRows;
    } catch {
      if (current()) setCatalogError(true);
      return [];
    } finally {
      if (current()) setCatalogLoading(false);
    }
  }, []);

  const reloadGoods = useCallback(async () => {
    const res = await browserJSON("/api/staff/goods");
    if (res.ok && Array.isArray(res.data)) {
      setGoods((res.data as unknown as ShopGood[]).map((good) => ({ ...good, images: good.images ?? [], listings: good.listings ?? [] })));
    }
  }, []);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => { void loadCatalog(() => active); }, 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [loadCatalog]);

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

  const eventGroups: EventGoodsGroup[] = useMemo(() => events.map((event) => ({
    event,
    listings: goods.flatMap((good) => good.listings
      .filter((listing) => listing.eventId === event.id)
      .map((listing) => ({
        ...listing, goodId: good.id, name: good.name, description: good.description, totalStock: good.totalStock,
        reviewStatus: good.reviewStatus, reviewNote: good.reviewNote, imageURL: good.imageURL, images: good.images,
      }))),
    bundles: bundlesByEvent[event.id] ?? [],
  })), [events, goods, bundlesByEvent]);

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
      goods.forEach((good) => {
        const key = normalizeGoodName(good.name);
        if (!shopGoodNames.has(key)) shopGoodNames.set(key, good.name);
      });
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
          || !Number.isInteger(item.totalStock) || item.totalStock < 1 || item.imageFiles.length > 8) {
          setMessage(`请完成套餐内商品“${item.name || "未命名"}”的名称、库存和价格设置。`, "error");
          return;
        }
        components.push({
          draftKey: item.draftKey,
          quantity: item.quantity,
          draft: {
            name: item.name.trim(), description: item.description, totalStock: item.totalStock,
            unitPriceMinor: draftUnitMinor, priceTiers: draftTiers, isListed: true,
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
        await loadCatalog();
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
      name, description, totalStock, isListed: true,
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
      await loadCatalog();
      if (createEvent) {
        const createdEvent = String(result.eventId ?? "");
        if (createdEvent) setEventId(createdEvent);
        setCreateEvent(false);
        setFresh(emptyEvent);
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
    setComponentQuantities({});
    setNewBundleGoods([]);
    setComponentSearch("");
    setImageFiles([]);
    if (imageInputRef.current) imageInputRef.current.value = "";
    setMessage("");
    setAddOpen(true);
  }

  function openGood(good: ShopGood) {
    setEditingGoodId(good.id);
    setGoodDraft({
      name: good.name,
      description: good.description ?? "",
      listings: Object.fromEntries(good.listings.map((listing) => [listing.listingId, {
      price: minorInput(listing.unitPriceMinor),
      tiers: (listing.priceTiers ?? []).map(tierDraft), isListed: listing.isListed,
    }])),
    });
    setRestockQty("");
    setAddEventId("");
    setAddEventPrice(good.listings[0] ? minorInput(good.listings[0].unitPriceMinor) : "19.90");
    setMessage("");
  }

  function closeGood() {
    setEditingGoodId(null);
    setGoodDraft(null);
  }

  async function addGoodImages(good: ShopGood, files: File[]) {
    if (files.length === 0) return;
    const error = imageSelectionError(files, good.images.length, 8);
    setExistingImageErrors((current) => ({ ...current, [good.id]: error }));
    if (error) return;
    setImagePending(good.id);
    setMessage("");
    try {
      const form = new FormData();
      files.forEach((file) => form.append("images", file));
      const response = await browserFormData(`/api/staff/goods/${good.id}/images`, "POST", form);
      if (!response.ok) {
        setExistingImageErrors((current) => ({ ...current, [good.id]: describeRequest(response) }));
        return;
      }
      await reloadGoods();
      setMessage("商品相册已更新，正在等待平台审核。");
    } catch {
      setExistingImageErrors((current) => ({ ...current, [good.id]: "无法连接服务，图片没有上传。" }));
    } finally {
      setImagePending("");
    }
  }

  async function deleteGoodImage(good: ShopGood, image: ProductImage) {
    setImagePending(good.id);
    setMessage("");
    try {
      const response = await browserJSON(`/api/staff/goods/${good.id}/images/${image.id}`, "DELETE");
      if (!response.ok) {
        setMessage(describeRequest(response), "error");
        return;
      }
      await reloadGoods();
      setMessage("商品相册已更新，正在等待平台审核。");
    } catch {
      setMessage("无法连接服务，图片没有删除。", "error");
    } finally {
      setImagePending("");
    }
  }

  async function moveGoodImage(good: ShopGood, index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= good.images.length) return;
    const images = [...good.images];
    [images[index], images[target]] = [images[target], images[index]];
    setImagePending(good.id);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${good.id}/images`, "PATCH", { imageIds: images.map((image) => image.id) });
      if (!res.ok) {
        setMessage("图片顺序没有保存，请刷新后重试。", "error");
        return;
      }
      await reloadGoods();
      setMessage("封面顺序已更新，正在等待平台审核。");
    } catch {
      setMessage("无法连接服务，图片顺序没有保存。", "error");
    } finally {
      setImagePending("");
    }
  }

  async function restockGood(good: ShopGood) {
    const add = Number(restockQty);
    if (!Number.isInteger(add) || add <= 0) { setMessage("请输入要新增的库存件数。", "error"); return; }
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${good.id}`, "PATCH", {
        name: good.name, description: good.description, totalStock: good.totalStock + add,
      });
      if (!res.ok) { setMessage("库存没有增加，请稍后重试。", "error"); return; }
      setRestockQty("");
      const nextTotal = good.totalStock + add;
      setGoods((prev) => prev.map((row) => row.id === good.id ? { ...row, totalStock: nextTotal } : row));
      setMessage(`已入库 ${add} 件，全店共享库存即时生效。`);
    } catch {
      setMessage("无法连接服务，请稍后重试。", "error");
    } finally {
      setPending(false);
    }
  }

  async function addGoodToEvent(good: ShopGood) {
    const priceMinor = majorToMinor(addEventPrice);
    if (!addEventId) {
      setMessage("请选择目标活动。", "error");
      return;
    }
    if (priceMinor == null || priceMinor <= 0) { setMessage("请填写该活动的有效单价。", "error"); return; }
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${good.id}/listings`, "POST", {
        eventId: addEventId, unitPriceMinor: priceMinor, priceTiers: [], isListed: true,
      });
      if (!res.ok) {
        setMessage(res.data.error === "conflict" ? "该活动已经有这件商品了。"
          : "加入活动失败，请稍后重试。", "error");
        return;
      }
      const newListingId = String(res.data.id ?? "");
      await reloadGoods();
      if (newListingId) {
        setGoodDraft((prev) => prev ? {
          ...prev,
          listings: { ...prev.listings, [newListingId]: { price: minorInput(priceMinor), tiers: [], isListed: true } },
        } : prev);
      }
      setAddEventId("");
      setMessage("已加入新活动并即时上架；弹窗里其他修改仍需点「保存修改」生效。");
    } catch {
      setMessage("无法连接服务，请检查网络后重试。", "error");
    } finally {
      setPending(false);
    }
  }

  function releaseListingQuota(listing: GoodListing) {
    setGoodDraft((prev) => {
      if (!prev) return prev;
      const draft = prev.listings[listing.listingId];
      if (!draft) return prev;
      return {
        ...prev,
        listings: {
          ...prev.listings,
          [listing.listingId]: { ...draft, isListed: false },
        },
      };
    });
  }

  async function saveGood(good: ShopGood) {
    if (!goodDraft) return;
    if (!goodDraft.name.trim()) { setMessage("商品名称不能为空。", "error"); return; }
    const listings: Record<string, unknown>[] = [];
    for (const listing of good.listings) {
      const draft = goodDraft.listings[listing.listingId];
      if (!draft) continue;
      const unitPriceMinor = majorToMinor(draft.price);
      const priceTiers = unitPriceMinor == null ? null : parseTiers(draft.tiers, unitPriceMinor);
      if (unitPriceMinor == null || unitPriceMinor <= 0 || priceTiers === null) {
        setMessage(`「${listing.eventName}」的价格或优惠包无效，请修正后保存。`, "error");
        return;
      }
      listings.push({ listingId: listing.listingId, unitPriceMinor, priceTiers, isListed: draft.isListed });
    }
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON(`/api/staff/goods/${good.id}`, "PATCH", {
        name: goodDraft.name.trim(), description: goodDraft.description, totalStock: good.totalStock, listings,
      });
      if (!res.ok) {
        setMessage(res.data.error === "stock_exceeded" ? "总库存不能低于待支付预占数量，本次保存整体未生效；请先入库或等待预占释放。"
          : res.data.error === "invalid_price_tiers" ? "优惠包总价必须低于对应数量的单买价格，本次保存整体未生效。"
            : `保存失败，所有修改都没有生效：${describeRequest(res)}`, "error");
        return;
      }
      closeGood();
      await loadCatalog();
      setMessage("商品已保存，资料与全部活动的上架设置一次性生效。");
    } catch {
      setMessage("无法连接服务，本次保存整体未生效，请重试。", "error");
    } finally {
      setPending(false);
    }
  }

  const editingGood = editingGoodId ? goods.find((good) => good.id === editingGoodId) ?? null : null;
  const editingPool = editingGood ? goodStockPool(editingGood) : { total: 0, sold: 0, reserved: 0, free: 0 };
  const addableEvents = editingGood
    ? events.filter((event) => !editingGood.listings.some((listing) => listing.eventId === event.id))
    : [];

  const selectedEvent = events.find((event) => event.id === eventId);
  const selectedGroup = eventGroups.find((group) => group.event.id === eventId);
  const currency = createEvent ? fresh.currency : selectedEvent?.currency ?? "MYR";
  const selectedComponentSubtotal = (selectedGroup?.listings ?? []).reduce((sum, item) =>
    sum + item.unitPriceMinor * (componentQuantities[item.goodId] ?? 0), 0)
    + newBundleGoods.reduce((sum, item) => sum + (majorToMinor(item.price) ?? 0) * item.quantity, 0);
  const bundleUploadBytes = imageFiles.reduce((sum, file) => sum + file.size, 0)
    + newBundleGoods.reduce((sum, item) => sum + item.imageFiles.reduce((total, file) => total + file.size, 0), 0);
  const normalizedSearch = search.trim().toLocaleLowerCase();
  const visibleGoods = goods.filter((good) => !normalizedSearch
    || good.name.toLocaleLowerCase().includes(normalizedSearch)
    || good.listings.some((listing) => listing.eventName.toLocaleLowerCase().includes(normalizedSearch)));
  const visibleGroups = eventGroups.map((group) => {
    const eventMatches = group.event.name.toLocaleLowerCase().includes(normalizedSearch);
    const listings = !normalizedSearch || eventMatches
      ? group.listings
      : group.listings.filter((item) => item.name.toLocaleLowerCase().includes(normalizedSearch));
    const bundles = !normalizedSearch || eventMatches
      ? group.bundles
      : group.bundles.filter((item) => item.name.toLocaleLowerCase().includes(normalizedSearch));
    return { ...group, listings, bundles, eventMatches };
  }).filter((group) => !normalizedSearch || group.eventMatches || group.listings.length > 0 || group.bundles.length > 0);
  const bundleCount = Object.values(bundlesByEvent).reduce((sum, rows) => sum + rows.length, 0);

  const shopGoodCandidates = (() => {
    return goods.map((good) => {
      const current = good.listings.find((listing) => listing.eventId === eventId);
      const otherEvents = good.listings.filter((listing) => listing.eventId !== eventId).map((listing) => listing.eventName);
      const listing: Listing | undefined = current ? {
        ...current, goodId: good.id, name: good.name, description: good.description, totalStock: good.totalStock,
        reviewStatus: good.reviewStatus, reviewNote: good.reviewNote, imageURL: good.imageURL, images: good.images,
      } : undefined;
      return { goodId: good.id, name: good.name, imageURL: good.imageURL, current: listing, otherEvents };
    })
      .filter((candidate) => (componentQuantities[candidate.goodId] ?? 0) === 0)
      .filter((candidate) => !componentPickerSearch.trim() || candidate.name.toLocaleLowerCase().includes(componentPickerSearch.trim().toLocaleLowerCase()))
      .sort((a, b) => Number(Boolean(b.current?.isListed)) - Number(Boolean(a.current?.isListed)));
  })();

  const shopGoodNameIndex = (() => {
    const names = new Map<string, string>();
    goods.forEach((good) => {
      const key = normalizeGoodName(good.name);
      if (!names.has(key)) names.set(key, good.name);
    });
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
      <p>一件商品只维护一份资料，全店库存共享；同一件商品可以参加多场活动，各活动单独定价与分配。商品内容和价格调整后需要平台审核，库存数量变化即时保存。</p>
      <ModalDialog
        ref={addDialogRef}
        kicker="商品目录"
        title={editingBundle ? "编辑固定套餐" : "添加商品或套餐"}
        description="单品可独立出售；固定套餐会和单品一起提交平台审核。"
        labelledById="goods-modal-title"
        onClose={() => setAddOpen(false)}
        closeDisabled={pending}
      >
          <form className="goods-form goods-form--modal" aria-labelledby="goods-modal-title" onSubmit={submit}>
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
            {catalogLoading && events.length === 0 ? <p role="status">正在读取活动…</p> : null}
            {catalogError ? <p role="alert">活动列表暂时无法读取。<button type="button" onClick={() => void loadCatalog()}>重试</button></p> : null}
            {!catalogLoading && !catalogError && events.length === 0 ? <p>当前店铺还没有活动，请先新建活动。</p> : null}
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
                draftKey: `draft-${crypto.randomUUID()}`, name: "", description: "", totalStock: 1,
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
        </div> : null}
        {message ? <p className={`form-message ${messageKind === "error" ? "form-message--error" : "form-message--success"}`} role="status">{message}</p> : null}
        <footer className="goods-modal__footer">
          <button className="button button--outline" type="button" disabled={pending} onClick={() => addDialogRef.current?.close()}>取消</button>
          <button className="button button--dark" type="submit" disabled={pending || (!createEvent && !eventId)}>{pending ? "正在提交…" : editingBundle ? "保存并重新提交审核" : itemKind === "bundle" ? "提交套餐审核" : "提交平台审核"}</button>
        </footer>
      </form>
      </ModalDialog>
      {!addOpen && !editingGoodId && message ? <p className={`form-message ${messageKind === "error" ? "form-message--error" : "form-message--success"}`} role="status">{message}</p> : null}

      <div className="workspace-list-heading">
        <h2>{view === "goods" ? "全部商品" : "活动与商品"}</h2>
        <span>{view === "goods" ? `${goods.length} 件商品 · ${bundleCount} 套套餐` : `${events.length} 场 · ${goods.length} 件商品 · ${bundleCount} 套套餐`}</span>
      </div>
      <div className="goods-toolbar">
        <div className="goods-search">
          <input type="search" aria-label="搜索商品或活动" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索商品名称或活动名称" />
          {search ? <button type="button" onClick={() => setSearch("")}>清除</button> : <span aria-hidden="true">⌕</span>}
        </div>
        <div className="goods-view-tabs" role="group" aria-label="商品浏览方式">
          <button type="button" aria-pressed={view === "goods"} className={view === "goods" ? "is-active" : ""} onClick={() => setView("goods")}>按商品</button>
          <button type="button" aria-pressed={view === "events"} className={view === "events" ? "is-active" : ""} onClick={() => setView("events")}>按活动</button>
        </div>
        <button className="button button--dark goods-add-button" type="button" onClick={startNewItem}>＋ 添加商品或套餐</button>
      </div>
      {catalogError ? (
        <div className="event-goods-alert" role="alert">
          <span>商品目录暂时无法读取。</span>
          <button className="button button--outline" type="button" disabled={catalogLoading} onClick={() => void loadCatalog()}>{catalogLoading ? "正在重试…" : "重试"}</button>
        </div>
      ) : null}
      {catalogLoading && goods.length === 0 && events.length === 0 ? <p className="event-goods-loading" role="status">正在读取商品目录…</p> : null}
      {!catalogLoading && !catalogError && events.length === 0 ? (
        <div className="empty-state empty-state--compact"><p>先创建一场活动，再为它安排商品和优惠。</p></div>
      ) : null}

      {view === "goods" ? (
        <>
          {!catalogLoading && !catalogError && goods.length === 0 && events.length > 0 ? (
            <div className="empty-state empty-state--compact"><p>店里还没有商品，点上方「添加商品」创建第一件。</p></div>
          ) : null}
          {goods.length > 0 && normalizedSearch && visibleGoods.length === 0 ? (
            <div className="empty-state empty-state--compact"><p>没有找到匹配的商品，试试其他关键词。</p></div>
          ) : null}
          <ul className="goods-card-grid">
            {visibleGoods.map((good) => {
              const pool = goodStockPool(good);
              const activeListings = good.listings.filter((listing) => listing.isListed);
              return (
                <li className="goods-card" key={good.id}>
                  <div className="goods-card__media">
                    {good.imageURL
                      ? <SafeImage src={good.imageURL} alt="" fill unoptimized fallback={<em>无图</em>} />
                      : <em>无图</em>}
                    <span className={`review-badge review-badge--${good.reviewStatus}`}>{statusLabel(good.reviewStatus)}</span>
                  </div>
                  <div className="goods-card__body">
                    <h3 className="goods-card__name">{good.name}</h3>
                    <p className="goods-card__meta">总库存 {good.totalStock} 件 · 参加 {good.listings.length} 场活动</p>
                    {good.reviewNote ? <p className="review-note">平台意见：{good.reviewNote}</p> : null}
                    {good.listings.length > 0 ? (
                      <ul className="goods-card__chips" aria-label={`${good.name} 参加的活动`}>
                        {good.listings.map((listing) => (
                          <li key={listing.listingId} className={listing.isListed ? "" : "is-unlisted"}>
                            <span>{listing.eventName}</span>
                            <i aria-hidden="true">·</i>
                            <em>{listing.isListed ? `${formatMoney(listing.currency, listing.unitPriceMinor)} · 余 ${listing.remaining}` : "已下架"}</em>
                          </li>
                        ))}
                      </ul>
                    ) : <p className="goods-card__noevent">未参加任何活动；编辑商品即可加入活动上架。</p>}
                    <p className="goods-card__stock">
                      <span>已售 <b>{pool.sold}</b></span>
                      <span>预占 <b>{pool.reserved}</b></span>
                      <span>可售 <b>{pool.free}</b></span>
                      {activeListings.length > 0 && activeListings.every((listing) => listing.remaining <= 0) ? (
                        <em className="goods-card__stockout">共享库存售罄</em>
                      ) : null}
                    </p>
                  </div>
                  <div className="goods-card__foot">
                    <button type="button" className="button button--outline" aria-label={`编辑商品 ${good.name}`} onClick={() => openGood(good)}>编辑商品</button>
                  </div>
                </li>
              );
            })}
          </ul>
        </>
      ) : (
        <>
          {!catalogLoading && !catalogError && events.length > 0 && normalizedSearch && visibleGroups.length === 0 ? (
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
                  {catalogLoading && group.listings.length === 0 ? <p className="event-goods-loading" role="status">正在读取本场商品…</p> : null}
                  {!catalogLoading && group.listings.length === 0 && group.bundles.length === 0 ? <p className="event-goods-empty">本场还没有上架商品。</p> : null}
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
                        const soldOut = item.remaining <= 0;
                        const good = goods.find((row) => row.id === item.goodId);
                        return (
                          <li className={`listing-admin-card${soldOut ? " listing-admin-card--soldout" : ""}`} key={item.listingId}>
                            <div className="listing-admin-card__heading">
                              <div className="listing-admin-card__identity">
                                <span className="listing-admin-card__thumb" aria-hidden="true">
                                  {item.imageURL ? <SafeImage src={item.imageURL} alt="" width={64} height={64} unoptimized fallback={<em>无图</em>} /> : <em>无图</em>}
                                </span>
                                <div><h3>{item.name}</h3><p>单品 · {formatMoney(item.currency, item.unitPriceMinor)} · {item.remaining} 件可售{item.isListed ? "" : " · 已下架"}</p></div>
                              </div>
                              <span className={`review-badge review-badge--${item.reviewStatus}`}>{statusLabel(item.reviewStatus)}</span>
                            </div>
                            {soldOut ? <p className="listing-admin-card__stockout" role="status">本场配额已售罄</p> : null}
                            {item.reviewNote ? <p className="review-note">平台意见：{item.reviewNote}</p> : null}
                            <button type="button" className="button button--outline" disabled={!good} onClick={() => good && openGood(good)}>编辑商品</button>
                          </li>
                        );
                      })}
                    </ul>
                  ) : null}
                </section>
              );
            })}
          </div>
        </>
      )}

      <ModalDialog
        ref={goodDialogRef}
        className="goods-modal--listing"
        kicker="商品 · 全店唯一"
        title={editingGood ? editingGood.name : "编辑商品"}
        description="资料与总库存全店共享；下方按活动分别设置价格与分配，保存时一次性生效。"
        labelledById="good-modal-title"
        onClose={closeGood}
        closeDisabled={pending}
      >
        {editingGood && goodDraft ? <div className="goods-form goods-form--modal listing-modal-body">
          <section className="listing-admin-section listing-admin-section--stock">
            <h4>商品资料与共享库存 <em className="field-tag">改一处，全部活动同步</em></h4>
            <div className="goods-image-editor" aria-label={`${editingGood.name}商品相册`}>
              <div className="goods-image-editor__heading"><strong>商品相册</strong><span>{editingGood.images.length}/8 张 · 第一张作为封面</span></div>
              {editingGood.images.length > 0 ? (
                <div className="goods-image-grid">
                  {editingGood.images.map((image, index) => (
                    <figure className="goods-image-item" key={image.id}>
                      <div className="goods-image-frame"><SafeImage src={image.url} alt={`${editingGood.name}，图片 ${index + 1}`} fill unoptimized fallbackClassName="goods-image-frame--fallback" /></div>
                      <figcaption>
                        <span>{index === 0 ? "封面" : `图片 ${index + 1}`}</span>
                        <div>
                          <button type="button" aria-label={`商品图片 ${index + 1} 上移`} disabled={imagePending === editingGood.id || index === 0} onClick={() => void moveGoodImage(editingGood, index, -1)}>↑</button>
                          <button type="button" aria-label={`商品图片 ${index + 1} 下移`} disabled={imagePending === editingGood.id || index === editingGood.images.length - 1} onClick={() => void moveGoodImage(editingGood, index, 1)}>↓</button>
                          <button type="button" aria-label={`删除商品图片 ${index + 1}`} disabled={imagePending === editingGood.id} onClick={() => setImageDeleteTarget({ good: editingGood, image })}>删除</button>
                        </div>
                      </figcaption>
                    </figure>
                  ))}
                </div>
              ) : <p className="goods-image-empty">还没有商品图片。</p>}
              <div className="image-picker-field"><span>{imagePending === editingGood.id ? "正在更新相册…" : "添加图片"} <small>JPG、PNG、WebP · 每张最大 6 MiB</small></span>
                <label className="image-picker" style={pickerStyle} data-disabled={imagePending === editingGood.id}><span>选择图片</span><input style={pickerInputStyle} aria-label={`为${editingGood.name}添加图片`} type="file" accept="image/jpeg,image/png,image/webp" multiple disabled={imagePending === editingGood.id} onChange={(change) => {
                  const input = change.currentTarget;
                  const files = Array.from(input.files ?? []);
                  input.value = "";
                  void addGoodImages(editingGood, files);
                }} /></label>
              </div>
              {existingImageErrors[editingGood.id] ? <p className="form-message form-message--error" role="alert">{existingImageErrors[editingGood.id]}</p> : null}
            </div>
            <label className="field"><span>商品名称</span><input value={goodDraft.name} onChange={(change) => setGoodDraft((prev) => prev ? { ...prev, name: change.target.value } : prev)} /></label>
            <label className="field"><span>商品说明</span><textarea rows={2} value={goodDraft.description} onChange={(change) => setGoodDraft((prev) => prev ? { ...prev, description: change.target.value } : prev)} /></label>
            <div className="stock-pool">
              <div className="stock-total"><span>商品总库存<em className="field-tag">全店共享</em></span><b>{editingGood.totalStock}<i>件</i></b></div>
              <div className="stock-restock">
                <label className="field"><span className="field-label">新增库存<em className="field-limit">件</em></span><input type="number" min="1" step="1" inputMode="numeric" placeholder="0" value={restockQty} onChange={(change) => setRestockQty(change.target.value)} /></label>
                <button type="button" className="button button--dark" disabled={pending || !(Number(restockQty) > 0)} onClick={() => void restockGood(editingGood)}>入库</button>
              </div>
              <StockMeter total={editingPool.total} sold={editingPool.sold} reserved={editingPool.reserved} free={editingPool.free} />
            </div>
          </section>

          <section className="listing-admin-section">
            <h4>活动上架与价格 <em className="field-tag">各活动独立</em></h4>
            {editingGood.listings.length === 0 ? <p className="goods-image-empty">这件商品还没有参加任何活动，在下方「加入新活动」上架。</p> : null}
            {editingGood.listings.map((listing) => {
              const draft = goodDraft.listings[listing.listingId];
              if (!draft) return null;
              const event = events.find((row) => row.id === listing.eventId);
              const draftMinor = majorToMinor(draft.price) ?? 0;
              const update = (patch: Partial<ListingDraft>) => setGoodDraft((prev) => prev ? {
                ...prev, listings: { ...prev.listings, [listing.listingId]: { ...draft, ...patch } },
              } : prev);
              return (
                <div className={`good-listing-editor${draft.isListed ? "" : " is-unlisted"}`} key={listing.listingId}>
                  <header className="good-listing-editor__heading">
                    <div>
                      <strong>{listing.eventName}</strong>
                      <span className={`event-status event-status--${event?.status ?? "scheduled"}`}><i aria-hidden="true" />{eventStatusLabel(event?.status)}</span>
                    </div>
                    <em>{listing.currency} · 已售 {listing.soldQty}{listing.reservedQty > 0 ? ` · 预占 ${listing.reservedQty}` : ""}</em>
                  </header>
                  <label className="check-field"><input type="checkbox" checked={draft.isListed} onChange={(change) => update({ isListed: change.target.checked })} /><span>在本场公开售卖</span></label>
                  <div className="goods-form__dates">
                    <label className="field"><span>基础单价 <small>{listing.currency}</small></span><input type="number" min="0.01" step="0.01" inputMode="decimal" value={draft.price} onChange={(change) => update({ price: change.target.value })} /></label>
                  </div>
                  <PriceTierEditor drafts={draft.tiers} currency={listing.currency} basePrice={draftMinor} onChange={(next) => update({ tiers: next })} />
                  {draft.isListed ? (
                    <button type="button" className="good-listing-editor__release" onClick={() => releaseListingQuota(listing)}>下架本场</button>
                  ) : <p className="good-listing-editor__released">已下架；共享库存由全部活动共用。</p>}
                </div>
              );
            })}
          </section>

          <section className="listing-admin-section listing-admin-section--alloc">
            <h4>加入新活动</h4>
            {addableEvents.length === 0 ? <p className="form-hint">这件商品已经参加了店里全部活动。</p> : (
              <>
                <label className="field"><span>目标活动</span>
                  <SelectMenu value={addEventId} placeholder="选择活动" ariaLabel="选择要加入的活动" options={addableEvents.map((event) => ({ value: event.id, label: `${event.name} · ${eventStatusLabel(event.status)} · ${event.currency}` }))} onChange={(value) => {
                    setAddEventId(value);
                    const target = events.find((row) => row.id === value);
                    const sameCurrency = editingGood.listings.find((listing) => listing.currency === target?.currency);
                    if (sameCurrency) setAddEventPrice(minorInput(sameCurrency.unitPriceMinor));
                  }} />
                </label>
                <div className="goods-form__dates">
                  <label className="field"><span>该活动单价 <small>{events.find((row) => row.id === addEventId)?.currency ?? ""}</small></span><input type="number" min="0.01" step="0.01" inputMode="decimal" value={addEventPrice} onChange={(change) => setAddEventPrice(change.target.value)} /></label>
                </div>
                <button type="button" className="button button--outline" disabled={pending || !addEventId} onClick={() => void addGoodToEvent(editingGood)}>加入活动并上架</button>
                <p className="form-hint">加入活动即时生效；名称、价格等其他修改仍要点下方「保存修改」。</p>
              </>
            )}
            {message ? <p className={`form-message ${messageKind === "error" ? "form-message--error" : "form-message--success"}`} role="status">{message}</p> : null}
          </section>
          <footer className="goods-modal__footer">
            <button type="button" className="button button--outline" disabled={pending} onClick={closeGood}>取消</button>
            <button type="button" className="button button--dark" disabled={pending} onClick={() => void saveGood(editingGood)}>{pending ? "正在保存…" : "保存修改（全部活动）"}</button>
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
        onConfirm={() => imageDeleteTarget && void deleteGoodImage(imageDeleteTarget.good, imageDeleteTarget.image).then(() => setImageDeleteTarget(null))}
        onCancel={() => setImageDeleteTarget(null)}
      />
    </main>
  );
}
