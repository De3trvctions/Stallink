"use client";

import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import type { StorefrontTheme } from "@/lib/storefront-theme";
import { AppNav } from "./app-nav";
import { WorkspaceShell } from "./workspace-shell";

const workspaceKinds = { admin: "shop", counter: "sales", platform: "platform" } as const;

export function SiteFrame({ theme, fallback = false, children }: { theme: StorefrontTheme; fallback?: boolean; children: ReactNode }) {
  const section = usePathname().split("/")[1];
  const kind = Object.hasOwn(workspaceKinds, section) ? workspaceKinds[section as keyof typeof workspaceKinds] : null;
  return (
    <div id="storefront-shell" data-storefront-theme={theme} data-theme-fallback={fallback ? "true" : undefined} data-surface={kind ? "workspace" : "public"}>
      <AppNav storefrontTheme={theme} isWorkspace={kind !== null} />
      <div id="main-content" tabIndex={-1}>
        {kind ? <WorkspaceShell kind={kind} themeFallback={fallback}>{children}</WorkspaceShell> : children}
      </div>
    </div>
  );
}
