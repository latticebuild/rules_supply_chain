"""Collect package metadata, record a policy verdict, and bind its replay test."""

load("@package_metadata//providers:package_metadata_info.bzl", "PackageMetadataInfo")
load("@rules_bound//bound:defs.bzl", "BOUND_TOOLCHAIN_TYPE", "bound_context")

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
    report = ctx.actions.declare_file(ctx.label.name + ".report.txt")
    status = ctx.actions.declare_file(ctx.label.name + ".status")
    ctx.actions.run(
        executable = ctx.executable._check,
        arguments = ["--manifest", manifest.path, "--report", report.path, "--status", status.path],
        inputs = depset([manifest, ctx.file.advisories, ctx.file.policy], transitive = files),
        outputs = [report, status],
        mnemonic = "SupplyChainCheck",
        progress_message = "Checking the supply chain for %{label}",
    )

    # Replay is built for the test platform.
    result = bound_context(ctx).bind(ctx.attr._replay, bundle = "private", args = ["--report", report, "--status", status])
    return [
        DefaultInfo(executable = result.executable),
        RunEnvironmentInfo(environment = {"BOUND_CACHE": "0"}),
    ]

supply_chain_test = rule(
    implementation = _supply_chain_test_impl,
    doc = """Checks the licence, advisories and source of third-party packages.

    Every package must have an allowed licence, no advisory the policy does not
    ignore, and, for Cargo, an allowed registry or Git repository. Each
    `packages` label must contribute at least one package.
    """,
    test = True,
    attrs = {
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
        # Replay retains the existing test execution configuration.
        "_replay": attr.label(
            default = Label("//supply_chain/private/tools/replay-verdict"),
            executable = True,
            cfg = config.exec("test"),
        ),
    },
    toolchains = [BOUND_TOOLCHAIN_TYPE],
)
