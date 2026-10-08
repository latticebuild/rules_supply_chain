# Policy and notice rules

Pin `latticebuild_supply_chain` with an immutable `git_override` in your root
MODULE.bazel. This project has no BCR release yet. Load `advisory_index`,
`supply_chain_test`, and `third_party_notices` from `//supply_chain:defs.bzl`.

```starlark
load("@latticebuild_supply_chain//supply_chain:defs.bzl", "advisory_index", "supply_chain_test", "third_party_notices")
advisory_index(name = "advisories", sources = {":pinned_osv_files": "crates.io"})
supply_chain_test(name = "policy_test", packages = [":package_metadata"], policy = "policy.json", advisories = ":advisories")
third_party_notices(name = "notices", package = "my-program", title = "My program", program = ":program", policy = "policy.json")
filegroup(name = "program_license", srcs = [":notices"], output_group = "license")
filegroup(name = "dependency_notices", srcs = [":notices"], output_group = "notices")
```

Use package_metadata's public rules to declare package URLs, SPDX license kinds,
and license text. Each check input must contribute metadata, directly or through
filegroup `srcs`. Advisory source values select `crates.io` or `npm`.
`supply_chain_test` requires the advisory index explicitly.

The JSON policy controls allowed license expressions, reviewed license overrides
and text hashes, advisory exceptions, and allowed Git sources. Inspect
[testdata/policy.json](../testdata/policy.json) and the policy reader's tests for
the schema and refusal behavior. License overrides require a reason; reviewed
text hashes protect the selected license bytes.

Rust programs expose their metadata through `package_metadata` attributes. The
notice aspect follows `deps` and `proc_macro_deps`, excludes Cargo build-script
dependencies, and requires the named program package to have license text.
Output groups select the own license and ThirdPartyNotices.txt separately.

Producers run on Linux or macOS. The replay tool and portable Go libraries also
support Windows; a Windows test platform may replay a verdict produced on a
supported execution platform. All inputs remain caller-owned and declared.
