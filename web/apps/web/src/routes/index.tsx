import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { COMPETITORS } from "../data/comparison";
import { Container, CtaRow, Faq, PricingTable, SiteFooter, Terminal } from "../components/marketing";
import { BurstCanvas, SidekickMark } from "../components/brand";

export const Route = createFileRoute("/")({ component: HomeComponent });

const QUICKSTART = `npm i -g @imbios/sidekick
sidekick login   # or SIDEKICK_TOKEN
sidekick agent-setup   # skill + MCP
sidekick xcode build . --scheme MyApp
sidekick ios create --device "iPhone 17"`;

const LOOP = `sidekick ui-tree         # @e refs on screen
# ...agent edits; reload in ~1s...
sidekick tap @e7         # drive the flow
sidekick screenshot      # looks right? ship
sidekick record --stop   # demo video for PR`;

const PLATFORMS = [
  { tab: "iOS", device: "iPhone 17 · iOS 26", rows: ["Destination detail", "Checkout flow", "Settings"], verb: "$ sidekick tap @e7 · screenshot · record — streamed at 60fps" },
  { tab: "Android", device: "Pixel 9 · API 35", rows: ["Explore", "Checkout flow", "Settings"], verb: "$ sidekick tap @e4 — local adb/Appium just work via tunnel" },
  { tab: "Apple TV", device: "Apple TV 4K · tvOS 26", rows: ["Home screen", "Player", "Search"], verb: "$ sidekick press dpad_center · tap-focused — focus-walk verbs" },
  { tab: "Android TV", device: "Android TV · API 34", rows: ["Launcher", "Player", "Settings"], verb: "$ sidekick press dpad_right — same verbs as phones" },
  { tab: "Roku", device: "Roku Ultra · beta pool", rows: ["Home", "Channel", "Search"], verb: "🛣️ In Roadmap: real-hardware pool, Q1 — watch it land in the open" },
];

function Eyebrow({ children }: { children: string }) {
  return (
    <p className="text-center text-xs font-semibold tracking-[0.2em] text-zinc-500 uppercase">{children}</p>
  );
}

function SectionHead({ eyebrow, title, sub }: { eyebrow: string; title: string; sub: string }) {
  return (
    <div className="mx-auto max-w-2xl text-center">
      <Eyebrow>{eyebrow}</Eyebrow>
      <h2 className="mt-4 text-4xl font-bold tracking-tighter text-white md:text-5xl">{title}</h2>
      <p className="mt-4 text-lg text-zinc-400">{sub}</p>
    </div>
  );
}

/**
 * Live-session panel: a browser window (traffic lights + share URL) instead
 * of a phone mock — closer to how the product actually looks when shared.
 */
function SessionPanel() {
  const [active, setActive] = useState(0);
  const p = PLATFORMS[active];
  return (
    <div className="overflow-hidden rounded-3xl border border-white/10 bg-zinc-950 shadow-2xl">
      <div className="flex items-center gap-3 border-b border-white/10 px-5 py-3.5">
        <div className="flex gap-1.5" aria-hidden="true">
          <span className="h-3 w-3 rounded-full bg-zinc-700" />
          <span className="h-3 w-3 rounded-full bg-zinc-700" />
          <span className="h-3 w-3 rounded-full bg-zinc-700" />
        </div>
        <p className="mx-auto hidden rounded-full border border-white/10 bg-white/5 px-4 py-1 font-mono text-xs text-zinc-400 sm:block">
          sidekick.imbios.dev/s/sim_8f3a
        </p>
        <span className="rounded-full border border-emerald-400/20 bg-emerald-400/10 px-2.5 py-0.5 text-xs font-medium text-emerald-300">
          ● Live
        </span>
      </div>
      <div className="flex flex-wrap justify-center gap-1.5 px-5 pt-5" role="tablist" aria-label="Platforms">
        {PLATFORMS.map((t, i) => (
          <button
            key={t.tab}
            role="tab"
            aria-selected={i === active}
            onClick={() => setActive(i)}
            className={`rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${i === active ? "bg-white text-black" : "text-zinc-500 hover:text-white"}`}
          >
            {t.tab}
          </button>
        ))}
      </div>
      <div className="grid gap-6 p-6 md:grid-cols-2 md:p-8">
        <div>
          <p className="font-mono text-xs text-zinc-500">{p.device}</p>
          <ul className="mt-3 space-y-2.5">
            {p.rows.map((t, i) => (
              <li key={t} className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-zinc-200">
                {t}
                <span className="font-mono text-xs text-zinc-600">@e{i + 1}</span>
              </li>
            ))}
          </ul>
        </div>
        <div className="flex flex-col justify-center rounded-2xl border border-emerald-400/15 bg-emerald-400/[0.06] p-5">
          <p className="font-mono text-sm text-emerald-300">✓ agent verified: tap @e7 → diff 3 elements</p>
          <p className="mt-3 font-mono text-xs leading-relaxed text-zinc-500">{p.verb}</p>
        </div>
      </div>
    </div>
  );
}

function HomeComponent() {
  return (
    <div className="bg-black text-zinc-100">
      {/* HERO */}
      <div className="relative overflow-hidden">
        <BurstCanvas className="pointer-events-none absolute inset-0 h-full w-full opacity-50" />
        <Container>
          <div className="relative flex flex-col items-center px-4 pt-24 pb-20 text-center md:pt-32 md:pb-24">
            <SidekickMark size={104} />
            <p className="mt-8 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1.5 text-xs font-medium text-zinc-400">
              Source-available (FSL) · self-hostable · converts to Apache-2.0
            </p>
            <h1 className="mt-6 max-w-3xl text-5xl font-bold tracking-tighter text-white md:text-7xl">
              Every agent needs a sidekick to hold the phone.
            </h1>
            <p className="mt-6 max-w-2xl text-lg leading-relaxed text-zinc-400 md:text-xl">
              Cloud Xcode, iOS, Android and TV devices your coding agent builds, verifies and submits —
              on a live device it can see and touch. You watch along in the browser and take over anytime.
            </p>
            <div className="mt-10"><CtaRow /></div>
            <p className="mt-8 font-mono text-xs text-zinc-500">
              skill: <span className="text-zinc-200">sidekick agent-setup</span> · MCP: <span className="text-zinc-200">sidekick mcp</span> · spec: <a className="underline hover:text-zinc-200" href="/llms.txt">llms.txt</a> · <a className="underline hover:text-zinc-200" href="/openapi.json">openapi.json</a>
            </p>
          </div>
        </Container>
      </div>

      {/* LIVE SESSION */}
      <Container>
        <div className="mx-auto max-w-4xl">
          <SessionPanel />
          <p className="mt-6 text-center text-sm text-zinc-500">
            From first build to final check · iOS · Android · Apple TV · Android TV · Roku (roadmap) · from Claude Code, Cursor, Codex or any MCP client
          </p>
        </div>
      </Container>

      {/* BENTO — the loop as one grid */}
      <Container>
        <div className="py-24 md:py-32">
          <SectionHead
            eyebrow="How it works"
            title="From first build to final check."
            sub="One loop for phones, tablets and TVs. Your agent drives, your team watches, your CI keeps the receipts."
          />
          <div className="mt-14 grid gap-4 md:grid-cols-3">
            <div className="rounded-3xl border border-white/10 bg-zinc-950 p-8 md:col-span-2">
              <Eyebrow>Preview</Eyebrow>
              <h3 className="mt-3 text-2xl font-bold tracking-tight text-white">One link puts the live app in anyone&apos;s browser.</h3>
              <p className="mt-2 max-w-md text-zinc-400">Guests watch, take over control, and embed the session anywhere. No accounts, no installs — the device is a URL.</p>
              <p className="mt-5 inline-block rounded-full border border-white/10 bg-white/5 px-4 py-1.5 font-mono text-xs text-zinc-300">sidekick.imbios.dev/s/sim_8f3a</p>
            </div>
            <div className="rounded-3xl border border-white/10 bg-zinc-950 p-8">
              <Eyebrow>Build</Eyebrow>
              <h3 className="mt-3 text-2xl font-bold tracking-tight text-white">Cloud Xcode, no Mac.</h3>
              <p className="mt-2 text-zinc-400">Remote xcodebuild and Gradle with a warm cache. Signing handled, TestFlight and Play submit from the terminal.</p>
            </div>
            <div className="rounded-3xl border border-white/10 bg-zinc-950 p-8">
              <Eyebrow>Test</Eyebrow>
              <h3 className="mt-3 text-2xl font-bold tracking-tight text-white">Record once, replay everywhere.</h3>
              <p className="mt-2 text-zinc-400">Visual regression and CI checks keep running after the agent leaves.</p>
            </div>
            <div className="rounded-3xl border border-white/10 bg-zinc-950 p-8">
              <Eyebrow>TV</Eyebrow>
              <h3 className="mt-3 text-2xl font-bold tracking-tight text-white">The living room too.</h3>
              <p className="mt-2 text-zinc-400">Apple TV and Android TV with d-pad verbs and focus-walk. Roku on real hardware, in the open.</p>
            </div>
            <div className="rounded-3xl border border-white bg-white p-8 text-black">
              <p className="text-xs font-semibold tracking-[0.2em] text-zinc-500 uppercase">Self-host</p>
              <h3 className="mt-3 text-2xl font-bold tracking-tight">$0 on your own minis.</h3>
              <p className="mt-2 text-zinc-600">BYO Mac fleet with <span className="font-mono text-sm">docker compose up</span>. Cloud plans just rent you ours.</p>
            </div>
          </div>
        </div>
      </Container>

      {/* AGENT TERMINAL */}
      <div className="border-y border-white/10 bg-zinc-950/50">
        <Container>
          <div className="grid items-center gap-12 py-24 md:grid-cols-2 md:py-32">
            <div>
              <Eyebrow>Agent-native</Eyebrow>
              <h2 className="mt-4 text-4xl font-bold tracking-tighter text-white md:text-5xl">Your agent gets the controls.</h2>
              <p className="mt-4 text-lg leading-relaxed text-zinc-400">Register Sidekick as an MCP server once. Your agent launches the app, reads the UI tree, taps, types and screenshots — while you watch and take over anytime.</p>
              <div className="mt-6 space-y-2 font-mono text-sm text-zinc-300">
                <p><span className="text-zinc-600">$</span> sidekick agent-setup <span className="text-zinc-600"># skill + MCP</span></p>
                <p><span className="text-zinc-600">$</span> sidekick dev --detach --json <span className="text-zinc-600"># live device</span></p>
              </div>
              <div className="mt-6 flex flex-wrap gap-2 text-sm">
                {["ui-tree", "tap", "type", "screenshot", "record", "logs", "press", "drag"].map((v) => (
                  <code key={v} className="rounded-md border border-white/10 bg-white/5 px-2 py-1 font-mono text-xs text-zinc-300">{v}</code>
                ))}
              </div>
            </div>
            <Terminal lines={QUICKSTART + "\n\n" + LOOP} />
          </div>
        </Container>
      </div>

      {/* PRICING */}
      <Container>
        <div className="py-24 md:py-32">
          <SectionHead
            eyebrow="Pricing"
            title="Start free. Self-host free forever."
            sub="Every plan is monthly with streaming minutes and builds included. Overage draws from credit at $0.03/min."
          />
          <div className="mt-14"><PricingTable /></div>
          <p className="mt-8 text-center text-sm text-zinc-500">
            <Link to="/pricing" className="font-medium text-zinc-300 underline hover:text-white">Full pricing →</Link>
          </p>
        </div>
      </Container>

      {/* SCOREBOARD TEASER */}
      <Container>
        <div className="rounded-3xl border border-white/10 bg-zinc-950 p-8 md:p-12">
          <div className="mx-auto max-w-2xl text-center">
            <Eyebrow>Comparisons</Eyebrow>
            <h2 className="mt-4 text-3xl font-bold tracking-tighter text-white md:text-4xl">Don&apos;t take our word for it. Take the table.</h2>
            <p className="mt-4 text-zinc-400">Head-to-head comparisons with VibeView, EAS Simulator, BrowserStack, EAS Build, Limrun, Appetize, AWS Device Farm, LambdaTest and Sauce Labs — sourced, dated, explicit about where each one beats us.</p>
          </div>
          <div className="mt-8 flex flex-wrap justify-center gap-2">
            {COMPETITORS.map((c) => (
              <Link key={c.slug} to="/alternatives/$slug" params={{ slug: c.slug }} className="rounded-full border border-white/15 bg-white/5 px-4 py-2 text-sm text-zinc-200 hover:border-white/40 hover:text-white">
                {c.name}
              </Link>
            ))}
          </div>
        </div>
      </Container>

      {/* FAQ */}
      <Container>
        <div className="mx-auto max-w-3xl py-24 md:py-32">
          <SectionHead
            eyebrow="FAQ"
            title="Common questions"
            sub="The short answers. Longer ones live on the comparison pages."
          />
          <div className="mt-10"><Faq items={[
            { q: "Do I need a Mac to test an iOS app?", a: "No. The simulator runs on our fleet (or your self-hosted Mac mini). Your machine and your agent's sandbox only need the sidekick CLI." },
            { q: "Are these real phones?", a: "Simulators and emulators, like Appetize and Limrun. Roku runs on real hardware (roadmap). For sensor-grade real-device matrices, BrowserStack remains the answer — we say so on their page." },
            { q: "How is this different from Limrun?", a: "Same high-level interface so migration is trivial — plus TV devices, post-agent testing, Play submission, public pricing, self-hosting, and a source-available license. Full breakdown on the Limrun alternative page." },
            { q: "What does the free tier include?", a: "30 streaming minutes and 10 cloud builds a month, no card. Self-hosting is unlimited and free." },
          ]} /></div>
        </div>
      </Container>

      <SiteFooter />
    </div>
  );
}
