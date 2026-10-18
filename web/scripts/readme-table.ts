// Regenerates the full comparison table in the root README from the
// single source of truth (apps/web/src/data/comparison.ts).
// Run from web/: bun scripts/readme-table.ts
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { COMPETITORS, FEATURES } from "../apps/web/src/data/comparison";
import type { Cell } from "../apps/web/src/data/comparison";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..");

function badge(cell: Cell, note?: string): string {
  switch (cell) {
    case "yes":
      return "✅ Yes";
    case "roadmap":
      return `🛣️ Roadmap${note ? ` (${note})` : ""}`;

    case "partial":
      return "➖ Partial";
    default:
      return "❌ No";
  }
}

for (const row of FEATURES) {
  for (const text of [row.feature, row.sidekickNote ?? ""]) {
    if (text.includes("|")) throw new Error(`pipe in cell text breaks markdown: ${text}`);
  }
}

const header = `| Feature | Sidekick | ${COMPETITORS.map((c) => c.name).join(" | ")} |`;
const sep = `| --- | ${COMPETITORS.map(() => "---").join(" | ")} |`;
const lines = FEATURES.map(
  (r) =>
    `| ${r.feature} | ${badge(r.sidekick, r.sidekickNote)} | ${COMPETITORS.map((c) => badge(r.cells[c.slug])).join(" | ")} |`,
);

const table = [
  "<details>",
  "<summary>Full 28-row comparison (all 9 vendors) — generated from <code>web/apps/web/src/data/comparison.ts</code>, last updated 18 Oct 2026</summary>",
  "",
  header,
  sep,
  ...lines,
  "",
  "Legend: ✅ Yes · 🛣️ Roadmap = public, lands in the open · ➖ Partial · ❌ No.",
  "</details>",
].join("\n");

const readmePath = join(root, "README.md");
const readme = readFileSync(readmePath, "utf8");
const start = "<!-- COMPARISON-START -->";
const end = "<!-- COMPARISON-END -->";
if (!readme.includes(start) || !readme.includes(end)) {
  throw new Error("README markers missing");
}
const next = readme.slice(0, readme.indexOf(start) + start.length) + "\n" + table + "\n" + readme.slice(readme.indexOf(end));
writeFileSync(readmePath, next);
console.log(`wrote ${FEATURES.length} rows x ${COMPETITORS.length + 1} cols`);
