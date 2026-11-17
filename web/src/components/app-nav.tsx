"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";
import { clearCounterDraftsForAccount } from "@/lib/counter-draft";

type Account = { id: string; role: string; shopSlug?: string };

const destinations: Record<string, { href: string; label: string }> = {
  customer: { href: "/me", label: "我的账号" },
  sales: { href: "/counter", label: "收银工作台" },
  shop_admin: { href: "/admin", label: "店铺工作台" },
  platform_admin: { href: "/platform", label: "平台工作台" },
};

export function AppNav() {
  const pathname = usePathname();
  const router = useRouter();
  const [account, setAccount] = useState<Account | null | undefined>(undefined);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [theme, setTheme] = useState<"light" | "dark" | null>(null);

  useEffect(() => {
    const preference = window.matchMedia("(prefers-color-scheme: dark)");
    const resolve = (): "light" | "dark" => {
      let saved: string | null = null;
      try { saved = window.localStorage.getItem("stallink.theme"); } catch { /* Use system preference. */ }
      return saved === "light" || saved === "dark" ? saved : preference.matches ? "dark" : "light";
    };
    const sync = () => {
      const next = resolve();
      document.documentElement.dataset.theme = next;
      setTheme(next);
    };
    sync();
    preference.addEventListener("change", sync);
    return () => preference.removeEventListener("change", sync);
  }, []);

  function toggleTheme() {
    const next = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    setTheme(next);
    try { window.localStorage.setItem("stallink.theme", next); } catch { /* Keep this session's choice. */ }
  }

  useEffect(() => {
    let active = true;
    browserJSON("/api/auth/me")
      .then((res) => {
        if (active) {
          setAccount(res.ok ? { id: String(res.data.id ?? ""), role: String(res.data.role ?? ""), shopSlug: String(res.data.shopSlug ?? "") } : null);
        }
      })
      .catch(() => {
        if (active) setAccount(null);
      });
    return () => {
      active = false;
    };
  }, [pathname]);

  async function logout() {
    setLoading(true);
    setError("");
    try {
      const res = await browserJSON("/api/auth/logout", "POST");
      if (!res.ok) {
        setError("退出失败，请重试。");
        return;
      }
      if (account?.id) clearCounterDraftsForAccount(account.id);
      setAccount(null);
      router.replace("/login");
    } catch {
      setError("网络连接失败，请检查连接后重试。");
    } finally {
      setLoading(false);
    }
  }

  const destination = account ? destinations[account.role] : undefined;
  const isWorkspace = pathname.startsWith("/admin") || pathname.startsWith("/counter") || pathname.startsWith("/platform");
  const routeSlug = pathname.startsWith("/c/") ? pathname.split("/")[2] : "";
  const storeSlug = routeSlug || account?.shopSlug;
  const brandHref = storeSlug ? `/c/${storeSlug}` : "/";

  return (
    <header className={`site-header${isWorkspace ? " site-header--workspace" : ""}`}>
      <div className="site-header__inner">
        <Link href={brandHref} className="site-brand" aria-label={storeSlug ? "返回当前摊位" : "Stallink 首页"}>
          <span className="site-brand__mark" aria-hidden="true">s.</span>
          <span>Stallink</span>
        </Link>
        <p className="site-header__note">{isWorkspace ? "现场订单 · 库存管理" : "独立摊位 · 现场好找"}</p>
        <nav className="site-header__actions" aria-label="账号导航">
          {theme ? <button type="button" className="theme-toggle" onClick={toggleTheme} aria-label={theme === "dark" ? "切换至浅色模式" : "切换至深色模式"}>{theme === "dark" ? "浅色" : "深色"}</button> : null}
          {account === undefined ? <><span className="header-loading" aria-hidden="true" /><span className="sr-only" role="status">正在加载账号…</span></> : account ? (
            <>
              {account.shopSlug ? <Link href={`/c/${account.shopSlug}`} className="header-link">查看店铺</Link> : null}
              {destination ? <Link href={destination.href} className="header-account">{destination.label}<span aria-hidden="true">↗</span></Link> : null}
              <button type="button" className="header-logout" onClick={logout} disabled={loading}>
                {loading ? "退出中" : "退出"}
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="header-link">登录</Link>
              <Link href="/register" className="header-account">创建账号<span aria-hidden="true">↗</span></Link>
            </>
          )}
        </nav>
        {error ? <p className="site-header__error" role="alert">{error}</p> : null}
      </div>
    </header>
  );
}
