"use client";

import { useState } from "react";
import { SafeImage } from "@/components/safe-image";
import type { GoodImage } from "@/lib/api";

export function ProductGallery({ name, images }: { name: string; images: GoodImage[] }) {
  const [activeIndex, setActiveIndex] = useState(0);
  const safeIndex = Math.min(activeIndex, Math.max(images.length - 1, 0));
  const activeImage = images[safeIndex];
  const placeholder = (
    <div className="product-detail__placeholder">
      <span className="product-detail__placeholder-label">STALLINK / ITEM</span>
      <strong>{name}</strong>
      <span>摊位现场可见实物</span>
    </div>
  );

  return (
    <div className="product-gallery">
      <figure className="product-detail__media">
        {activeImage ? (
          <SafeImage src={activeImage.url} alt={`${name}，图片 ${safeIndex + 1}`} fill sizes="(max-width: 800px) 100vw, 60vw" unoptimized className="product-detail__image" priority fallback={placeholder} />
        ) : placeholder}
      </figure>
      {images.length > 1 ? (
        <div className="product-gallery__thumbs" role="group" aria-label="商品相册">
          {images.map((image, index) => (
            <button key={image.id} type="button" aria-label={`查看商品图片 ${index + 1}`} aria-pressed={safeIndex === index} onClick={() => setActiveIndex(index)}>
              <SafeImage src={image.url} alt="" width={72} height={72} unoptimized fallbackClassName="product-gallery__thumb--fallback" />
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
