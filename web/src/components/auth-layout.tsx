import type { ReactNode } from "react";
import Link from "next/link";

export function AuthLayout({
  eyebrow,
  title,
  description,
  children,
  footer,
  register = false,
  profile = false,
  exhibitionTitle,
}: {
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
  footer: ReactNode;
  register?: boolean;
  profile?: boolean;
  exhibitionTitle?: string;
}) {
  return (
    <main className={`auth-page${register ? " auth-page--register" : ""}${profile ? " exhibition-profile" : ""}`}>
      <aside className="auth-story" aria-label="关于 Stallink">
        <Link href="/" className="auth-story__brand"><span>s.</span>Stallink</Link>
        <div className="auth-story__copy">
          <h2>喜欢的作品，<br />现场再见。</h2>
          <p>每一个创作者都有自己的摊位链接。记住链接，看看他们这次带来了什么。</p>
        </div>
        <span className="auth-story__foot">摊位 · 商品 · 活动</span>
      </aside>

      <section className="auth-panel">
        <div className="auth-panel__top"><span>{eyebrow}</span><Link href="/">返回首页</Link></div>
        <div className="auth-panel__body">
          <div className="auth-panel__heading">
            <h1>{exhibitionTitle ? <><span className="classic-intro">{title}</span><span className="exhibition-only">{exhibitionTitle}</span></> : title}</h1>
            <p>{description}</p>
          </div>
          {children}
          <div className="auth-panel__footer">{footer}</div>
        </div>
        <div className="auth-panel__bottom"><span>STALLINK ACCOUNT</span><span>© Stallink</span></div>
      </section>
    </main>
  );
}
