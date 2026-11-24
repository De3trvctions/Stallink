import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFrame } from "@/components/site-frame";
import { FieldValidity } from "@/components/field-validity";
import { connection } from "next/server";
import { getStorefrontTheme } from "@/lib/api";
import "./globals.css";
import "./exhibition.css";
import "./exhibition-workspace.css";

export const metadata: Metadata = {
  title: "Stallink",
  description: "以独立链接作为摊位入口",
};

// 色板要先知道展场是否钉死浅色，根布局必须等这次读取结束再输出。
export const instant = false;

const themeBootstrap = `(function(){var meta=document.querySelector('meta[name="color-scheme"]');if(!meta||meta.getAttribute("data-pinned")==="light")return;try{var saved=localStorage.getItem("stallink.theme");if(saved==="light"||saved==="dark")meta.setAttribute("content",saved);}catch(e){}})();`;

export default async function RootLayout({ children }: LayoutProps<"/">) {
  await connection();
  const theme = await getStorefrontTheme();
  const pinnedLight = theme === "exhibition";
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <meta name="color-scheme" content={pinnedLight ? "light" : "light dark"} data-pinned={pinnedLight ? "light" : undefined} suppressHydrationWarning />
        <script dangerouslySetInnerHTML={{ __html: themeBootstrap }} />
      </head>
      <body>
        <a className="skip-link" href="#main-content">跳到主要内容</a>
        <FieldValidity />
        <Suspense fallback={<span className="sr-only" role="status">正在打开 Stallink…</span>}>
          <SiteFrame theme={theme}>{children}</SiteFrame>
        </Suspense>
      </body>
    </html>
  );
}
