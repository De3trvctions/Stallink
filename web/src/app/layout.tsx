import type { Metadata } from "next";
import { Suspense } from "react";
import { AppNav } from "@/components/app-nav";
import "./globals.css";

export const metadata: Metadata = {
  title: "Stallink",
  description: "以独立链接作为摊位入口",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="zh-CN">
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
