# Self-hosting from a clean machine

Proved 28 Oct 2026 on Linux (KVM present, docker 29.9.2): `docker compose
-f docker-compose.selfhost.yml up -d --build control-plane` boots
control-plane + postgres healthy (image builds from the fixed Dockerfiles),
`curl /v1/whoami` returns the org, `sidekick host join` enrolls the box,
and the emu claim loop runs for real — `android create` → host `ClaimNext`
→ `HostEmuReady` → `android get` reports `running` → `ui-tree`, `screenshot`
(1.4MB PNG), `logs` all serve → `android share` mints a signed guest link
whose public page/JSON/shot/tree read back with no auth. Run 1 used the
prebuilt local `emu-host --adopt` against a T3 AVD booted on the host
(boot_completed=1 in ~50s); claim binds to pool `linux-pool`, region
`asia-southeast1` heartbeat shows in `GET /v1/fleet/regions`.

Deliberately NOT run here: the `emulator` service image build (base
`us-docker.pkg.dev/android-emulator-helpers/images/emulator:latest` needs
GCP auth this box doesn't have — treat as private-registry, bake your own
or run emu-host on the metal as §3 does), and `minio` (quay.io needs auth
from this network, so it is `profiles: ["blobs"]` opt-in; the plane ignores
S3 until artifact offload lands — blobs stay inline in Postgres).

## 0. Prerequisites

- One Linux box (4+ cores, 16GB+ RAM, KVM: `test -r /dev/kvm && test -w
  /dev/kvm`) for control-plane + postgres + minio + Linux emulator.
- Mac minis (optional, for iOS): macOS 14+, Xcode 16+, `idb`
  (`pip install idb`), Tart (`brew install cirruslabs/cli/tart`).
- `docker` + `docker compose` plugin, `openssl`, `sidekick` CLI
  (`go build -o sidekick ./cmd/sidekick`).

## 1. Secrets (once)

```bash
export SIDEKICK_HOST_TOKEN=$(openssl rand -hex 32)   # fleet bearer
export SIDEKICK_API_KEYS="org_prod:$(openssl rand -hex 16)"  # agent keys
echo "HOST_TOKEN=$SIDEKICK_HOST_TOKEN" >> .env.secrets   # never commit
```

Empty `SIDEKICK_HOST_TOKEN` rejects **all** host traffic (401) — the most
common first-boot confusion (`control-plane/host.go` `withHost`).

## 2. Boot the stack

```bash
docker compose -f docker-compose.selfhost.yml up -d
curl -sf -H "Authorization: Bearer <agent-key>" \
  http://localhost:8080/v1/whoami   # -> {"org_id":"org_prod"}
```

What comes up: `control-plane` (:8080), `postgres:16` (data in `pgdata`
volume). NOT by default: `minio` (opt-in `profiles: ["blobs"]` — plane
ignores S3, blobs stay inline in Postgres) and `emulator` (base image needs
GCP artifact-registry auth; run emu-host on the metal per §3 until your
registry hosts it). Set `EMU_ADB_ADDR` to the machine's LAN IP — agents are
usually remote and a docker-internal IP won't dial
(`docker-compose.selfhost.yml` comments).

What came up (`docker compose up -d --build control-plane`, 28 Oct 2026):
`t3-37f5b575-postgres-1` (Up, healthy via `pg_isready`),
`t3-37f5b575-control-plane-1` (Up, `:8080→8080`, log
`control-plane: durable scheduler (postgres)`).
Fixes this run forced (all landed): compose build context is now repo root
for control-plane (was `build: ./control-plane`, which broke `COPY
control-plane/`); `golang:1.24`→`1.26` in both Dockerfiles (go.mod says
1.26); minio moved to `profiles: ["blobs"]` opt-in + pinned tag + root
creds (quay.io 401s from restricted networks, docker.io/minio is gone, and
minio refused to boot without `MINIO_ROOT_USER/PASSWORD`); plane got
`depends_on: postgres (service_healthy)` + healthchecks so boot order is
deterministic; the dead `S3_ENDPOINT` env was removed (nothing reads it).

## 3. Join fleet hosts

On a box with an AVD (or inside the `emulator` service once its base image
pulls — needs GCP artifact-registry auth, see header):

On each Mac mini / Linux KVM box:

```bash
sidekick host join --control http://<control-ip>:8080 \
  --token $SIDEKICK_HOST_TOKEN --pool mac-pool [--region asia-southeast1]
# writes ~/.config/sidekick/host.json (0600); daemons read it, or
# SIDEKICK_CONTROL/SIDEKICK_HOST_TOKEN/SIDEKICK_POOL env, or flags.
```

Then run the daemons (one process each; they poll
`POST /v1/host/claims/next` every `--interval`; every poll refreshes the
pool+region heartbeat read back by `GET /v1/fleet/regions` /
`sidekick fleet regions`):

```bash
./build-daemon --control ... --token ... --pool mac-pool &   # Mac only
./sim-daemon   --control ... --token ... --pool mac-pool &   # Mac only
./emu-host     --control ... --token ... --pool linux-pool & # Linux KVM
# No AVD to boot (CI, dogfood rig with a running emulator): add --adopt
# to attach to the already-booted emulator-5554 instead of launching one.
```

## 4. Verify the loop

```bash
export SIDEKICK_URL=http://<control-ip>:8080 SIDEKICK_TOKEN=<agent-key>
sidekick android create --profile Pixel_7_API_34   # -> id, provisioning
sidekick android get <id>                          # poll to running
sidekick android ui-tree <id> | head               # @e refs
sidekick android screenshot <id> --out shot.png
sidekick android delete <id>
```

iOS is the same shape once a Mac has joined (`xcode build`, `ios create`,
`ios act --tap @e5`, `ios record --stop --out demo.mp4`).

Guest share links (login-free, proved 28 Oct 2026 against the local plane):
mint with the org key, open with nothing:

```bash
export SIDEKICK_URL=http://<control-ip>:8080 SIDEKICK_TOKEN=<agent-key>
sidekick android share <emu-id> --ttl 24h
# -> http://<control-ip>:8080/s/sk_share_... (+ share id + revoke hint)
sidekick android share <emu-id> --revoke <share-id>   # kills the link
sidekick ios share <sim-id> [--ttl 12h] [--revoke <share-id>]
```

`GET /s/:token` serves the snapshot page (screenshot auto-refreshed every
5s, tree excerpt, logs tail) plus `?format=json`; `/s/:token/screenshot`,
`/tree`, `/logs?tail=N`, `/video` (sims only) serve the raw blobs — all
with no `Authorization` header. TTL default 24h, max 720h. Revoked/expired/
forged tokens 404 uniformly. Instance (`sk_inst_`) keys get 403 on mint;
foreign orgs get 404 on mint/revoke (no oracle). NOT live video: no media
server exists in this system — the page shows host-posted snapshots. The
branded viewer lives at web route `/s/$token` (`?api=` points it at any
plane); the plane's own `/s/:token` HTML works standalone for self-host.

## Failure modes (code-grounded; rows marked 28 Oct 2026 were hit live this run)

| Symptom | Cause | Fix |
|---|---|---|
| `401 invalid host token` on daemon start | `SIDEKICK_HOST_TOKEN` empty/mismatched | same value in compose env **and** `host join --token` |
| `need control+token+pool` from daemon | no flags, no `host.json`, no env | run `host join` first (`internal/fleet/client.go` `Resolve`) |
| emu stuck `provisioning`, host log `no booted emulator` | no usable KVM | `test -r/-w /dev/kvm`; needs nested virt or bare metal |
| emu boots then uiautomator dies, tree 404s | background dexopt starves it | `adb shell cmd package bg-dexopt-job --cancel` (throwaway) or bake dexopt into image |
| shell exits 137 mid-run | host OOM (software-emulated Android) | give the box RAM, `-memory 3072`, watch `dmesg \| grep -i "out of memory"` |
| `record --stop` 404s video | daemon hasn't posted MP4 yet | poll `recording` until `has_video:true` |
| sim action hangs | host restarted, UDID map lost | recreate the sim (map is in-memory — known gap) |
| adb can't reach emulator | `EMU_ADB_ADDR` is docker-internal | set to host LAN IP, publish 5555 |
| `PUT /s/:token/...` etc 404 with `no screenshot yet` | host hasn't posted state | boot the emu/sim and wait a refresh interval; page shows placeholders until then |
| `minio` exits `MINIO_ROOT_USER not set` | fresh minio refuses keyless boot | fixed 28 Oct 2026: compose sets throwaway root creds |
| minio/postgres data lost | `docker compose down -v` | don't pass `-v`; `pgdata` volume holds state |

## What wasn't run here

The `emulator` service image build (base needs GCP artifact-registry auth;
see header) and hosted `minio` (quay.io 401s from restricted networks —
left `profiles: ["blobs"]` opt-in; the plane ignores S3, blobs stay inline
in Postgres). Everything else above ran for real on 28 Oct 2026.
