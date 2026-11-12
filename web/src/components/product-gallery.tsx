"use client";

import Image from "next/image";
import { useState } from "react";
import type { GoodImage } from "@/lib/api";

export function ProductGallery({ name, images }: { name: string; images: GoodImage[] }) {
  const [activeIndex, setActiveIndex] = useState(0);
  const safeIndex = Math.min(activeIndex, Math.max(images.length - 1, 0));
  const activeImage = images[safeIndex];

  return (
    <div className="product-gallery">
      <figure className="product-detail__media">
        {activeImage ? (
          <Image src={activeImage.url} alt={`${name}，图片 ${safeIndex + 1}`} fill sizes="(max-width: 800px) 100vw, 60vw" unoptimized className="product-detail__image" priority />
        ) : (
          <div className="product-detail__placeholder">
            <span className="product-detail__placeholder-label">STALLINK / ITEM</span>
            <strong>{name}</strong>
            <span>摊位现场可见实物</span>
          </div>
        )}
      </figure>
      {images.length > 1 ? (
        <div className="product-gallery__thumbs" role="group" aria-label="商品相册">
          {images.map((image, index) => (
            <button key={image.id} type="button" aria-label={`查看商品图片 ${index + 1}`} aria-pressed={safeIndex === index} onClick={() => setActiveIndex(index)}>
              <Image src={image.url} alt="" width={72} height={72} unoptimized />
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
