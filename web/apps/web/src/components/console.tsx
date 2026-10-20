import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import { useState } from "react";
import type { ReactNode } from "react";
import { toast } from "sonner";

import { authClient } from "@/lib/auth-client";
import { SidekickMark } from "./brand";

const NAV = [
  { to: "/console", label: "Get started", exact: true },
  { to: "/simulators", label: "Simulators", exact: false },
  { to: "/builds", label: "Builds", exact: false },
  { to: "/settings", label: "API keys", exact: false },
] as const;

/** Console shell: Limrun-pattern sidebar (CodeNest dark). Wraps every /console/* page. */
export function ConsoleShell({ children }: { children: ReactNode }) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const navigate = useNavigate();
  const { data: session } = authClient.useSession();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="min-h-svh bg-ink text-zinc-100 md:grid md:grid-cols-[240px_minmax(0,1fr)]">
      <aside className="border-b border-white/10 md:border-r md:border-b-0">
        <div className="flex items-center justify-between px-5 py-4">
          <Link to="/" className="flex items-center gap-2" aria-label="Sidekick home">
            <SidekickMark size={24} />
            {!collapsed && <span className="font-bold tracking-tight text-white">Sidekick</span>}
          </Link>
          <button
            type="button"
            onClick={() => setCollapsed((v) => !v)}
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            className="hidden rounded-md px-2 py-1 text-xs text-zinc-500 hover:bg-white/5 hover:text-white md:block"
          >
            {collapsed ? "»" : "«"}
          </button>
        </div>
        <nav className="flex gap-1 overflow-x-auto px-3 pb-3 md:flex-col md:pb-0" aria-label="Console">
          {NAV.map(({ to, label, exact }) => {
            const active = exact ? pathname === to : pathname.startsWith(to);
            return (
              <Link
                key={to}
                to={to}
                className={`rounded-lg px-3 py-2 text-sm whitespace-nowrap transition-colors ${
                  active
                    ? "bg-white/10 font-medium text-white"
                    : "text-zinc-400 hover:bg-white/5 hover:text-white"
                }`}
              >
                {collapsed ? label.slice(0, 1) : label}
              </Link>
            );
          })}
          <a
            href="/llms.txt"
            className="rounded-lg px-3 py-2 text-sm whitespace-nowrap text-zinc-400 transition-colors hover:bg-white/5 hover:text-white"
          >
            {collapsed ? "D" : "Docs"}
          </a>
        </nav>
        <div className="hidden border-t border-white/10 p-4 md:block">
          <p className="truncate text-sm font-medium text-white">{session?.user.name ?? ""}</p>
          <p className="truncate text-xs text-zinc-500">{session?.user.email ?? ""}</p>
          <button
            type="button"
            onClick={() =>
              authClient.signOut({ fetchOptions: { onSuccess: () => navigate({ to: "/" }) } })
            }
            className="mt-2 text-xs text-zinc-500 underline hover:text-white"
          >
            Sign out
          </button>
        </div>
      </aside>
      <main className="min-w-0 px-6 py-10 md:px-12">{children}</main>
    </div>
  );
}

/** Copy button for keys and commands. */
export function CopyButton({ text, label = "Copy" }: { text: string; label?: string }) {
  return (
    <button
      type="button"
      onClick={() => {
        void navigator.clipboard.writeText(text).then(
          () => toast.success("Copied to clipboard"),
          () => toast.error("Copy failed — select and copy manually"),
        );
      }}
      className="shrink-0 rounded-md border border-white/15 bg-white/5 px-3 py-1.5 text-xs font-medium text-zinc-200 transition-colors hover:bg-white/10 hover:text-white"
    >
      {label}
    </button>
  );
}

/** Terminal block with copy (Limrun-style quickstart). */
export function CodeBlock({ lines, title }: { lines: string; title?: string }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-white/10 bg-zinc-950">
      {title && (
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
          <p className="text-xs font-medium text-zinc-500">{title}</p>
          <CopyButton text={lines} />
        </div>
      )}
      <pre className="overflow-x-auto p-4 font-mono text-[13px] leading-relaxed text-zinc-200">
        {lines}
      </pre>
    </div>
  );
}

/** Numbered onboarding step. */
export function Step({
  n,
  title,
  children,
}: {
  n: number;
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="flex gap-4">
      <span
        aria-hidden="true"
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-nest/30 bg-nest/10 font-mono text-sm font-bold text-nest"
      >
        {n}
      </span>
      <div className="min-w-0 flex-1">
        <h3 className="font-semibold text-white">{title}</h3>
        <div className="mt-2 space-y-3 text-sm text-zinc-400">{children}</div>
      </div>
    </section>
  );
}
