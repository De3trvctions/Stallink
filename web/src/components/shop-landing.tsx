import Link from "next/link";
import { SafeImage } from "@/components/safe-image";
import type { Good } from "@/lib/api";

function PhotoCard({ good, hero = false }: { good: Good; hero?: boolean }) {
  return (
    <Link className="shop-photo-card" href={`/c/${good.shopSlug}/listings/${good.listingId}`}>
      <div className="shop-photo-card__image">
        <SafeImage src={good.imageURL!} alt={good.name} fill unoptimized loading={hero ? "eager" : "lazy"}
          sizes={hero ? "(max-width: 640px) 40vw, 200px" : "(max-width: 640px) 45vw, 280px"}
          fallback={<span className="shop-photo-card__fallback">暂无商品图片</span>} />
      </div>
      <div className="shop-photo-card__copy">
        <span className="shop-photo-card__shop">{good.shopName}</span>
        <h3>{good.name}</h3>
        {!hero ? <p>{good.eventName}</p> : null}
        <div className="shop-photo-card__details"><span>{good.priceLabel}</span><span>余 {good.remaining}</span></div>
      </div>
    </Link>
  );
}

function photoGoods(goods: Good[]) {
  return goods.filter((good) => good.imageURL && good.shopSlug && !good.soldOut && good.remaining > 0);
}

export function ShopHero({ goods }: { goods: Good[] }) {
  const photos = photoGoods(goods).slice(0, 5);
  return (
    <div className="shop-only shop-hero">
      {photos.length > 0 ? (
        <div className="shop-hero__constellation" aria-label="发现摊位商品">
          {photos.map((good) => <PhotoCard key={good.listingId} good={good} hero />)}
        </div>
      ) : null}
      <h1 className="shop-hero__wordmark">Stallink<span>.</span></h1>
      <p className="shop-hero__intro">从一件喜欢的作品，找到你的下一站。</p>
      <p className="shop-hero__hint">搜商品、摊位或活动，现场相见。</p>
    </div>
  );
}

export function ShopSpotlight({ goods, unavailable = false }: { goods: Good[]; unavailable?: boolean }) {
  // The catalogue already explains a failed load; avoid a second, misleading empty message.
  if (unavailable) return null;
  // Spread the spotlight across real events, without inventing categories or rankings.
  const seen = new Set<string>();
  const featured = photoGoods(goods).filter((good) => {
    const key = `${good.shopSlug}:${good.eventId}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  }).slice(0, 4);
  return (
    <section className="shop-only shop-spotlight" aria-label="摊位在售好物">
      <div className="shop-section-heading"><h2>摊位在售好物</h2><span>从这些作品开始逛</span></div>
      {featured.length > 0 ? (
        <div className="shop-spotlight__grid">
          {featured.map((good) => <PhotoCard key={good.listingId} good={good} />)}
        </div>
      ) : (
        <div className="empty-state empty-state--compact">
          <span className="empty-state__mark" aria-hidden="true">✳</span>
          <p>还没有带图片的在售好物，可以在下方「全部商品」里逛逛。</p>
        </div>
      )}
    </section>
  );
}
