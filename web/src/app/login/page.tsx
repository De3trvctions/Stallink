"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { AuthLayout } from "@/components/auth-layout";
import { browserJSON } from "@/lib/api";

const landing: Record<string, string> = {
  customer: "/me",
  sales: "/counter",
  shop_admin: "/admin",
  platform_admin: "/platform",
};

export default function LoginPage() {
  const router = useRouter();
  const [loginName, setLoginName] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPending(true);
    try {
      const res = await browserJSON("/api/auth/login", "POST", { loginName, password });
      if (!res.ok) {
        setError(res.status >= 500 ? "暂时无法登录，请稍后重试。" : "账号或密码不正确，请检查后重试。");
        return;
      }
      const role = String(res.data.role ?? "");
      router.replace(landing[role] ?? "/");
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthLayout
      eyebrow="账号登录"
      title="欢迎回来"
      description="登录后会根据你的账号权限打开对应页面。"
      footer={<>还没有账号？ <Link href="/register">创建顾客账号 <span aria-hidden="true">↗</span></Link></>}
    >
      <form className="auth-form" onSubmit={submit}>
        <label className="field" htmlFor="login-name">
          <span>账号</span>
          <input
            id="login-name"
            name="username"
            type="text"
            autoComplete="username"
            autoCapitalize="none"
            spellCheck={false}
            required
            value={loginName}
            onChange={(event) => setLoginName(event.target.value)}
            placeholder="输入你的账号"
          />
        </label>
        <div className="field">
          <label className="field__label" htmlFor="login-password">密码</label>
          <span className="field__control">
            <input
              id="login-password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="输入你的密码"
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
        </div>
        {error ? <p className="form-message form-message--error" role="alert">{error}</p> : null}
        <button className="button button--dark auth-form__submit" type="submit" disabled={pending}>
          {pending ? <><span className="button-spinner" aria-hidden="true" />正在登录</> : <>登录 <span aria-hidden="true">→</span></>}
        </button>
        <p className="auth-form__hint">销售、店铺管理员和平台管理员也使用各自的 Stallink 账号登录。</p>
      </form>
    </AuthLayout>
  );
}
