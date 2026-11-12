import Image from "next/image";
import type { Booth } from "@/lib/api";
import { BundlesGrid, EventList, GoodsGrid } from "@/components/goods-grid";

export function BoothView({ booth }: { booth: Booth }) {
  const activeEvent = booth.events.live?.[0] ?? booth.events.scheduled?.[0];
  const featuredGood = booth.goods[0];
  const isLive = Boolean(booth.events.live?.length);

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
                {activeEvent ? (isLive ? "本场摊位 / 进行中" : "下一场活动") : "摊位信息"}
              </span>
              <div className="ed-pass__name">{activeEvent ? activeEvent.name : "线上浏览，现场见面"}</div>
              <div className="ed-pass__meta">
                {activeEvent ? (
                  <>
                    <span>{[activeEvent.city, activeEvent.venue].filter(Boolean).join(" / ") || "地点待公布"}</span>
                    <span>摊位 {activeEvent.boothNumber || "待公布"} / {activeEvent.startsOn} – {activeEvent.endsOn}</span>
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
          </div>
        </div>

        <figure className="ed-figure ed-store__figure">
          <div className="ed-figure__frame">
            {featuredGood?.imageURL ? (
              <Image
                src={featuredGood.imageURL}
                alt={featuredGood.name}
                fill
                sizes="(max-width: 900px) 100vw, 45vw"
                unoptimized
                priority
              />
            ) : (
              <div className="ed-store__poster">
                <span>{booth.name}</span>
              </div>
            )}
          </div>
          <figcaption className="ed-placard">
            <strong>{featuredGood ? `本场主推 / ${featuredGood.name}` : "摊位主视觉待上传"}</strong>
            <span>{isLive ? "LIVE" : "NEXT"}</span>
          </figcaption>
        </figure>
      </section>

      <section className="ed-shelf" id="goods">
        <div className="ed-shelf__head">
          <h2>正在贩售</h2>
          <span className="ed-mono">{booth.goods.length} 件商品</span>
        </div>
        <GoodsGrid slug={booth.slug} goods={booth.goods} />
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
