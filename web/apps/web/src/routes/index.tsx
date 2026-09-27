import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useTRPC } from "@/utils/trpc";

export const Route = createFileRoute("/")({ component: Home });

const SNIPPET = `npm i -g openlim
export OPENLIM_TOKEN=...
openlim xcode build . --scheme MyApp
openlim ios create --device "iPhone 16, iOS 18.2"`;

function Home() {
  const trpc = useTRPC();
  const health = useQuery(trpc.healthCheck.queryOptions());
  return (
    <div className="mx-auto max-w-6xl px-6 py-12">
      <p className="text-sm text-muted-foreground">FSL source-available · self-hostable · converts to Apache-2.0 in 2y</p>
      <h1 className="mt-2 text-5xl font-bold tracking-tight">Your cloud agent can build mobile apps now.</h1>
      <p className="mt-4 max-w-2xl text-lg text-muted-foreground">
        Remote Xcode, iOS simulators, Android emulators + TV/Roku. One CLI + skill file, no sandbox change.
        Watch along in the browser, share preview links on PRs.
      </p>
      <div className="mt-6 flex gap-3">
        <Link to="/login" className="rounded bg-black px-5 py-2.5 text-white">Start free</Link>
        <a href="/docs" className="rounded border px-5 py-2.5">Docs</a>
        <a href="/llms.txt" className="rounded border px-5 py-2.5 font-mono text-sm">llms.txt · MCP · skill</a>
      </div>
      <pre className="mt-8 overflow-x-auto rounded-lg bg-zinc-950 p-4 font-mono text-sm text-green-300">{SNIPPET}</pre>

      <div className="mt-12 grid gap-4 md:grid-cols-3">
        {[["Build", "Cloud Xcode + Gradle. Signing + TestFlight/Play submit. Warm NVMe cache."],
          ["Preview", "Live share links, iframe embed, collab guests. Take over anytime."],
          ["Test", "AX/UI tree, tap/type/screenshot, 60fps record, record-replay, visual regression."]].map(([t, d]) => (
          <div key={t} className="rounded-lg border p-5"><h3 className="font-semibold">{t}</h3><p className="mt-1 text-sm text-muted-foreground">{d}</p></div>
        ))}
      </div>

      <div className="mt-12">
        <h2 className="text-2xl font-semibold">Every screen · including TV</h2>
        <div className="mt-3 flex gap-2 text-sm">
          {["iOS", "Android", "Apple TV", "Android TV", "Roku (beta)"].map((p) => (
            <span key={p} className="rounded-full border px-3 py-1">{p}</span>
          ))}
        </div>
        <p className="mt-3 text-sm text-muted-foreground">Same verbs: ui-tree, tap, tap-focused, d-pad, screenshot. Foldables: posture + rotate.</p>
      </div>

      <div className="mt-12 rounded-lg border p-4 text-sm">
        <span className="font-medium">API:</span> {health.data ? "connected" : health.isLoading ? "checking…" : "disconnected"}
        <span className="ml-4 font-mono">MCP: /mcp · openapi.json · skill/SKILL.md</span>
      </div>
    </div>
  );
}
