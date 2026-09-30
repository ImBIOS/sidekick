import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import { COMPETITORS, FEATURES, score } from "../data/comparison";
import { CellBadge, CompareTable, Container, CtaRow, Faq, SiteFooter, Terminal } from "../components/marketing";

export const Route = createFileRoute("/alternatives/$slug")({
  loader: ({ params }) => {
    const c = COMPETITORS.find((x) => x.slug === params.slug);
    if (!c) throw notFound();
    return { c };
  },
  component: AlternativePage,
  notFoundComponent: () => (
    <div className="bg-black p-10 text-zinc-100">
      Unknown comparison. <Link to="/alternatives" className="underline">All alternatives</Link>
    </div>
  ),
});

function AlternativePage() {
  const { c } = Route.useLoaderData();
  const s = score(c.slug);
  const rows = FEATURES.map((r) => ({
    feature: r.feature,
    category: r.category,
    note: r.sidekick === "roadmap" ? `Sidekick: ${r.sidekickNote}` : undefined,
    cells: [
      <CellBadge key="a" cell={r.sidekick} note={r.sidekickNote} />,
      <CellBadge key="b" cell={r.cells[c.slug]} />,
    ],
  }));

  return (
    <div className="bg-black text-zinc-100">
      <Container>
        <p className="pt-10 text-sm text-zinc-500">
          <Link to="/">Home</Link> / <Link to="/alternatives">Alternatives</Link> / {c.name}
        </p>
        <h1 className="mt-2 text-4xl font-bold tracking-tight text-white">{c.name} alternative: what Sidekick does differently</h1>
        <p className="mt-3 max-w-3xl text-lg text-zinc-400">{c.tagline}</p>
        <div className="mt-5"><CtaRow /></div>

        <div className="mt-8 space-y-4 text-[15px] leading-relaxed text-zinc-300">
          {c.intro.map((p, i) => (
            <p key={i}>{p}</p>
          ))}
        </div>

        <Terminal lines={`$ sidekick agent-setup   # skill + MCP, one command\n$ sidekick xcode build . --scheme MyApp   # no Mac needed\n$ sidekick ios create   # agent drives, you watch in the browser`} />

        <h2 className="mt-10 text-2xl font-bold tracking-tight text-white">
          Sidekick vs {c.name} <span className="ml-2 rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1 text-sm font-semibold text-emerald-300">{s.sidekick.toFixed(0)} – {s.them.toFixed(0)} across {s.rows} rows</span>
        </h2>
        <div className="mt-4">
          <CompareTable
            columns={[
              { key: "f", header: "What you are deciding" },
              { key: "s", header: "Sidekick" },
              { key: "t", header: c.name },
            ]}
            rows={rows}
            footnote={<>✅ Yes · 🛣️ In Roadmap = public, dated, lands in the open (self-host nightlies available) · ➖ Partial · ❌ No. {c.name} claims checked {c.checked} at {c.url}. {c.name} pricing: {c.price}.</>}
          />
        </div>

        <h2 className="mt-10 text-2xl font-bold tracking-tight text-white">Where {c.name} is stronger</h2>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          {c.strengths.map((st) => (
            <div key={st.title} className="rounded-2xl border border-white/10 bg-zinc-950 p-5">
              <h3 className="font-semibold text-white">{st.title}</h3>
              <p className="mt-1.5 text-sm text-zinc-400">{st.body}</p>
            </div>
          ))}
        </div>

        <h2 className="mt-10 text-2xl font-bold tracking-tight text-white">Which is yours: a decision guide</h2>
        <div className="mt-4 space-y-3">
          {c.decideGuide.map((d) => (
            <div key={d.label} className="flex gap-3 rounded-2xl border border-white/10 bg-zinc-950 p-4">
              <span className={`h-fit rounded-full px-2.5 py-0.5 text-xs font-bold whitespace-nowrap ${d.winner === "sidekick" ? "bg-emerald-400/15 text-emerald-300" : "bg-white/10 text-zinc-300"}`}>
                {d.winner === "sidekick" ? "Sidekick" : c.name}
              </span>
              <p className="text-sm text-zinc-300"><strong className="text-white">{d.label}:</strong> {d.body}</p>
            </div>
          ))}
        </div>

        <h2 className="mt-10 text-2xl font-bold tracking-tight text-white">Frequently asked questions</h2>
        <div className="mt-4"><Faq items={c.faqs} /></div>
        <p className="mt-6 text-xs text-zinc-600">Published Sep 2026 · {c.name} homepage and docs verified {c.checked} ·t Scores count roadmap as delivered because every roadmap item is public, dated and source-available.</p>
      </Container>
      <SiteFooter />
    </div>
  );
}
