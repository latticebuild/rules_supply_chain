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

Native CI runs these gates on Ubuntu 24.04, macOS 15, and Windows 2025.
See [usage.md](usage.md) for setup and supported inputs.
