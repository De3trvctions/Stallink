"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";
import { clearCounterDraftsForAccount } from "@/lib/counter-draft";

type WorkspaceKind = "shop" | "sales" | "platform";

type RailIconName =
  | "grid" | "calendar" | "box" | "receipt" | "chart" | "scan" | "bag"
  | "card" | "users" | "store" | "history" | "shop" | "review" | "appearance";

type NavItem = { href: string; label: string; index?: string; icon?: RailIconName };
type NavGroup = { label: string; items: NavItem[] };

const railIcons: Record<RailIconName, string> = {
  grid: "M4 4h7v7H4z M13 4h7v7h-7z M4 13h7v7H4z M13 13h7v7h-7z",
  calendar: "M4 6h16v14H4z M8 3v6 M16 3v6 M4 11h16",
  box: "M3 8l9-5 9 5v8l-9 5-9-5z M3 8l9 5 9-5 M12 13v8",
  receipt: "M6 3h12v18l-3-2-3 2-3-2-3 2z M9 8h6 M9 12h6",
  chart: "M4 20V10 M10 20V4 M16 20V13 M2 20h20",
  scan: "M4 4h6v6H4z M14 4h6v6h-6z M4 14h6v6H4z M14 16h6 M17 13v6",
  bag: "M6 8h12l-1 12H7z M9 8a3 3 0 0 1 6 0",
  card: "M3 7h18v10H3z M3 11h18",
  users: "M9 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z M3 20v-2a5 5 0 0 1 10 0v2 M17 11a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z M17 14a4 4 0 0 1 4 4v2",
  store: "M4 9h16l-2-5H6z M5 9v11h14V9 M9 20v-6h6v6",
  history: "M12 7v5l3 2 M21 12a9 9 0 1 1-3-6.7 M21 4v5h-5",
  shop: "M4 10 12 4l8 6v10H4z M9 20v-5h6v5",
  review: "M4 4h16v16H4z M8 12l3 3 5-6",
  appearance: "M12 3l2.4 5.8 6.3.5-4.8 4.2 1.5 6.1-5.4-3.4-5.4 3.4 1.5-6.1L3.3 9.3l6.3-.5z",
};

function RailIcon({ name }: { name: RailIconName }) {
  return (
    <svg aria-hidden="true" width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round">
      <path d={railIcons[name]} />
    </svg>
  );
}

const cashierGroup: NavGroup = {
  label: "现场收银",
  items: [
    { href: "/counter", label: "现场开单", icon: "scan" },
    { href: "/counter/orders", label: "本场订单", icon: "bag" },
  ],
};

const workspace = {
  shop: {
    label: "店铺工作台",
    eyebrow: "STALLINK / SHOP DESK",
    description: "活动、商品、订单与现场收银",
    groups: [
      { label: "经营", items: [
        { href: "/admin", label: "总览", icon: "grid" },
        { href: "/admin/events", label: "活动", index: "01", icon: "calendar" },
        { href: "/admin/goods", label: "商品", index: "02", icon: "box" },
        { href: "/admin/orders", label: "订单记录", index: "03", icon: "receipt" },
        { href: "/admin/reports", label: "销售日报", index: "04", icon: "chart" },
      ] },
      cashierGroup,
      { label: "店铺", items: [
        { href: "/admin/pay-codes", label: "收款方式", index: "05", icon: "card" },
        { href: "/admin/sales", label: "销售账号", index: "06", icon: "users" },
        { href: "/admin/shop", label: "店铺资料", index: "07", icon: "store" },
        { href: "/admin/audit", label: "操作记录", index: "08", icon: "history" },
      ] },
    ],
  },
  sales: {
    label: "现场工作台",
    eyebrow: "STALLINK / POS",
    description: "开单、收款与交付",
    groups: [cashierGroup],
  },
  platform: {
    label: "平台工作台",
    eyebrow: "STALLINK / PLATFORM",
    description: "店铺管理与商品审核",
    groups: [
      { label: "平台", items: [
        { href: "/platform", label: "店铺管理", index: "01", icon: "shop" },
        { href: "/platform/goods", label: "商品审核", index: "02", icon: "review" },
        { href: "/platform/appearance", label: "平台外观", index: "03", icon: "appearance" },
      ] },
    ],
  },
} satisfies Record<WorkspaceKind, { label: string; eyebrow: string; description: string; groups: NavGroup[] }>;

export function WorkspaceShell({ kind, themeFallback = false, children }: { kind: WorkspaceKind; themeFallback?: boolean; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [account, setAccount] = useState<{ role: string; id: string } | null>(null);
  const [authError, setAuthError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [logoutError, setLogoutError] = useState("");
  const [shopName, setShopName] = useState("");
  const [loggingOut, setLoggingOut] = useState(false);
  const current = workspace[kind === "sales" && account?.role === "shop_admin" ? "shop" : kind];
  const activeItem = current.groups
    .flatMap((group) => group.items)
    .filter(({ href }) => pathname === href || pathname.startsWith(`${href}/`))
    .sort((left, right) => right.href.length - left.href.length)[0];

  useEffect(() => {
    let active = true;
    browserJSON("/api/auth/me")
      .then((res) => {
        if (!active) return;
        if (!res.ok) {
          if (res.status === 401) router.replace("/login");
          else setAuthError(true);
          return;
        }
        setAccount({ role: String(res.data.role ?? ""), id: String(res.data.id ?? "") });
        const slug = res.data.shopSlug;
        if (typeof slug === "string" && slug) {
          browserJSON(`/api/public/shops/${encodeURIComponent(slug)}`).then((shop) => {
            if (active && shop.ok && typeof shop.data.name === "string") setShopName(shop.data.name);
          });
        }
      }).catch(() => {
        if (active) setAuthError(true);
      });
    return () => { active = false; };
  }, [retry, router]);

  const role = account?.role;
  const allowed = kind === "sales" ? role === "sales" || role === "shop_admin"
    : kind === "shop" ? role === "shop_admin" : role === "platform_admin";
  useEffect(() => {
    if (role && !allowed) {
      const home: Record<string, string> = { customer: "/me", sales: "/counter", shop_admin: "/admin", platform_admin: "/platform" };
      router.replace(home[role] ?? "/login");
    }
  }, [role, allowed, router]);

  useEffect(() => {
    if (!allowed) return;
    const nav = document.getElementById("workspace-navigation");
    const current = nav?.querySelector<HTMLElement>("[aria-current='page']");
    if (!nav || !current) return;

    const alignCurrentIntoView = () => {
      const style = getComputedStyle(nav);
      const navRect = nav.getBoundingClientRect();
      const currentRect = current.getBoundingClientRect();
      if (style.overflowX === "auto" || style.overflowX === "scroll") {
        nav.scrollLeft += currentRect.left - navRect.left - (nav.clientWidth - currentRect.width) / 2;
      }
      if (style.overflowY === "auto" || style.overflowY === "scroll") {
        nav.scrollTop += currentRect.top - navRect.top - (nav.clientHeight - currentRect.height) / 2;
      }
    };

    alignCurrentIntoView();
    const frame = requestAnimationFrame(alignCurrentIntoView);
    return () => cancelAnimationFrame(frame);
  }, [pathname, allowed]);

  async function logout() {
    setLoggingOut(true);
    setLogoutError("");
    try {
      const response = await browserJSON("/api/auth/logout", "POST");
      if (!response.ok && response.status !== 401) {
        setLogoutError("退出失败，请重试。");
        return;
      }
      if (account?.id) clearCounterDraftsForAccount(account.id);
      setAccount(null);
      router.replace("/login");
    } finally {
      setLoggingOut(false);
    }
  }

  if (!allowed || authError) {
    return (
      <main className="gate-state" role={authError ? "alert" : "status"}>
        <span className={authError ? "gate-state__mark gate-state__mark--error" : "loading-mark"} aria-hidden="true">{authError ? "!" : null}</span>
        <div>
          <strong>{authError ? "无法验证账号权限" : "正在检查工作台权限"}</strong>
          <p>{authError ? "服务暂时无法连接，请重试。" : "确认账号权限后打开对应的工作空间。"}</p>
        </div>
        {authError ? <button type="button" className="button button--outline" onClick={() => { setAuthError(false); setRetry((value) => value + 1); }}>重新检查</button> : null}
      </main>
    );
  }

  return (
    <div className="workspace-shell">
      <aside className="workspace-sidebar">
        <Link href="/" className="workspace-sidebar__brand" aria-label="Stallink 首页">
          <span className="workspace-sidebar__logo" aria-hidden="true">S</span>
          <span className="workspace-sidebar__wordmark"><strong>STALLINK</strong><small>{current.label}</small></span>
        </Link>
        <div className="workspace-sidebar__identity">
          <strong>{shopName || current.label}</strong>
          <span>{shopName ? "店铺 · 现场运营" : current.eyebrow}</span>
        </div>
        <div className="workspace-sidebar__heading">
          <span className="workspace-sidebar__eyebrow">{current.eyebrow}</span>
          <h2>{current.label}</h2>
          <p>{current.description}</p>
        </div>
        <nav
          id="workspace-navigation"
          className="workspace-nav"
          aria-label={`${current.label}导航`}
        >
          {current.groups.map((group) => (
            <div className="workspace-nav__group" key={group.label}>
              <span className="workspace-nav__label">{group.label}</span>
              {group.items.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="workspace-nav__link"
                  data-active={activeItem?.href === item.href}
                  aria-current={activeItem?.href === item.href ? "page" : undefined}
                >
                  {item.icon ? <span className="workspace-nav__icon"><RailIcon name={item.icon} /></span> : null}
                  {item.index ? <span className="workspace-nav__index" aria-hidden="true">{item.index}</span> : null}
                  <span className="workspace-nav__text">{item.label}</span>
                </Link>
              ))}
            </div>
          ))}
        </nav>
        <div className="workspace-sidebar__foot">
          <Link href="/" className="workspace-sidebar__back"><span aria-hidden="true">←</span> 返回 Stallink</Link>
          <button type="button" className="workspace-sidebar__logout" disabled={loggingOut} onClick={() => void logout()}>{loggingOut ? "退出中…" : "退出登录"}</button>
          {logoutError && <p role="alert">{logoutError}</p>}
        </div>
      </aside>
      <div className="workspace-main">
        <div className="workspace-topbar">
          <span><i aria-hidden="true" />工作空间已连接</span>
          <Link href="/" className="workspace-topbar__link">返回 Stallink ↗</Link>
        </div>
        {themeFallback && account?.role === "platform_admin" ? (
          <div className="platform-theme-fallback" role="status">
            <div>
              <strong>外观设置暂时读取失败，全站临时使用「现有风格」</strong>
              <p>保存的设置没有丢，通常是外观服务短暂超时，刷新后会重新读取。这条提醒只有平台管理员能看到。</p>
            </div>
            <button type="button" className="button button--outline" onClick={() => router.refresh()}>重新检查</button>
          </div>
        ) : null}
        <div className="workspace-page">{children}</div>
      </div>
    </div>
  );
}
