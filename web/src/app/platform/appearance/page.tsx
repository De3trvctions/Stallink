"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { appearancePath, isStorefrontTheme, type StorefrontTheme } from "@/lib/storefront-theme";
import "./appearance.css";

const options = [
  { value: "classic", name: "现有风格", detail: "纸色背景、编辑式排版，保留当前的浏览体验。" },
  { value: "exhibition", name: "展场风格", detail: "顾客浏览展场海报，工作人员使用深青色固定侧栏。" },
  { value: "shop", name: "店面风格", detail: "白底浮卡，圆角搜索，紫色只用于标识和主操作。" },
] as const;

function AppearanceForm() {
  const router = useRouter();
  const [saved, setSaved] = useState<StorefrontTheme | null>(null);
  const [selection, setSelection] = useState<StorefrontTheme | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    browserJSON(appearancePath).then((res) => {
      if (!active) return;
      if (res.ok && isStorefrontTheme(res.data.theme)) {
        setSaved(res.data.theme);
        setSelection(res.data.theme);
      } else {
        setError(res.ok ? "无法识别当前风格，请重新读取。" : describeRequest(res));
      }
    });
    return () => { active = false; };
  }, [attempt]);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!saved || !selection || selection === saved || saving) return;
    setSaving(true);
    setError("");
    setNotice("");
    try {
      const res = await browserJSON(appearancePath, "PUT", { theme: selection });
      if (!res.ok || !isStorefrontTheme(res.data.theme)) {
        setError(res.ok ? "未能确认发布结果，请刷新后检查。" : describeRequest(res));
        return;
      }
      setSaved(res.data.theme);
      setSelection(res.data.theme);
      setNotice("已发布。打开或刷新平台页面即可看到新风格。");
      router.refresh();
    } finally {
      setSaving(false);
    }
  }

  return (
    <main>
      <h1>平台外观</h1>
      <p>选择全平台使用的界面风格。发布后会应用到顾客逛展、摊主管理、现场收银和平台工作台。</p>
      <form onSubmit={save}>
        <fieldset className="appearance-options" disabled={!saved || saving}>
          <legend className="sr-only">选择平台风格</legend>
          {options.map(({ value, name, detail }) => (
            <label className="appearance-option" key={value}>
              <span aria-hidden="true" className={`appearance-preview appearance-${value}`}>
                <i /><b>{value === "classic" ? "Stallink." : value === "exhibition" ? "Artist Alley." : "Stallink"}</b>
              </span>
              <span className="appearance-label">
                <input type="radio" name="storefront-theme" value={value} checked={selection === value} onChange={() => { setSelection(value); setError(""); setNotice(""); }} />
                <strong>{name}</strong>{saved === value && <small>使用中</small>}
              </span>
              <span className="appearance-detail">{detail}</span>
            </label>
          ))}
        </fieldset>
        {!saved && !error ? <p role="status">正在读取当前风格…</p> : null}
        {error ? <p role="alert">{error} {!saved && <button type="button" className="button button--outline" onClick={() => { setError(""); setAttempt(attempt + 1); }}>重试</button>}</p> : null}
        {notice ? <p role="status">{notice}</p> : null}
        <div className="appearance-actions">
          <button type="submit" className="button button--dark" disabled={!saved || selection === saved || saving}>{saving ? "发布中…" : "发布所选风格"}</button>
          <Link href="/" prefetch={false} className="button button--outline">查看前台 ↗</Link>
        </div>
      </form>
    </main>
  );
}

export default function AppearancePage() {
  return <AppearanceForm />;
}
