import type { ReactNode } from "react";

export function shelfStock(remaining: number, soldOut: boolean) {
  return `${soldOut ? "售罄" : `余 ${remaining}`} · 全店共用`;
}

export function ShelfCopy({
  title,
  eventLine,
  eventClassName,
  classicPrice,
  unitPrice,
  remaining,
  soldOut,
  tier,
  tag,
  classicExtra,
}: {
  title: string;
  eventLine?: ReactNode;
  eventClassName?: string;
  classicPrice: ReactNode;
  unitPrice: string;
  remaining: number;
  soldOut: boolean;
  tier?: string | null;
  tag?: string | null;
  classicExtra?: ReactNode;
}) {
  return (
    <div className="product-card__info">
      {tag ? <span className="exhibition-only product-card__tag">{tag}</span> : null}
      <h3>{title}</h3>
      {eventLine ? <p className={eventClassName ? `classic-intro ${eventClassName}` : "classic-intro"}>{eventLine}</p> : null}
      <div className="product-card__meta">
        <span className="product-card__price">
          <span className="classic-intro">{classicPrice}</span>
          <span className="exhibition-only">{unitPrice}</span>
        </span>
        <span className="classic-intro product-card__stock">{soldOut ? "售罄" : `余 ${remaining}`}</span>
      </div>
      {classicExtra}
      {tier ? <p className="exhibition-only product-card__tier">{tier}</p> : null}
      <p className="exhibition-only product-card__stockline">{shelfStock(remaining, soldOut)}</p>
    </div>
  );
}
