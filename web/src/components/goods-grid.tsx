import Image from "next/image";
import Link from "next/link";
import type { BoothEvent, BundleOffer, Good } from "@/lib/api";
import { formatMoney, lowestUnitRate } from "@/lib/pricing";

const statusLabel: Record<string, string> = {
  scheduled: "即将开始",
  live: "现场进行中",
  ended: "已结束",
};

export function GoodsGrid({ slug, goods }: { slug: string; goods: Good[] }) {
  if (goods.length === 0) {
    return (
      <div className="empty-state">
        <span className="empty-state__mark" aria-hidden="true">✳</span>
        <p>这家摊位还没有上架商品。</p>
      </div>
    );
  }

  return (
    <ul className="product-grid">
      {goods.map((good) => {
        const offers = (good.priceTiers ?? []).filter((tier) => tier.quantity <= good.remaining);
        return (
          <li key={good.listingId}>
          <Link href={`/c/${slug}/listings/${good.listingId}`} className="product-card">
            <figure className="product-card__media">
              {good.imageURL ? (
                <Image
                  src={good.imageURL}
                  alt={good.name}
                  fill
                  sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw"
                  unoptimized
                  className="product-card__image"
                />
              ) : (
                <div className="product-placeholder" aria-label={`${good.name} 暂无商品图片`}>
                  <span className="product-placeholder__name">{good.name}</span>
                  <span className="product-placeholder__note">暂无商品图片</span>
                </div>
              )}
            </figure>
            <div className="product-card__info">
              <h3>{good.name}</h3>
              <p>{good.eventName}</p>
              <div className="product-card__meta">
                <span className="product-card__price">
                  {offers.length > 0 ? <>{formatMoney(good.currency, Math.round(lowestUnitRate(good.unitPriceMinor, offers)))}<em className="product-card__from">起</em></> : good.priceLabel}
                </span>
                <span className="product-card__stock">{good.remaining > 0 ? `余 ${good.remaining}` : "售罄"}</span>
              </div>
              {offers.length > 0 ? <small className="product-card__bundle">{offers.map((tier) => `${tier.quantity} 件 ${formatMoney(good.currency, tier.totalPriceMinor)}`).join(" · ")}</small> : null}
            </div>
          </Link>
          </li>
        );
      })}
    </ul>
  );
}

export function BundlesGrid({ slug, bundles }: { slug: string; bundles: BundleOffer[] }) {
  if (bundles.length === 0) return null;
  return (
    <ul className="bundle-grid">
      {bundles.map((bundle) => {
        const images = bundle.images?.length ? bundle.images : bundle.components.flatMap((component) => component.images ?? []).slice(0, 6);
        return <li className="bundle-card" key={bundle.bundleListingId}>
          <Link href={`/c/${slug}/bundles/${bundle.bundleListingId}`} className="bundle-card__link">
            <div className="bundle-card__media" aria-label={`${bundle.name}套餐图片`}>
              {images.length > 0 ? images.slice(0, 4).map((image, index) => <Image key={image.id} src={image.url} alt={`${bundle.name}，图片 ${index + 1}`} fill={false} width={160} height={160} unoptimized />)
                : <div className="bundle-card__placeholder"><span>固定组合</span><strong>{bundle.name}</strong></div>}
              {bundle.remaining <= 0 ? <span className="bundle-card__soldout">暂时售罄</span> : null}
            </div>
            <div className="bundle-card__body">
              <p className="eyebrow">{bundle.eventName} · 固定套餐</p>
              <h3>{bundle.name}</h3>
              {bundle.description ? <p className="bundle-card__description">{bundle.description}</p> : null}
              <p className="bundle-card__components">{bundle.components.map((component) => `${component.name} × ${component.quantity}`).join(" · ")}</p>
              <div className="bundle-card__price"><strong>{bundle.priceLabel}</strong><span>{bundle.remaining > 0 ? `${bundle.remaining} 套可售` : "等商品补货"}</span></div>
              {bundle.priceTiers?.length ? <small className="bundle-card__tiers">多套优惠：{bundle.priceTiers.map((tier) => `${tier.quantity} 套 ${formatMoney(bundle.currency, tier.totalPriceMinor)}`).join(" · ")}</small> : null}
            </div>
          </Link>
        </li>;
      })}
    </ul>
  );
}

export function EventList({ slug, events }: { slug: string; events: BoothEvent[] }) {
  if (events.length === 0) {
    return <p className="empty-state empty-state--compact">还没有公开活动。</p>;
  }

  return (
    <ul className="event-list">
      {events.map((event) => (
        <li key={event.id}>
          <Link href={`/c/${slug}/events/${event.id}`} className="event-card">
            <div className="event-card__date">
              <span>{event.startsOn.slice(5, 10).replace("-", ".") || "待定"}</span>
            </div>
            <div className="event-card__body">
              <span className={`event-status event-status--${event.status}`}>
                <i aria-hidden="true" />{statusLabel[event.status] ?? event.status}
              </span>
              <h3>{event.name}</h3>
              <p>{[event.city, event.venue].filter(Boolean).join(" · ") || "地点待公布"}</p>
            </div>
            <span className="event-card__booth">摊位 {event.boothNumber || "待公布"}<b aria-hidden="true">↗</b></span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
