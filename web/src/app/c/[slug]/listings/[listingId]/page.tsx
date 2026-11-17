import Link from "next/link";
import { Suspense } from "react";
import { getListing } from "@/lib/api";
import { formatMoney, lowestUnitRate } from "@/lib/pricing";
import { ProductGallery } from "@/components/product-gallery";

async function Detail({ params }: { params: Promise<{ slug: string; listingId: string }> }) {
  const { slug, listingId } = await params;
  const good = await getListing(slug, listingId);
  if (!good) {
    return (
      <div className="empty-page">
        <p className="eyebrow">页面不存在</p>
        <h1>没有找到这件商品</h1>
        <Link href={`/c/${slug}`} className="button button--dark">返回摊位</Link>
      </div>
    );
  }
  const soldOut = good.soldOut ?? good.remaining <= 0;
  const offers = (good.priceTiers ?? []).filter((tier) => tier.quantity <= good.remaining);

  return (
    <main className="wrap product-page">
      <nav className="breadcrumbs" aria-label="面包屑导航">
        <Link href={`/c/${slug}`}>{slug}</Link><span aria-hidden="true">/</span><span>{good.name}</span>
      </nav>
      <article className="product-detail">
        <ProductGallery name={good.name} images={good.images ?? (good.imageURL ? [{ id: "cover", url: good.imageURL, sortOrder: 0 }] : [])} />
        <div className="product-detail__body">
          <p className="eyebrow"><span className="eyebrow__dot" />{good.eventName}</p>
          <h1>{good.name}</h1>
          <div className="product-detail__price-row">
            <strong>{offers.length > 0 ? `${formatMoney(good.currency, Math.round(lowestUnitRate(good.unitPriceMinor, offers)))} 起` : good.priceLabel}</strong>
            <span className={soldOut ? "stock-label stock-label--empty" : "stock-label"}>
              {soldOut ? "本场售罄" : `现场余量 ${good.remaining}`}
            </span>
          </div>
          {good.description ? <p className="product-detail__description">{good.description}</p> : null}
          {offers.length > 0 ? (
            <section className="product-detail__bundles" aria-label="组合优惠">
              <h2>组合优惠</h2>
              <ul>
                {offers.map((tier) => (
                  <li key={tier.quantity}>
                    <strong>{tier.quantity} 件一组</strong>
                    <span>{formatMoney(good.currency, tier.totalPriceMinor)}</span>
                    <small>比单买省 {formatMoney(good.currency, Math.max(0, good.unitPriceMinor * tier.quantity - tier.totalPriceMinor))}</small>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
          <div className="product-detail__note">
            <span className="product-detail__note-mark" aria-hidden="true">✳</span>
            <div>
              <strong>现场购买</strong>
              <p>线上查看商品与余量，到活动摊位由工作人员为你开单。</p>
            </div>
          </div>
          <Link href={`/c/${slug}/events/${good.eventId}`} className="button button--dark">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}`} className="text-link">返回 {slug} 摊位</Link>
        </div>
      </article>
    </main>
  );
}

export default function Page({ params }: { params: Promise<{ slug: string; listingId: string }> }) {
  return (
    <Suspense fallback={<main className="wrap page-loading"><span className="loading-mark" />正在打开商品…</main>}>
      <Detail params={params} />
    </Suspense>
  );
}
