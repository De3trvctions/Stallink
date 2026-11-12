"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { browserJSON } from "@/lib/api";

export function usePagedList<T>(endpoint: string, enabled = true) {
  const [input, setInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<T[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [version, setVersion] = useState(0);
  const sequence = useRef(0);
  useEffect(() => { const timer = window.setTimeout(() => { setPage(1); setQuery(input.trim()); }, 300); return () => window.clearTimeout(timer); }, [input]);
  useEffect(() => {
    const tracker = sequence;
    const id = ++tracker.current;
    if (!enabled) return;
    const timer = window.setTimeout(() => {
      setLoading(true); setError("");
      const params = new URLSearchParams({ page: String(page), limit: "20", q: query });
      browserJSON(`${endpoint}${endpoint.includes("?") ? "&" : "?"}${params}`).then((res) => {
        if (id !== sequence.current) return;
        if (!res.ok || !Array.isArray(res.data.items)) throw new Error("request failed");
        setItems(res.data.items as T[]); setHasMore(res.data.hasMore === true);
      }).catch(() => { if (id === sequence.current) { setError("加载失败，请检查网络后重试。"); setItems([]); setHasMore(false); } })
        .finally(() => { if (id === sequence.current) setLoading(false); });
    }, 0);
    return () => { ++tracker.current; window.clearTimeout(timer); };
  }, [endpoint, page, query, version, enabled]);
  const refresh = useCallback(() => setVersion((n) => n + 1), []);
  const reset = useCallback(() => setPage(1), []);
  return { input, setInput, page, setPage, items, hasMore, loading, error, refresh, reset };
}

export function ListControls({ value, onChange, page, setPage, hasMore, loading, placeholder }: {
  value: string; onChange: (value: string) => void; page: number; setPage: (page: number) => void; hasMore: boolean; loading: boolean; placeholder: string;
}) {
  return <div className="paged-search"><label className="field"><span>搜索</span><input type="search" maxLength={100} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} /></label>
    <nav className="paged-nav" aria-label="分页"><button type="button" onClick={() => setPage(page - 1)} disabled={page <= 1 || loading}>上一页</button><span aria-live="polite">第 {page} 页</span><button type="button" onClick={() => setPage(page + 1)} disabled={!hasMore || loading}>下一页</button></nav>
  </div>;
}
