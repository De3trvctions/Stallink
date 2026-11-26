import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { StorefrontLoading } from "@/components/storefront-loading";
import { getListing, getShop, type Booth } from "@/lib/api";
import { customerBoothLabel } from "@/lib/booth";
import { formatMoney, lowestUnitRate } from "@/lib/pricing";
import { ProductGallery } from "@/components/product-gallery";

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

async function Detail({ params }: { params: Promise<{ slug: string; listingId: string }> }) {
  const { slug, listingId } = await params;
  const good = await getListing(slug, listingId);
  if (!good) notFound();
  const soldOut = good.soldOut ?? good.remaining <= 0;
  const offers = (good.priceTiers ?? []).filter((tier) => tier.quantity <= good.remaining);
  const spot = await boothSpot(slug, good.eventId);
  const pickup = spot.number || "摊位";

  return (
    <main className="wrap product-page">
      <nav className="breadcrumbs" aria-label="面包屑导航">
        <Link href={`/c/${slug}`}>{slug}</Link><span aria-hidden="true">/</span><span>{good.name}</span>
      </nav>
      <article className="product-detail">
        <ProductGallery name={good.name} images={good.images ?? (good.imageURL ? [{ id: "cover", url: good.imageURL, sortOrder: 0 }] : [])} />
        <div className="product-detail__body">
          <p className="classic-intro eyebrow"><span className="eyebrow__dot" />{good.eventName}</p>
          <p className="exhibition-only product-kicker">{[spot.number, spot.name].filter(Boolean).join(" · ")}</p>
          <p className="exhibition-only product-lead-price">{good.priceLabel}</p>
          <h1>{good.name}</h1>
          <div className="classic-intro product-detail__price-row">
            <strong>{offers.length > 0 ? `${formatMoney(good.currency, Math.round(lowestUnitRate(good.unitPriceMinor, offers)))} 起` : good.priceLabel}</strong>
            <span className={soldOut ? "stock-label stock-label--empty" : "stock-label"}>
              {soldOut ? "本场售罄" : `现场余量 ${good.remaining}`}
            </span>
          </div>
          {offers.length > 0 ? <p className="exhibition-only product-subprice">阶梯价另计：{offers.map((tier) => `${tier.quantity} 件一组 ${formatMoney(good.currency, tier.totalPriceMinor)}`).join(" · ")}</p> : null}
          {good.description ? <p className="product-detail__description">{good.description}</p> : null}
          <p className="exhibition-only product-stockline">{soldOut ? "售罄。" : `本场可售 ${good.remaining}。`}全店还剩这些，本场和其他场一起卖。</p>
          {offers.length > 0 ? (
            <section className="classic-intro product-detail__bundles" aria-label="组合优惠">
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
          <dl className="exhibition-only product-facts">
            <div><dt>取货</dt><dd>到 {pickup} 现场购买</dd></div>
            <div><dt>邮寄</dt><dd>不邮寄</dd></div>
            <div><dt>开单</dt><dd>店员开单，公开页不收款</dd></div>
            <div><dt>库存</dt><dd>各场共用这一批货</dd></div>
          </dl>
          <div className="classic-intro product-detail__note">
            <span className="product-detail__note-mark" aria-hidden="true">✳</span>
            <div>
              <strong>现场购买</strong>
              <p>线上查看商品与余量，到活动摊位由工作人员为你开单。</p>
            </div>
          </div>
          <Link href={`/c/${slug}/events/${good.eventId}`} className="exhibition-only button button--dark product-buy">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}/events/${good.eventId}`} className="classic-intro button button--dark">查看活动与摊位 <span aria-hidden="true">↗</span></Link>
          <Link href={`/c/${slug}`} className="classic-intro text-link">返回 {slug} 摊位</Link>
          <Link href={`/c/${slug}`} className="exhibition-only product-jump"><span>返回 {spot.name} 摊位</span>{spot.number ? <strong>{spot.number}</strong> : null}</Link>
        </div>
      </article>
    </main>
  );
}

export default function Page({ params }: { params: Promise<{ slug: string; listingId: string }> }) {
  return (
    <Suspense fallback={<StorefrontLoading>正在打开商品…</StorefrontLoading>}>
      <Detail params={params} />
    </Suspense>
  );
}
