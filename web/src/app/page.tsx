import { Suspense } from "react";
import { BoothView } from "@/components/booth-view";
import { MarketingLanding } from "@/components/marketing-landing";
import { getLanding } from "@/lib/api";

async function Landing() {
  try {
    const booth = await getLanding();
    if (booth) return <BoothView booth={booth} />;
  } catch (error) {
    console.error(error);
    return (
      <main className="ed-frame empty-page">
        <p className="eyebrow">首页</p>
        <h1>摊位暂时打不开</h1>
        <p>首页没有拿到摊位数据。再打开一次即可；如果还是这样，稍后再来。</p>
      </main>
    );
  }
  return <MarketingLanding />;
}

export default function Home() {
  return (
    <Suspense fallback={<div className="ed-frame page-loading"><span className="loading-mark" />正在开门…</div>}>
      <Landing />
    </Suspense>
  );
}
