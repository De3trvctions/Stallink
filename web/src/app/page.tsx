import { Suspense } from "react";
import { CatalogExplorer } from "@/components/catalog-explorer";
import { getCatalog, type Good } from "@/lib/api";

async function CatalogData() {
  let goods: Good[] = [];
  try {
    const result = await getCatalog();
    goods = result.goods;
  } catch {
    goods = [];
  }
  return <CatalogExplorer initialGoods={goods} />;
}

export default function HomePage() {
  return (
    <main className="ed-frame">
      <section className="ed-hero ed-hero--compact">
        <div>
          <div className="ed-kicker">
            <span className="ed-dash" aria-hidden="true" />
            <span className="ed-mono">独立摊位 / 一场一会</span>
          </div>
          <h1 className="ed-title">逛遍这场展的摊位好货。</h1>
          <p className="ed-desc">搜索商品、按价格排序，先在线上看图和价，再走到摊位现场相见。</p>
        </div>
        <div className="ed-hero__aside">
          <a className="ed-link" href="/about">了解 Stallink ↗</a>
        </div>
      </section>
      <Suspense fallback={<div className="empty-state"><span className="empty-state__mark" aria-hidden="true">✳</span><p>正在加载商品…</p></div>}>
        <CatalogData />
      </Suspense>
    </main>
  );
}
