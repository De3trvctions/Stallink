export function StorefrontLoading({ children }: { children: React.ReactNode }) {
  return (
    <main className="wrap page-loading">
      <p role="status"><span className="loading-mark" aria-hidden="true" />{children}</p>
    </main>
  );
}
