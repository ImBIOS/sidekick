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

type Ray = { angle: number; len: number; dots: { d: number; r: number; a: number }[] };

/**
 * Expo-style particle burst: fine rays radiating from center with dots.
 * Rendered once as a static frame (no animation loop): crisp at any size,
 * zero repaint cost, and nothing that can blink.
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
        r: 0.6 + rand() * 1.1,
        a: 0.35 + rand() * 0.4,
      }));
      return { angle, len, dots };
    });

    const draw = () => {
      // NOTE: use clientWidth/Height (CSS size), never canvas.width/height
      // (buffer size) — mixing them up blurs the whole field into grey mush.
      const w = canvas.clientWidth;
      const h = canvas.clientHeight;
      if (w === 0 || h === 0) return;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, w, h);
      const cx = w / 2;
      const cy = h / 2;
      const R = Math.min(w, h) / 2;

      ctx.strokeStyle = "rgba(255,255,255,0.13)";
      ctx.lineWidth = 1;
      for (const ray of field) {
        ctx.beginPath();
        ctx.moveTo(cx + Math.cos(ray.angle) * R * 0.12, cy + Math.sin(ray.angle) * R * 0.12);
        ctx.lineTo(cx + Math.cos(ray.angle) * R * ray.len, cy + Math.sin(ray.angle) * R * ray.len);
        ctx.stroke();
        for (const dot of ray.dots) {
          ctx.fillStyle = `rgba(255,255,255,${dot.a.toFixed(3)})`;
          ctx.beginPath();
          ctx.arc(cx + Math.cos(ray.angle) * R * ray.len * dot.d, cy + Math.sin(ray.angle) * R * ray.len * dot.d, dot.r, 0, Math.PI * 2);
          ctx.fill();
        }
      }
    };

    draw();
    const ro = new ResizeObserver(draw);
    ro.observe(canvas);
    return () => ro.disconnect();
  }, [rays]);

  return <canvas ref={ref} className={className} aria-hidden="true" />;
}
