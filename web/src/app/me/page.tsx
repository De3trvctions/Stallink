"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AuthLayout } from "@/components/auth-layout";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";

export default function MePage() {
  const router = useRouter();
  const [displayName, setDisplayName] = useState("");
  const [contact, setContact] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const { message, setError, setSuccess } = useMessage();

  useEffect(() => {
    let active = true;
    browserJSON("/api/auth/me")
      .then((res) => {
        if (!active) return;
        if (!res.ok) {
          router.replace("/login");
          return;
        }
        if (res.data.role !== "customer") {
          router.replace(res.data.role === "sales" ? "/counter" : res.data.role === "platform_admin" ? "/platform" : "/admin");
          return;
        }
        setDisplayName(String(res.data.displayName ?? ""));
        setContact(String(res.data.contact ?? ""));
        setLoading(false);
      })
      .catch(() => {
        if (active) router.replace("/login");
      });
    return () => { active = false; };
  }, [router]);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    try {
      const res = await browserJSON("/api/me", "PATCH", { displayName, contact });
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess("资料已保存。");
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AuthLayout
      eyebrow="顾客账号"
      title="我的资料"
      description="更新现场开单时使用的称呼和联系方式。"
      footer={<Link href="/">继续逛摊位 <span aria-hidden="true">↗</span></Link>}
    >
      {loading ? (
        <div className="profile-loading" role="status"><span className="loading-mark" />正在读取账号资料…</div>
      ) : (
        <form className="auth-form" onSubmit={submit}>
          <label className="field" htmlFor="profile-display-name">
            <span>称呼</span>
            <input id="profile-display-name" autoComplete="nickname" value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="你希望别人怎么称呼你" />
          </label>
          <label className="field" htmlFor="profile-contact">
            <span>联系方式</span>
            <input id="profile-contact" autoComplete="tel" value={contact} onChange={(event) => setContact(event.target.value)} placeholder="电话、社交账号等" />
          </label>
          {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
          <button className="button button--dark auth-form__submit" type="submit" disabled={saving}>
            {saving ? <><span className="button-spinner" aria-hidden="true" />正在保存</> : <>保存资料 <span aria-hidden="true">→</span></>}
          </button>
        </form>
      )}
    </AuthLayout>
  );
}
