# OpenLim — source-available Limrun alternative (FSL-1.1-ALv2)

Remote Xcode, iOS simulators, Android emulators for cloud agents. Your agent in E2B/Daytona/Docker builds mobile apps with no Mac.

`lim xcode build .` → stream logs → `lim ios create` → tap/type/screenshot → preview link + demo video on PR. Same loop as Limrun, TV/Roku breadth borrowed from VibeView, self-hostable.

## Why

Coding agents moved to Linux cloud sandboxes. No Xcode, no simulator, no emulator, no GPU Unity. OpenLim exposes them as remote services: one `lim` CLI + skill file, no sandbox change.

## Limrun vs VibeView — what we copied, what we do better

| | Limrun | VibeView | OpenLim |
|---|---|---|---|
| Xcode remote build | ✅ `lim xcode build`, warm S3 cache, idle/build split pricing | ✅ `build --cloud` RN only, $1.50 iOS / $0.75 Android | ✅ Limrun model (any Xcode + Gradle sandbox), warm NVMe + sccache, self-host BYO Mac |
| iOS / Android | ✅ sim + emu, AX tree, 60fps record, batch ms actions, adb tunnel | ✅ + Apple TV, Android TV, Roku HW, foldables (iPhone Duo posture/rotate) | ✅ both, plus TV verbs (`tap-focused`, d-pad) + foldable posture from VibeView |
| Agent UX | ✅ `lim` npm/brew, per-instance MCP (HTTP+OAuth), skill index, llms.txt, openapi | ✅ `vibeview agent-setup`, skill git + MCP stdio, `.vibeview/` state, CLAUDE.md nudge | ✅ both: `lim` + skill + MCP stdio/HTTP, `llms.txt`, `/openapi.json` |
| Human UX | preview links, demo video, `<RemoteControl/>` embed | ✅ better: share links, live collab guests, iframe embed, record-replay, visual regression, AI testing, signing + TestFlight/Play submit | ✅ Limrun infra + VibeView human layer: share/embed/collab, signing/submit, record-replay + visual diff |
| Platforms | 2 (iOS, Android) | 5 (iOS, Android, tvOS, Android TV, Roku) | 5, Roku via real HW or scrcpy bridge |
| Pricing | usage (idle+build) | sub + metered: $0/$19/$49/$249, $0.04-0.06/min | same shape, cheaper self-host: $0 self-host, cloud mirrors VibeView tiers |
| License | closed | closed | FSL-1.1-ALv2 → Apache-2.0 after 2y |

Verdict: copy Limrun's IA (agent-first, control/data plane split, per-instance tokens), steal VibeView's live device embed + pricing transparency + TV/foldable/testing sections. Beat both on self-host + open spec.

## Web design decision

- Structure from Limrun: hero with CLI snippet + MCP/skill/llms.txt above fold, docs/console/demo CTAs, YC-style minimal dev-infra.
- Sections from VibeView: live sandbox hero image, platform tabs (iOS/Android/TV/Roku), Build→Preview→Test loop, agent terminal shot, pricing table, FAQ.
- Stack: official `create-better-t-stack` (Bun + TanStack Start + Hono + tRPC + Better Auth + Drizzle Postgres + Tailwind + shadcn + Turborepo + Fumadocs) — not the 71-star fork. Type-safe end-to-end, `web/` in this repo.

## Self-hosting? Yes.

Mac fleet cost is the moat. Teams with office Mac minis must self-host; closed competitors can't serve them. FSL needs a deploy story for adoption → paid hosted conversion.
- `docker compose -f docker-compose.selfhost.yml up` → control-plane + postgres + minio + Linux emulator.
- BYO Mac: `lim host join` runs build-daemon + sim-daemon (Tart VMs, warm NVMe). No K8s required for MVP; K8s controller optional at scale.

## Layout

```
web/               # better-t-stack: marketing + console + tRPC API + docs (fumadocs)
cmd/lim/           # CLI (Go, cobra)
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
npm i -g openlim # or brew install ImBIOS/tap/lim
lim login # or VIBEVIEW-style: export OPENLIM_TOKEN=...
lim xcode build . --scheme MyApp
lim ios create --device "iPhone 16, iOS 18.2"
lim ios act sim_abc --tap @e5
```

Self-host: `docker compose -f docker-compose.selfhost.yml up -d` then join a Mac.

## License

FSL-1.1-ALv2, converts to Apache-2.0 on 2nd anniversary per version. No competing-use SaaS. See LICENSE.md.
