import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { Menu, X } from "lucide-react";
import { HeroCodenest } from "../components/hero-codenest";

export const Route = createFileRoute("/design/codenest")({
  component: CodeNestDemo,
});

const LINKS = ["PROJECTS", "BLOG", "ABOUT", "RESUME"];

function DemoHeader() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <header className="absolute inset-x-0 top-0 z-30">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
          <a
            href="/design/codenest"
            className="text-lg font-extrabold tracking-tight text-white"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            CodeNest
          </a>
          <nav className="hidden items-center gap-8 md:flex" aria-label="Primary">
            {LINKS.map((l) => (
              <a
                key={l}
                href="#"
                className="text-[16px] font-medium text-white transition-colors hover:text-[#5ed29c]"
                style={{ fontFamily: "Inter, sans-serif" }}
              >
                {l}
              </a>
            ))}
          </nav>
          <div className="hidden md:block">
            <a
              href="#"
              className="text-[16px] font-semibold text-white transition-colors hover:text-[#5ed29c]"
              style={{ fontFamily: "Inter, sans-serif" }}
            >
              [REGISTER NOW]
            </a>
          </div>
          <button
            className="text-white md:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            onClick={() => setOpen((v) => !v)}
          >
            {open ? <X size={24} /> : <Menu size={24} />}
          </button>
        </div>
      </header>
      {open && (
        <div className="fixed inset-0 z-40 flex flex-col items-center justify-center gap-8 bg-[#070b0a]/95 md:hidden">
          <button
            className="absolute top-5 right-6 text-white"
            aria-label="Close menu"
            onClick={() => setOpen(false)}
          >
            <X size={28} />
          </button>
          {LINKS.map((l) => (
            <a
              key={l}
              href="#"
              onClick={() => setOpen(false)}
              className="text-2xl font-bold tracking-tight text-white"
              style={{ fontFamily: "Inter, sans-serif" }}
            >
              {l}
            </a>
          ))}
          <a
            href="#"
            onClick={() => setOpen(false)}
            className="text-2xl font-bold tracking-tight text-[#5ed29c]"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            [REGISTER NOW]
          </a>
        </div>
      )}
    </>
  );
}

function CodeNestDemo() {
  return (
    <HeroCodenest
      headerRow={<DemoHeader />}
      cardTag="[ 2025 ]"
      cardTitle={
        <>
          Taught by <em style={{ fontFamily: "'Instrument Serif', serif" }}>Industry</em> Professionals
        </>
      }
      cardDesc="Mentorship and instruction from top tech software engineers."
      eyebrow="Career-Ready Curriculum"
      headline={
        <>
          Launch your coding career<span style={{ color: "#5ed29c" }}>.</span>
        </>
      }
      description="Master in-demand coding skills with hands-on projects, guided lessons, and real-world mentorship. Whether you're a beginner or looking to upskill, CodeNest helps you go from zero to hire-ready."
      ctaLabel="Get Started"
      ctaTo="/design/codenest"
    />
  );
}
