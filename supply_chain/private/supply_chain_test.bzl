"""Collect package metadata, record a policy verdict, and bind its replay test."""

load("@io_bazel_rules_go//go:def.bzl", "GoInfo", "go_context", "go_rule", "new_go_info")
load("@package_metadata//providers:package_metadata_info.bzl", "PackageMetadataInfo")

visibility("//...")

_PackagesInfo = provider(
    doc = "The package_metadata a target or its filegroup srcs provide.",
    fields = {
        "files": "Depset of the metadata files, their attribute files and licence texts.",
        "metadata": "Depset of each package's metadata JSON file.",
    },
)

def _packages_aspect_impl(target, ctx):
    metadata = []
    files = []
    if PackageMetadataInfo in target:
        info = target[PackageMetadataInfo]
        metadata.append(depset([info.metadata]))
        files.append(info.files)
    for src in getattr(ctx.rule.attr, "srcs", []):
        if _PackagesInfo in src:
            metadata.append(src[_PackagesInfo].metadata)
            files.append(src[_PackagesInfo].files)
    return [_PackagesInfo(metadata = depset(transitive = metadata), files = depset(transitive = files))]

_packages_aspect = aspect(
    implementation = _packages_aspect_impl,
    attr_aspects = ["srcs"],
    doc = "Collects the package_metadata of a target and of its filegroup srcs.",
)

def _pure_replay_impl(_settings, _attr):
    return {
        "@io_bazel_rules_go//go/config:pure": True,
        "@io_bazel_rules_go//go/config:race": False,
        "@io_bazel_rules_go//go/config:msan": False,
    }

_pure_replay = transition(
    implementation = _pure_replay_impl,
    inputs = [],
    outputs = [
        "@io_bazel_rules_go//go/config:pure",
        "@io_bazel_rules_go//go/config:race",
        "@io_bazel_rules_go//go/config:msan",
    ],
)

_ReplayContextInfo = provider(
    doc = "Pure Go context data selected for the test execution platform.",
    fields = {"context": "The pure Go context data on the test execution platform."},
)

def _replay_source_impl(ctx):
    return [ctx.attr.source[GoInfo], _ReplayContextInfo(context = ctx.attr._go_context_data)]

replay_source = rule(
    implementation = _replay_source_impl,
    cfg = _pure_replay,
    attrs = {
        "source": attr.label(providers = [GoInfo], mandatory = True),
        "_go_context_data": attr.label(default = Label("@io_bazel_rules_go//:go_context_data")),
        "_allowlist_function_transition": attr.label(default = Label("@bazel_tools//tools/allowlists/function_transition_allowlist")),
    },
    provides = [GoInfo, _ReplayContextInfo],
)

def _supply_chain_test_impl(ctx):
    sources = []
    files = []
    for target in ctx.attr.packages:
        packages = target[_PackagesInfo]
        sources.append({
            # Main-repository labels print without Bazel's canonical `@@`.
            "label": str(target.label) if target.label.repo_name else str(target.label).removeprefix("@@"),
            "metadata": [file.path for file in packages.metadata.to_list()],
        })
        files.append(packages.files)
    manifest = ctx.actions.declare_file(ctx.label.name + ".manifest.json")
    ctx.actions.write(manifest, json.encode({
        "advisories": ctx.file.advisories.path,
        "policy": ctx.file.policy.path,
        "sources": sources,
    }))
    report = ctx.actions.declare_file(ctx.label.name + ".verdict/report.txt")
    status = ctx.actions.declare_file(ctx.label.name + ".verdict/status")
    ctx.actions.run(
        executable = ctx.executable._check,
        arguments = ["--manifest", manifest.path, "--report", report.path, "--status", status.path],
        inputs = depset([manifest, ctx.file.advisories, ctx.file.policy], transitive = files),
        outputs = [report, status],
        mnemonic = "SupplyChainCheck",
        progress_message = "Checking the supply chain for %{label}",
    )

    return replay_verdict(ctx, report, status)

def replay_verdict(ctx, report, status):
    """Compiles a report and status into a native executable on the test platform.

    Args:
        ctx: Rule context with REPLAY_ATTRIBUTES and the Go rule toolchain.
        report: Report File beside embed.go in the target's verdict directory.
        status: Recorded status File beside embed.go in that directory.

    Returns:
        Executable DefaultInfo and compilation, analyzer and fix output groups.
    """

    # The verdict is part of the native test executable; replay never extracts it.
    generated = ctx.actions.declare_file(ctx.label.name + ".verdict/embed.go")
    ctx.actions.write(generated, """package main
import _ "embed"
//go:embed report.txt
var embeddedReport []byte
//go:embed status
var embeddedStatus []byte
func init() { embeddedVerdict = &verdictData{report: embeddedReport, status: embeddedStatus} }
""")
    replay = ctx.attr._replay[GoInfo]
    go = go_context(
        ctx,
        embed = [ctx.attr._replay],
        go_context_data = ctx.attr._replay[_ReplayContextInfo].context,
        goos = replay.mode.goos,
        goarch = replay.mode.goarch,
        maybe_needs_cc_toolchain = False,
    )

    if go.mode != replay.mode:
        fail("replay source and test compiler modes must match")

    def verdict_inputs(_go, _attr, source, _merge):
        source["embedsrcs"].extend([report, status])

    source = new_go_info(
        go,
        struct(embed = [ctx.attr._replay]),
        generated_srcs = [generated],
        resolver = verdict_inputs,
        importable = False,
        is_main = True,
    )
    archive, executable, runfiles = go.binary(go, name = ctx.label.name, source = source)
    return [
        DefaultInfo(executable = executable, runfiles = runfiles),
        OutputGroupInfo(
            compilation_outputs = depset([archive.data.file]),
            _validation = depset([archive.data._validation_output] if archive.data._validation_output else []),
            nogo_fix = depset([archive.data._nogo_diagnostics] if archive.data._nogo_diagnostics else []),
        ),
    ]

REPLAY_ATTRIBUTES = {
    "_replay": attr.label(
        default = Label("//supply_chain/private/tools/replay-verdict:source_context"),
        providers = [GoInfo],
        cfg = config.exec("test"),
    ),
    "_nogo": attr.label(default = Label("@io_bazel_rules_nogo//:nogo"), cfg = "exec"),
}

supply_chain_test = go_rule(
    implementation = _supply_chain_test_impl,
    doc = """Checks the licence, advisories and source of third-party packages.

    Every package must have an allowed licence, no advisory the policy does not
    ignore, and, for Cargo, an allowed registry or Git repository. Each
    `packages` label must contribute at least one package.
    """,
    test = True,
    attrs = REPLAY_ATTRIBUTES | {
        "advisories": attr.label(
            doc = "The `advisory_index` to check against.",
            allow_single_file = [".json"],
            mandatory = True,
        ),
        "packages": attr.label_list(
            doc = "Targets providing `PackageMetadataInfo`, directly or through a filegroup's `srcs`.",
            aspects = [_packages_aspect],
            allow_empty = False,
            mandatory = True,
        ),
        "policy": attr.label(
            doc = "The caller-owned policy JSON.",
            allow_single_file = [".json"],
            mandatory = True,
        ),
        "_check": attr.label(default = Label("//supply_chain/private/tools/check-packages"), executable = True, cfg = "exec"),
    },
)
