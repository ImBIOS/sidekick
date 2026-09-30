import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { COMPETITORS, score } from "../data/comparison";
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
  { tab: "iOS", device: "iPhone 17, iOS 26", rows: ["Destination detail", "Checkout flow", "Settings"], verb: "$ sidekick tap @e7 · screenshot · record — streamed at 60fps" },
  { tab: "Android", device: "Pixel 9, API 35", rows: ["Explore", "Checkout flow", "Settings"], verb: "$ sidekick tap @e4 — local adb/Appium just work via tunnel" },
  { tab: "Apple TV", device: "Apple TV 4K, tvOS 26", rows: ["Home screen", "Player", "Search"], verb: "$ sidekick press dpad_center · tap-focused — focus-walk verbs" },
  { tab: "Android TV", device: "Android TV, API 34", rows: ["Launcher", "Player", "Settings"], verb: "$ sidekick press dpad_right — same verbs as phones" },
  { tab: "Roku", device: "Roku Ultra (beta pool)", rows: ["Home", "Channel", "Search"], verb: "🛣️ In Roadmap: real-hardware pool, Q1 — watch it land in the open" },
];

function HeroSandbox() {
  const [active, setActive] = useState(0);
  const p = PLATFORMS[active];
  return (
    <div className="rounded-3xl border border-white/10 bg-zinc-950/80 p-4 shadow-2xl backdrop-blur">
      <div className="flex flex-wrap justify-center gap-1.5 pb-3" role="tablist" aria-label="Platforms">
        {PLATFORMS.map((t, i) => (
          <button
            key={t.tab}
            role="tab"
            aria-selected={i === active}
            onClick={() => setActive(i)}
            className={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${i === active ? "bg-white text-black" : "border border-white/10 bg-white/5 text-zinc-400 hover:border-white/25 hover:text-white"}`}
          >
            {t.tab}
          </button>
        ))}
      </div>
      <div className="mx-auto max-w-[280px] rounded-[2rem] border-8 border-zinc-800 bg-black p-4">
        <div className="mx-auto mb-3 h-4 w-20 rounded-full bg-zinc-800" />
        <p className="mb-2 text-center font-mono text-[10px] text-zinc-500">{p.device}</p>
        <div className="space-y-2">
          <div className="flex h-8 items-center rounded-lg bg-white px-3 text-[10px] font-medium text-black">Explore · @e1…@e9</div>
          {p.rows.map((t) => (
            <div key={t} className="rounded-lg border border-white/10 p-2 text-xs text-zinc-300">{t}</div>
          ))}
          <div className="rounded-lg border border-emerald-400/20 bg-emerald-400/10 p-2 text-xs text-emerald-300">✓ agent verified: tap @e7 → diff 3 elements</div>
        </div>
      </div>
      <p className="pt-3 text-center font-mono text-xs text-zinc-500">{p.verb}</p>
    </div>
  );
}

function HomeComponent() {
  return (
    <div className="bg-black text-zinc-100">
      {/* HERO — Expo language: burst + mark + huge centered headline + pill CTAs */}
      <div className="relative overflow-hidden">
        <BurstCanvas className="pointer-events-none absolute inset-0 h-full w-full opacity-70" />
        <Container>
          <div className="relative flex flex-col items-center px-4 pt-20 pb-14 text-center md:pt-28">
            <SidekickMark size={104} />
            <p className="mt-8 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1.5 text-xs font-medium text-zinc-400">
              Source-available (FSL) · self-hostable · converts to Apache-2.0
            </p>
            <h1 className="mt-6 max-w-3xl text-5xl font-bold tracking-tighter text-white md:text-7xl">
              Every agent needs a sidekick to hold the phone.
            </h1>
            <p className="mt-6 max-w-2xl text-lg text-zinc-400">
              Cloud Xcode, iOS, Android and TV devices your coding agent builds, verifies and submits —
              on a live device it can see and touch. You watch along in the browser and take over anytime.
            </p>
            <div className="mt-8"><CtaRow /></div>
            <p className="mt-6 font-mono text-xs text-zinc-500">
              skill: <span className="text-zinc-200">sidekick agent-setup</span> · MCP: <span className="text-zinc-200">sidekick mcp</span> · spec: <a className="underline hover:text-zinc-200" href="/llms.txt">llms.txt</a> · <a className="underline hover:text-zinc-200" href="/openapi.json">openapi.json</a>
            </p>
          </div>
        </Container>
      </div>

      {/* DEVICE — Expo shows the product right under the hero */}
      <Container>
        <div className="mx-auto max-w-xl">
          <HeroSandbox />
        </div>
      </Container>

      {/* TRUST STRIP */}
      <div className="mt-14 border-y border-white/10 bg-zinc-950/60">
        <Container>
          <p className="py-4 text-center text-sm text-zinc-500">
            From first build to final check · iOS · Android · Apple TV · Android TV · Roku (roadmap) · from Claude Code, Cursor, Codex or any MCP client
          </p>
        </Container>
      </div>

      {/* LOOP */}
      <Container>
        <h2 className="pt-14 text-center text-3xl font-bold tracking-tight text-white">From first build to final check.</h2>
        <div className="mt-6 grid gap-4 md:grid-cols-3">
          {[
            ["Build", "Cloud Xcode + Gradle without a Mac. Signing handled, TestFlight + Play submit from the terminal."],
            ["Preview", "One link puts the live app in anyone's browser. Guests watch, take over, embed anywhere."],
            ["Test", "Record once, replay everywhere. Visual regression + CI checks keep running after the agent leaves."],
          ].map(([t, d]) => (
            <div key={t} className="rounded-2xl border border-white/10 bg-zinc-950 p-6">
              <h3 className="font-semibold text-white">{t}</h3>
              <p className="mt-1 text-sm text-zinc-400">{d}</p>
            </div>
          ))}
        </div>
      </Container>

      {/* AGENT TERMINAL */}
      <Container>
        <div className="mt-14 grid items-center gap-8 md:grid-cols-2">
          <div>
            <h2 className="text-3xl font-bold tracking-tight text-white">Your agent gets the controls.</h2>
            <p className="mt-3 text-zinc-400">Register Sidekick as an MCP server once. Your agent launches the app, reads the UI tree, taps, types and screenshots — while you watch and take over anytime.</p>
            <div className="mt-4 space-y-2 font-mono text-sm text-zinc-300">
              <p><span className="text-zinc-600">$</span> sidekick agent-setup <span className="text-zinc-600"># skill + MCP</span></p>
              <p><span className="text-zinc-600">$</span> sidekick dev --detach --json <span className="text-zinc-600"># live device</span></p>
            </div>
            <div className="mt-4 flex flex-wrap gap-2 text-sm">
              {["ui-tree", "tap", "type", "screenshot", "record", "logs", "press", "drag"].map((v) => (
                <code key={v} className="rounded-md border border-white/10 bg-white/5 px-2 py-1 font-mono text-xs text-zinc-300">{v}</code>
              ))}
            </div>
          </div>
          <Terminal lines={QUICKSTART + "\n\n" + LOOP} />
        </div>
      </Container>

      {/* TV */}
      <Container>
        <div className="mt-14 rounded-3xl border border-white/10 bg-zinc-950 p-8 md:p-10">
          <h2 className="text-2xl font-bold tracking-tight text-white">The same loop reaches the living room.</h2>
          <p className="mt-2 text-zinc-400">Apple TV and Android TV with d-pad verbs and focus-walk. Roku on real hardware (roadmap). Same verbs: ui-tree, press, tap-focused, screenshot.</p>
          <div className="mt-4 flex flex-wrap gap-2 font-mono text-xs">
            <code className="rounded-md border border-white/10 bg-white/5 px-2 py-1 text-zinc-300">sidekick dev --platform tvos</code>
            <code className="rounded-md border border-white/10 bg-white/5 px-2 py-1 text-zinc-300">sidekick press dpad_center</code>
          </div>
        </div>
      </Container>

      {/* PRICING */}
      <Container>
        <div className="mt-14 flex flex-wrap items-end justify-between gap-3">
          <h2 className="text-3xl font-bold tracking-tight text-white">Start free. Self-host free forever.</h2>
          <Link to="/pricing" className="text-sm font-medium text-zinc-400 underline hover:text-white">Full pricing →</Link>
        </div>
        <div className="mt-6"><PricingTable /></div>
        <p className="mt-3 text-center text-sm text-zinc-500">Self-hosting on your own Mac minis is $0 forever under FSL. Cloud plans just rent you our fleet.</p>
      </Container>

      {/* SCOREBOARD TEASER */}
      <Container>
        <div className="mt-14 rounded-3xl border border-white/10 bg-zinc-950 p-8">
          <h2 className="text-2xl font-bold tracking-tight text-white">Don&apos;t take our word for it. Take the table.</h2>
          <p className="mt-2 text-zinc-400">Head-to-head comparisons with VibeView, EAS Simulator, BrowserStack, EAS Build, Limrun, Appetize, AWS Device Farm, LambdaTest and Sauce Labs — sourced, dated, explicit about where each one beats us.</p>
          <div className="mt-4 flex flex-wrap gap-2">
            {COMPETITORS.map((c) => {
              const s = score(c.slug);
              return (
                <Link key={c.slug} to="/alternatives/$slug" params={{ slug: c.slug }} className="rounded-full border border-white/15 bg-white/5 px-3 py-1.5 text-sm text-zinc-200 hover:border-white/40 hover:text-white">
                  Sidekick {s.sidekick.toFixed(0)} – {s.them.toFixed(0)} {c.name}
                </Link>
              );
            })}
          </div>
        </div>
      </Container>

      {/* FAQ */}
      <Container>
        <div className="mx-auto max-w-3xl">
          <h2 className="pt-14 text-center text-2xl font-bold tracking-tight text-white">Common questions</h2>
          <div className="mt-4"><Faq items={[
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
