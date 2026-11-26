"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { appearancePath, isStorefrontTheme, type StorefrontTheme } from "@/lib/storefront-theme";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { appearancePublishLabel, type AppearanceMeta } from "@/lib/appearance-status";
import "./appearance.css";

const options = [
  {
    value: "classic" as const,
    name: "现有风格",
    detail: "顾客端是纸色背景与编辑式排版；工作台沿用同一套纸色和衬线标题。",
    swatches: ["#faf8f3", "#16171a", "#c6bdaa"],
    swatchLabel: "纸色 · 墨色 · 暖灰线",
  },
  {
    value: "exhibition" as const,
    name: "展场风格",
    detail: "顾客端是展场海报与炭黑竖栏；工作台使用深青色固定侧栏。",
    swatches: ["#f4f0e6", "#262820", "#8c3c2d", "#e3f7a2", "#215e58"],
    swatchLabel: "纸色 · 炭黑 · 赭红 · 柠绿 · 深青",
  },
  {
    value: "shop" as const,
    name: "店面风格",
    detail: "顾客端是白底浮卡与圆角搜索；工作台沿用同一套白底和紫色主操作。",
    swatches: ["#f2f4f5", "#000000", "#4524db"],
    swatchLabel: "浅灰底 · 黑色 · 紫色",
  },
];

function themeName(value: StorefrontTheme) {
  return options.find((item) => item.value === value)?.name ?? value;
}

type ErrorKind = "load" | "save" | "auth" | "forbidden" | "";

function AppearanceForm() {
  const router = useRouter();
  const [saved, setSaved] = useState<StorefrontTheme | null>(null);
  const [selection, setSelection] = useState<StorefrontTheme | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [offline, setOffline] = useState(false);
  const [error, setError] = useState("");
  const [errorKind, setErrorKind] = useState<ErrorKind>("");
  const [notice, setNotice] = useState("");
  const [remoteUpdated, setRemoteUpdated] = useState("");
  const [attempt, setAttempt] = useState(0);
  const [publishMeta, setPublishMeta] = useState<AppearanceMeta | null>(null);
  const [justPublished, setJustPublished] = useState(false);
  const messageRef = useRef<HTMLDivElement>(null);
  const savedRef = useRef<StorefrontTheme | null>(null);

  useEffect(() => {
    const sync = () => setOffline(typeof navigator !== "undefined" && navigator.onLine === false);
    sync();
    const onOffline = () => setOffline(true);
    const onOnline = () => {
      setOffline(false);
      setAttempt((value) => value + 1);
    };
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", onOnline);
    return () => {
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", onOnline);
    };
  }, []);

  useEffect(() => {
    let active = true;
    browserJSON(appearancePath).then((res) => {
      const previousSaved = savedRef.current;
      if (!active) return;
      if (res.category === "unauthorized") {
        setSaved(null);
        setSelection(null);
        setError("登录已过期，请重新登录。");
        setErrorKind("auth");
        return;
      }
      if (res.category === "forbidden") {
        setSaved(null);
        setSelection(null);
        setError("只有平台管理员可以修改平台外观");
        setErrorKind("forbidden");
        return;
      }
      if (res.ok && isStorefrontTheme(res.data.theme)) {
        const next = res.data.theme;
        if (previousSaved && previousSaved !== next) {
          setRemoteUpdated(`当前风格刚刚被更新为「${themeName(next)}」`);
        }
        savedRef.current = next;
        setSaved(next);
        setSelection((current) => (current && previousSaved && current !== previousSaved ? current : next));
        setPublishMeta({
          updatedAt: typeof res.data.updatedAt === "string" ? res.data.updatedAt : null,
          updatedBy: res.data.updatedBy && typeof res.data.updatedBy === "object" ? (res.data.updatedBy as AppearanceMeta["updatedBy"]) : null,
        });
        setJustPublished(false);
        return;
      }
      setSaved(null);
      setSelection(null);
      setError(res.ok ? "无法识别当前风格，请重新读取。" : describeRequest(res));
      setErrorKind("load");
    });
    return () => { active = false; };
  }, [attempt]);

  useEffect(() => {
    if (error || notice) messageRef.current?.focus();
  }, [error, notice]);

  async function publish(skipConfirm = false) {
    if (!saved || !selection || selection === saved || saving || offline) return;
    if (!skipConfirm) {
      setConfirmOpen(true);
      return;
    }
    setConfirmOpen(false);
    setSaving(true);
    setError("");
    setErrorKind("");
    setNotice("");
    try {
      const res = await browserJSON(appearancePath, "PUT", { theme: selection });
      if (res.category === "unauthorized") {
        setError("登录已过期，请重新登录。");
        setErrorKind("auth");
        return;
      }
      if (res.category === "forbidden") {
        setError("只有平台管理员可以修改平台外观");
        setErrorKind("forbidden");
        return;
      }
      if (!res.ok || !isStorefrontTheme(res.data.theme)) {
        setError(res.ok ? "未能确认发布结果，请刷新后检查。" : describeRequest(res));
        setErrorKind("save");
        return;
      }
      setSaved(res.data.theme);
      setSelection(res.data.theme);
      setPublishMeta({
        updatedAt: typeof res.data.updatedAt === "string" ? res.data.updatedAt : null,
        updatedBy: res.data.updatedBy && typeof res.data.updatedBy === "object" ? (res.data.updatedBy as AppearanceMeta["updatedBy"]) : null,
      });
      setJustPublished(true);
      setNotice(themeName(res.data.theme));
      router.refresh();
    } finally {
      setSaving(false);
    }
  }

  const dirty = Boolean(saved && selection && selection !== saved);
  const blocked = errorKind === "auth" || errorKind === "forbidden";
  const loadFailed = errorKind === "load";

  return (
    <main className="workspace-subpage appearance-page">
      {offline ? <p className="appearance-offline" role="status">网络已断开，恢复后会自动重新读取。</p> : null}
      <p className="eyebrow">平台</p>
      <h1>平台外观</h1>
      <p>选择全平台使用的界面风格。发布后会应用到顾客逛展、摊主管理、现场收银和平台工作台。</p>

      {blocked ? (
        <div className="appearance-message appearance-message--error" role="alert" ref={messageRef} tabIndex={-1}>
          <strong>{errorKind === "auth" ? "登录已过期，请重新登录。" : "只有平台管理员可以修改平台外观"}</strong>
          {errorKind === "forbidden" ? <p>你的账号没有这项权限，或者登录身份已变更。请换用平台管理员账号登录。</p> : null}
          <div className="appearance-message__actions">
            <a className="button button--dark" href="/login?next=/platform/appearance">重新登录</a>
            {errorKind === "forbidden" ? <Link className="button button--outline" href="/">返回 Stallink</Link> : null}
          </div>
        </div>
      ) : (
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void publish(false);
          }}
        >
          {saved && !loadFailed ? (
            <div className="appearance-status">
              <p>当前使用：{themeName(saved)}</p>
              <p>{(() => {
                const publishStatus = appearancePublishLabel(publishMeta, { justPublished });
                return publishStatus === "你 · 刚刚" || publishStatus === "尚无记录" ? `上次发布：${publishStatus}` : publishStatus;
              })()}</p>

              {remoteUpdated ? <p aria-live="polite">{remoteUpdated}</p> : null}
              <p>发布后，顾客端和所有工作台在下次打开或刷新页面时切换。</p>
            </div>
          ) : null}

          <fieldset className="appearance-options" disabled={!saved || saving || loadFailed} aria-busy={saving || undefined}>
            <legend className="sr-only">选择平台风格</legend>
            {!saved && !error ? (
              <p className="appearance-status" role="status">正在读取当前风格…</p>
            ) : null}
            {options.map(({ value, name, detail, swatchLabel }) => (
              <label className="appearance-option" key={value}>
                <span aria-hidden="true" className={`appearance-preview appearance-preview--${value}`}>
                  <span className="appearance-preview__face appearance-preview__face--public">
                    <span className="appearance-preview__tag">顾客端</span>
                    <b>{value === "classic" ? "Stallink." : value === "exhibition" ? "Artist Alley." : "Stallink"}</b>
                  </span>
                  <span className="appearance-preview__face appearance-preview__face--workspace">
                    <span className="appearance-preview__tag">工作台</span>
                    <i />
                  </span>
                </span>
                <span className="appearance-label">
                  <input
                    type="radio"
                    name="storefront-theme"
                    value={value}
                    checked={selection === value}
                    onChange={() => { setSelection(value); setError(""); setNotice(""); }}
                  />
                  <strong>{name}</strong>
                  {saved === value && !loadFailed ? <span className="appearance-tag appearance-tag--live">使用中</span> : null}
                  {dirty && selection === value ? <span className="appearance-tag appearance-tag--draft">已选 · 未发布</span> : null}
                </span>
                <span className="appearance-detail" id={`appearance-detail-${value}`}>{detail}</span>
                <span className="appearance-swatches">{swatchLabel}</span>
              </label>
            ))}
          </fieldset>

          {loadFailed ? (
            <div className="appearance-message appearance-message--error" role="alert" ref={messageRef} tabIndex={-1}>
              <strong>没能读取当前风格</strong>
              <p>{error} 读取成功前不能发布，以免覆盖别人刚发布的设置。</p>
              <button type="button" className="button button--outline" onClick={() => setAttempt((value) => value + 1)}>重新读取</button>
            </div>
          ) : null}

          {offline && saved ? (
            <div className="appearance-message appearance-message--info" role="status">
              <p>当前离线，暂时不能发布。你的选择已保留。</p>
            </div>
          ) : null}

          {errorKind === "save" ? (
            <div className="appearance-message appearance-message--error" role="alert" ref={messageRef} tabIndex={-1}>
              <strong>发布没有成功，前台仍是「{saved ? themeName(saved) : "当前风格"}」</strong>
              <p>{error} 你的选择已保留。</p>
              <button type="button" className="button button--dark" onClick={() => void publish(true)} disabled={offline || saving}>重新发布</button>
            </div>
          ) : null}

          {notice ? (
            <div className="appearance-message appearance-message--ok" role="status" ref={messageRef} tabIndex={-1}>
              <strong>已发布「{notice}」</strong>
              <p>本页已换成新风格。顾客和工作人员下次打开或刷新页面时生效，正在收银的页面不会被打断。</p>
              <a href="/" target="_blank" rel="noopener" className="button button--outline">在新标签页查看前台 ↗</a>
            </div>
          ) : null}

          {saving ? <p className="sr-only" aria-live="assertive">正在发布{selection ? themeName(selection) : ""}</p> : null}

          {!loadFailed && saved ? (
            <div className="appearance-actions">
              <button type="submit" className="button button--dark" disabled={!dirty || saving || offline}>
                {saving ? "发布中…" : "发布所选风格"}
              </button>
              {saving ? <span>正在发布，请勿关闭页面。</span> : null}
              {!dirty && !saving ? <span>选择与当前不同的风格后才能发布。</span> : null}
              {dirty ? (
                <button type="button" className="text-link" onClick={() => { setSelection(saved); setError(""); setNotice(""); }}>还原为当前风格</button>
              ) : null}
              <a href="/" target="_blank" rel="noopener" className="button button--outline">查看前台 ↗</a>
            </div>
          ) : null}
        </form>
      )}

      <ConfirmDialog
        open={confirmOpen}
        danger={false}
        initialFocus="cancel"
        pending={saving}
        title={selection ? `把全平台换成「${themeName(selection)}」？` : "确认发布"}
        description={(
          <ul>
            <li>顾客端、摊主工作台、现场收银和平台工作台都会换成新风格</li>
            <li>已打开的页面不会被打断，刷新或重新打开后生效</li>
            <li>可以随时回到这里改回「{saved ? themeName(saved) : "当前风格"}」</li>
          </ul>
        )}
        confirmLabel="确认发布"
        onConfirm={() => void publish(true)}
        onCancel={() => setConfirmOpen(false)}
      />
    </main>
  );
}

export default function AppearancePage() {
  return <AppearanceForm />;
}
