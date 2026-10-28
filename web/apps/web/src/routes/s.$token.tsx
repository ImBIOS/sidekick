import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Container, SiteFooter } from "../components/marketing";

// Guest share viewer: GET /s/$token serves a read-only snapshot page
// (latest screenshot auto-refreshed, AX/UI tree, logs tail, recorded video
// when present). No login — the signed `sk_share_...` token IS the
// capability (scoped to one sim/emu, expiring, revocable).
//
// This TanStack route is the branded viewer around the same public JSON
// endpoints the control-plane already serves; `?api=` points it at any
// self-hosted plane (default: same-origin `/s/...`, which only works when
// web and plane share an origin — hosted deploys proxy /s to the plane).

export const Route = createFileRoute("/s/$token")({
  component: ShareViewer,
  head: ({ params }) => ({
    meta: [{ title: `Shared device ${params.token.slice(0, 12)}… — Sidekick` }],
  }),
});

interface Snapshot {
  kind: string;
  id: string;
  status: string;
  recording: boolean;
  has_shot: boolean;
  has_tree: boolean;
  has_video: boolean;
  expires_at: string;
}

function apiBase(): string {
  const q = new URLSearchParams(window.location.search).get("api");
  if (q) return q.replace(/\/$/, "");
  return "";
}

function ShareViewer() {
  const { token } = Route.useParams();
  const [tab, setTab] = useState<"tree" | "logs">("tree");
  const base = apiBase();
  const snap = useQuery({
    queryKey: ["share", token, base],
    queryFn: async (): Promise<Snapshot> => {
      const r = await fetch(`${base}/s/${token}?format=json`);
      if (!r.ok) throw new Error(`share ${r.status}`);
      return r.json();
    },
    refetchInterval: 5000,
    retry: false,
  });
  const text = useQuery({
    queryKey: ["share-text", token, base, tab],
    queryFn: async (): Promise<string> => {
      const r = await fetch(`${base}/s/${token}/${tab}`);
      if (!r.ok) throw new Error(`${tab} ${r.status}`);
      return r.text();
    },
    refetchInterval: 5000,
    retry: false,
  });

  return (
    <div className="bg-black text-zinc-100">
      <Container>
        <p className="pt-10 text-sm text-zinc-500">Sidekick shared session</p>
        {snap.isPending && <p className="mt-6 text-zinc-400">Loading shared session…</p>}
        {snap.isError && (
          <div className="mt-6">
            <h1 className="text-2xl font-bold">Link unavailable</h1>
            <p className="mt-2 text-zinc-400">
              This link is invalid, expired, or was revoked by its owner.
            </p>
          </div>
        )}
        {snap.data && (
          <div className="mt-6 pb-16">
            <h1 className="text-2xl font-bold">
              {snap.data.kind} <span className="font-mono text-nest">{snap.data.id}</span>
            </h1>
            <p className="mt-1 text-sm text-zinc-500">
              status: {snap.data.status}
              {snap.data.recording ? " · ● recording" : ""} · expires{" "}
              {new Date(snap.data.expires_at).toLocaleString()} · refreshes every 5s
            </p>
            {snap.data.has_shot ? (
              <img
                key={Date.now()}
                src={`${base}/s/${token}/screenshot`}
                alt="latest device screenshot"
                className="mt-6 max-w-full rounded-xl border border-white/10"
              />
            ) : (
              <p className="mt-6 text-zinc-500">No screenshot yet.</p>
            )}
            {snap.data.has_video && (
              <p className="mt-3">
                <a className="text-nest underline" href={`${base}/s/${token}/video`}>
                  Download recording (mp4)
                </a>
              </p>
            )}
            <div className="mt-6 flex gap-2">
              {(["tree", "logs"] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => setTab(t)}
                  className={`rounded-lg px-3 py-1 text-sm ${tab === t ? "bg-nest text-black" : "bg-zinc-900 text-zinc-300"}`}
                >
                  {t === "tree" ? "UI tree" : "Logs"}
                </button>
              ))}
            </div>
            <pre className="mt-2 max-h-[40vh] overflow-auto rounded-xl bg-zinc-950 p-4 text-xs whitespace-pre-wrap text-zinc-300">
              {text.isPending ? "Loading…" : text.isError ? "Not available yet." : text.data?.slice(0, 8192)}
            </pre>
            <p className="mt-4 text-xs text-zinc-600">
              Read-only guest view. No live video is streamed — this page shows
              snapshots the host posts every refresh interval.
            </p>
          </div>
        )}
      </Container>
      <SiteFooter />
    </div>
  );
}
