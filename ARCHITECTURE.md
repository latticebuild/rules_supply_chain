# Supply-chain checks and program notices

A build needs a verdict that reflects its declared packages, policy, and
advisory snapshot. The check reads package metadata and pinned OSV files in
Bazel actions. It performs no live advisory or registry lookup. The caller owns
the policy, package sources, and database updates.

The producer writes a report and status artifact even when policy rejects a
package. A native Go test executable embeds the report and status and replays them on
the test execution platform. It performs no runtime file extraction or temporary
directory creation and rejects arguments that could replace the recorded verdict. Keeping computation separate from replay supports remote execution
without turning a policy violation into an action failure that hides the report.

The notice aspect follows Rust deps and proc_macro_deps, then stops at Cargo
build scripts. This includes linked libraries and procedural macros while
excluding build-only packages. Package URLs define identity and stable ordering;
the writer preserves the program's own license bytes and emits the dependency
licenses selected by policy overrides.

Producer tools support Linux and macOS. Portable readers and replay tests also
run on Windows. The replay source and compiler use the same pure Go mode on the
test execution platform; consumer-configured Go analysis remains a build validation. The development Rust graph uses upstream rules only and checks
real metadata edges, output groups, and build-script exclusion. Bundled SPDX
data keeps its upstream license and attribution files.
