import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";

/**
 * SSR-safe fade-up on scroll without a runtime animation library:
 * IntersectionObserver toggles a class, CSS owns the transition
 * (compositor-only opacity + transform, runs once, reduced-motion safe).
 * UI-UX-Pro-Max guidance: motion for entrance only, never for state.
 */
export function Reveal({
  children,
  delay = 0,
  className,
}: {
  children: ReactNode;
  delay?: number;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [shown, setShown] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (typeof IntersectionObserver === "undefined") {
      setShown(true);
      return;
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setShown(true);
          io.disconnect();
        }
      },
      { rootMargin: "-80px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <div
      ref={ref}
      className={`reveal${shown ? " reveal-shown" : ""}${className ? ` ${className}` : ""}`}
      style={{ transitionDelay: `${delay}s` }}
    >
      {children}
    </div>
  );
}
