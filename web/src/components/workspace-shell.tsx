"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";

type WorkspaceKind = "shop" | "sales" | "platform";

type NavItem = { href: string; label: string; index: string };
type NavGroup = { label: string; items: NavItem[] };

const workspace = {
  shop: {
    label: "店铺工作台",
    eyebrow: "STALLINK / SHOP DESK",
    description: "活动、商品、订单与现场收银",
    groups: [
      { label: "经营", items: [
        { href: "/admin", label: "总览", index: "00" },
        { href: "/admin/events", label: "活动", index: "01" },
        { href: "/admin/goods", label: "商品", index: "02" },
        { href: "/admin/orders", label: "订单", index: "03" },
        { href: "/admin/reports", label: "销售日报", index: "04" },
      ] },
      { label: "现场", items: [
        { href: "/counter", label: "现场收银", index: "05" },
        { href: "/admin/pay-codes", label: "收款方式", index: "06" },
      ] },
      { label: "店铺", items: [
        { href: "/admin/sales", label: "销售账号", index: "07" },
        { href: "/admin/shop", label: "店铺资料", index: "08" },
        { href: "/admin/audit", label: "操作记录", index: "09" },
      ] },
    ],
  },
  sales: {
    label: "现场工作台",
    eyebrow: "STALLINK / POS",
    description: "开单、收款与交付",
    groups: [
      { label: "本场", items: [
        { href: "/counter", label: "创建订单", index: "POS" },
        { href: "/counter/orders", label: "本场订单", index: "ORD" },
      ] },
    ],
  },
  platform: {
    label: "平台工作台",
    eyebrow: "STALLINK / PLATFORM",
    description: "店铺管理与商品审核",
    groups: [
      { label: "平台", items: [
        { href: "/platform", label: "店铺管理", index: "01" },
        { href: "/platform/goods", label: "商品审核", index: "02" },
      ] },
    ],
  },
} satisfies Record<WorkspaceKind, { label: string; eyebrow: string; description: string; groups: NavGroup[] }>;

export function WorkspaceShell({ kind, children }: { kind: WorkspaceKind; children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [checking, setChecking] = useState(true);
  const [authError, setAuthError] = useState(false);
  const [retry, setRetry] = useState(0);
  const current = workspace[kind];
  const activeHref = current.groups
    .flatMap((group) => group.items.map((item) => item.href))
    .filter((href) => pathname === href || pathname.startsWith(`${href}/`))
    .sort((left, right) => right.length - left.length)[0];

  useEffect(() => {
    let active = true;
    browserJSON("/api/auth/me")
      .then((res) => {
        if (!active) return;
        if (!res.ok) {
          router.replace("/login");
          return;
        }
        const role = String(res.data.role ?? "");
        const allowed = kind === "shop"
          ? role === "shop_admin"
          : kind === "sales"
            ? role === "sales" || role === "shop_admin"
            : role === "platform_admin";
        if (allowed) {
          setChecking(false);
          return;
        }
        const landing: Record<string, string> = {
          customer: "/me",
          sales: "/counter",
          shop_admin: "/admin",
          platform_admin: "/platform",
        };
        router.replace(landing[role] ?? "/login");
      })
      .catch(() => {
        if (active) {
          setChecking(false);
          setAuthError(true);
        }
      });
    return () => { active = false; };
  }, [kind, retry, router]);

  if (checking || authError) {
    return (
      <main className="gate-state" role={authError ? "alert" : "status"}>
        <span className={authError ? "gate-state__mark gate-state__mark--error" : "loading-mark"} aria-hidden="true">{authError ? "!" : null}</span>
        <div>
          <strong>{authError ? "无法验证账号权限" : "正在检查工作台权限"}</strong>
          <p>{authError ? "服务暂时无法连接，请重试。" : "确认账号权限后打开对应的工作空间。"}</p>
        </div>
        {authError ? <button type="button" className="button button--outline" onClick={() => { setChecking(true); setAuthError(false); setRetry((value) => value + 1); }}>重新检查</button> : null}
      </main>
    );
  }

  return (
    <div className="workspace-shell">
      <aside className="workspace-sidebar">
        <div className="workspace-sidebar__heading">
          <span className="workspace-sidebar__eyebrow">{current.eyebrow}</span>
          <h2>{current.label}</h2>
          <p>{current.description}</p>
        </div>
        <nav className="workspace-nav" aria-label={`${current.label}导航`}>
          {current.groups.map((group) => (
            <div className="workspace-nav__group" key={group.label}>
              <span className="workspace-nav__label">{group.label}</span>
              {group.items.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="workspace-nav__link"
                  data-active={activeHref === item.href}
                  aria-current={activeHref === item.href ? "page" : undefined}
                >
                  <span className="workspace-nav__index" aria-hidden="true">{item.index}</span>
                  <span className="workspace-nav__text">{item.label}</span>
                </Link>
              ))}
            </div>
          ))}
        </nav>
        <Link href="/" className="workspace-sidebar__back"><span aria-hidden="true">←</span> 返回 Stallink</Link>
      </aside>
      <div className="workspace-main">
        <div className="workspace-topbar">
          <span><i aria-hidden="true" />工作空间已连接</span>
          <Link href="/" className="workspace-topbar__link">返回 Stallink ↗</Link>
        </div>
        <div className="workspace-page">{children}</div>
      </div>
    </div>
  );
}
