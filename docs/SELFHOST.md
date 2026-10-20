# Self-hosting from a clean machine

Proved 18 Oct 2026 up to the docker boundary: control-plane boot,
`host join`, and the emu-host claim loop were run for real on Linux;
`docker compose up` itself was **not** run here (no docker daemon) — steps
below are exact from `docker-compose.selfhost.yml`, failure modes from the
code paths they hit.

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
volume), `minio` (:9000), `emulator` (privileged, `/dev/kvm`, adb :5555).
Set `EMU_ADB_ADDR` to the machine's LAN IP — agents are usually remote and
a docker-internal IP won't dial (`docker-compose.selfhost.yml` comments).

## 3. Join fleet hosts

On each Mac mini / Linux KVM box:

```bash
sidekick host join --control http://<control-ip>:8080 \
  --token $SIDEKICK_HOST_TOKEN --pool mac-pool [--region asia-southeast1]
# writes ~/.config/sidekick/host.json (0600); daemons read it, or
# SIDEKICK_CONTROL/SIDEKICK_HOST_TOKEN/SIDEKICK_POOL env, or flags.
```

Then run the daemons (one process each; they poll
`POST /v1/host/claims/next` every `--interval`):

```bash
./build-daemon --control ... --token ... --pool mac-pool &   # Mac only
./sim-daemon   --control ... --token ... --pool mac-pool &   # Mac only
./emu-host     --control ... --token ... --pool linux-pool & # Linux KVM
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

## Failure modes (code-grounded)

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
| minio/postgres data lost | `docker compose down -v` | don't pass `-v`; `pgdata` volume holds state |

## What wasn't run here

`docker compose up` end-to-end (no docker daemon on this box). Compose file
lint: services/ports/env verified by reading; `control-plane` image builds
from `control-plane/Dockerfile`, emulator from `deploy/emulator.Dockerfile`.
First stranger run will surface image-build issues — file them, don't ask.
