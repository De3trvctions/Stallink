import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { StorefrontLoading } from "@/components/storefront-loading";
import { ProductGallery } from "@/components/product-gallery";
import { SafeImage } from "@/components/safe-image";
import { getBundle, getShop, type Booth } from "@/lib/api";
import { customerBoothLabel } from "@/lib/booth";
import { formatMoney } from "@/lib/pricing";

async function boothSpot(slug: string, eventId: string) {
  let booth: Booth | null = null;
  try {
    booth = await getShop(slug);
  } catch {
    booth = null;
  }
  const events = booth ? [...booth.events.live, ...booth.events.scheduled, ...booth.events.ended] : [];
  return {
    name: booth?.name || slug,
    number: (() => { const event = events.find((item) => item.id === eventId); return event ? customerBoothLabel(event) : ""; })(),
  };
}

async function Detail({ params }: { params: Promise<{ slug: string; bundleListingId: string }> }) {
  const { slug, bundleListingId } = await params;
  const bundle = await getBundle(slug, bundleListingId);
  if (!bundle) notFound();
  const images = bundle.images?.length ? bundle.images : bundle.components.flatMap((component) => component.images ?? []);
  const baseComponentTotal = bundle.components.reduce((sum, component) => {
    return sum + component.unitPriceMinor * component.quantity;
  }, 0);
  const spot = await boothSpot(slug, bundle.eventId);
  const pickup = spot.number || "摊位";
  const saved = Math.max(0, baseComponentTotal - bundle.unitPriceMinor);

  return (
    <main className="wrap product-page">
      <nav className="breadcrumbs" aria-label="面包屑导航">
        <Link href={`/c/${slug}`}>{slug}</Link><span aria-hidden="true">/</span>
        <Link href={`/c/${slug}/events/${bundle.eventId}`}>{bundle.eventName}</Link><span aria-hidden="true">/</span>
        <span>{bundle.name}</span>
      </nav>
      <article className="product-detail bundle-detail">
        <ProductGallery name={bundle.name} images={images} />
        <div className="product-detail__body">
          <p className="classic-intro eyebrow"><span className="eyebrow__dot" />{bundle.eventName} · 固定套餐</p>
          <p className="exhibition-only product-kicker">{[spot.number, spot.name].filter(Boolean).join(" · ")}</p>
          <p className="exhibition-only product-lead-price">{bundle.priceLabel}</p>
          <h1>{bundle.name}</h1>
          <div className="classic-intro product-detail__price-row">
            <strong>{bundle.priceLabel} / 套</strong>
            <span className={bundle.remaining > 0 ? "stock-label" : "stock-label stock-label--empty"}>
              {bundle.remaining > 0 ? `整套余量 ${bundle.remaining}` : "套餐暂时售罄"}
            </span>
          </div>
          {baseComponentTotal > 0 ? <p className="exhibition-only product-subprice">单买 {formatMoney(bundle.currency, baseComponentTotal)}{saved > 0 ? `，这套省 ${formatMoney(bundle.currency, saved)}` : ""}</p> : null}
          {bundle.description ? <p className="product-detail__description">{bundle.description}</p> : null}
          <p className="exhibition-only product-stockline">{bundle.remaining > 0 ? `这套还能组 ${bundle.remaining} 套。卖出一套会同时减少套餐里的每件商品。` : "售罄。"}</p>
          <section className="bundle-detail__components" aria-label="套餐内商品">
            <h2>套餐包含</h2>
            <ul>
              {bundle.components.map((component, index) => (
                <li key={`${component.name}-${component.quantity}-${index}`}>
                  {component.images?.[0] ? <SafeImage src={component.images[0].url} alt="" width={52} height={52} unoptimized fallback={<span className="bundle-detail__component-placeholder" aria-hidden="true">✳</span>} /> : <span className="bundle-detail__component-placeholder" aria-hidden="true">✳</span>}
                  <div><strong>{component.name}</strong><small>每套 {component.quantity} 件 · 单买 {formatMoney(bundle.currency, component.unitPriceMinor)} / 件</small></div>
                </li>
              ))}
            </ul>
          </section>
          {bundle.priceTiers?.length ? (
            <section className="product-detail__bundles" aria-label="套餐多买优惠">
              <h2>多套优惠</h2>
              <ul>
                {bundle.priceTiers.map((tier) => (
                  <li key={tier.quantity}>
                    <strong>{tier.quantity} 套一组</strong>
                    <span>{formatMoney(bundle.currency, tier.totalPriceMinor)}</span>
                    <small>比单买省 {formatMoney(bundle.currency, Math.max(0, bundle.unitPriceMinor * tier.quantity - tier.totalPriceMinor))}</small>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
          {baseComponentTotal > 0 ? <p className="classic-intro field-hint">组件单买合计 {formatMoney(bundle.currency, baseComponentTotal)} · 套餐价 {formatMoney(bundle.currency, bundle.unitPriceMinor)}</p> : null}
          <dl className="exhibition-only product-facts">
            <div><dt>取货</dt><dd>到 {pickup} 现场购买</dd></div>
            <div><dt>邮寄</dt><dd>不邮寄</dd></div>
            <div><dt>开单</dt><dd>店员开单，公开页不收款</dd></div>
            <div><dt>库存</dt><dd>各场共用这一批货</dd></div>
          </dl>
          <div className="classic-intro product-detail__note">
            <span className="product-detail__note-mark" aria-hidden="true">✳</span>
            <div><strong>现场购买</strong><p>线上查看套餐与余量，到活动摊位由工作人员为你开单。</p></div>
          </div>
          <Link href={`/c/${slug}/events/${bundle.eventId}`} className="exhibition-only button button--dark product-buy">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}/events/${bundle.eventId}`} className="classic-intro button button--dark">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}`} className="classic-intro text-link">返回 {slug} 摊位</Link>
          <Link href={`/c/${slug}`} className="exhibition-only product-jump"><span>返回 {spot.name} 摊位</span>{spot.number ? <strong>{spot.number}</strong> : null}</Link>
        </div>
      </article>
    </main>
  );
}

export default function Page({ params }: { params: Promise<{ slug: string; bundleListingId: string }> }) {
  return (
    <Suspense fallback={<StorefrontLoading>正在打开套餐…</StorefrontLoading>}>
      <Detail params={params} />
    </Suspense>
  );
}
