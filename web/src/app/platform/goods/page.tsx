"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { SafeImage } from "@/components/safe-image";
import { useRouter } from "next/navigation";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";
import { formatMoney, type PriceTier } from "@/lib/pricing";

type Listing = {
  listingId: string; eventName: string; currency: string; priceLabel: string;
  priceTiers: PriceTier[]; remaining: number; reservedQty: number; isListed: boolean;
};
type ProductImage = { id: string; url: string; sortOrder: number };
type Good = { id: string; name: string; description: string; reviewStatus: string; reviewNote: string; revision: number; totalStock: number; listings: Listing[]; images: ProductImage[] };
type Bundle = {
  bundleId: string; bundleListingId: string; name: string; description: string; reviewStatus: string; reviewNote: string;
  revision: number;
  priceLabel: string; currency: string; unitPriceMinor: number; remaining: number; isListed: boolean;
  eventName: string; priceTiers: PriceTier[]; images: ProductImage[];
  components: { goodId: string; name: string; quantity: number; unitPriceMinor: number; reviewStatus: string; isListed: boolean; remaining: number; images: ProductImage[] }[];
};
type ShopGroup = { shopId: string; shopName: string; shopSlug: string; goods: Good[]; bundles: Bundle[] };
type Filter = "all" | "pending" | "approved" | "rejected";
type KindFilter = "all" | "good" | "bundle";

const statusText: Record<string, string> = { pending: "待审核", approved: "已通过", rejected: "已拒绝" };

export default function PlatformGoodsPage() {
  const router = useRouter();
  const [shops, setShops] = useState<ShopGroup[]>([]);
  const [filter, setFilter] = useState<Filter>("pending");
  const [kindFilter, setKindFilter] = useState<KindFilter>("all");
  const [rejectID, setRejectID] = useState("");
  const [note, setNote] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const { message, setError, setSuccess } = useMessage();
  const [saving, setSaving] = useState(false);
  const [autoApprove, setAutoApprove] = useState(false);
  const [switchSaving, setSwitchSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError("");
    try {
      const res = await browserJSON("/api/platform/goods");
      if (res.category === "unauthorized" || res.category === "forbidden") {
        router.replace("/login");
        return;
      }
      if (!res.ok) { setLoadError(describeRequest(res)); return; }
      setShops(Array.isArray(res.data) ? (res.data as ShopGroup[]).map((shop) => ({
        ...shop,
        goods: shop.goods.map((good) => ({ ...good, images: good.images ?? [] })),
        bundles: (shop.bundles ?? []).map((bundle) => ({
          ...bundle, images: bundle.images ?? [],
          components: (bundle.components ?? []).map((component) => ({ ...component, images: component.images ?? [] })),
        })),
      })) : []);
    } catch {
      setLoadError("无法连接服务，请稍后重试。");
    } finally {
      setLoading(false);
    }
  }, [router]);

  const loadAutoApprove = useCallback(async () => {
    try {
      const res = await browserJSON("/api/platform/review/auto-approve");
      if (res.ok && typeof (res.data as { enabled?: boolean })?.enabled === "boolean") {
        setAutoApprove(Boolean((res.data as { enabled: boolean }).enabled));
      }
    } catch {
      // Keep the default (off) when the flag cannot be read.
    }
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => { void load(); void loadAutoApprove(); }, 0);
    return () => window.clearTimeout(timer);
  }, [load, loadAutoApprove]);

  async function toggleAutoApprove() {
    if (switchSaving) return;
    const next = !autoApprove;
    setSwitchSaving(true);
    setError(null);
    try {
      const res = await browserJSON("/api/platform/review/auto-approve", "PATCH", { enabled: next });
      if (!res.ok) {
        setError(describeRequest(res));
        return;
      }
      setAutoApprove(Boolean((res.data as { enabled?: boolean })?.enabled ?? next));
    } catch {
      setError("无法连接服务，开关没有保存。");
    } finally {
      setSwitchSaving(false);
    }
  }

  const visibleShops = useMemo(() => shops.map((shop) => ({
    ...shop,
    goods: kindFilter === "bundle" ? [] : shop.goods.filter((good) => filter === "all" || good.reviewStatus === filter),
    bundles: kindFilter === "good" ? [] : shop.bundles.filter((bundle) => filter === "all" || bundle.reviewStatus === filter),
  })), [shops, filter, kindFilter]);
  const pendingCount = shops.reduce((count, shop) => count + shop.goods.filter((good) => good.reviewStatus === "pending").length + shop.bundles.filter((bundle) => bundle.reviewStatus === "pending").length, 0);

  async function review(good: Good, status: "approved" | "rejected", reason = "") {
    setSaving(true);
    setError(null);
    try {
      const res = await browserJSON(`/api/platform/goods/${good.id}/review`, "PATCH", { status, note: reason, expectedRevision: good.revision });
      if (!res.ok) {
        setError(describeRequest(res));
        return;
      }
      setRejectID("");
      setNote("");
      setSuccess(status === "approved" ? "商品已通过审核。" : "商品已拒绝并通知店铺查看原因。");
      await load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setSaving(false);
    }
  }

  async function reviewBundle(bundleID: string, status: "approved" | "rejected", reason = "") {
    setSaving(true);
    setError(null);
    try {
      const bundle = shops.flatMap((shop) => shop.bundles).find((candidate) => candidate.bundleId === bundleID);
      if (!bundle) {
        setError("找不到这条套餐审核记录，请刷新后重试。");
        return;
      }
      const res = await browserJSON(`/api/platform/bundles/${bundleID}/review`, "PATCH", {
        status, note: reason, expectedRevision: bundle.revision,
      });
      if (!res.ok) {
        setError(describeRequest(res));
        return;
      }
      setRejectID("");
      setNote("");
      setSuccess(status === "approved" ? "套餐已通过审核。" : "套餐已拒绝并通知店铺查看原因。");
      await load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <main className="platform-review-page">
        <p className="eyebrow">平台内容管理</p>
        <h1>商品审核</h1>
        <p>按店铺查看商品、活动价格和库存。这里只提供商品审核，不展示店铺销售账号或顾客联系方式。</p>
        <div className="review-toolbar">
          <div className="review-filter" role="group" aria-label="筛选审核状态">
            {(["pending", "approved", "rejected", "all"] as const).map((value) => (
              <button key={value} type="button" data-active={filter === value} aria-pressed={filter === value} onClick={() => setFilter(value)}>
                {value === "all" ? "全部" : statusText[value]}
                {value === "pending" ? <span>{pendingCount}</span> : null}
              </button>
            ))}
          </div>
          <div className="review-filter" role="group" aria-label="筛选商品类型">
            {(["all", "good", "bundle"] as const).map((value) => <button key={value} type="button" data-active={kindFilter === value} aria-pressed={kindFilter === value} onClick={() => setKindFilter(value)}>
              {value === "all" ? "全部类型" : value === "good" ? "单品" : "套餐"}
            </button>)}
          </div>
          <button
            className="review-switch"
            type="button"
            role="switch"
            aria-checked={autoApprove}
            aria-label="新提交自动通过审核"
            data-on={autoApprove}
            disabled={switchSaving}
            onClick={() => void toggleAutoApprove()}
          >
            <span className="review-switch__track" aria-hidden="true"><i /></span>
            <span className="review-switch__copy"><strong>自动通过</strong><small>{autoApprove ? "新提交直接通过" : "新提交进待审核"}</small></span>
          </button>
          <button className="button button--outline review-refresh" type="button" onClick={() => void load()} disabled={loading}>刷新目录</button>
        </div>
        {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
        {loading ? <p className="review-loading" role="status">正在读取各店商品…</p> : loadError ? (
          <p className="review-loading" role="alert">{loadError} <button type="button" className="button button--outline" onClick={() => void load()}>重试</button></p>
        ) : visibleShops.length === 0 ? <p className="review-loading">还没有店铺。</p> : (
          <div className="review-shops">
            {visibleShops.map((shop) => (
              <section className="review-shop" key={shop.shopId}>
                <header className="review-shop__heading">
                  <div><p className="eyebrow">/c/{shop.shopSlug}</p><h2>{shop.shopName}</h2></div>
                  <span>{shop.goods.length} 件商品 · {shop.bundles.length} 套套餐</span>
                </header>
                {shop.goods.length === 0 && shop.bundles.length === 0 ? <p className="review-shop__empty">{filter === "pending" ? "当前没有待审核商品或套餐。" : "此状态下暂无商品或套餐。"}</p> : null}
                {shop.goods.length > 0 ? (
                  <ul className="review-good-list">
                    {shop.goods.map((good) => (
                      <li className="review-good" key={good.id}>
                        <div className="review-good__heading">
                          <div><span className={`review-badge review-badge--${good.reviewStatus}`}>{statusText[good.reviewStatus] ?? good.reviewStatus}</span><h3>{good.name}</h3></div>
                          <p>总库存 {good.totalStock}</p>
                        </div>
                        {good.images.length > 0 ? (
                          <div className="review-good__images" aria-label={`${good.name}商品图片`}>
                            {good.images.map((image, index) => (
                              <a href={image.url} target="_blank" rel="noreferrer" key={image.id} aria-label={`查看${good.name}图片 ${index + 1}`}>
                                <SafeImage src={image.url} alt={`${good.name}，图片 ${index + 1}`} width={112} height={112} unoptimized fallbackClassName="review-good__image--fallback" />
                                {index === 0 ? <span>封面</span> : null}
                              </a>
                            ))}
                          </div>
                        ) : <p className="review-good__no-images">此商品尚未上传图片。</p>}
                        {good.description ? <p className="review-good__description">{good.description}</p> : null}
                        {good.reviewNote ? <p className="review-note"><strong>上次拒绝原因：</strong>{good.reviewNote}</p> : null}
                        <div className="review-listings">
                          {good.listings.map((listing) => (
                            <article key={listing.listingId} className="review-listing">
                              <div><strong>{listing.eventName || "未关联活动"}</strong><span>{listing.isListed ? "公开上架" : "已下架"} · 可售 {listing.remaining} · 预留 {listing.reservedQty}</span></div>
                              <div className="review-listing__price"><b>{listing.priceLabel}</b>{listing.priceTiers.map((tier) => <span key={tier.quantity}>{tier.quantity} 件合计 {formatMoney(listing.currency, tier.totalPriceMinor)}</span>)}</div>
                            </article>
                          ))}
                          {good.listings.length === 0 ? <p className="review-listings__empty">尚未创建活动上架。</p> : null}
                        </div>
                        <div className="review-actions">
                          <button type="button" className="button button--dark" disabled={saving || good.reviewStatus === "approved"} onClick={() => void review(good, "approved")}>通过商品</button>
                          {rejectID === good.id ? (
                            <div className="review-reject-form">
                              <label className="field"><span>拒绝原因 <small>{note.length}/500</small></span><textarea rows={3} maxLength={500} required value={note} onChange={(event) => setNote(event.target.value)} placeholder="说明需要调整的商品信息或定价" /></label>
                              <button type="button" className="button button--outline" disabled={saving || !note.trim()} onClick={() => void review(good, "rejected", note.trim())}>确认拒绝</button>
                              <button type="button" className="review-cancel" onClick={() => { setRejectID(""); setNote(""); }}>取消</button>
                            </div>
                          ) : (
                            <button type="button" className="button button--outline" disabled={saving} onClick={() => { setRejectID(good.id); setNote(""); }}>拒绝并说明原因</button>
                          )}
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : null}
                {shop.bundles.length > 0 ? <ul className="review-bundle-list">
                  {shop.bundles.map((bundle) => {
                    const rejectKey = `bundle:${bundle.bundleId}`;
                    return <li className="review-bundle" key={bundle.bundleId}>
                      <div className="review-good__heading">
                        <div><span className={`review-badge review-badge--${bundle.reviewStatus}`}>{statusText[bundle.reviewStatus] ?? bundle.reviewStatus}</span><h3>{bundle.name}</h3></div>
                        <strong>{bundle.priceLabel} / 套</strong>
                      </div>
                      {bundle.images.length > 0 ? <div className="review-good__images" aria-label={`${bundle.name}套餐图片`}>
                        {bundle.images.map((image, index) => <a href={image.url} target="_blank" rel="noreferrer" key={image.id} aria-label={`查看套餐图片 ${index + 1}`}>
                          <SafeImage src={image.url} alt={`${bundle.name}，图片 ${index + 1}`} width={112} height={112} unoptimized fallbackClassName="review-good__image--fallback" />{index === 0 ? <span>封面</span> : null}
                        </a>)}
                      </div> : <p className="review-good__no-images">此套餐尚未上传图片。</p>}
                      {bundle.description ? <p className="review-good__description">{bundle.description}</p> : null}
                      {bundle.reviewNote ? <p className="review-note"><strong>上次拒绝原因：</strong>{bundle.reviewNote}</p> : null}
                      <div className="review-bundle__component-list" aria-label={`${bundle.name}组成商品审核状态`}>
                        {bundle.components.map((component) => <div className="review-bundle__component" key={component.goodId}>
                          {component.images[0] ? <SafeImage src={component.images[0].url} alt="" width={44} height={44} unoptimized fallback={<span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />} /> : <span className="bundle-component-picker__thumb-placeholder" aria-hidden="true" />}
                          <div><strong>{component.name} × {component.quantity}</strong><small>{component.reviewStatus === "approved" ? "商品审核通过" : statusText[component.reviewStatus] ?? "未审核"} · {component.isListed ? `本场可售 ${component.remaining}` : "本场已下架"} · {formatMoney(bundle.currency, component.unitPriceMinor)} / 件</small></div>
                        </div>)}
                      </div>
                      <article className="review-listing">
                        <div><strong>{bundle.eventName || "未关联活动"}</strong><span>{bundle.isListed ? "公开上架" : "已下架"} · 可售余量 {bundle.remaining}</span></div>
                        <div className="review-listing__price"><b>{bundle.priceLabel}</b>{bundle.priceTiers.map((tier) => <span key={tier.quantity}>{tier.quantity} 套合计 {formatMoney(bundle.currency, tier.totalPriceMinor)}</span>)}</div>
                      </article>
                      <div className="review-actions">
                        <button type="button" className="button button--dark" disabled={saving || bundle.reviewStatus === "approved"} onClick={() => void reviewBundle(bundle.bundleId, "approved")}>通过套餐</button>
                        {rejectID === rejectKey ? <div className="review-reject-form">
                          <label className="field"><span>拒绝原因 <small>{note.length}/500</small></span><textarea rows={3} maxLength={500} required value={note} onChange={(event) => setNote(event.target.value)} placeholder="说明套餐组成、图片或定价需要调整的内容" /></label>
                          <button type="button" className="button button--outline" disabled={saving || !note.trim()} onClick={() => void reviewBundle(bundle.bundleId, "rejected", note.trim())}>确认拒绝</button>
                          <button type="button" className="review-cancel" onClick={() => { setRejectID(""); setNote(""); }}>取消</button>
                        </div> : <button type="button" className="button button--outline" disabled={saving} onClick={() => { setRejectID(rejectKey); setNote(""); }}>拒绝并说明原因</button>}
                      </div>
                    </li>;
                  })}
                </ul> : null}
              </section>
            ))}
          </div>
        )}
      </main>
    </>
  );
}
