import Link from "next/link";

export default function NotFound() {
  return (
    <main className="wrap empty-page public-page-state">
      <p className="eyebrow">404 · 链接失效</p>
      <h1>这个页面找不到了</h1>
      <p>链接可能已更改，或该摊位、活动和作品已下架。</p>
      <Link href="/" className="button button--dark">返回首页逛展</Link>
    </main>
  );
}
