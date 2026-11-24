"use client";

import { useEffect } from "react";
import Link from "next/link";

export default function GlobalError({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
  useEffect(() => {
    // 兜底记录；不向顾客展示内部错误细节。
    console.error(error);
  }, [error]);

  return (
    <main className="wrap empty-page public-page-state">
      <p className="eyebrow">页面出错</p>
      <h1>页面暂时无法打开</h1>
      <p>页面加载失败。你可以重试，或返回首页。</p>
      <div className="error-page__actions">
        <button type="button" className="button button--dark" onClick={retry}>重试</button>
        <Link href="/" className="button button--outline">返回首页</Link>
      </div>
    </main>
  );
}
