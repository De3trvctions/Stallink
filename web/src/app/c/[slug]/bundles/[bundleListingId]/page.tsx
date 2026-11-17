import Link from "next/link";
import { Suspense } from "react";
import { ProductGallery } from "@/components/product-gallery";
import { SafeImage } from "@/components/safe-image";
import { getBundle } from "@/lib/api";
import { formatMoney } from "@/lib/pricing";

async function Detail({ params }: { params: Promise<{ slug: string; bundleListingId: string }> }) {
  const { slug, bundleListingId } = await params;
  const bundle = await getBundle(slug, bundleListingId);
  if (!bundle) {
    return (
      <div className="empty-page">
        <p className="eyebrow">页面不存在</p>
        <h1>这个套餐暂时不可用</h1>
        <Link href={`/c/${slug}`} className="button button--dark">返回摊位</Link>
      </div>
    );
  }
  const images = bundle.images?.length ? bundle.images : bundle.components.flatMap((component) => component.images ?? []);
  const baseComponentTotal = bundle.components.reduce((sum, component) => {
    return sum + component.unitPriceMinor * component.quantity;
  }, 0);

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
          <p className="eyebrow"><span className="eyebrow__dot" />{bundle.eventName} · 固定套餐</p>
          <h1>{bundle.name}</h1>
          <div className="product-detail__price-row">
            <strong>{bundle.priceLabel} / 套</strong>
            <span className={bundle.remaining > 0 ? "stock-label" : "stock-label stock-label--empty"}>
              {bundle.remaining > 0 ? `整套余量 ${bundle.remaining}` : "套餐暂时售罄"}
            </span>
          </div>
          {bundle.description ? <p className="product-detail__description">{bundle.description}</p> : null}
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
          {baseComponentTotal > 0 ? <p className="field-hint">组件单买合计 {formatMoney(bundle.currency, baseComponentTotal)} · 套餐价 {formatMoney(bundle.currency, bundle.unitPriceMinor)}</p> : null}
          <div className="product-detail__note">
            <span className="product-detail__note-mark" aria-hidden="true">✳</span>
            <div><strong>现场购买</strong><p>线上查看套餐与余量，到活动摊位由工作人员为你开单。</p></div>
          </div>
          <Link href={`/c/${slug}/events/${bundle.eventId}`} className="button button--dark">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}`} className="text-link">返回 {slug} 摊位</Link>
        </div>
      </article>
    </main>
  );
}

export default function Page({ params }: { params: Promise<{ slug: string; bundleListingId: string }> }) {
  return (
    <Suspense fallback={<main className="wrap page-loading"><span className="loading-mark" />正在打开套餐…</main>}>
      <Detail params={params} />
    </Suspense>
  );
}
