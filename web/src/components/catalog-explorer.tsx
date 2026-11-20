"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SafeImage } from "@/components/safe-image";
import { type Good } from "@/lib/api";
import { formatMoney, lowestUnitRate } from "@/lib/pricing";
type SortKey = "relevance" | "price_asc" | "price_desc" | "newest";

const sortOptions: { key: SortKey; label: string }[] = [
  { key: "relevance", label: "综合" },
  { key: "price_asc", label: "价格从低到高" },
  { key: "price_desc", label: "价格从高到低" },
  { key: "newest", label: "最新上架" },
];

export function CatalogExplorer({ initialGoods }: { initialGoods: Good[] }) {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortKey>("relevance");
  const [inStockOnly, setInStockOnly] = useState(false);
  const [goods, setGoods] = useState<Good[]>(initialGoods);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [focused, setFocused] = useState(false);
  const debounceRef = useRef<number | undefined>(undefined);
  const inputRef = useRef<HTMLInputElement>(null);

  const load = useCallback((q: string, s: SortKey) => {
    window.clearTimeout(debounceRef.current);
    debounceRef.current = window.setTimeout(async () => {
      setLoading(true);
      setError("");
      try {
        const params = new URLSearchParams();
        if (q.trim()) params.set("q", q.trim());
        if (s !== "relevance") params.set("sort", s);
        const qs = params.toString();
        const res = await fetch(`/api/public/catalog${qs ? `?${qs}` : ""}`, { cache: "no-store" });
        if (!res.ok) throw new Error("catalog request failed");
        const data: { goods?: Good[] } = await res.json();
        setGoods(Array.isArray(data.goods) ? data.goods : []);
      } catch {
        setError("商品加载失败，请稍后重试。");
      } finally {
        setLoading(false);
      }
    }, 280);
  }, []);

  useEffect(() => () => window.clearTimeout(debounceRef.current), []);

  useEffect(() => {
    load(query, sort);
  }, [query, sort, load]);

  const visible = useMemo(() => {
    const list = inStockOnly ? goods.filter((good) => !good.soldOut && good.remaining > 0) : goods;
    return list;
  }, [goods, inStockOnly]);

  const suggestions = useMemo(() => {
    const q = query.trim();
    if (!q) return [];
    const seen = new Set<string>();
    const out: string[] = [];
    for (const good of goods) {
      if (good.name.toLowerCase().includes(q.toLowerCase()) && !seen.has(good.name)) {
        seen.add(good.name);
        out.push(good.name);
      }
      if (out.length >= 6) break;
    }
    return out;
  }, [goods, query]);

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "Enter") inputRef.current?.blur();
  };

  return (
    <section className="catalog" aria-label="在售商品">
      <div className="catalog__search">
        <div className={`catalog__box${focused ? " catalog__box--focus" : ""}`}>
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M7 13A6 6 0 1 0 7 1a6 6 0 0 0 0 12zm4-1 3.5 3.5" /></svg>
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onFocus={() => setFocused(true)}
            onBlur={() => window.setTimeout(() => setFocused(false), 120)}
            onKeyDown={onKeyDown}
            placeholder="搜商品、摊位或活动…"
            aria-label="搜索商品"
            aria-busy={loading}
          />
          {query ? (
            <button type="button" className="catalog__clear" onClick={() => { setQuery(""); inputRef.current?.focus(); }} aria-label="清空搜索">×</button>
          ) : null}
          {suggestions.length > 0 && focused ? (
            <ul className="catalog__suggest" role="listbox">
              {suggestions.map((name) => (
                <li key={name}>
                  <button type="button" onMouseDown={(event) => { event.preventDefault(); setQuery(name); }} role="option" aria-selected={false}>
                    <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="7" cy="7" r="4" /><path d="M10 10l3 3" /></svg>
                    <span>{name}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        <div className="catalog__toolbar">
          <div className="catalog__sorts" role="tablist" aria-label="排序方式">
            {sortOptions.map((option) => (
              <button
                key={option.key}
                type="button"
                role="tab"
                aria-selected={sort === option.key}
                className={`catalog__sort${sort === option.key ? " catalog__sort--active" : ""}`}
                onClick={() => setSort(option.key)}
              >
                {option.label}
              </button>
            ))}
          </div>
          <label className="catalog__stock">
            <input type="checkbox" checked={inStockOnly} onChange={(event) => setInStockOnly(event.target.checked)} />
            仅看有货
          </label>
          <span className="catalog__count ed-mono">{visible.length} 件商品</span>
        </div>
      </div>

      {error ? <p className="catalog__error" role="alert">{error}</p> : null}

      {visible.length === 0 ? (
        <div className="empty-state">
          <span className="empty-state__mark" aria-hidden="true">✳</span>
          <p>{query ? `没有找到与「${query}」相关的商品。` : "当前还没有上架商品。"}</p>
        </div>
      ) : (
        <ul className="product-grid catalog__grid">
          {visible.map((good) => {
            const soldOut = good.soldOut ?? good.remaining <= 0;
            const offers = (good.priceTiers ?? []).filter((tier) => tier.quantity <= good.remaining);
            const placeholder = (
              <div className="product-placeholder" aria-label={`${good.name} 暂无商品图片`}>
                <span className="product-placeholder__name">{good.name}</span>
                <span className="product-placeholder__note">暂无商品图片</span>
              </div>
            );
            const href = good.shopSlug ? `/c/${good.shopSlug}/listings/${good.listingId}` : "#";
            return (
              <li key={good.listingId}>
                <Link href={href} className={`product-card${soldOut ? " product-card--soldout" : ""}`}>
                  <figure className="product-card__media">
                    {good.imageURL ? (
                      <SafeImage src={good.imageURL} alt={good.name} fill sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, 25vw" unoptimized className="product-card__image" fallback={placeholder} />
                    ) : placeholder}
                    {soldOut ? <span className="product-card__soldout-badge">售罄</span> : null}
                  </figure>
                  <div className="product-card__info">
                    <h3>{good.name}</h3>
                    <p className="catalog__meta">{[good.shopName, good.eventName].filter(Boolean).join(" · ")}</p>
                    <div className="product-card__meta">
                      <span className="product-card__price">
                        {offers.length > 0 ? <>{formatMoney(good.currency, Math.round(lowestUnitRate(good.unitPriceMinor, offers)))}<em className="product-card__from">起</em></> : good.priceLabel}
                      </span>
                      <span className="product-card__stock">{soldOut ? "售罄" : `余 ${good.remaining}`}</span>
                    </div>
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
