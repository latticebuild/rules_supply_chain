# rules_supply_chain

[![CI](https://github.com/latticebuild/rules_supply_chain/actions/workflows/ci.yml/badge.svg)](https://github.com/latticebuild/rules_supply_chain/actions/workflows/ci.yml)
[![Bazel](https://img.shields.io/badge/Bazel-9.2.0-43A047?logo=bazel&logoColor=white)](MODULE.bazel)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Bazel rules for advisory indexing, dependency policy checks and third-party notices. Metadata, license policy and advisory inputs belong to the caller.

## Setup

Use Bazel 9.2 with Bzlmod. These source repositories have no registry release yet.
Pin your chosen revision in your root MODULE.bazel:

```starlark
bazel_dep(name = "latticebuild_supply_chain", version = "0.0.0")
git_override(
    module_name = "latticebuild_supply_chain",
    remote = "https://github.com/latticebuild/rules_supply_chain.git",
    commit = "FULL_COMMIT_SHA",
)
```

Replace FULL_COMMIT_SHA with the full commit hash of that revision. Copy the
Latticebuild dependency overrides from [MODULE.bazel](MODULE.bazel) into the
consuming root too; overrides declared by a dependency do not propagate.

## Usage

```starlark
load("@latticebuild_supply_chain//supply_chain:defs.bzl", "advisory_index", "supply_chain_test", "third_party_notices")
advisory_index(name = "advisories", sources = {":pinned_osv_files": "crates.io"})
supply_chain_test(name = "policy_test", packages = [":package_metadata"], policy = "policy.json", advisories = ":advisories")
third_party_notices(name = "notices", package = "my-program", title = "My program", program = ":program", policy = "policy.json")
filegroup(name = "program_license", srcs = [":notices"], output_group = "license")
filegroup(name = "dependency_notices", srcs = [":notices"], output_group = "notices")
```

See [docs/usage.md](docs/usage.md) for attributes, required tool inputs and
consumer setup. The public API lives in [supply_chain/](supply_chain/);
implementation files under its private/ directory are repository-local.

Policy and notice producers support Linux and macOS. Portable algorithm tests
and verdict replay also run on Windows. Policies select allowed licenses,
reviewed license texts, advisory exceptions and allowed source URLs. The notice
aspect follows Rust dependencies and proc-macro edges while excluding Cargo
build-script dependencies. Bundled SPDX data retains its upstream licensing
under [the checker’s assets](supply_chain/private/tools/check-packages/assets/spdx/).

<details>
<summary>Repository map</summary>

| Area | Location |
| --- | --- |
| Public API | [supply_chain/defs.bzl](supply_chain/defs.bzl) |
| Implementation | [supply_chain/private/](supply_chain/private/) |
| Examples and fixtures | [testdata/](testdata/) |
| Owning checks | [tests/](tests/) |
| Consumer guide | [docs/usage.md](docs/usage.md) |

</details>

## Development

Install [Mise](https://mise.jdx.dev/), then prepare this checkout:

```sh
mise trust
mise run bootstrap
hk validate
hk test
hk check --all --slow
bazel build //:artifacts
bazel test //:test
```

Tools and dependency versions are pinned in [mise.toml](mise.toml) and
[MODULE.bazel](MODULE.bazel). CI runs these gates on native Linux, macOS and
Windows runners. Repositories with a race suite also run it on Linux and macOS.
See [docs/development.md](docs/development.md) for owning checks and platform
constraints, and [ARCHITECTURE.md](ARCHITECTURE.md) for implementation decisions.

## License

[Apache License 2.0](LICENSE).
