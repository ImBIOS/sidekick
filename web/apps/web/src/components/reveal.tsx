import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

/**
 * SSR-safe fade-up on scroll. Compositor-only (opacity + transform),
 * runs once, disabled when the user prefers reduced motion.
 * UI-UX-Pro-Max guidance applied: motion for entrance only, never for
 * conveying state; content is fully visible with JS disabled (SSR HTML).
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
  const reduce = useReducedMotion();
  if (reduce) return <div className={className}>{children}</div>;
  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 24 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-80px" }}
      transition={{ duration: 0.6, delay, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </motion.div>
  );
}
