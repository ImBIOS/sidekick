import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { SidekickMark } from "./brand";
import UserMenu from "./user-menu";

const LINKS = [
  { to: "/", label: "Product" },
  { to: "/pricing", label: "Pricing" },
  { to: "/alternatives", label: "Alternatives" },
] as const;

function formatStars(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  return `${n}`;
}

let starsPromise: Promise<number | null> | null = null;
function fetchStars(): Promise<number | null> {
  if (!starsPromise) {
    starsPromise = fetch("https://api.github.com/repos/ImBIOS/sidekick")
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => (typeof j?.stargazers_count === "number" ? j.stargazers_count : null))
      .catch(() => null);
  }
  return starsPromise;
}

/** GitHub star pill, Expo-style. Hides silently when the API is unreachable. */
export function StarCounter() {
  const [stars, setStars] = useState<number | null>(null);
  useEffect(() => {
    fetchStars().then(setStars);
  }, []);
  return (
    <a
      href="https://github.com/ImBIOS/sidekick"
      target="_blank"
      rel="noreferrer"
      className="hidden items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-3.5 py-1.5 text-sm text-zinc-300 transition-colors hover:bg-white/10 hover:text-white sm:inline-flex"
      aria-label="Star Sidekick on GitHub"
    >
      <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
      </svg>
      {stars !== null ? <span className="font-mono text-xs">{formatStars(stars)}</span> : null}
      <span className="hidden lg:inline">Star</span>
    </a>
  );
}

export default function Header() {
  const [open, setOpen] = useState(false);
  return (
    <header className="sticky top-0 z-50 border-b border-white/10 bg-black/80 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
        <Link to="/" className="flex items-center gap-2.5" aria-label="Sidekick home">
          <SidekickMark size={28} />
          <span className="text-lg font-bold tracking-tight text-white">Sidekick</span>
        </Link>
        <nav className="hidden items-center gap-7 text-sm text-zinc-400 md:flex">
          {LINKS.map(({ to, label }) => (
            <Link key={to} to={to} className="transition-colors hover:text-white">
              {label}
            </Link>
          ))}
        </nav>
        <div className="hidden items-center gap-2 md:flex">
          <StarCounter />
          <UserMenu />
        </div>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-label="Menu"
          aria-expanded={open}
          className="flex h-9 w-9 items-center justify-center rounded-full border border-white/15 text-white md:hidden"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
            {open ? <path d="M3 3l10 10M13 3L3 13" /> : <path d="M2 5h12M2 8h12M2 11h12" />}
          </svg>
        </button>
      </div>
      {open && (
        <nav className="border-t border-white/10 px-6 py-3 md:hidden">
          <div className="flex flex-col gap-1">
            {LINKS.map(({ to, label }) => (
              <Link
                key={to}
                to={to}
                onClick={() => setOpen(false)}
                className="rounded-lg px-2 py-2.5 text-[15px] text-zinc-300 hover:bg-white/5 hover:text-white"
              >
                {label}
              </Link>
            ))}
            <div className="flex items-center gap-2 px-2 py-2" onClick={() => setOpen(false)}>
              <UserMenu />
            </div>
          </div>
        </nav>
      )}
    </header>
  );
}
