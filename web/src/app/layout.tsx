import type { Metadata } from "next";
import { Suspense } from "react";
import { AppNav } from "@/components/app-nav";
import "./globals.css";

export const metadata: Metadata = {
  title: "Stallink",
  description: "以独立链接作为摊位入口",
};

const themeBootstrap = `(function(){try{var t=window.localStorage.getItem("stallink.theme");if(t!=="light"&&t!=="dark"){t=window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";}document.documentElement.dataset.theme=t;}catch(e){document.documentElement.dataset.theme="light";}})();`;

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeBootstrap }} />
      </head>
      <body>
        <a className="skip-link" href="#main-content">跳到主要内容</a>
        <Suspense fallback={null}>
          <AppNav />
        </Suspense>
        <div id="main-content" tabIndex={-1}>{children}</div>
      </body>
    </html>
  );
}
