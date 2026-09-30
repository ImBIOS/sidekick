import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import type { Cell, PriceTier } from "../data/comparison";
import { PRICING } from "../data/comparison";

export function Container({ children, wide }: { children: ReactNode; wide?: boolean }) {
  return <div className={`mx-auto px-6 ${wide ? "max-w-7xl" : "max-w-6xl"}`}>{children}</div>;
}

export function cellLabel(c: Cell): { icon: string; text: string; cls: string } {
  switch (c) {
    case "yes":
      return { icon: "✅", text: "Yes", cls: "border-emerald-400/20 bg-emerald-400/10 text-emerald-300" };
    case "roadmap":
      return { icon: "🛣️", text: "In Roadmap", cls: "border-sky-400/20 bg-sky-400/10 text-sky-300" };
    case "partial":
      return { icon: "➖", text: "Partial", cls: "border-amber-400/20 bg-amber-400/10 text-amber-300" };
    default:
      return { icon: "❌", text: "No", cls: "border-white/10 bg-white/5 text-zinc-500" };
  }
}

export function CellBadge({ cell, note }: { cell: Cell; note?: string }) {
  const l = cellLabel(cell);
  return (
    <span title={note} className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium whitespace-nowrap ${l.cls}`}>
      <span>{l.icon}</span> {l.text}
    </span>
  );
}

export interface TableColumn {
  key: string;
  header: ReactNode;
}

export function CompareTable({
  columns,
  rows,
  footnote,
}: {
  columns: TableColumn[];
  rows: { feature: string; category: string; cells: ReactNode[]; note?: string }[];
  footnote?: ReactNode;
}) {
  let lastCat = "";
  return (
    <div className="overflow-x-auto rounded-2xl border border-white/10 bg-zinc-950">
      <table className="w-full min-w-[720px] border-collapse text-sm">
        <thead>
          <tr className="bg-white/[0.03] text-left">
            {columns.map((c) => (
              <th key={c.key} className="border-b border-white/10 px-4 py-3 font-semibold text-white first:sticky first:left-0 first:bg-zinc-950 first:z-10">
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const showCat = r.category !== lastCat;
            lastCat = r.category;
            return (
              <>
                {showCat && (
                  <tr key={`cat-${r.category}`}>
                    <td colSpan={columns.length} className="bg-white/[0.02] px-4 py-1.5 text-xs font-bold tracking-wide text-zinc-500 uppercase">
                      {r.category}
                    </td>
                  </tr>
                )}
                <tr key={r.feature} className="border-t border-white/5 hover:bg-white/[0.02]">
                  <td className="px-4 py-2.5 font-medium text-zinc-100 sticky left-0 bg-zinc-950">
                    {r.feature}
                    {r.note && <span className="block text-xs font-normal text-sky-400">{r.note}</span>}
                  </td>
                  {r.cells.map((cell, i) => (
                    <td key={i} className="px-4 py-2.5 text-center text-zinc-300">{cell}</td>
                  ))}
                </tr>
              </>
            );
          })}
        </tbody>
      </table>
      {footnote && <p className="border-t border-white/10 bg-white/[0.02] px-4 py-2.5 text-xs text-zinc-500">{footnote}</p>}
    </div>
  );
}

export function Terminal({ lines }: { lines: string }) {
  return (
    <pre className="overflow-x-auto rounded-2xl border border-white/10 bg-zinc-950 p-5 font-mono text-sm leading-relaxed text-zinc-200 shadow-2xl">
      {lines}
    </pre>
  );
}

/** Expo-style pill CTAs: solid white primary, dim dark secondary. */
export function CtaRow() {
  return (
    <div className="flex flex-wrap items-center justify-center gap-3">
      <Link to="/pricing" className="rounded-full bg-white px-7 py-3 font-medium text-black transition-opacity hover:opacity-85">
        Start free →
      </Link>
      <a href="https://github.com/ImBIOS/sidekick" className="rounded-full border border-white/10 bg-white/5 px-7 py-3 font-medium text-white transition-colors hover:bg-white/10">
        GitHub
      </a>
      <Link to="/alternatives" className="rounded-full border border-white/10 bg-white/5 px-7 py-3 font-medium text-white transition-colors hover:bg-white/10">
        Compare alternatives
      </Link>
    </div>
  );
}

export function PricingTable({ tiers = PRICING }: { tiers?: PriceTier[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-4">
      {tiers.map((t) => (
        <div
          key={t.name}
          className={`flex flex-col rounded-2xl border p-6 ${
            t.featured
              ? "border-white bg-white text-black shadow-2xl"
              : "border-white/10 bg-zinc-950 text-white"
          }`}
        >
          <h3 className="font-semibold">{t.name}</h3>
          <p className="mt-1 text-3xl font-bold tracking-tight">
            {t.price}
            <span className={`text-sm font-normal ${t.featured ? "text-zinc-600" : "text-zinc-500"}`}>{t.period}</span>
          </p>
          <ul className={`mt-4 flex-1 space-y-2 text-sm ${t.featured ? "text-zinc-700" : "text-zinc-400"}`}>
            {t.features.map((f) => (
              <li key={f} className="flex gap-2">
                <span className={t.featured ? "text-black" : "text-emerald-400"}>✓</span>{f}
              </li>
            ))}
          </ul>
          <Link
            to="/pricing"
            className={`mt-6 rounded-full px-4 py-2.5 text-center text-sm font-medium transition-opacity hover:opacity-85 ${
              t.featured ? "bg-black text-white" : "border border-white/15 bg-white/5 text-white hover:bg-white/10"
            }`}
          >
            {t.cta}
          </Link>
        </div>
      ))}
    </div>
  );
}

export function Faq({ items }: { items: { q: string; a: string }[] }) {
  return (
    <div className="space-y-3">
      {items.map((f) => (
        <details key={f.q} className="group rounded-2xl border border-white/10 bg-zinc-950 px-5 py-4">
          <summary className="cursor-pointer font-medium text-white">{f.q}</summary>
          <p className="mt-2 text-sm leading-relaxed text-zinc-400">{f.a}</p>
        </details>
      ))}
    </div>
  );
}

export function SiteFooter() {
  const alts: [string, string][] = [
    ["VibeView alternative", "/alternatives/vibeview"],
    ["EAS Simulator alternative", "/alternatives/eas-simulator"],
    ["BrowserStack alternative", "/alternatives/browserstack"],
    ["EAS Build alternative", "/alternatives/eas-build"],
    ["Limrun alternative", "/alternatives/limrun"],
    ["Appetize alternative", "/alternatives/appetize"],
    ["AWS Device Farm alternative", "/alternatives/aws-device-farm"],
    ["LambdaTest alternative", "/alternatives/lambdatest"],
    ["Sauce Labs alternative", "/alternatives/sauce-labs"],
  ];
  return (
    <footer className="mt-20 border-t border-white/10 bg-black">
      <Container>
        <div className="grid gap-8 py-12 md:grid-cols-4">
          <div>
            <p className="font-bold text-white">Sidekick</p>
            <p className="mt-2 text-sm text-zinc-500">Every agent needs a sidekick to hold the phone. Source-available (FSL → Apache-2.0).</p>
          </div>
          <div>
            <p className="text-sm font-semibold text-white">Product</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-500">
              <li><Link to="/" className="hover:text-white">Home</Link></li>
              <li><Link to="/pricing" className="hover:text-white">Pricing</Link></li>
              <li><a href="https://github.com/ImBIOS/sidekick" className="hover:text-white">GitHub</a></li>
            </ul>
          </div>
          <div>
            <p className="text-sm font-semibold text-white">Alternatives</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-500">
              {alts.slice(0, 5).map(([l, to]) => (
                <li key={to}><Link to={to} className="hover:text-white">{l}</Link></li>
              ))}
            </ul>
          </div>
          <div>
            <p className="text-sm font-semibold text-white">More comparisons</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-500">
              {alts.slice(5).map(([l, to]) => (
                <li key={to}><Link to={to} className="hover:text-white">{l}</Link></li>
              ))}
              <li><Link to="/alternatives" className="hover:text-white">All alternatives</Link></li>
            </ul>
          </div>
        </div>
        <p className="border-t border-white/10 py-6 text-xs text-zinc-600">© 2026 Sidekick · FSL-1.1-ALv2, converts to Apache-2.0 · All competitor claims sourced and dated on each page.</p>
      </Container>
    </footer>
  );
}
