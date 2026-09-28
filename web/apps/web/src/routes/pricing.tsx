import { createFileRoute, Link } from "@tanstack/react-router";
import { Container, PricingTable, SiteFooter } from "../components/marketing";

export const Route = createFileRoute("/pricing")({ component: Pricing });

function Pricing() {
  return (
    <div className="bg-white text-zinc-900">
      <Container>
        <p className="pt-10 text-sm text-zinc-500"><Link to="/">Home</Link> / Pricing</p>
        <h1 className="mt-2 text-4xl font-bold tracking-tight">Test your app. Pick your plan.</h1>
        <p className="mt-3 max-w-2xl text-zinc-600">
          Upload your iOS, Android or TV build and drive it from your agent or browser.
          Every plan is monthly with streaming minutes and builds included — overage draws from credit at $0.03/min.
          Self-hosting on your own Mac minis is $0 forever.
        </p>
        <div className="mt-8"><PricingTable /></div>
        <div className="mt-8 grid gap-4 md:grid-cols-3 text-sm">
          <div className="rounded-xl border border-zinc-200 p-5"><h3 className="font-semibold">How rivals price it</h3><p className="mt-1 text-zinc-600">BrowserStack App Automate $199/mo per parallel · Sauce Real $199/mo · Appetize Starter $59/mo · AWS Device Farm $0.17/device-min (~$3,400/mo at 20k min). We price the org, not the seat or parallel.</p></div>
          <div className="rounded-xl border border-zinc-200 p-5"><h3 className="font-semibold">What counts as usage</h3><p className="mt-1 text-zinc-600">Streaming minutes run while a device session is open (concurrent caps per plan). Builds are per cloud compile. Overage $0.03/min — under every legacy cloud's per-minute math.</p></div>
          <div className="rounded-xl border border-zinc-200 p-5"><h3 className="font-semibold">Self-host = $0</h3><p className="mt-1 text-zinc-600">BYO Mac minis + <span className="font-mono text-xs">docker compose up</span>. FSL license: internal use, research and services all permitted purposes.</p></div>
        </div>
      </Container>
      <SiteFooter />
    </div>
  );
}
