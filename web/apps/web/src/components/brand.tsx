import { useEffect, useRef } from "react";

/** White rounded-square mark with a black "S" — the Expo-Λ slot in our hero. */
export function SidekickMark({ size = 112 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 112 112" role="img" aria-label="Sidekick">
      <rect width="112" height="112" rx="26" fill="#fff" />
      <text
        x="56"
        y="78"
        textAnchor="middle"
        fontFamily="Inter, -apple-system, 'Segoe UI', sans-serif"
        fontWeight={900}
        fontSize="64"
        fill="#000"
      >
        S
      </text>
    </svg>
  );
}

/** Deterministic PRNG so the burst is stable across renders. */
function mulberry32(seed: number) {
  return () => {
    seed |= 0;
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

type Ray = { angle: number; len: number; dots: { d: number; r: number; tw: number }[] };

/**
 * Expo-style particle burst: fine rays radiating from center with drifting
 * dots. Static frame when prefers-reduced-motion; pauses offscreen.
 */
export function BurstCanvas({ rays = 170, className }: { rays?: number; className?: string }) {
  const ref = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const rand = mulberry32(7);
    const field: Ray[] = Array.from({ length: rays }, () => {
      const angle = rand() * Math.PI * 2;
      const len = 0.35 + rand() * 0.65;
      const dots = Array.from({ length: 2 + Math.floor(rand() * 4) }, () => ({
        d: rand(),
        r: 0.6 + rand() * 1.4,
        tw: rand() * Math.PI * 2,
      }));
      return { angle, len, dots };
    });

    let raf = 0;
    let running = true;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    const draw = (t: number) => {
      if (!running) return;
      const { width: w, height: h } = canvas;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      if (canvas.width !== w * dpr) {
        canvas.width = w * dpr;
        canvas.height = h * dpr;
      }
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);
      const cx = w / 2;
      const cy = h / 2;
      const R = Math.min(w, h) / 2;
      const spin = reduced ? 0 : t / 90000;

      ctx.strokeStyle = "rgba(255,255,255,0.16)";
      ctx.lineWidth = 1;
      for (const ray of field) {
        const a = ray.angle + spin;
        ctx.beginPath();
        ctx.moveTo(cx + Math.cos(a) * R * 0.12, cy + Math.sin(a) * R * 0.12);
        ctx.lineTo(cx + Math.cos(a) * R * ray.len, cy + Math.sin(a) * R * ray.len);
        ctx.stroke();
        for (const dot of ray.dots) {
          const twinkle = reduced ? 1 : 0.55 + 0.45 * Math.sin(t / 900 + dot.tw);
          ctx.fillStyle = `rgba(255,255,255,${(0.75 * twinkle).toFixed(3)})`;
          ctx.beginPath();
          ctx.arc(cx + Math.cos(a) * R * ray.len * dot.d, cy + Math.sin(a) * R * ray.len * dot.d, dot.r, 0, Math.PI * 2);
          ctx.fill();
        }
      }
      if (!reduced) raf = requestAnimationFrame(draw);
    };

    const io = new IntersectionObserver(([entry]) => {
      if (entry?.isIntersecting) {
        running = true;
        raf = requestAnimationFrame(draw);
      } else {
        running = false;
        cancelAnimationFrame(raf);
      }
    });
    io.observe(canvas);
    raf = requestAnimationFrame(draw);
    return () => {
      running = false;
      cancelAnimationFrame(raf);
      io.disconnect();
    };
  }, [rays]);

  return <canvas ref={ref} className={className} aria-hidden="true" />;
}
