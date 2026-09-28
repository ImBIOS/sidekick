import { createFileRoute, Link } from "@tanstack/react-router";
import { COMPETITORS, FEATURES, score } from "../data/comparison";
import { CellBadge, CompareTable, Container, SiteFooter } from "../components/marketing";

export const Route = createFileRoute("/alternatives/")({ component: AlternativesHub });

function AlternativesHub() {
  const tableRows = FEATURES.map((r) => ({
    feature: r.feature,
    category: r.category,
    note: r.sidekick === "roadmap" ? `Sidekick: ${r.sidekickNote}` : undefined,
    cells: [
      <CellBadge key="s" cell={r.sidekick} note={r.sidekickNote} />,
      ...COMPETITORS.map((c) => <CellBadge key={c.slug} cell={r.cells[c.slug]} />),
    ],
  }));

  return (
    <div className="bg-white text-zinc-900">
      <Container>
        <p className="pt-10 text-sm text-zinc-500"><Link to="/">Home</Link> / Alternatives</p>
        <h1 className="mt-2 text-4xl font-bold tracking-tight">Mobile app testing alternatives</h1>
        <p className="mt-3 max-w-3xl text-zinc-600">
          Sidekick compared with VibeView, EAS Simulator, BrowserStack, Expo EAS Build, Limrun,
          Appetize, AWS Device Farm, LambdaTest and Sauce Labs — sourced, dated, and explicit
          about where each one is the better choice.
        </p>
        <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-5 text-sm text-amber-900">
          <p className="font-semibold">What these pages are</p>
          <p className="mt-1">Head-to-head comparisons written by us. We are not a neutral party, so we make every claim checkable:
          each page carries sources with check dates, names the rows where the competitor wins, and never uses ratings or scores —
          just a 28-row feature table where 🛣️ means a public, dated roadmap item (open source, so you can watch it land).</p>
        </div>

        <h2 className="mt-10 text-2xl font-bold">Which one to read</h2>
        <div className="mt-4 space-y-3 text-[15px] text-zinc-700">
          <p><strong>If you're choosing between the two agent-native loops:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "vibeview" }}>VibeView</Link> is the closest full-stack rival — the page concedes Roku, Duo and record-replay, and shows where any-stack builds and self-host answer.</p>
          <p><strong>If you're an Expo shop:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "eas-simulator" }}>EAS Simulator</Link> is the shortest path inside the preview — the page covers the waitlist, the missing price, and what phones-only means for TV apps.</p>
          <p><strong>If the blocker is the build itself:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "eas-build" }}>EAS Build</Link> compiles and submits RN apps — the page prices per build and shows what a device adds.</p>
          <p><strong>If the blocker is cost per person:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "aws-device-farm" }}>AWS Device Farm</Link> meters the device; <Link className="underline" to="/alternatives/$slug" params={{ slug: "lambdatest" }}>LambdaTest</Link> prices per parallel. Both pages work out what the fourth teammate costs.</p>
          <p><strong>If the blocker is who can get in:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "sauce-labs" }}>Sauce Labs</Link> requires logins to watch; our share links need no account.</p>
          <p><strong>If you weigh hardware against reach:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "browserstack" }}>BrowserStack</Link> puts builds on real hardware; we put them in front of non-engineers.</p>
          <p><strong>If you're already sold on embedding:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "appetize" }}>Appetize</Link> is the closest embed comparison — two differences pointing opposite ways.</p>
          <p><strong>If you're evaluating the other agent-native incumbent:</strong> <Link className="underline" to="/alternatives/$slug" params={{ slug: "limrun" }}>Limrun</Link> is the closest CLI match — the page concedes video, tokens and regions.</p>
        </div>

        <h2 className="mt-10 text-2xl font-bold">Alternatives guides</h2>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          {COMPETITORS.map((c) => {
            const s = score(c.slug);
            return (
              <Link key={c.slug} to="/alternatives/$slug" params={{ slug: c.slug }} className="group rounded-xl border border-zinc-200 p-5 hover:border-zinc-900">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold group-hover:underline">{c.name} alternative</h3>
                  <span className="rounded-full bg-emerald-50 border border-emerald-200 px-2.5 py-0.5 text-xs font-medium text-emerald-800">
                    {s.sidekick.toFixed(0)}–{s.them.toFixed(0)}
                  </span>
                </div>
                <p className="mt-1.5 text-sm text-zinc-600">{c.cardBlurb}</p>
                <p className="mt-2 font-mono text-xs text-zinc-400">claims checked {c.checked} · {c.price}</p>
              </Link>
            );
          })}
        </div>

        <h2 className="mt-10 text-2xl font-bold">Every rival, every row, one table</h2>
        <p className="mt-2 max-w-3xl text-[15px] text-zinc-600">
          The full 28-row comparison across all 9 vendors. Sidekick's column is ✅ Yes, 🛣️ In Roadmap
          (public, dated) or ❌ No — nothing else. Scroll sideways to audit every cell against the per-page sources.
        </p>
        <div className="mt-4">
          <CompareTable
            columns={[
              { key: "f", header: "What you are deciding" },
              { key: "s", header: "Sidekick" },
              ...COMPETITORS.map((c) => ({ key: c.slug, header: c.name })),
            ]}
            rows={tableRows}
            footnote={<>✅ Yes · 🛣️ In Roadmap = public, dated, lands in the open (self-host nightlies available) · ➖ Partial · ❌ No. Competitor claims checked Sep 2026 — see each guide for sources, prices and check dates.</>}
          />
        </div>
      </Container>
      <SiteFooter />
    </div>
  );
}
