"use client";

import Image from "next/image";
import { useState, type ComponentProps, type ReactNode } from "react";

type SafeImageProps = Omit<ComponentProps<typeof Image>, "onError"> & {
  /** 图片加载失败时渲染的既有占位内容；缺省渲染 fallbackClassName 容器。 */
  fallback?: ReactNode;
  /** 无自定义占位内容时使用的占位容器类名。 */
  fallbackClassName?: string;
};

/**
 * 远端商品图统一降级：加载失败（404 / 网络错误）时不再显示浏览器破图图标，
 * 而是回退到各页面既有的占位样式。文件预览（object URL）无需使用本组件。
 */
export function SafeImage({ fallback, fallbackClassName, alt, ...props }: SafeImageProps) {
  const [failed, setFailed] = useState(false);

  if (failed) {
    if (fallback) return <>{fallback}</>;
    return <div className={fallbackClassName} role="img" aria-label={alt} aria-hidden={!alt} />;
  }

  return <Image {...props} alt={alt} onError={() => setFailed(true)} />;
}

export default SafeImage;
