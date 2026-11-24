"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { AuthLayout } from "@/components/auth-layout";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ loginName: "", password: "", displayName: "", contact: "" });
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  function set(key: keyof typeof form, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      const res = await browserJSON("/api/auth/register", "POST", form);
      if (!res.ok) {
        setError(res.errorCode === "login_taken"
          ? "这个账号已经被使用，请换一个账号。"
          : describeRequest(res));
        return;
      }
      router.replace("/me");
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthLayout
      register
      eyebrow="创建顾客账号"
      title="创建你的账号"
      exhibitionTitle="创建逛展账号"
      description="保存你希望摊位记住的称呼和联系方式。"
      footer={<>已经有账号？ <Link href="/login">直接登录 <span aria-hidden="true">↗</span></Link></>}
    >
      <form className="auth-form" onSubmit={submit}>
        <label className="field" htmlFor="register-login-name">
          <span>账号 <small>3-32 个字符，不含空格</small></span>
          <input
            id="register-login-name"
            name="username"
            autoComplete="username"
            autoCapitalize="none"
            spellCheck={false}
            minLength={3}
            maxLength={32}
            pattern="\S{3,32}"
            required
            value={form.loginName}
            onChange={(event) => set("loginName", event.target.value)}
            placeholder="例如：mika_art"
            data-error="register-login-error"
          />
          <span id="register-login-error" className="field__validity">错误：账号需要 3 到 32 个字符，且不能包含空格。</span>
        </label>
        <div className="field">
          <label className="field__label" htmlFor="register-password"><span>密码</span><small>至少 8 个字符</small></label>
          <span className="field__control">
            <input
              id="register-password"
              name="new-password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              minLength={8}
              maxLength={72}
              required
              value={form.password}
              onChange={(event) => set("password", event.target.value)}
              placeholder="创建一个密码"
              data-error="register-password-error"
            />
            <button
              type="button"
              className="field__toggle"
              aria-label={showPassword ? "隐藏密码" : "显示密码"}
              aria-pressed={showPassword}
              onClick={() => setShowPassword((visible) => !visible)}
            >
              {showPassword ? "隐藏" : "显示"}
            </button>
          </span>
          <span id="register-password-error" className="field__validity">错误：密码至少需要 8 个字符。</span>
        </div>
        <div className="auth-form__divider"><span>个人资料</span><i /></div>
        <label className="field" htmlFor="register-display-name">
          <span>称呼 <small>选填</small></span>
          <input id="register-display-name" autoComplete="nickname" value={form.displayName} onChange={(event) => set("displayName", event.target.value)} placeholder="别人怎么称呼你" />
        </label>
        <label className="field" htmlFor="register-contact">
          <span>联系方式 <small>选填</small></span>
          <input id="register-contact" autoComplete="tel" value={form.contact} onChange={(event) => set("contact", event.target.value)} placeholder="电话、社交账号等" />
        </label>
        {error ? <p className="form-message form-message--error" role="alert">{error}</p> : null}
        <button className="button button--dark auth-form__submit" type="submit" disabled={pending}>
          {pending ? <><span className="button-spinner" aria-hidden="true" />正在创建</> : <>创建账号 <span aria-hidden="true">→</span></>}
        </button>
        <p className="auth-form__hint">顾客账号免费。摊位工作人员请使用店主提供的账号登录。</p>
      </form>
    </AuthLayout>
  );
}
