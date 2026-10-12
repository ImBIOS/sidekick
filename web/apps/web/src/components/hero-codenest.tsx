import { useEffect, useRef } from "react";
import type { ReactNode, RefObject } from "react";
import { Link } from "@tanstack/react-router";
import Hls from "hls.js";
import { ArrowRight } from "lucide-react";

export const HERO_STREAM =
  "https://stream.mux.com/tLkHO1qZoaaQOUeVWo8hEBeGQfySP02EPS02BmnNFyXys.m3u8";

/**
 * HLS background video with belt and suspenders:
 * native playback where supported (Safari), hls.js elsewhere with
 * enableWorker:false for sandboxed environments, explicit play() on
 * canplay (autoplay attributes alone are unreliable once hypers),
 * and a fatal-error fallback to a direct src attempt.
 * A static CSS scene sits underneath, so a stalled stream still
 * renders the dark-green CodeNest look instead of pure black.
 */
export function useHeroVideo(videoRef: RefObject<HTMLVideoElement | null>) {
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    let hls: Hls | null = null;
    const tryPlay = () => {
      video.play().catch(() => {});
    };
    video.addEventListener("canplay", tryPlay);
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = HERO_STREAM;
    } else if (Hls.isSupported()) {
      hls = new Hls({ enableWorker: false });
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal) return;
        try {
          hls?.destroy();
        } catch {
          /* noop */
        }
        hls = null;
        video.src = HERO_STREAM;
      });
      hls.loadSource(HERO_STREAM);
      hls.attachMedia(video);
    } else {
      video.src = HERO_STREAM;
    }
    return () => {
      video.removeEventListener("canplay", tryPlay);
      try {
        hls?.destroy();
      } catch {
        /* noop */
      }
    };
  }, [videoRef]);
}

/** Static scene under the video: green masses + faint code-rain bars. */
function FallbackScene() {
  return (
    <div className="absolute inset-0" aria-hidden="true">
      <div
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(46% 34% at 38% 22%, rgba(35,120,80,0.34), transparent 70%), radial-gradient(30% 26% at 66% 30%, rgba(94,210,156,0.16), transparent 70%), radial-gradient(52% 40% at 50% 108%, rgba(7,11,10,0.9), transparent 70%), #070b0a",
        }}
      />
      <div
        className="absolute inset-0 opacity-[0.55]"
        style={{
          background:
            "repeating-linear-gradient(to right, transparent 0 46px, rgba(94,210,156,0.05) 46px 48px, transparent 48px 96px)",
          maskImage: "linear-gradient(to right, black 0%, transparent 62%)",
          WebkitMaskImage: "linear-gradient(to right, black 0%, transparent 62%)",
        }}
      />
    </div>
  );
}

export interface HeroCodenestCopy {
  /** Glass card above the headline. Pass null to omit (hero stays full-screen). */
  card: {
    tag: string;
    title: ReactNode;
    desc: string;
  } | null;
  eyebrow: string;
  headline: ReactNode;
  description: string;
  ctaLabel: string;
  ctaTo: string;
  microcopy?: ReactNode;
  /** In-hero header row (brand + links + bracket CTA). Omit on pages that already have a site header. */
  headerRow?: ReactNode;
}

/** Full-bleed CodeNest hero geometry. Copy via props — Sidekick words, same bones. */
export function HeroCodenest(props: HeroCodenestCopy) {
  const videoRef = useRef<HTMLVideoElement>(null);
  useHeroVideo(videoRef);

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
        <FallbackScene />
        <video
          ref={videoRef}
          className="absolute inset-0 h-full w-full object-cover opacity-60"
          autoPlay
          muted
          loop
          playsInline
          preload="auto"
          poster="/hero-poster.jpg"
          aria-hidden="true"
        />
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
        <div className="absolute inset-0 hidden md:block" aria-hidden="true">
          {[25, 50, 75].map((x) => (
            <span key={x} className="absolute inset-y-0 w-px bg-white/10" style={{ left: `${x}%` }} />
          ))}
        </div>
        <svg
          className="absolute top-[-12%] left-1/2 w-[1000px] max-w-none -translate-x-1/2"
          viewBox="0 0 1000 340"
          aria-hidden="true"
        >
          <defs>
            <filter id="hero-glow" x="-50%" y="-50%" width="200%" height="200%">
              <feGaussianBlur stdDeviation="25" />
            </filter>
          </defs>
          <ellipse cx="500" cy="150" rx="360" ry="100" fill="#5ed29c" opacity="0.3" filter="url(#hero-glow)" />
          <ellipse cx="380" cy="130" rx="190" ry="70" fill="#1d7a52" opacity="0.5" filter="url(#hero-glow)" />
          <ellipse cx="640" cy="165" rx="150" ry="60" fill="#0e3b2e" opacity="0.6" filter="url(#hero-glow)" />
        </svg>

        {props.headerRow}

        <div className="relative z-10 mx-auto flex w-full max-w-7xl flex-1 flex-col justify-center px-6 pt-32 pb-20">
          {props.card && (
          <div className="liquid-card relative h-[200px] w-[200px] translate-y-[-50px] p-5">
            <p className="text-[14px] font-medium text-white/80" style={{ fontFamily: "Inter, sans-serif" }}>
              {props.card.tag}
            </p>
            <p className="mt-2 text-[18px] leading-snug font-semibold text-white" style={{ fontFamily: "Inter, sans-serif" }}>
              {props.card.title}
            </p>
            <p className="mt-2 text-[11px] leading-relaxed text-white/60" style={{ fontFamily: "Inter, sans-serif" }}>
              {props.card.desc}
            </p>
            <ArrowRight size={16} className="absolute right-5 bottom-5 text-white/70" aria-hidden="true" />
          </div>
          )}
          <p
            className="text-[11px] font-bold tracking-[0.18em] text-[#5ed29c] uppercase"
            style={{ fontFamily: "'Plus Jakarta Sans', sans-serif" }}
          >
            {props.eyebrow}
          </p>
          <h1
            className="mt-4 max-w-5xl text-[40px] leading-[1.02] font-extrabold tracking-tight text-white uppercase md:text-[72px]"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            {props.headline}
          </h1>
          <p
            className="mt-5 max-w-[512px] text-[14px] leading-relaxed text-white/70"
            style={{ fontFamily: "Inter, sans-serif" }}
          >
            {props.description}
          </p>
          <div className="mt-8">
            <Link
              to={props.ctaTo}
              className="inline-flex items-center gap-2 rounded-full bg-[#5ed29c] px-7 py-3 text-sm font-bold tracking-wide text-[#070b0a] uppercase transition-opacity hover:opacity-90"
              style={{ fontFamily: "Inter, sans-serif" }}
            >
              {props.ctaLabel} <ArrowRight size={18} />
            </Link>
          </div>
          {props.microcopy}
        </div>
      </section>
    </div>
  );
}
