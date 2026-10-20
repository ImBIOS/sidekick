# Gaps vs the Limrun job req — honest status (18 Oct 2026)

Hiring-signal mapping: Go · K8s · distributed systems · on-call ·
AI-assisted devices. Each row: what exists, how it's proved, what's missing.

## Go

- **Done**: control-plane scheduler (`ClaimNext` + region pin), scoped
  instance tokens, tenant isolation, fleet client, daemons — all unit-tested,
  `go vet` clean, `-race` clean (`go test -race ./control-plane/ ./cmd/...`).
- **Proved under load**: `control-plane/scheduler_concurrent_test.go` —
  200 builds × 16 racers claimed exactly once; 64 tokens minted/looked-up/
  revoked concurrently; token→create→regional-claim E2E over real HTTP
  (10 Tokyo-pinned + 10 unpinned × 8 racing hosts, no cross-region theft,
  no double-claim). Load: startup ~1µs, `ClaimNext` p50 ~0.8ms / p95
  ~1.0ms over 5k queued (O(n) scan — index past ~10k queued).
- **Missing**: `PGStore` proved by shape only (pgxmock); no live-Postgres
  soak, no multi-instance control-plane (single process, no leader election).

## K8s

- **Done**: `controller/` reconciler (create → mirror status → TTL release),
  `crds/claims.yaml` (3 CRDs + status subresource + examples),
  `deploy/controller.yaml` (RBAC + Deployment). Tests incl. fake
  dynamic-client watch→reconcile→status roundtrip, `-race` clean.
- **Missing**: never applied to a live cluster (no cluster in CI, no envtest).
  Controller patterns are demonstrated, not production-hardened (no leader
  election — Deployment `replicas: 1` is load-bearing; no metrics).

## Distributed systems (scheduler, tokens, region pin)

- **Done**: claim protocol (queue → claim → logs → finish/ready), idempotent
  reconcile, region stickiness, per-instance credentials with TTL+revocation.
- **Missing**: no fence against two control-planes (single-writer assumed);
  host heartbeats don't exist (a dead host's `running` stays `running`);
  sim UDID map is in-memory (restart orphans sims). These are the next three
  on-call pages waiting to happen — listed in ARCH.md, not hidden.

## On-call

- **Done**: ARCH.md documents caps, failure modes (KVM gate, dexopt,
  OOM vitals), and what queue-depth growth means. android-e2e samples
  vitals for SIGKILL forensics.
- **Missing**: no alerting, no dashboards, no SLOs, no runbook beyond ARCH.
  Never carried a pager for this system.

## AI-assisted devices (record/replay, visual diff, NL)

- **Done**: `test record/replay/diff` + `test nl` (parse+ground unit-tested);
  record/replay/diff run against a real emulator in `scripts/android-e2e.sh`.
- **Missing**: NL steps are not in the CI e2e path (unit-only); no
  flakiness numbers; iOS verbs are `fakeDevice` unit tests (no Mac).

## What was cut to get here (this session)

1. **iOS video loop** — cut: no Mac in this environment. The iOS path is
   contract-tested (fakeDevice) and CLI-complete, but `xcodebuild`/`simctl`
   never ran here. The recorded loop is Android (same scheduler/CLI/daemon
   machinery, real emulator on KVM).
2. **Live-cluster K8s proof** — cut: no cluster, no docker. Fake-client
   roundtrip + parse-checked manifests instead.
3. **Live-Postgres proof** — cut: no docker. pgxmock shapes + `SKIP LOCKED`
   SQL review instead; compose-up documented, not run.
4. **2-min edited video** — cut to artifacts: terminal transcript
   (`docs/demo/`), device screenshots, and the loop's own MP4 recording.
   No display server to screen-capture; no ffmpeg to render one.
5. **Multi-region fleet witness** — cut: region pin is unit+E2E proved over
   HTTP with fake regions; "2 GCP regions" was an unwitnessed claim and was
   removed from the comparison table.

## Stranger-run checklist (the bar)

- [ ] `docker compose -f docker-compose.selfhost.yml up` boots (needs docker;
      steps + failure modes in `docs/SELFHOST.md`)
- [ ] `sidekick host join` enrolls a Mac/Linux host (works today, §SELFHOST)
- [ ] Android loop: `android create → get → ui-tree → screenshot → record →
      delete` against one emu-host (proved 18 Oct 2026, transcript in
      `docs/demo/`)
- [ ] iOS loop: same verbs, needs a Mac — NOT stranger-runnable yet
- [ ] K8s claims: `kubectl apply -f crds/claims.yaml` + controller —
      manifests parse-checked, NOT cluster-applied yet
