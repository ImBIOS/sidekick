#!/usr/bin/env bash
# Android end-to-end (run INSIDE reactivecircus/android-emulator-runner).
# Single file on purpose: the action executes script lines in separate
# shells, so all state (EID, loops, set -eu) must live in one process.
set -eu
export SIDEKICK_URL=http://localhost:8080 SIDEKICK_TOKEN=ci-key
BIN="$GITHUB_WORKSPACE/bin"
# emu-host and the CLI shell `adb`; the runner does not guarantee it on PATH.
for d in "$ANDROID_HOME" "$ANDROID_SDK_ROOT" /usr/local/lib/android/sdk /opt/android-sdk; do
  if [ -n "$d" ] && [ -x "$d/platform-tools/adb" ]; then export PATH="$d/platform-tools:$d/emulator:$PATH"; break; fi
done
command -v adb
adb version | head -1

SIDEKICK_API_KEYS="org_ci:ci-key" SIDEKICK_HOST_TOKEN=ci-host PORT=8080 "$BIN/control-plane" > /tmp/cp.log 2>&1 &
for i in $(seq 1 30); do
  curl -sf -H "Authorization: Bearer ci-key" http://localhost:8080/v1/whoami && break
  sleep 2
done
curl -sf -H "Authorization: Bearer ci-key" http://localhost:8080/v1/whoami

SIDEKICK_POOL=linux-pool EMU_ADB_ADDR=127.0.0.1 "$BIN/emu-host" --control http://localhost:8080 --token ci-host --adopt --interval 10s > /tmp/emu-host.log 2>&1 &
EID=$("$BIN/sidekick" android create --profile Pixel_7_API_34 | python3 -c "import json,sys; print(json.JSONDecoder().raw_decode(sys.stdin.read())[0]['id'])")
test -n "$EID"
echo "EID=$EID"

for i in $(seq 1 60); do
  STATUS=$("$BIN/sidekick" android get "$EID" | python3 -c "import json,sys; print(json.load(sys.stdin)['status'])")
  echo "emu status: $STATUS"
  if [ "$STATUS" = "running" ]; then break; fi
  sleep 10
done
[ "$STATUS" = "running" ]
echo "--- host memory ---"; free -m | head -2
echo "--- device memory ---"; adb -s emulator-5554 shell cat /proc/meminfo | head -3
echo "--- focused window ---"; adb -s emulator-5554 shell dumpsys window windows | grep -i mCurrentFocus | head -2 || true
# Background vitals sampler: if something gets SIGKILLed later, this log
# shows what grew. Harmless if it dies with the shell.
( for i in $(seq 1 40); do echo "== $(date -u +%T) =="; free -m | head -2; ps aux --sort=-%mem 2>/dev/null | head -6 | awk '{print $4"% "$11}'; dmesg 2>/dev/null | grep -ci "out of memory"; sleep 30; done > /tmp/vitals.log 2>&1 & )
# Fresh boots run background dexopt for many minutes; it starves big
# transient processes (uiautomator gets LMK-killed). Cancel it: the image
# is throwaway CI state, not a battery-powered phone.
adb -s emulator-5554 shell cmd package bg-dexopt-job --cancel || true
echo "--- direct uiautomator probe (bypass Go wrapper, non-fatal) ---"
adb -s emulator-5554 shell uiautomator dump /sdcard/probe.xml || echo "probe failed, continuing"
adb -s emulator-5554 shell ls -l /sdcard/probe.xml || true
adb -s emulator-5554 shell dumpsys power | grep -iE "wakefulness|screenon" | head -3 || true

# Tree/screenshot arrive on the host refresh loop; poll until present.
# Attempts stay visible: a silent tree is the failure we are hunting.
for i in $(seq 1 18); do
  echo "tree attempt $i"
  if "$BIN/sidekick" android ui-tree "$EID" > tree.json; then break; fi
  if [ $((i % 6)) -eq 0 ]; then echo "--- emu-host log ---"; tail -10 /tmp/emu-host.log || true; fi
  sleep 10
done
grep -q "@e" tree.json
for i in $(seq 1 18); do
  "$BIN/sidekick" android screenshot "$EID" --out shot.png 2>/dev/null && break
  sleep 10
done
test -s shot.png
"$BIN/sidekick" android logs "$EID" --tail 5 | tee logs.txt
test -s logs.txt
"$BIN/sidekick" android record "$EID" --out demo.mp4 --duration 20
test -s demo.mp4
"$BIN/sidekick" android netcap --out cap.pcap --duration 15 --adb-serial emulator-5554
test -s cap.pcap
"$BIN/sidekick" android rotate "$EID" 90 --adb-serial emulator-5554
"$BIN/sidekick" android rotate "$EID" 0 --adb-serial emulator-5554
REF=$(python3 -c "import json; d=json.load(open('tree.json')); print(d['nodes'][0]['children'][0]['ref'] if d['nodes'][0].get('children') else d['nodes'][0]['ref'])")
echo "tapping $REF"
"$BIN/sidekick" test record --out t.json tap "$REF"
"$BIN/sidekick" test replay --in t.json --emu "$EID" --adb-serial emulator-5554 --threshold 0.5
"$BIN/sidekick" test diff shot.png shot.png --threshold 0.01
"$BIN/sidekick" android delete "$EID"
echo "--- emu-host log ---"
tail -20 /tmp/emu-host.log || true
echo "--- vitals ---"
tail -40 /tmp/vitals.log || true
echo "--- host OOM check ---"
dmesg 2>/dev/null | grep -i "out of memory\|killed process" | tail -3 || true
echo "E2E GREEN"
