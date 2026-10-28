# Sidekick — source-available Limrun alternative (Apache-2.0 client + FSL server)

Remote Xcode, iOS simulators, Android emulators for cloud agents. Your agent in E2B/Daytona/Docker builds mobile apps with no Mac.

`sidekick xcode build .` → stream logs → `sidekick ios create` → tap/type/screenshot → preview link + demo video on PR. Same loop as Limrun, TV/Roku breadth borrowed from VibeView, self-hostable.

## Why

Coding agents moved to Linux cloud sandboxes. No Xcode, no simulator, no emulator, no GPU Unity. Sidekick exposes them as remote services: one `sidekick` CLI + skill file, no sandbox change.

## Limrun vs VibeView — what we copied, what we do better

<!-- COMPARISON-START -->
<details>
<summary>Full 28-row comparison (all 9 vendors) — generated from <code>web/apps/web/src/data/comparison.ts</code>, last updated 18 Oct 2026</summary>

| Feature | Sidekick | VibeView | EAS Simulator | BrowserStack | Expo EAS Build | Limrun | Appetize | AWS Device Farm | LambdaTest (TestMu AI) | Sauce Labs |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Remote Xcode build (no Mac needed) | 🛣️ Roadmap (Claim-logs-finish loop unit-tested (fakeRunner); no Mac fleet witnessed) | ➖ Partial | ❌ No | ❌ No | ➖ Partial | ✅ Yes | ❌ No | ❌ No | ❌ No | ❌ No |
| Android / Gradle cloud build | 🛣️ Roadmap (Daemon runs xcodebuild only; no Gradle path implemented) | ✅ Yes | ❌ No | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ❌ No | ❌ No |
| Signing handled (certs, profiles, keystores) | 🛣️ Roadmap (No signing code yet) | ✅ Yes | ❌ No | ❌ No | ✅ Yes | ➖ Partial | ❌ No | ❌ No | ❌ No | ❌ No |
| Store submit from terminal (TestFlight + Play) | 🛣️ Roadmap (Play dry-run validated in unit tests only; no live upload witnessed) | ✅ Yes | ❌ No | ❌ No | ✅ Yes | ➖ Partial | ❌ No | ❌ No | ❌ No | ❌ No |
| iOS simulators | 🛣️ Roadmap (Provision/verb loop unit-tested (fakeDevice); needs a Mac fleet) | ✅ Yes | ✅ Yes | ✅ Yes | ➖ Partial | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes | ✅ Yes |
| Android emulators + adb tunnel | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes | ➖ Partial |
| Apple TV / tvOS + Android TV | 🛣️ Roadmap (No TV code yet) | ✅ Yes | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No |
| Roku (real hardware) | 🛣️ Roadmap (Backlog: needs real hardware pool) | ➖ Partial | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No |
| Foldables (posture + rotate verbs) | ➖ Partial | ✅ Yes | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No |
| Real physical devices | ❌ No | ➖ Partial | ❌ No | ✅ Yes | ❌ No | ❌ No | ❌ No | ✅ Yes | ✅ Yes | ✅ Yes |
| CLI with device verbs (tap, type, screenshot) | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ✅ Yes | ❌ No | ❌ No | ❌ No | ➖ Partial |
| Agent skill file (skill install) | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ✅ Yes | ❌ No | ❌ No | ➖ Partial | ❌ No |
| MCP server | ✅ Yes | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ❌ No | ✅ Yes |
| UI / accessibility tree reads | ✅ Yes | ✅ Yes | ✅ Yes | ➖ Partial | ❌ No | ✅ Yes | ➖ Partial | ❌ No | ➖ Partial | ➖ Partial |
| Batch actions (ms precision) | ➖ Partial | ➖ Partial | ❌ No | ❌ No | ❌ No | ✅ Yes | ❌ No | ❌ No | ❌ No | ❌ No |
| Session video recording | ✅ Yes | ➖ Partial | ➖ Partial | ✅ Yes | ❌ No | ✅ Yes | ➖ Partial | ✅ Yes | ✅ Yes | ✅ Yes |
| App logs + network capture | ✅ Yes | ➖ Partial | ➖ Partial | ✅ Yes | ➖ Partial | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| Live share links (no login needed) | ➖ Partial | ✅ Yes | ➖ Partial | ❌ No | ➖ Partial | ✅ Yes | ✅ Yes | ❌ No | ➖ Partial | ❌ No |
| Embeddable player (iframe / SDK) | 🛣️ Roadmap (No iframe/SDK implemented) | ✅ Yes | ❌ No | ❌ No | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ❌ No | ❌ No |
| Live collaboration + human takeover | 🛣️ Roadmap (Not implemented) | ✅ Yes | ➖ Partial | ➖ Partial | ❌ No | ➖ Partial | ➖ Partial | ❌ No | ➖ Partial | ➖ Partial |
| Record once, replay everywhere | ✅ Yes | ✅ Yes | ➖ Partial | ❌ No | ➖ Partial | ❌ No | ✅ Yes | ❌ No | ➖ Partial | ❌ No |
| Visual regression per step | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes | ❌ No | ❌ No | ➖ Partial | ❌ No | ✅ Yes | ✅ Yes |
| AI / plain-English testing | ✅ Yes | ✅ Yes | ❌ No | ➖ Partial | ❌ No | ❌ No | ❌ No | ❌ No | ✅ Yes | ➖ Partial |
| CI runner + GitHub checks | 🛣️ Roadmap (Actions + checks) | ➖ Partial | ➖ Partial | ✅ Yes | ✅ Yes | ➖ Partial | ➖ Partial | ✅ Yes | ✅ Yes | ✅ Yes |
| Self-hostable (BYO Mac minis) | ✅ Yes | ❌ No | ❌ No | ❌ No | ➖ Partial | ❌ No | ❌ No | ❌ No | ➖ Partial | ❌ No |
| Source-available (FSL → Apache-2.0) | ✅ Yes | ❌ No | ❌ No | ❌ No | ➖ Partial | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No |
| Public pricing + free tier, no card | ✅ Yes | ✅ Yes | ➖ Partial | ➖ Partial | ✅ Yes | ❌ No | ➖ Partial | ➖ Partial | ➖ Partial | ❌ No |
| Per-instance sandbox tokens + region pin | ✅ Yes | ➖ Partial | ❌ No | ➖ Partial | ❌ No | ✅ Yes | ❌ No | ❌ No | ❌ No | ➖ Partial |

Legend: ✅ Yes · 🛣️ Roadmap = public, lands in the open · ➖ Partial · ❌ No.
</details>
<!-- COMPARISON-END -->

Verdict: copy Limrun's IA (agent-first, control/data plane split, per-instance tokens), steal VibeView's live device embed + pricing transparency + TV/foldable/testing sections. Beat both on self-host + open spec.

Regenerate the table above after editing cells: `cd web && bun scripts/readme-table.ts` (single source of truth: `web/apps/web/src/data/comparison.ts`).

## Web design decision

- Structure from Limrun: hero with CLI snippet + MCP/skill/llms.txt above fold, docs/console/demo CTAs, YC-style minimal dev-infra.
- Sections from VibeView: live sandbox hero image, platform tabs (iOS/Android/TV/Roku), Build→Preview→Test loop, agent terminal shot, pricing table, FAQ.
- Stack: official `create-better-t-stack` (Bun + TanStack Start + Hono + tRPC + Better Auth + Drizzle Postgres + Tailwind + shadcn + Turborepo + Fumadocs) — not the 71-star fork. Type-safe end-to-end, `web/` in this repo.

## Self-hosting? Yes.

Mac fleet cost is the moat. Teams with office Mac minis must self-host; closed competitors can't serve them. FSL needs a deploy story for adoption → paid hosted conversion.
- `docker compose -f docker-compose.selfhost.yml up` → control-plane + postgres + minio + Linux emulator.
- BYO Mac: `sidekick host join` runs build-daemon + sim-daemon (Tart VMs, warm NVMe). No K8s required for MVP; K8s controller optional at scale.

## Layout

```
web/               # better-t-stack: marketing + console + tRPC API + docs (fumadocs)
cmd/sidekick/           # CLI (Go, cobra)
control-plane/     # Go scheduler + REST (builds/sims/emus) + JWT/WireGuard
build-daemon/      # Mac: xcodebuild + warm cache + SSE logs
sim-daemon/        # Mac: simctl + ScreenCaptureKit->LiveKit + idb AX tree
adb-proxy/         # Linux + client adb tunnel
skill/SKILL.md     # agent skill
crds/claims.yaml   # SimulatorClaim/BuildClaim/EmulatorClaim
deploy/            # docker + k8s + ansible for Mac minis
```

## Quickstart

```bash
npm i -g sidekick # or brew install ImBIOS/tap/sidekick
sidekick login # or VIBEVIEW-style: export SIDEKICK_TOKEN=...
sidekick xcode build . --scheme MyApp
sidekick ios create --device "iPhone 16, iOS 18.2"
sidekick ios act sim_abc --tap @e5
```

Self-host: `docker compose -f docker-compose.selfhost.yml up -d` then join a Mac.

## License

Apache-2.0 for the distribution surface, FSL-1.1-ALv2 for the hosted infra. All in this repo, no closed `pro/` module.

- `cmd/sidekick/`, `skill/SKILL.md`, `crds/` — Apache-2.0, see `LICENSE-APACHE.md`. Copy them anywhere, agents included.
- `control-plane/`, `build-daemon/`, `sim-daemon/`, `adb-proxy/`, `deploy/`, `docker-compose.selfhost.yml`, `web/` — FSL-1.1-ALv2, converts to Apache-2.0 on 2nd anniversary per version. Self-host and internal use are fine, competing SaaS is not. See `LICENSE.md`.
