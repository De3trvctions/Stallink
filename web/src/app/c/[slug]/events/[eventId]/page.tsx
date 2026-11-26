import { customerBoothLabel } from "@/lib/booth";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { BundlesGrid, GoodsGrid } from "@/components/goods-grid";
import { StorefrontLoading } from "@/components/storefront-loading";
import { getEvent } from "@/lib/api";

function eventIsEnded(event: { status: string; endsOn: string | null }): boolean {
  if (event.status === "ended") return true;
  if (!event.endsOn) return false;
  return new Date(`${event.endsOn}T23:59:59`).getTime() < Date.now();
}

async function EventPage({ params }: { params: Promise<{ slug: string; eventId: string }> }) {
  const { slug, eventId } = await params;
  const data = await getEvent(slug, eventId);
  if (!data) notFound();
  const event = data.event;
  const ended = eventIsEnded(event);
  return (
    <main className="wrap exhibition-event">
      <nav className="breadcrumbs" aria-label="面包屑导航">
        <Link href={`/c/${slug}`}>{slug}</Link><span aria-hidden="true">/</span><span>{event.name}</span>
      </nav>
      <header className="event-head event-head--detail">
        <div className="event-head__copy">
          <p className="eyebrow"><i aria-hidden="true" />{event.city || "线下活动"} · {event.venue || "地点待公布"}</p>
          <h1>{event.name}{ended ? <span className="event-ended-badge">已结束</span> : null}</h1>
          <div className="event-head__meta">
            <span><time dateTime={event.startsOn}>{event.startsOn}</time> 至 <time dateTime={event.endsOn}>{event.endsOn}</time></span>
            <span>摊位 {customerBoothLabel(event)}</span>
            <span>币种 {event.currency}</span>
          </div>
          <Link className="exhibition-only exhibition-action" href={`/c/${slug}`}>查看摊位 <span aria-hidden="true">↗</span></Link>
        </div>
      </header>
      {ended ? <p className="event-ended-notice">本场活动已结束，可继续浏览作品。</p> : event.status === "scheduled" ? <p className="event-ended-notice">本场活动尚未开始，可先浏览作品。</p> : null}
      <div className="section-heading section-heading--compact">
        <div><p className="eyebrow">仅限本场</p><h2>活动商品</h2></div>
        <span>{data.goods.length} 件</span>
      </div>
      <GoodsGrid slug={slug} goods={data.goods} />
      {data.bundles.length > 0 ? <section className="store-section store-section--bundles" id="bundles">
        <div className="section-heading">
          <div><p className="eyebrow">固定商品组合</p><h2>套餐优惠</h2></div>
          <span>{data.bundles.length} 套</span>
        </div>
        <BundlesGrid slug={slug} bundles={data.bundles} />
      </section> : null}
      <Link href={`/c/${slug}`} className="text-link event-back">
        ← 返回摊位
      </Link>
    </main>
  );
}

export default function Page({ params }: { params: Promise<{ slug: string; eventId: string }> }) {
  return (
    <Suspense fallback={<StorefrontLoading>正在打开活动…</StorefrontLoading>}>
      <EventPage params={params} />
    </Suspense>
  );
}
