"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState, useSyncExternalStore } from "react";
import { browserJSON } from "@/lib/api";
import { clearCounterDraftsForAccount } from "@/lib/counter-draft";
import type { StorefrontTheme } from "@/lib/storefront-theme";

type Account = { id: string; role: string; shopSlug?: string };

function subscribeToFragment(notify: () => void) {
  window.addEventListener("hashchange", notify);
  return () => window.removeEventListener("hashchange", notify);
}

function readStoredScheme() {
  try {
    const saved = window.localStorage.getItem("stallink.theme");
    return saved === "light" || saved === "dark" ? saved : null;
  } catch {
    return null;
  }
}

function storedScheme() {
  return readStoredScheme() ?? "light dark";
}

const destinations: Record<string, { href: string; label: string }> = {
  customer: { href: "/me", label: "我的账号" },
  sales: { href: "/counter", label: "收银工作台" },
  shop_admin: { href: "/admin", label: "店铺工作台" },
  platform_admin: { href: "/platform", label: "平台工作台" },
};

const railIcons = {
  home: "M3 10 12 3l9 7v10H3z",
  catalog: "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
  about: "M12 10v7 M12 7v.2 M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z",
  shop: "M4 10v11h16V10 M2 10l3-7h14l3 7z M9 21v-7h6v7",
  account: "M16 7a4 4 0 1 1-8 0 4 4 0 0 1 8 0z M4 21v-2a8 8 0 0 1 16 0v2",
};

function RailIcon({ name }: { name: keyof typeof railIcons }) {
  return <svg aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round"><path d={railIcons[name]} /></svg>;
}

export function AppNav({ storefrontTheme, isWorkspace }: { storefrontTheme: StorefrontTheme; isWorkspace: boolean }) {
  const pathname = usePathname();
  const hash = useSyncExternalStore(subscribeToFragment, () => window.location.hash, () => "");
  const router = useRouter();
  const [account, setAccount] = useState<Account | null | undefined>(undefined);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const meta = document.querySelector<HTMLMetaElement>('meta[name="color-scheme"]');
    if (!meta) return;
    if (storefrontTheme === "exhibition") {
      meta.dataset.pinned = "light";
      meta.content = "light";
      return () => {
        delete meta.dataset.pinned;
        meta.content = storedScheme();
      };
    }
    const onStorage = (event: StorageEvent) => {
      if (event.key !== "stallink.theme") return;
      meta.content = event.newValue === "light" || event.newValue === "dark" ? event.newValue : "light dark";
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, [storefrontTheme]);

  function toggleTheme() {
    const meta = document.querySelector<HTMLMetaElement>('meta[name="color-scheme"]');
    if (!meta || meta.dataset.pinned === "light") return;
    const system = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
    const saved = readStoredScheme();
    const rendered = saved ?? system;
    const target = rendered === "dark" ? "light" : "dark";
    try {
      if (target === system) window.localStorage.removeItem("stallink.theme");
      else window.localStorage.setItem("stallink.theme", target);
    } catch { /* Keep this session's choice. */ }
    meta.content = target === system ? "light dark" : target;
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
  const routeSlug = pathname.startsWith("/c/") ? pathname.split("/")[2] : "";
  const storeSlug = routeSlug || account?.shopSlug;
  const brandHref = storeSlug ? `/c/${storeSlug}` : "/";

  if (storefrontTheme === "exhibition" && isWorkspace) {
    return null;
  }

  if (storefrontTheme === "exhibition" && !isWorkspace) {
    const HomeLink = pathname === "/" ? "a" : Link;
    return (
      <header className="exhibition-nav">
        <Link href="/" className="exhibition-nav__brand" aria-label="Stallink 首页">Stallink<span aria-hidden="true">.</span></Link>
        <nav className="exhibition-nav__links" aria-label="展场导航">
          <HomeLink href={pathname === "/" ? "#" : "/"} aria-current={pathname === "/" && hash !== "#catalog" ? "page" : undefined}><RailIcon name="home" /><span>逛展</span></HomeLink>
          <HomeLink href={pathname === "/" ? "#catalog" : "/#catalog"} aria-current={pathname === "/" && hash === "#catalog" ? "location" : undefined}><RailIcon name="catalog" /><span>作品</span></HomeLink>
          {storeSlug ? <Link href={brandHref} aria-current={pathname === brandHref ? "page" : undefined}><RailIcon name="shop" /><span>摊位</span></Link> : null}
          <Link href="/about" aria-current={pathname === "/about" ? "page" : undefined}><RailIcon name="about" /><span>关于</span></Link>
        </nav>
        <div className="exhibition-nav__settings">
          {account === undefined ? <span className="sr-only" role="status">正在加载账号…</span> : account ? (
            <>
              {destination ? <Link href={destination.href} aria-current={pathname === destination.href ? "page" : undefined}><RailIcon name="account" /><span>{account.role === "customer" ? "我的" : "工作台"}</span></Link> : null}
              <button type="button" disabled={loading} onClick={logout}>{loading ? "退出中" : "退出"}</button>
            </>
          ) : (
            <>
              <Link href="/login" aria-current={pathname === "/login" ? "page" : undefined}><RailIcon name="account" /><span>登录</span></Link>
              <Link href="/register" aria-current={pathname === "/register" ? "page" : undefined}>注册</Link>
            </>
          )}
          {error ? <span role="alert">{error}</span> : null}
        </div>
      </header>
    );
  }

  return (
    <header className={`site-header${isWorkspace ? " site-header--workspace" : ""}`}>
      <div className="site-header__inner">
        <Link href={brandHref} className="site-brand" aria-label={storeSlug ? "返回当前摊位" : "Stallink 首页"}>
          <span className="site-brand__mark" aria-hidden="true">s.</span>
          <span>Stallink</span>
        </Link>
        <p className="site-header__note">{isWorkspace ? "现场订单 · 库存管理" : "独立摊位 · 现场好找"}</p>
        <nav className="site-header__actions" aria-label="账号导航">
          <button type="button" className="theme-toggle" onClick={toggleTheme}>
            <span className="theme-toggle__to-light"><span className="sr-only">切换至</span>浅色<span className="sr-only">模式</span></span>
            <span className="theme-toggle__to-dark"><span className="sr-only">切换至</span>深色<span className="sr-only">模式</span></span>
          </button>
          {account === undefined ? <><span className="header-loading" aria-hidden="true" /><span className="sr-only" role="status">正在加载账号…</span></> : account ? (
            <>
              <Link href="/about" className="header-link">关于</Link>
              {account.shopSlug ? <Link href={`/c/${account.shopSlug}`} className="header-link">查看店铺</Link> : null}
              {destination ? <Link href={destination.href} className="header-account">{destination.label}<span aria-hidden="true">↗</span></Link> : null}
              <button type="button" className="header-logout" onClick={logout} disabled={loading}>
                {loading ? "退出中" : "退出"}
              </button>
            </>
          ) : (
            <>
              <Link href="/about" className="header-link">关于</Link>
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
