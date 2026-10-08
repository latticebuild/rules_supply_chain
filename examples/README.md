# Runnable examples

Prepare the repository with `mise run bootstrap`, then run:

```sh
bazel build //examples:artifacts
bazel test //examples:test
```

| Feature | Source | Command | Expected result |
| --- | --- | --- | --- |
| Rust metadata and notice output groups | [rust/BUILD.bazel](rust/BUILD.bazel) | `bazel build //examples/rust:own_output //examples/rust:notice_output` | Separates the program license and third-party notices; includes linked/proc-macro deps and excludes build-only deps. |
| Pinned advisory index and caller policy | [rust/policy.json](rust/policy.json) | `bazel test //examples/rust:passing_test` | Checks declared metadata against licence, advisory and source policy. |
| npm policies and refusal cases | [../testdata/BUILD.bazel](../testdata/BUILD.bazel) | `bazel test //tests:check_test` | The wrapper requires meaningful rejection for invalid licensing, advisories and source policies. |
| Notice ordering and graph boundaries | [../tests/BUILD.bazel](../tests/BUILD.bazel) | `bazel test //tests:notices_test` | Checks real license and notice bytes, linked dependencies, and build-script exclusion. Producers require Linux/macOS. |
| Standalone verdict replay | [../tests/replay_test.go](../tests/replay_test.go) | `bazel test //tests:replay_test` | Runs accepted and refused verdicts, copies their executables away from runfiles, and rejects overrides on Linux, macOS and Windows. |

The root artifact/test gates include these examples. Deliberately invalid subjects
remain in test fixtures; their owner tests require the expected refusals. Fix and
editor-write commands modify the invoking checkout only when run explicitly.
Automated mutation cases use disposable invoking workspaces.
