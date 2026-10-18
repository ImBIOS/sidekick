# Sidekick design system — CodeNest language, whole product

Single source of truth for brand + UI. Agents and humans follow this file;
`web/scripts/design-lint.sh` enforces the machine-checkable parts in CI.

## Tokens (`web/packages/ui/src/styles/globals.css`)

| Token | Value | Use |
|---|---|---|
| `--color-nest` | `#5ed29c` | The ONLY accent: CTAs, eyebrows, periods, yes-badges, verified panels, active states |
| `--color-nest-deep` | `#0e3b2e` | Glow cores, dark green washes |
| `--color-ink` | `#070b0a` | Page bg, header tint (`bg-ink/80`, `bg-ink/40`) |
| `--font-sans` | Inter | Everything except below |
| `--font-jakarta` | Plus Jakarta Sans | Eyebrows only (11px, bold, tracking 0.18em, uppercase) |
| `--font-serif-display` | Instrument Serif italic | Single accent words inside headlines/cards |

Tailwind v4 generates `text-nest`, `bg-nest/10`, `border-nest/20`, `font-jakarta` from these. Fonts load once globally in `__root.tsx` — never per-component `<link>` tags.

## Voice (applies to copy AND visuals)

- Agent-first, PM-friendly, casual. Competitor names/prices live in `comparison.ts` — update cells there, never in prose.
- Honesty pattern: sourced + dated claims, a real "Where X is stronger" section, decision guide, FAQ. Roadmap cells (🛣️) count toward Sidekick's score ONLY with a dated note — never silently flip roadmap to yes.
- Emojis ARE our badge language (✅ 🛣️ ➖ ❌) — the one place they are allowed. Everywhere else: lucide icons or SVG, never emoji.

## Page patterns

- Hero: CodeNest geometry — full-bleed dark media (video w/ poster fallback), grid lines (desktop), green glow, glass card (200px, mask-composite border), Jakarta eyebrow, Inter ExtraBold uppercase headline 40→72px with green period, 14px/70% description (max 512px), single green pill CTA.
- Sections: `py-24/32`, centered eyebrow + headline + descriptive subcopy.
- Tables: `CompareTable` (sticky first column, category bands, `compact` on landing). Footnotes carry legend + per-guide check dates.
- Motion: entrance-only (fade-up once via `Reveal`, tab fades via CSS). Compositor properties only (opacity/transform), `prefers-reduced-motion` respected. No animation loops except the hero video. No `backdrop-blur` over animated canvases (caused the grey-wash blink).
- Header: transparent ink glass at top (`bg-ink/40`), denser on scroll (`bg-ink/80`), sticky all page (root grid is `min-h-svh`, never `h-svh`).

## Anti-slop (CI-enforced, see design-lint.sh)

1. No purple/blue gradients (`from-purple`, `to-blue`, …) — green monochrome + ink only.
2. No lorem ipsum / placeholder copy — every string ships real.
3. No emoji outside comparison badges — lucide/SVG elsewhere.
4. No per-component font `<link>` tags — root only.
5. No `TODO`/`FIXME` in shipped UI code.
6. No animation runtime for entrances — CSS + IntersectionObserver only.
7. No hardcoded competitor claims in prose — `comparison.ts` is the source.
