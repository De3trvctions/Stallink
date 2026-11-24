"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { SafeImage } from "@/components/safe-image";
import { ShelfCopy } from "@/components/shelf-copy";
import { type Good } from "@/lib/api";
import { formatMoney, lowestUnitRate } from "@/lib/pricing";
type SortKey = "relevance" | "price_asc" | "price_desc" | "newest";

const sortOptions: { key: SortKey; label: string }[] = [
  { key: "relevance", label: "综合" },
  { key: "price_asc", label: "价格从低到高" },
  { key: "price_desc", label: "价格从高到低" },
  { key: "newest", label: "最新上架" },
];

const unavailableMessage = "暂时无法加载商品目录，请重试。";

export function CatalogExplorer({ initialGoods, initialUnavailable }: { initialGoods: Good[]; initialUnavailable: boolean }) {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortKey>("relevance");
  const [inStockOnly, setInStockOnly] = useState(false);
  const [goods, setGoods] = useState<Good[]>(initialGoods);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(initialUnavailable ? unavailableMessage : "");
  const [request, setRequest] = useState(0);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (request === 0) return;
    const controller = new AbortController();
    const debounce = window.setTimeout(async () => {
      try {
        const params = new URLSearchParams();
        if (query.trim()) params.set("q", query.trim());
        if (sort !== "relevance") params.set("sort", sort);
        const qs = params.toString();
        const res = await fetch(`/api/public/catalog${qs ? `?${qs}` : ""}`, { cache: "no-store", signal: controller.signal });
        if (!res.ok) throw new Error("catalog request failed");
        const data: { goods?: Good[] } = await res.json();
        if (!Array.isArray(data.goods)) throw new Error("catalog response has no goods");
        if (!controller.signal.aborted) setGoods(data.goods);
      } catch {
        if (!controller.signal.aborted) setError(unavailableMessage);
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 280);
    return () => {
      window.clearTimeout(debounce);
      controller.abort();
    };
  }, [query, sort, request]);

  function refresh() {
    setError("");
    setLoading(true);
    setRequest((previous) => previous + 1);
  }

  function changeQuery(value: string) {
    if (value === query) return;
    setQuery(value);
    refresh();
  }

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

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== "Enter") return;
    if (event.nativeEvent.isComposing || event.keyCode === 229) return;
    event.preventDefault();
    inputRef.current?.blur();
  };

  return (
    <section className="catalog" aria-label="在售商品">
      <div className="catalog__search">
        <div
          className={`catalog__box${focused ? " catalog__box--focus" : ""}`}
          onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false); }}
          onKeyDown={(event) => {
            if (event.key !== "Escape" || !focused || event.nativeEvent.isComposing) return;
            event.preventDefault();
            inputRef.current?.focus();
            setFocused(false);
          }}
        >
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M7 13A6 6 0 1 0 7 1a6 6 0 0 0 0 12zm4-1 3.5 3.5" /></svg>
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(event) => { setFocused(true); changeQuery(event.target.value); }}
            onFocus={() => setFocused(true)}
            onKeyDown={onKeyDown}
            placeholder="搜商品、摊位或活动…"
            aria-label="搜索商品"
          />
          {query ? (
            <button type="button" className="catalog__clear" onClick={() => { changeQuery(""); inputRef.current?.focus(); }} aria-label="清空搜索">×</button>
          ) : null}
          {suggestions.length > 0 && focused ? (
            <ul className="catalog__suggest" role="list" aria-label="搜索建议">
              {suggestions.map((name) => (
                <li key={name}>
                  <button type="button" onClick={() => { changeQuery(name); inputRef.current?.focus(); setFocused(false); }}>
                    <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="7" cy="7" r="4" /><path d="M10 10l3 3" /></svg>
                    <span>{name}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
        <div className="catalog__toolbar">
          <div className="catalog__sorts" role="group" aria-label="排序方式">
            {sortOptions.map((option) => (
              <button
                key={option.key}
                type="button"
                aria-pressed={sort === option.key}
                className={`catalog__sort${sort === option.key ? " catalog__sort--active" : ""}`}
                onClick={() => { if (sort !== option.key) { setSort(option.key); refresh(); } }}
              >
                {option.label}
              </button>
            ))}
          </div>
          <label className="catalog__stock">
            <input type="checkbox" checked={inStockOnly} onChange={(event) => setInStockOnly(event.target.checked)} />
            仅看有货
          </label>
          {loading ? <span className="catalog__count ed-mono">正在加载商品…</span> : null}
          <span className={loading ? "sr-only" : "catalog__count ed-mono"} role="status">{!loading && !error ? `${visible.length} 件商品` : ""}</span>
        </div>
      </div>

      {error ? (
        <div className="catalog__failure">
          <p className="catalog__error" role="alert">{error}</p>
          <button type="button" className="button button--outline" onClick={() => { inputRef.current?.focus(); refresh(); }}>重新加载商品</button>
        </div>
      ) : null}

      {visible.length === 0 && !loading && !error ? (
        <div className="empty-state">
          <span className="empty-state__mark" aria-hidden="true">✳</span>
          <p>{inStockOnly ? "当前筛选没有有货商品，可取消「仅看有货」查看其他作品。" : query ? `没有找到与「${query}」相关的商品。` : "当前还没有上架商品。"}</p>
        </div>
      ) : visible.length > 0 && !error ? (
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
                  <ShelfCopy
                    title={good.name}
                    eventLine={[good.shopName, good.eventName].filter(Boolean).join(" · ") || undefined}
                    eventClassName="catalog__meta"
                    classicPrice={offers.length > 0 ? <>{formatMoney(good.currency, Math.round(lowestUnitRate(good.unitPriceMinor, offers)))}<em className="product-card__from">起</em></> : good.priceLabel}
                    unitPrice={good.priceLabel}
                    remaining={good.remaining}
                    soldOut={soldOut}
                    tier={offers[0] ? `${offers[0].quantity} 件一组 ${formatMoney(good.currency, offers[0].totalPriceMinor)}` : null}
                  />
                </Link>
              </li>
            );
          })}
        </ul>
      ) : null}
    </section>
  );
}
