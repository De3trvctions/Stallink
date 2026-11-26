import Link from "next/link";
import type { Booth } from "@/lib/api";
import { customerBoothLabel } from "@/lib/booth";
import { SafeImage } from "@/components/safe-image";
import { BundlesGrid, EventList, GoodsGrid } from "@/components/goods-grid";

export function BoothView({ booth }: { booth: Booth }) {
  const featuredEventId = booth.featuredEventId ?? "";
  const onStage = [...(booth.events.live ?? []), ...(booth.events.scheduled ?? [])];
  const activeEvent = onStage.find((event) => event.id === featuredEventId) ?? booth.events.live?.[0] ?? booth.events.scheduled?.[0];
  const isLive = Boolean(booth.events.live?.length);
  const allEvents = [...(booth.events.live ?? []), ...(booth.events.scheduled ?? []), ...(booth.events.ended ?? [])];
  const statusRank: Record<string, number> = { live: 0, scheduled: 1, ended: 2 };
  const sections = (() => {
    const map = new Map<string, { id: string; name: string; status: string; place: string; boothNumber: string; dates: string; goods: typeof booth.goods }>();
    booth.goods.forEach((good) => {
      const event = allEvents.find((item) => item.id === good.eventId);
      const existing = map.get(good.eventId);
      if (existing) {
        existing.goods.push(good);
        return;
      }
      map.set(good.eventId, {
        id: good.eventId,
        name: event?.name ?? good.eventName,
        status: event?.status ?? "scheduled",
        place: [event?.city, event?.venue].filter(Boolean).join(" / "),
        boothNumber: event ? customerBoothLabel(event) : "",
        dates: event ? `${event.startsOn.slice(5).replace("-", ".")}–${event.endsOn.slice(5).replace("-", ".")}` : "",
        goods: [good],
      });
    });
    return [...map.values()].sort((a, b) => {
      if (a.id === featuredEventId) return -1;
      if (b.id === featuredEventId) return 1;
      return (statusRank[a.status] ?? 9) - (statusRank[b.status] ?? 9);
    });
  })();
  const featuredGood = sections.find((section) => section.id === featuredEventId)?.goods[0] ?? booth.goods[0];

  return (
    <main className="ed-frame ed-booth">
      <div className="ed-crumb">
        <span className="ed-mono">独立摊位</span>
        <span className="ed-mono ed-sep" aria-hidden="true">/</span>
        <span className="ed-mono" style={{ color: "var(--ink)" }}>{booth.slug}</span>
      </div>

      <section className="ed-store">
        <div>
          <div className="ed-live">
            {isLive ? <i aria-hidden="true" /> : null}
            <span className="ed-mono" style={{ color: "var(--ink)" }}>
              {isLive ? "正在营业 / 今日在展" : "创作者独立店铺"}
            </span>
          </div>
          <h1 className="ed-title ed-title--store">{booth.name}</h1>
          <p className="ed-store__notice">{booth.notice || "原创周边、活动现场信息与库存，一页就能找到。"}</p>

          <div className="ed-pass">
            <div className="ed-pass__body">
              <span className="ed-pass__label">
                {activeEvent ? (activeEvent.status === "live" ? "本场摊位 / 进行中" : "即将开展") : "摊位信息"}
              </span>
              <div className="ed-pass__name">{activeEvent ? activeEvent.name : "线上浏览，现场见面"}</div>
              <div className="ed-pass__meta">
                {activeEvent ? (
                  <>
                    <span>{[activeEvent.city, activeEvent.venue].filter(Boolean).join(" / ") || "地点待公布"}</span>
                    <span>摊位 {customerBoothLabel(activeEvent)} / {activeEvent.startsOn} – {activeEvent.endsOn}</span>
                  </>
                ) : (
                  <span>活动安排将在这里更新。</span>
                )}
              </div>
            </div>
            <div className="ed-pass__stub" aria-hidden="true">ADMIT ONE</div>
          </div>

          <div className="ed-store__actions">
            <a className="ed-link" href="#goods">看本场商品 ↓</a>
            <span className="ed-mono">{booth.goods.length} 件 / 现场自取</span>
            {activeEvent ? <Link className="exhibition-only exhibition-action" href={`/c/${booth.slug}/events/${activeEvent.id}`}>查看活动 <span aria-hidden="true">↗</span></Link> : null}
          </div>
        </div>

        <figure className="ed-figure ed-store__figure">
          <div className="ed-figure__frame">
            {featuredGood?.imageURL ? (
              <SafeImage
                src={featuredGood.imageURL}
                alt={featuredGood.name}
                fill
                sizes="(max-width: 900px) 100vw, 45vw"
                unoptimized
                priority
                fallback={<div className="ed-store__poster"><span>{booth.name}</span></div>}
              />
            ) : (
              <div className="ed-store__poster">
                <span>{booth.name}</span>
              </div>
            )}
          </div>
          {featuredGood ? <figcaption className="ed-placard"><strong>{featuredGood.name}</strong></figcaption> : null}
        </figure>
      </section>

      <section className="ed-shelf" id="goods">
        <div className="ed-shelf__head">
          <div>
            <h2>正在贩售</h2>
            <p className="ed-shelf__meta">按活动分开陈列，同一个商品可在多场活动中分别上架。</p>
          </div>
          <span className="ed-mono">{booth.goods.length} 件 · {sections.length} 场</span>
        </div>
        {sections.length === 0 ? (
          <div className="empty-state">
            <span className="empty-state__mark" aria-hidden="true">✳</span>
            <p>摊位暂未上架单件商品。</p>
          </div>
        ) : sections.map((section) => (
          <section className="ed-event-shelf" id={`event-${section.id}`} key={section.id}>
            <div className="ed-shelf__head">
              <div>
                <h3>{section.name}{section.id === featuredEventId ? <em className="ed-shelf__flag">主推</em> : null}</h3>
                <p className="ed-shelf__meta">
                  {section.status === "live" ? "现场进行中" : section.status === "ended" ? "已结束" : "即将开始"}
                  {section.place ? ` · ${section.place}` : ""}
                  {section.boothNumber ? ` · 摊位 ${section.boothNumber}` : ""}
                  {section.dates ? ` · ${section.dates}` : ""}
                </p>
              </div>
              <span className="ed-mono">{section.goods.length} 件</span>
            </div>
            <GoodsGrid slug={booth.slug} goods={section.goods} showEventName={false} />
          </section>
        ))}
      </section>

      {booth.bundles.length > 0 ? (
        <section className="ed-shelf" id="bundles">
          <div className="ed-shelf__head">
            <h2>套餐推荐</h2>
            <span className="ed-mono">{booth.bundles.length} 套</span>
          </div>
          <BundlesGrid slug={booth.slug} bundles={booth.bundles} />
        </section>
      ) : null}

      <section className="ed-shelf" id="events">
        <div className="ed-shelf__head">
          <h2>活动与摊位</h2>
          <span className="ed-mono">信息以现场安排为准</span>
        </div>
        <EventList slug={booth.slug} events={[...(booth.events.live ?? []), ...(booth.events.scheduled ?? []), ...(booth.events.ended ?? [])]} />
      </section>

      <footer className="store-footer">
        <span>由 Stallink 摊位链接提供</span>
        <span>/c/{booth.slug}</span>
      </footer>
    </main>
  );
}
