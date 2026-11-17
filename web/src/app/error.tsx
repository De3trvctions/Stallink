"use client";

import { useEffect } from "react";
import Link from "next/link";

export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // 兜底记录；不向顾客展示内部错误细节。
    console.error(error);
  }, [error]);

  return (
    <main className="wrap empty-page">
      <p className="eyebrow">页面出错</p>
      <h1>页面暂时无法打开</h1>
      <p>发生了意外错误，可以重试；如果反复出现，请稍后再访问。</p>
      <div className="error-page__actions">
        <button type="button" className="button button--dark" onClick={reset}>重试</button>
        <Link href="/" className="button button--outline">返回首页</Link>
      </div>
    </main>
  );
}
