# Demo loop — Android on local KVM (18 Oct 2026, no mocks)

One working loop, every step executed for real on this machine:
control-plane + emu-host + CLI built from this repo at `ebc06d5`,
AVD `T3` (Android 14, x86_64) booted on `/dev/kvm`, host token
`demo-host`, org key `demo-key`. Timings: claim→boot ~4 min (cold AVD),
verbs seconds each.

## Build half (HTTP contract; execution needs a Mac)

```
$ sidekick xcode build . --scheme MyApp
syncing . -> control-plane, scheme=MyApp
build bld_20261018-3 (queued)
$ # Mac-daemon side, real HTTP:
$ curl -H "Bearer demo-host" POST /v1/host/claims/next {"pool":"mac-1","kinds":["build"]}
claimed: build bld_20261018-3
$ curl ... POST /v1/host/builds/bld_20261018-3/logs {"lines":["$ xcodebuild -scheme MyApp build","** BUILD SUCCEEDED **"]}
{"ok":true}
$ curl ... POST /v1/host/builds/bld_20261018-3/finish {"status":"succeeded"}
{"ok":true,"status":"succeeded"}
$ curl ... GET /v1/builds/bld_20261018-3/logs?format=json
{"lines":["[sidekick] queued; waiting for a build host","[sidekick] build claimed by mac-1",
 "$ xcodebuild -scheme MyApp build","** BUILD SUCCEEDED **"],"status":"succeeded"}
```

`xcodebuild` itself never ran here (no Mac). Everything around it —
queue, claim, log stream, terminal poll — did, over real HTTP with real
binaries. Full typescript: `build.typescript`.

## Device half (fully real, KVM)

```
$ sidekick android create --profile T3
{"id": "emu_20261018-2", "adb_target": "", "status": "provisioning"}
$ sidekick android get emu_20261018-2        # ...poll...
{"status": "running", "adb_target": "127.0.0.1:5555", ...}
$ sidekick android ui-tree emu_20261018-2    # -> tree.json, refs: 17 (@e1..@e17)
$ sidekick android screenshot emu_20261018-2 --out shot.png
saved 1374041 bytes to shot.png
$ sidekick android logs emu_20261018-2 --tail 5   # real logcat lines
$ sidekick android record emu_20261018-2 --out demo.mp4 --duration 20s --adb-serial emulator-5554
saved 122787 bytes to demo.mp4
$ sidekick android netcap --out cap.pcap --duration 15s --adb-serial emulator-5554
saved 4096 bytes to cap.pcap
$ sidekick android rotate emu_20261018-2 90  --adb-serial emulator-5554  # -> rotated to 90
$ sidekick android rotate emu_20261018-2 0   --adb-serial emulator-5554  # -> rotated to 0
$ sidekick test record --out t.json tap "@e2"
recorded 1 steps to t.json
$ sidekick test replay --in t.json --emu emu_20261018-2 --adb-serial emulator-5554 --threshold 0.5
passed 1 steps
$ sidekick test diff shot.png shot.png --threshold 0.01
diff 0.00% (threshold 1.00%)
$ sidekick android delete emu_20261018-2
deleted emu_20261018-2        # emus left: 0
```

Artifacts in this dir: `shot.png` (device screenshot), `demo.mp4`
(device recording — the "recorded mp4 on PR"), `tree.json` (17-ref UI
tree), `t.json` (recorded tap), transcripts (`*.typescript`).

## Bugs this loop caught

1. **Fleet 204 bug** (`internal/fleet/client.go`): idle `ClaimNext` polls
   (204 + empty body) decoded as empty claims — emu-host spun on phantom
   work. Fixed + regression-tested (`internal/fleet/client_test.go`).
   Before the fix the demo could not proceed past provisioning.

## What's still Mac-gated (not shown)

`ios create → ax-tree → act --tap → screenshot → record --stop → preview
link`: CLI + server contract unit-tested (`TestSimProvisionAndVerbs`),
never executed against `simctl`/`idb` here. The "preview link" (`webrtc_url`)
is plumbed but has no media server behind it — scaffolding, not a product.
