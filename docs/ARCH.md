# Sidekick architecture

Source of truth: this repo at `d669b92` (18 Oct 2026). Anything here that
isn't backed by code or a test is marked **[unwitnessed]**.

## Control plane vs data plane

```
agents/CI ──HTTPS──> control-plane (:8080) ──HTTPS──> fleet daemons
  sidekick CLI          Go, stateless REST        build-daemon (Mac)
  MCP server            JWT/WireGuard edge        sim-daemon (Mac)
  skill file            Postgres (durable)        emu-host (Linux KVM)
                        in-memory Store (dev)
```

- **Control plane** (`control-plane/`): REST API only. Agents create
  builds/sims/emus, queue verbs, read trees/screenshots/logs/videos.
  Fleet daemons poll `POST /v1/host/claims/next` and post results back.
  The control plane never executes device verbs — it queues; daemons drain.
- **Data plane**: `build-daemon` runs `xcodebuild` on Mac minis;
  `sim-daemon` drives `simctl`+`idb` and posts AX trees/PNGs/MP4s;
  `emu-host` boots AVDs on Linux KVM and reports over the adb tunnel
  (`adb-proxy` lets agents use plain `adb` against the tunnel).
- **Scheduler** (`control-plane/devices.go` `ClaimNext`): oldest actionable
  claim wins — queued builds, then provisioning sims/emus, then pending
  sim-actions. `kinds` filters per daemon. Pinned resources
  (`region` set) are only claimable in that region; unpinned resources go
  to whoever claims first and **stick** to that region.
  Measured (in-memory Store, 8-core Linux, `go test -race`, Oct 2026):
  `NewStore` ~1µs; `ClaimNext` p50 ~0.8ms / p95 ~1.0ms / max ~1.6–1.8ms
  over 5,000 queued builds. Cost is O(queue) per claim (linear scan) —
  fine at hundreds queued; add an index or move to `PGStore` past ~10k.
- **Durable scheduler** (`control-plane/pgstore.go`): same method set on
  Postgres. `ClaimNext` uses `SELECT … FOR UPDATE SKIP LOCKED` so
  concurrent hosts in one pool never double-claim. Proved by shape in
  `pgstore_test.go` (pgxmock) — **[unwitnessed]** against live Postgres.

## Tokens

Two key kinds, enforced in `control-plane/tokens.go` (`withAuthz`):

- **Org keys** (`SIDEKICK_API_KEYS=org:key,…`): full access, including
  `POST /v1/tokens`. Verified per request (static map; `CONTROL_PLANE_SECRET`
  delegates to web in hosted mode — see `auth.go`).
- **Instance tokens** (`sk_inst_…`): minted per sandboxed agent
  (`sidekick token create --name ci-4821 --scope sims,emus --ttl 2h`),
  scoped to `builds|sims|emus`, TTL-capped at 720h. Only the sha256 hash is
  stored; the secret shows once. Reads need any valid token; writes need
  the matching scope; `/v1/tokens/*` rejects instance tokens outright.
  Revocation is immediate (`Revoked=true`).
- **Fleet token** (`SIDEKICK_HOST_TOKEN`): shared bearer for
  `/v1/host/*` only. Empty value rejects all host traffic — the operator
  must opt in. Org keys are rejected on host endpoints and vice versa
  (`host_test.go` `TestHostAuth`).

## Tenant isolation

Every store lookup is org-scoped: `GetBuild/GetSim/GetEmu/Delete…` return
`(nil,false)` → 404 for foreign orgs, so org B can't tell org A's resource
IDs exist. Instance tokens inherit their org; hosts are trusted
infrastructure and address resources by ID across orgs (narrow
claim/result calls only). Tests: `TestOrgIsolation`, cross-org tree 404 in
`TestSimProvisionAndVerbs`, token scope matrix in `TestTokenLifecycle`.

## K8s operator (optional at scale)

No K8s required for MVP (compose + `host join` is the default path).
At scale: `crds/claims.yaml` defines `SimulatorClaim/BuildClaim/EmulatorClaim`
(CRD + status subresource + TTL). `cmd/controller` watches them via a
dynamic informer, `controller.Reconciler` creates the control-plane
resource, mirrors state into `.status` (`Pending → Provisioning → Running →
Succeeded/Failed`, `Expired` on TTL, default 2h), and releases sims/emus on
expiry. Reconcile is reentrant/idempotent; out-of-band deletes fail the
claim instead of recreating spend. Tests: `controller/*_test.go` incl. a
fake-dynamic-client watch→status roundtrip, all `-race` clean.
`deploy/controller.yaml` has RBAC + Deployment.
**[Unwitnessed]**: never applied to a live cluster (no cluster in CI);
manifests are parse-checked only.

## Observability / on-call notes

- **Logs**: control-plane logs to stdout (`control-plane`, `emu-host`
  `> *.log` convention); daemons stream device logs via
  `POST /v1/host/*/logs`, agents read `sidekick ios logs --tail N`.
  Build logs stream as SSE (`/v1/builds/{id}/logs`) with `?format=json`
  polling fallback (what the CLI `--wait` uses).
- **What pages**: queue depth growing (`GET /v1/builds`, `/v1/sims`,
  `/v1/emus` stuck in `queued`/`provisioning`) = fleet too small or
  daemons wedged; check daemon claim errors first. `dmesg` OOM lines on
  emu hosts = brownout (the android-e2e job samples vitals for exactly
  this — `free`, top mem, OOM count — see `scripts/android-e2e.sh`).
- **Known failure modes**:
  - Emulator without usable KVM: 16-min boot, wedged accessibility stack,
    host OOM-kills our own shell (exit 137). Gate: `test -r/-w /dev/kvm`
    before booting (CI does this in seconds).
  - Fresh AVDs run background dexopt for minutes and starve uiautomator
    (LMK-kill). CI cancels it (`cmd package bg-dexopt-job --cancel`).
    Dogfood rigs should bake dexopt into the image instead.
  - `sim-daemon` keeps UDID map in memory: a host restart orphans sims
    (actions resolve to "host restarted; recreate the sim"). Reboot
    recovery is a backlog item — persist the map.
  - Builds: `execRunner` shells `xcodebuild` with a warm DerivedData path;
    only `fakeRunner` is CI-tested (no Mac in CI). First Mac-fleet boot
    will surface the real failure modes (signing, derived-data size).
- **Caps**: tree JSON 1MB, screenshots 4MB, recordings 32MB
  (`control-plane/host.go`). Instance TTL max 720h. Recording MP4 is
  posted whole (no chunked upload) — fine at 32MB, revisit for long sessions.

## Honest gaps vs the marketing table

See `web/apps/web/src/data/comparison.ts` — every `sidekick:"yes"` there
is backed by a unit test or `scripts/android-e2e.sh`. Everything else is
`roadmap`/`partial` with the reason in `sidekickNote`. The short version:
iOS/Xcode/TV/sharing/embed/collab need a Mac fleet and media servers that
don't exist yet; Gradle/signing have no code; posture is unit-only;
self-host compose-up isn't CI-run. `docs/GAPS.md` maps each to the Limrun
job req.
