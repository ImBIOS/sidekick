import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { ArrowRight, Menu, X } from "lucide-react";

export const Route = createFileRoute("/design/codenest")({
  component: CodeNestHero,
});

const STREAM = "https://stream.mux.com/tLkHO1qZoaaQOUeVWo8hEBeGQfySP02EPS02BmnNFyXys.m3u8";
const ACCENT = "#5ed29c";

function useHls(videoRef: React.RefObject<HTMLVideoElement | null>) {
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    let hls: Hls | null = null;
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = STREAM;
    } else if (Hls.isSupported()) {
      hls = new Hls({ enableWorker: false });
      hls.loadSource(STREAM);
      hls.attachMedia(video);
    } else {
      video.src = STREAM;
    }
    return () => {
      hls?.destroy();
    };
  }, [videoRef]);
}

function Header() {
  const [open, setOpen] = useState(false);
  const links = ["PROJECTS", "BLOG", "ABOUT", "RESUME"];
  return (
    <>
      <header className="absolute inset-x-0 top-0 z-30">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
          <a href="/design/codenest" className="text-lg font-extrabold tracking-tight text-white" style={{ fontFamily: "Inter, sans-serif" }}>
            CodeNest
          </a>
          <nav className="hidden items-center gap-8 md:flex" aria-label="Primary">
            {links.map((l) => (
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
          {links.map((l) => (
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
        </div>
      )}
    </>
  );
}

function LiquidGlassCard() {
  return (
    <div className="liquid-card relative h-[200px] w-[200px] translate-y-[-50px] p-5">
      <p className="text-[14px] font-medium text-white/80" style={{ fontFamily: "Inter, sans-serif" }}>
        [ 2025 ]
      </p>
      <p className="mt-2 text-[18px] leading-snug font-semibold text-white" style={{ fontFamily: "Inter, sans-serif" }}>
        Taught by <em style={{ fontFamily: "'Instrument Serif', serif" }}>Industry</em> Professionals
      </p>
      <p className="mt-2 text-[11px] leading-relaxed text-white/60" style={{ fontFamily: "Inter, sans-serif" }}>
        Mentorship and instruction from top tech software engineers.
      </p>
    </div>
  );
}

function CodeNestHero() {
  const videoRef = useRef<HTMLVideoElement>(null);
  useHls(videoRef);

  return (
    <div className="bg-[#070b0a] text-white">
      <link
        rel="stylesheet"
        href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Plus+Jakarta+Sans:wght@700&family=Instrument+Serif:ital@0;1&display=swap"
      />
      <style>{`
        .liquid-card {
          background: rgba(255, 255, 255, 0.01);
          background-blend-mode: luminosity;
          backdrop-filter: blur(4px);
          -webkit-backdrop-filter: blur(4px);
          box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.1);
          border-radius: 20px;
        }
        .liquid-card::before {
          content: "";
          position: absolute;
          inset: 0;
          border-radius: 20px;
          padding: 1.4px;
          background: linear-gradient(180deg, rgba(255,255,255,0.55), rgba(255,255,255,0.06));
          -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
          -webkit-mask-composite: xor;
          mask-composite: exclude;
          pointer-events: none;
        }
      `}</style>

      <section className="relative flex min-h-screen flex-col overflow-hidden">
        {/* Background video */}
        <video
          ref={videoRef}
          className="absolute inset-0 h-full w-full object-cover opacity-60"
          autoPlay
          muted
          loop
          playsInline
          aria-hidden="true"
        />
        {/* Readability gradients */}
        <div
          className="absolute inset-0"
          style={{ background: "linear-gradient(to right, #070b0a 0%, rgba(7,11,10,0.55) 35%, transparent 70%)" }}
          aria-hidden="true"
        />
        <div
          className="absolute inset-0"
          style={{ background: "linear-gradient(to top, #070b0a 0%, transparent 45%)" }}
          aria-hidden="true"
        />
        {/* Desktop grid lines */}
        <div className="absolute inset-0 hidden md:block" aria-hidden="true">
          {[25, 50, 75].map((x) => (
            <span key={x} className="absolute inset-y-0 w-px bg-white/10" style={{ left: `${x}%` }} />
          ))}
        </div>
        {/* Central glow */}
        <svg
          className="absolute top-[-10%] left-1/2 w-[900px] max-w-none -translate-x-1/2"
          viewBox="0 0 900 320"
          aria-hidden="true"
        >
          <defs>
            <filter id="codenest-glow" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="25" />
            </filter>
          </defs>
          <ellipse cx="450" cy="140" rx="330" ry="90" fill="#5ed29c" opacity="0.28" filter="url(#codenest-glow)" />
          <ellipse cx="450" cy="150" rx="200" ry="55" fill="#0e3b2e" opacity="0.55" filter="url(#codenest-glow)" />
        </svg>

        <Header />

        {/* Hero content */}
        <div className="relative z-10 mx-auto flex w-full max-w-7xl flex-1 flex-col justify-center px-6 pt-32 pb-20">
          <LiquidGlassCard />
          <p
            className="text-[11px] font-bold tracking-[0.18em] text-[#5ed29c] uppercase"
            style={{ fontFamily: "'Plus Jakarta Sans', sans-serif" }}
          >
            Career-Ready Curriculum
          </p>
          <h1
            className="mt-4 max-w-4xl text-[40px] leading-[1.02] font-extrabold tracking-tight text-white uppercase md:text-[72px]"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            Launch your coding career<span style={{ color: ACCENT }}>.</span>
          </h1>
          <p
            className="mt-5 max-w-[512px] text-[14px] leading-relaxed text-white/70"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            Master in-demand coding skills with hands-on projects, guided lessons, and real-world
            mentorship. Whether you&apos;re a beginner or looking to upskill, CodeNest helps you go
            from zero to hire-ready.
          </p>
          <div className="mt-8">
            <a
              href="#"
              className="inline-flex items-center gap-2 rounded-full bg-[#5ed29c] px-7 py-3 text-sm font-bold tracking-wide text-[#070b0a] uppercase transition-opacity hover:opacity-90"
              style={{ fontFamily: "Inter, sans-serif" }}
            >
              Get Started <ArrowRight size={18} />
            </a>
          </div>
        </div>
      </section>
    </div>
  );
}
