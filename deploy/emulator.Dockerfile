# SPDX-License-Identifier: FSL-1.1-ALv2
# Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
FROM --platform=linux/amd64 us-docker.pkg.dev/android-emulator-helpers/images/emulator:latest
# KVM required on host. Exposes grpc 8550 + adb 5555; control-plane proxies both.
CMD ["emulator", "-avd", "Pixel_7_API_34", "-no-window", "-no-audio", "-gpu", "swiftshader_indirect", "-no-snapshot", "-grpc", "8550"]
