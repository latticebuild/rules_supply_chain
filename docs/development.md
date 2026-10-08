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

On Windows, use a temporary root with its canonical long path. Vite rejects 8.3
aliases in served paths. CI selects LOCALAPPDATA/Temp/latticebuild before dependency preparation and
forwards TMP/TEMP through Bazel tests; private runtime trees remain inside that
root. Keep this path out of installed source and dependency directories.
