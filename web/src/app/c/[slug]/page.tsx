import { Suspense } from "react";
import { notFound } from "next/navigation";
import { BoothView } from "@/components/booth-view";
import { StorefrontLoading } from "@/components/storefront-loading";
import { getShop } from "@/lib/api";

async function Shop({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const booth = await getShop(slug);
  if (!booth) notFound();
  return <BoothView booth={booth} />;
}

export default function ShopPage({ params }: { params: Promise<{ slug: string }> }) {
  return (
    <Suspense fallback={<StorefrontLoading>正在打开摊位…</StorefrontLoading>}>
      <Shop params={params} />
    </Suspense>
  );
}
