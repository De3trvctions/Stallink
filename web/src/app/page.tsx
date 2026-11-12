import { Suspense } from "react";
import { BoothView } from "@/components/booth-view";
import { MarketingLanding } from "@/components/marketing-landing";
import { getLanding } from "@/lib/api";

async function Landing() {
  const booth = await getLanding();
  if (booth) return <BoothView booth={booth} />;
  return <MarketingLanding />;
}

export default function Home() {
  return (
    <Suspense fallback={<div className="ed-frame page-loading"><span className="loading-mark" />正在开门…</div>}>
      <Landing />
    </Suspense>
  );
}
