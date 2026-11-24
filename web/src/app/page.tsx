import Link from "next/link";
import { Suspense } from "react";
import { CatalogExplorer } from "@/components/catalog-explorer";
import { getCatalog, type Good } from "@/lib/api";

async function CatalogData() {
  let goods: Good[] = [];
  let unavailable = false;
  try {
    const result = await getCatalog();
    goods = result.goods;
  } catch {
    unavailable = true;
  }
  return (
    <>
      <div className="exhibition-only exhibition-catalog__head">
        <h2>展场里的作品</h2>
      </div>
      <CatalogExplorer initialGoods={goods} initialUnavailable={unavailable} />
    </>
  );
}

export default function HomePage() {
  return (
    <>
      <header className="exhibition-only exhibition-pagebar">
        <span className="exhibition-pagebar__brand">Stallink.</span>
        <span className="exhibition-pagebar__meta">全站 / 作品目录</span>
      </header>
      <main className="ed-frame exhibition-home">
        <section className="ed-hero ed-hero--compact classic-intro">
          <div>
            <div className="ed-kicker">
              <span className="ed-dash" aria-hidden="true" />
              <span className="ed-mono">独立摊位 / 一场一会</span>
            </div>
            <h1 className="ed-title">逛遍这场展的摊位好货。</h1>
            <p className="ed-desc">搜索商品、按价格排序，先在线上看图和价，再走到摊位现场相见。</p>
          </div>
          <div className="ed-hero__aside">
            <Link className="ed-link" href="/about">了解 Stallink <span aria-hidden="true">↗</span></Link>
          </div>
        </section>
        <section className="exhibition-only exhibition-poster">
          <div className="exhibition-poster__copy">
            <p className="exhibition-poster__label">STALLINK / 独立创作者展场</p>
            <h1 lang="en">Artist<br />Alley.</h1>
            <p className="exhibition-poster__tagline">在展场，遇见创作者。</p>
            <a className="exhibition-action" href="#catalog">逛展场作品 <span aria-hidden="true">↓</span></a>
          </div>
        </section>
        <div className="exhibition-only exhibition-strip" aria-hidden="true">
          <span>展前线上浏览</span>
          <span>多摊位 · 现场自取</span>
          <span>STALLINK · ARTIST ALLEY</span>
        </div>
        <div className="exhibition-catalog" id="catalog">
          <Suspense fallback={<div className="empty-state" role="status"><span className="empty-state__mark" aria-hidden="true">✳</span><p>正在加载商品…</p></div>}>
            <CatalogData />
          </Suspense>
        </div>
      </main>
    </>
  );
}
