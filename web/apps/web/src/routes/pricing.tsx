import { createFileRoute, Link } from "@tanstack/react-router";
import { CompareTable, Container, PricingTable, SiteFooter } from "../components/marketing";

export const Route = createFileRoute("/pricing")({ component: Pricing });

const PLAN_ROWS: [string, string[]][] = [
  ["Monthly price", ["$0", "$15", "$39", "$149"]],
  ["Streaming minutes / month", ["30", "200", "500", "3,000"]],
  ["Cloud builds / month", ["10", "25", "50", "200"]],
  ["Concurrent sessions", ["2", "2", "4", "25"]],
  ["Members", ["3", "3", "10", "Unlimited"]],
  ["Live guests per session", ["1", "1", "2", "5"]],
  ["Embed on your site", ["—", "—", "1 domain", "Unlimited + SSO"]],
  ["Streaming overage", ["$0.03/min", "$0.03/min", "$0.03/min", "$0.03/min"]],
];

function Pricing() {
  return (
    <div className="bg-black text-zinc-100">
      <Container>
        <p className="pt-10 text-sm text-zinc-500"><Link to="/">Home</Link> / Pricing</p>
        <h1 className="mt-2 text-4xl font-bold tracking-tight text-white">Test your app. Pick your plan.</h1>
        <p className="mt-3 max-w-2xl text-zinc-400">
          Upload your iOS, Android or TV build and drive it from your agent or browser.
          Every plan is monthly with streaming minutes and builds included — overage draws from credit at $0.03/min.
          Self-hosting on your own Mac minis is $0 forever.
        </p>
        <div className="mt-8"><PricingTable /></div>
        <h2 className="mt-10 text-2xl font-bold tracking-tight text-white">Compare plans in detail</h2>
        <div className="mt-4">
          <CompareTable
            columns={[
              { key: "f", header: "Feature" },
              { key: "free", header: "Free $0/mo" },
              { key: "dev", header: "Dev $15/mo" },
              { key: "team", header: "Team $39/mo" },
              { key: "scale", header: "Scale $149/mo" },
            ]}
            rows={PLAN_ROWS.map(([feature, cells]) => ({
              feature,
              category: "Plans",
              cells: cells.map((v, i) => <span key={i} className="text-zinc-300">{v}</span>),
            }))}
          />
        </div>
        <div className="mt-8 grid gap-4 md:grid-cols-3 text-sm">
          <div className="rounded-2xl border border-white/10 bg-zinc-950 p-5"><h3 className="font-semibold text-white">How rivals price it</h3><p className="mt-1 text-zinc-400">VibeView Dev $19/mo + $0.04/min overage · BrowserStack App Automate $199/mo per parallel · Sauce Real $199/mo · Appetize Starter $59/mo · AWS Device Farm $0.17/device-min (~$3,400/mo at 20k min). We price the org, not the seat or parallel.</p></div>
          <div className="rounded-2xl border border-white/10 bg-zinc-950 p-5"><h3 className="font-semibold text-white">What counts as usage</h3><p className="mt-1 text-zinc-400">Streaming minutes run while a device session is open (concurrent caps per plan). Builds are per cloud compile. Overage $0.03/min — under every legacy cloud's per-minute math.</p></div>
          <div className="rounded-2xl border border-white/10 bg-zinc-950 p-5"><h3 className="font-semibold text-white">Self-host = $0</h3><p className="mt-1 text-zinc-400">BYO Mac minis + <span className="font-mono text-xs">docker compose up</span>. FSL license: internal use, research and services all permitted purposes.</p></div>
        </div>
      </Container>
      <SiteFooter />
    </div>
  );
}
