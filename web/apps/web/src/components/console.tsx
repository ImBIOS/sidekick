import { Link, useLocation, useNavigate, useRouterState } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

import { authClient } from "@/lib/auth-client";
import { useTRPC } from "@/utils/trpc";
import { SidekickMark } from "./brand";

const SECTIONS: { title: string; items: { to: string; label: string; exact: boolean }[] }[] = [
  {
    title: "Platform",
    items: [
      { to: "/console", label: "Get started", exact: true },
      { to: "/simulators", label: "Simulators", exact: false },
      { to: "/builds", label: "Builds", exact: false },
    ],
  },
  {
    title: "Account",
    items: [{ to: "/settings", label: "API keys", exact: false }],
  },
];

const TITLES: [RegExp, string][] = [
  [/^\/console$/, "Get started"],
  [/^\/simulators/, "Simulators"],
  [/^\/builds/, "Builds"],
  [/^\/settings/, "API keys"],
];

function pageTitle(pathname: string): string {
  for (const [re, label] of TITLES) {
    if (re.test(pathname)) return label;
  }
  return "Console";
}

/** Console shell: shadcn dashboard anatomy (sidebar + header bar + main).
 * Sidebar sections, header breadcrumb, mobile top tabs. No fake data:
 * stats come from tRPC, everything else is actions and links. */
export function ConsoleShell({ children }: { children: React.ReactNode }) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const navigate = useNavigate();
  const { data: session } = authClient.useSession();
  const [open, setOpen] = useState(false);

  return (
    <div className="min-h-svh bg-ink text-zinc-100">
      {/* Header bar */}
      <header className="sticky top-0 z-40 border-b border-white/10 bg-ink/80 backdrop-blur">
        <div className="flex h-14 items-center gap-3 px-4 md:px-6">
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-label="Menu"
            aria-expanded={open}
            className="flex h-9 w-9 items-center justify-center rounded-md text-zinc-300 hover:bg-white/5 hover:text-white md:hidden"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
              {open ? <path d="M3 3l10 10M13 3L3 13" /> : <path d="M2 5h12M2 8h12M2 11h12" />}
            </svg>
          </button>
          <nav aria-label="Breadcrumb" className="flex min-w-0 items-center gap-2 text-sm">
            <span className="truncate text-zinc-500">Console</span>
            <span aria-hidden="true" className="text-zinc-700">/</span>
            <span className="truncate font-medium text-white">{pageTitle(pathname)}</span>
          </nav>
          <div className="ml-auto flex items-center gap-2">
            <a
              href="/llms.txt"
              className="hidden rounded-md px-3 py-1.5 text-sm text-zinc-400 transition-colors hover:bg-white/5 hover:text-white sm:block"
            >
              Docs
            </a>
            <a
              href="https://github.com/ImBIOS/sidekick"
              className="hidden items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3.5 py-1.5 text-sm text-zinc-300 transition-colors hover:bg-white/10 hover:text-white sm:inline-flex"
            >
              GitHub
            </a>
          </div>
        </div>
        {open && (
          <nav className="border-t border-white/10 px-4 py-2 md:hidden" aria-label="Console">
            {SECTIONS.flatMap((s) => s.items).map(({ to, label }) => (
              <Link
                key={to}
                to={to}
                onClick={() => setOpen(false)}
                className="block rounded-lg px-2 py-2.5 text-[15px] text-zinc-300 hover:bg-white/5 hover:text-white"
              >
                {label}
              </Link>
            ))}
          </nav>
        )}
      </header>

      <div className="md:grid md:grid-cols-[240px_minmax(0,1fr)]">
        {/* Sidebar */}
        <aside className="hidden border-r border-white/10 p-4 md:block">
          <Link to="/" className="flex items-center gap-2 px-2 py-2" aria-label="Sidekick home">
            <SidekickMark size={24} />
            <span className="font-bold tracking-tight text-white">Sidekick</span>
          </Link>
          <nav className="mt-4 space-y-5" aria-label="Console">
            {SECTIONS.map((section) => (
              <div key={section.title}>
                <p className="px-3 text-[11px] font-semibold tracking-[0.14em] text-zinc-600 uppercase">
                  {section.title}
                </p>
                <div className="mt-1 space-y-0.5">
                  {section.items.map(({ to, label, exact }) => {
                    const active = exact ? pathname === to : pathname.startsWith(to);
                    return (
                      <Link
                        key={to}
                        to={to}
                        className={`block rounded-lg px-3 py-2 text-sm transition-colors ${
                          active
                            ? "bg-white/10 font-medium text-white"
                            : "text-zinc-400 hover:bg-white/5 hover:text-white"
                        }`}
                      >
                        {label}
                      </Link>
                    );
                  })}
                </div>
              </div>
            ))}
          </nav>
          <div className="mt-6 border-t border-white/10 p-3">
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
    </div>
  );
}

/** Current location hook for console pages that need it. */
export function useConsolePath() {
  return useLocation().pathname;
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
  children: React.ReactNode;
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

/** Click-first device creation: split button from Limrun onboarding.
 * Primary creates an Android emulator (Pixel_7) — the path that boots
 * today via the linux-pool emu-host. iOS stays in the menu, disabled,
 * until a Mac fleet (sim-daemon + build-daemon) is joined.
 * The agent-first CLI steps stay the main path — this is the secondary
 * click path, backed by the devices router (ephemeral key, missing ids
 * resolve to null so empty states can offer creation). */
export function CreateDeviceSplitButton({ caption }: { caption?: string }) {
  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onCreated = (label: string) => {
    setError(null);
    setOpen(false);
    toast.success(label);
    queryClient.invalidateQueries({ queryKey: trpc.devices.listSims.queryKey() });
    queryClient.invalidateQueries({ queryKey: trpc.devices.listEmus.queryKey() });
    navigate({ to: "/simulators" });
  };

  const emu = useMutation(
    trpc.devices.createEmu.mutationOptions({
      onSuccess: () => onCreated("Emulator provisioning started"),
      onError: (e) => setError(e.message),
    }),
  );
  const pending = emu.isPending;

  return (
    <div>
      <div className="relative inline-flex">
        <button
          type="button"
          disabled={pending}
          onClick={() => emu.mutate({ profile: "Pixel_7" })}
          className="inline-flex items-center gap-2 rounded-l-xl bg-white px-4 py-2.5 text-sm font-semibold text-black transition-colors hover:bg-zinc-200 disabled:opacity-60"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
            <rect x="4" y="1.5" width="8" height="13" rx="1.5" />
            <line x1="7" y1="12.5" x2="9" y2="12.5" strokeLinecap="round" />
          </svg>
          {pending ? "Creating…" : "Create Android Emulator"}
        </button>
        <button
          type="button"
          disabled={pending}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-label="More device options"
          onClick={() => setOpen((v) => !v)}
          className="rounded-r-xl border-l border-black/10 bg-white px-3 text-black transition-colors hover:bg-zinc-200 disabled:opacity-60"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true" className={open ? "rotate-180" : ""}>
            <path d="M2.5 4.5L6 8l3.5-3.5" />
          </svg>
        </button>
        {open && (
          <div role="menu" className="absolute top-full left-0 z-10 mt-1 min-w-72 rounded-xl border border-white/10 bg-zinc-900 p-1 shadow-xl">
            <button
              type="button"
              role="menuitem"
              disabled
              title="iOS needs a joined Mac fleet (sim-daemon + build-daemon)"
              className="flex w-full cursor-not-allowed items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm whitespace-nowrap text-zinc-500"
            >
              <span>Create iOS Simulator</span>
              <span className="shrink-0 rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[11px] font-medium text-zinc-400">
                Coming soon · Mac fleet
              </span>
            </button>
          </div>
        )}
      </div>
      {caption && <p className="mt-2 text-sm text-zinc-500">{caption}</p>}
      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
    </div>
  );
}
