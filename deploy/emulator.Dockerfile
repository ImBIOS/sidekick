# SPDX-License-Identifier: FSL-1.1-ALv2
# Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
# Self-host Android fleet in one container: emu-host (Go) boots AVDs per
# claimed emu inside the emulator image, so adb is always localhost:5555.
# AVDs must exist in the image (bake with sdkmanager/avdmanager, e.g.
# Pixel_7_API_34) or be mounted in; emu-host fails claims loudly otherwise.
FROM golang:1.26 AS hostbuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY emu-host/ ./emu-host/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -o /emu-host ./emu-host

FROM --platform=linux/amd64 us-docker.pkg.dev/android-emulator-helpers/images/emulator:latest
# KVM required on host. emu-host reads SIDEKICK_CONTROL / SIDEKICK_HOST_TOKEN /
# SIDEKICK_POOL / EMU_ADB_ADDR from the environment (see
# docker-compose.selfhost.yml). Agents reach adb at $EMU_ADB_ADDR:5555.
COPY --from=hostbuild /emu-host /usr/local/bin/emu-host
ENTRYPOINT ["/usr/local/bin/emu-host"]
