FROM --platform=linux/amd64 us-docker.pkg.dev/android-emulator-helpers/images/emulator:latest
# KVM required on host. Exposes grpc 8550 + adb 5555; control-plane proxies both.
CMD ["emulator", "-avd", "Pixel_7_API_34", "-no-window", "-no-audio", "-gpu", "swiftshader_indirect", "-no-snapshot", "-grpc", "8550"]
