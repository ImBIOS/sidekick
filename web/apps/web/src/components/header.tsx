import { Link } from "@tanstack/react-router";
import { useState } from "react";

import { SidekickMark } from "./brand";
import UserMenu from "./user-menu";

const LINKS = [
  { to: "/", label: "Product" },
  { to: "/pricing", label: "Pricing" },
  { to: "/alternatives", label: "Alternatives" },
  { to: "/dashboard", label: "Dashboard" },
] as const;

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
            <div className="px-2 py-2">
              <UserMenu />
            </div>
          </div>
        </nav>
      )}
    </header>
  );
}
