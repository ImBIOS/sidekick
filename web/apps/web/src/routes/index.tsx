import { createFileRoute, Link } from "@tanstack/react-router";
import { COMPETITORS, score } from "../data/comparison";
import { Container, CtaRow, Faq, PricingTable, SiteFooter, Terminal } from "../components/marketing";

export const Route = createFileRoute("/")({ component: HomeComponent });

const QUICKSTART = `npm i -g @imbios/sidekick
sidekick login   # or export SIDEKICK_TOKEN=...
sidekick agent-setup   # skill + MCP, one command
sidekick xcode build . --scheme MyApp
sidekick ios create --device "iPhone 17, iOS 26"`;

const LOOP = `sidekick ui-tree                      # what's on screen? (@e refs)
# ...agent edits code; hot-reload pushes in ~1s...
sidekick tap @e7                        # keep going: tap through the flow
sidekick screenshot --out ./check.png   # looks right? ship it
sidekick record --stop                  # demo video for the PR`;

function HomeComponent() {
  return (
    <div className="bg-white text-zinc-900">
      {/* HERO — VibeView structure, Limrun agent wiring */}
      <Container>
        <div className="grid items-center gap-10 py-14 md:grid-cols-2">
          <div>
            <p className="inline-flex items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1 text-xs font-medium text-zinc-600">
              🛣️ Source-available (FSL) · self-hostable · converts to Apache-2.0
            </p>
            <h1 className="mt-4 text-5xl font-bold tracking-tight">Every agent needs a sidekick to hold the phone.</h1>
            <p className="mt-4 text-lg text-zinc-600">
              Cloud Xcode, iOS, Android and TV devices your coding agent builds, verifies and submits —
              on a live device it can see and touch. You watch along in the browser and take over anytime.
            </p>
            <div className="mt-6"><CtaRow /></div>
            <p className="mt-4 font-mono text-xs text-zinc-500">
              skill: <span className="text-zinc-800">sidekick agent-setup</span> · MCP: <span className="text-zinc-800">sidekick mcp</span> · spec: <a className="underline" href="/llms.txt">llms.txt</a> · <a className="underline" href="/openapi.json">openapi.json</a>
            </p>
          </div>
          <div>
            {/* CSS device mock in the spirit of competitor hero sandboxes */}
            <div className="rounded-2xl border border-zinc-200 bg-zinc-50 p-4 shadow-xl">
              <div className="flex gap-1.5 pb-3">
                {["iOS", "Android", "Apple TV", "Android TV", "Roku"].map((p, i) => (
                  <span key={p} className={`rounded-full px-2.5 py-1 text-xs font-medium ${i === 0 ? "bg-zinc-900 text-white" : "bg-white text-zinc-600 border border-zinc-200"}`}>{p}</span>
                ))}
              </div>
              <div className="mx-auto max-w-[280px] rounded-[2rem] border-8 border-zinc-900 bg-white p-4">
                <div className="mx-auto mb-3 h-4 w-20 rounded-full bg-zinc-900" />
                <div className="space-y-2">
                  <div className="h-8 rounded-lg bg-zinc-900 text-[10px] text-white flex items-center px-3">Explore · @e1…@e9</div>
                  {["Destination detail", "Checkout flow", "Settings"].map((t) => (
                    <div key={t} className="rounded-lg border border-zinc-200 p-2 text-xs text-zinc-600">{t}</div>
                  ))}
                  <div className="rounded-lg bg-emerald-50 border border-emerald-200 p-2 text-xs text-emerald-800">✓ agent verified: tap @e7 → diff 3 elements</div>
                </div>
              </div>
              <p className="pt-3 text-center font-mono text-xs text-zinc-500">$ sidekick tap @e7 · screenshot · record — streamed at 60fps</p>
            </div>
          </div>
        </div>
      </Container>

      {/* TRUST STRIP */}
      <div className="border-y border-zinc-200 bg-zinc-50">
        <Container>
          <p className="py-4 text-center text-sm text-zinc-500">
            From first build to final check · iOS · Android · Apple TV · Android TV · Roku (roadmap) · from Claude Code, Cursor, Codex or any MCP client
          </p>
        </Container>
      </div>

      {/* LOOP */}
      <Container>
        <h2 className="pt-14 text-3xl font-bold">From first build to final check.</h2>
        <div className="mt-6 grid gap-4 md:grid-cols-3">
          {[
            ["Build", "Cloud Xcode + Gradle without a Mac. Signing handled, TestFlight + Play submit from the terminal."],
            ["Preview", "One link puts the live app in anyone's browser. Guests watch, take over, embed anywhere."],
            ["Test", "Record once, replay everywhere. Visual regression + CI checks keep running after the agent leaves."],
          ].map(([t, d]) => (
            <div key={t} className="rounded-xl border border-zinc-200 p-5">
              <h3 className="font-semibold">{t}</h3>
              <p className="mt-1 text-sm text-zinc-600">{d}</p>
            </div>
          ))}
        </div>
      </Container>

      {/* AGENT TERMINAL — Limrun pattern */}
      <Container>
        <div className="mt-14 grid gap-8 md:grid-cols-2">
          <div>
            <h2 className="text-3xl font-bold">Your agent gets the controls.</h2>
            <p className="mt-3 text-zinc-600">Register Sidekick as an MCP server once. Your agent launches the app, reads the UI tree, taps, types and screenshots — while you watch and take over anytime.</p>
            <div className="mt-4 space-y-2 font-mono text-sm">
              <p><span className="text-zinc-400">$</span> sidekick agent-setup <span className="text-zinc-400"># skill + MCP</span></p>
              <p><span className="text-zinc-400">$</span> sidekick dev --detach --json <span className="text-zinc-400"># live device</span></p>
            </div>
            <div className="mt-4 flex flex-wrap gap-2 text-sm">
              {["ui-tree", "tap", "type", "screenshot", "record", "logs", "press", "drag"].map((v) => (
                <code key={v} className="rounded bg-zinc-100 px-2 py-1 font-mono text-xs">{v}</code>
              ))}
            </div>
          </div>
          <Terminal lines={QUICKSTART + "\n\n" + LOOP} />
        </div>
      </Container>

      {/* TV */}
      <Container>
        <div className="mt-14 rounded-2xl bg-zinc-950 p-8 text-white">
          <h2 className="text-2xl font-bold">The same loop reaches the living room.</h2>
          <p className="mt-2 text-zinc-400">Apple TV and Android TV with d-pad verbs and focus-walk. Roku on real hardware (roadmap). Same verbs: ui-tree, press, tap-focused, screenshot.</p>
          <div className="mt-4 flex gap-2 font-mono text-xs">
            <code className="rounded bg-zinc-800 px-2 py-1">sidekick dev --platform tvos</code>
            <code className="rounded bg-zinc-800 px-2 py-1">sidekick press dpad_center</code>
          </div>
        </div>
      </Container>

      {/* PRICING */}
      <Container>
        <div className="mt-14 flex items-end justify-between">
          <h2 className="text-3xl font-bold">Start free. Self-host free forever.</h2>
          <Link to="/pricing" className="text-sm font-medium underline">Full pricing →</Link>
        </div>
        <div className="mt-6"><PricingTable /></div>
        <p className="mt-3 text-sm text-zinc-500">Self-hosting on your own Mac minis is $0 forever under FSL. Cloud plans just rent you our fleet.</p>
      </Container>

      {/* SCOREBOARD TEASER */}
      <Container>
        <div className="mt-14 rounded-2xl border border-zinc-200 bg-zinc-50 p-8">
          <h2 className="text-2xl font-bold">Don't take our word for it. Take the table.</h2>
          <p className="mt-2 text-zinc-600">Head-to-head comparisons with BrowserStack, EAS Build, Limrun, Appetize, AWS Device Farm, LambdaTest and Sauce Labs — sourced, dated, explicit about where each one beats us.</p>
          <div className="mt-4 flex flex-wrap gap-2">
            {COMPETITORS.map((c) => {
              const s = score(c.slug);
              return (
                <Link key={c.slug} to="/alternatives/$slug" params={{ slug: c.slug }} className="rounded-full border border-zinc-300 bg-white px-3 py-1.5 text-sm hover:border-zinc-900">
                  Sidekick {s.sidekick.toFixed(0)} – {s.them.toFixed(0)} {c.name}
                </Link>
              );
            })}
          </div>
        </div>
      </Container>

      {/* FAQ */}
      <Container>
        <h2 className="pt-14 text-2xl font-bold">Common questions</h2>
        <div className="mt-4"><Faq items={[
          { q: "Do I need a Mac to test an iOS app?", a: "No. The simulator runs on our fleet (or your self-hosted Mac mini). Your machine and your agent's sandbox only need the sidekick CLI." },
          { q: "Are these real phones?", a: "Simulators and emulators, like Appetize and Limrun. Roku runs on real hardware (roadmap). For sensor-grade real-device matrices, BrowserStack remains the answer — we say so on their page." },
          { q: "How is this different from Limrun?", a: "Same high-level interface so migration is trivial — plus TV devices, post-agent testing, Play submission, public pricing, self-hosting, and a source-available license. Full breakdown on the Limrun alternative page." },
          { q: "What does the free tier include?", a: "30 streaming minutes and 10 cloud builds a month, no card. Self-hosting is unlimited and free." },
        ]} /></div>
      </Container>

      <SiteFooter />
    </div>
  );
}
