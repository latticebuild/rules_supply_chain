# Development

Install Mise, then run `mise trust` and `mise run bootstrap` in this checkout.
Use the owned MODULE configuration and frozen dependency locks.

```sh
hk check --all --slow
hk validate
hk test
bazel build //:artifacts
bazel test //:test
```

On Linux and macOS also run `bazel test //:race_test`.

Native CI runs these gates on Ubuntu 24.04, macOS 27, and Windows 2025.
See [usage.md](usage.md) for setup and supported inputs.

On Windows, CI creates LOCALAPPDATA/Temp/latticebuild before Mise installs
tools. This uses a canonical long path on the installation drive and forwards
TMP/TEMP through Bazel tests. Private runtime trees remain inside that root.

Rust examples use the downloaded toolchain sysroot. The generated sysroot mode
in rules_rust 0.74.0 passes an execroot-relative path to Windows rustdoc tests;
the downloaded sysroot lets rustdoc locate the same standard libraries from
its runfiles executable. Every doctest still compiles, links and runs.
