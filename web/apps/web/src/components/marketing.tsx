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
      return { icon: "✅", text: "Yes", cls: "bg-emerald-50 text-emerald-800 border-emerald-200" };
    case "roadmap":
      return { icon: "🛣️", text: "Roadmap", cls: "bg-sky-50 text-sky-800 border-sky-200" };
    case "partial":
      return { icon: "➖", text: "Partial", cls: "bg-amber-50 text-amber-800 border-amber-200" };
    default:
      return { icon: "❌", text: "No", cls: "bg-zinc-100 text-zinc-500 border-zinc-200" };
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
    <div className="overflow-x-auto rounded-xl border border-zinc-200">
      <table className="w-full min-w-[720px] border-collapse bg-white text-sm">
        <thead>
          <tr className="bg-zinc-50 text-left">
            {columns.map((c) => (
              <th key={c.key} className="border-b border-zinc-200 px-4 py-3 font-semibold text-zinc-900 first:sticky first:left-0 first:bg-zinc-50 first:z-10">
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
                    <td colSpan={columns.length} className="bg-zinc-100/70 px-4 py-1.5 text-xs font-bold tracking-wide text-zinc-500 uppercase">
                      {r.category}
                    </td>
                  </tr>
                )}
                <tr key={r.feature} className="border-t border-zinc-100 hover:bg-zinc-50/60">
                  <td className="px-4 py-2.5 font-medium text-zinc-900 sticky left-0 bg-white">
                    {r.feature}
                    {r.note && <span className="block text-xs font-normal text-sky-700">{r.note}</span>}
                  </td>
                  {r.cells.map((cell, i) => (
                    <td key={i} className="px-4 py-2.5 text-center">{cell}</td>
                  ))}
                </tr>
              </>
            );
          })}
        </tbody>
      </table>
      {footnote && <p className="border-t border-zinc-200 bg-zinc-50 px-4 py-2.5 text-xs text-zinc-500">{footnote}</p>}
    </div>
  );
}

export function Terminal({ lines }: { lines: string }) {
  return (
    <pre className="overflow-x-auto rounded-xl bg-zinc-950 p-5 font-mono text-sm leading-relaxed text-green-300 shadow-lg">
      {lines}
    </pre>
  );
}

export function CtaRow() {
  return (
    <div className="flex flex-wrap gap-3">
      <Link to="/pricing" className="rounded-lg bg-zinc-900 px-5 py-2.5 font-medium text-white hover:bg-zinc-700">
        Start free
      </Link>
      <a href="https://github.com/ImBIOS/sidekick" className="rounded-lg border border-zinc-300 px-5 py-2.5 font-medium text-zinc-900 hover:bg-zinc-100">
        GitHub
      </a>
      <Link to="/alternatives" className="rounded-lg border border-zinc-300 px-5 py-2.5 font-medium text-zinc-900 hover:bg-zinc-100">
        Compare alternatives
      </Link>
    </div>
  );
}

export function PricingTable({ tiers = PRICING }: { tiers?: PriceTier[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-4">
      {tiers.map((t) => (
        <div key={t.name} className={`flex flex-col rounded-xl border p-5 ${t.featured ? "border-zinc-900 shadow-lg" : "border-zinc-200"}`}>
          <h3 className="font-semibold">{t.name}</h3>
          <p className="mt-1 text-3xl font-bold">{t.price}<span className="text-sm font-normal text-zinc-500">{t.period}</span></p>
          <ul className="mt-4 flex-1 space-y-2 text-sm text-zinc-600">
            {t.features.map((f) => (
              <li key={f} className="flex gap-2"><span className="text-emerald-600">✓</span>{f}</li>
            ))}
          </ul>
          <Link to="/pricing" className={`mt-5 rounded-lg px-4 py-2 text-center text-sm font-medium ${t.featured ? "bg-zinc-900 text-white" : "border border-zinc-300"}`}>
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
        <details key={f.q} className="group rounded-xl border border-zinc-200 bg-white px-5 py-4">
          <summary className="cursor-pointer font-medium text-zinc-900">{f.q}</summary>
          <p className="mt-2 text-sm leading-relaxed text-zinc-600">{f.a}</p>
        </details>
      ))}
    </div>
  );
}

export function SiteFooter() {
  const alts: [string, string][] = [
    ["BrowserStack alternative", "/alternatives/browserstack"],
    ["EAS Build alternative", "/alternatives/eas-build"],
    ["Limrun alternative", "/alternatives/limrun"],
    ["Appetize alternative", "/alternatives/appetize"],
    ["AWS Device Farm alternative", "/alternatives/aws-device-farm"],
    ["LambdaTest alternative", "/alternatives/lambdatest"],
    ["Sauce Labs alternative", "/alternatives/sauce-labs"],
  ];
  return (
    <footer className="mt-20 border-t border-zinc-200 bg-zinc-50">
      <Container>
        <div className="grid gap-8 py-12 md:grid-cols-4">
          <div>
            <p className="font-bold">Sidekick</p>
            <p className="mt-2 text-sm text-zinc-500">Every agent needs a sidekick to hold the phone. Source-available (FSL → Apache-2.0).</p>
          </div>
          <div>
            <p className="text-sm font-semibold">Product</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-600">
              <li><Link to="/">Home</Link></li>
              <li><Link to="/pricing">Pricing</Link></li>
              <li><a href="https://github.com/ImBIOS/sidekick">GitHub</a></li>
            </ul>
          </div>
          <div>
            <p className="text-sm font-semibold">Alternatives</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-600">
              {alts.slice(0, 4).map(([l, to]) => (
                <li key={to}><Link to={to}>{l}</Link></li>
              ))}
            </ul>
          </div>
          <div>
            <p className="text-sm font-semibold">More comparisons</p>
            <ul className="mt-2 space-y-1.5 text-sm text-zinc-600">
              {alts.slice(4).map(([l, to]) => (
                <li key={to}><Link to={to}>{l}</Link></li>
              ))}
              <li><Link to="/alternatives">All alternatives</Link></li>
            </ul>
          </div>
        </div>
        <p className="border-t border-zinc-200 py-6 text-xs text-zinc-400">© 2026 Sidekick · FSL-1.1-ALv2, converts to Apache-2.0 · All competitor claims sourced and dated on each page.</p>
      </Container>
    </footer>
  );
}
