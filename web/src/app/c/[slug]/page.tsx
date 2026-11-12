import { Suspense } from "react";
import { BoothView } from "@/components/booth-view";
import { getShop } from "@/lib/api";

async function Shop({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const booth = await getShop(slug);
  if (!booth) {
    return (
      <main className="mx-auto max-w-xl px-5 py-16">
        <h1 className="text-3xl font-semibold">这个链接没有摊位</h1>
        <p className="mt-4 text-stone-600">没有找到这一家，页面也不会列出其他 coser。</p>
      </main>
    );
  }
  return <BoothView booth={booth} />;
}

export default function ShopPage({ params }: { params: Promise<{ slug: string }> }) {
  return (
    <Suspense fallback={<p className="px-5 py-16 text-sm text-stone-500">正在打开摊位…</p>}>
      <Shop params={params} />
    </Suspense>
  );
}
