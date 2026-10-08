"""Collect the program dependency graph and write its licence artifacts."""

load("@package_metadata//providers:package_metadata_info.bzl", "PackageMetadataInfo")

visibility("//...")

_CratesInfo = provider(
    doc = "The package_metadata of a program's build graph.",
    fields = {
        "files": "Depset of the metadata files, their attribute files and licence texts.",
        "metadata": "Depset of each package's metadata JSON file.",
    },
)

def _crates_aspect_impl(_target, ctx):
    # A build script's dependencies build the crate; none reaches the program.
    if ctx.rule.kind == "cargo_build_script":
        return [_CratesInfo(metadata = depset(), files = depset())]
    metadata = []
    files = []
    for provider in getattr(ctx.rule.attr, "package_metadata", None) or []:
        if PackageMetadataInfo in provider:
            info = provider[PackageMetadataInfo]
            metadata.append(depset([info.metadata]))
            files.append(info.files)
    for attribute in ("deps", "proc_macro_deps"):
        for dependency in getattr(ctx.rule.attr, attribute, None) or []:
            if _CratesInfo in dependency:
                metadata.append(dependency[_CratesInfo].metadata)
                files.append(dependency[_CratesInfo].files)
    return [_CratesInfo(metadata = depset(transitive = metadata), files = depset(transitive = files))]

_crates_aspect = aspect(
    implementation = _crates_aspect_impl,
    attr_aspects = ["deps", "proc_macro_deps"],
    doc = "Collects the package_metadata of a Rust program and its dependencies, without build-script dependencies.",
)

def _third_party_notices_impl(ctx):
    packages = ctx.attr.program[_CratesInfo]
    manifest = ctx.actions.declare_file(ctx.label.name + ".manifest.json")
    notices = ctx.actions.declare_file(ctx.label.name + "/ThirdPartyNotices.txt")
    license = ctx.actions.declare_file(ctx.label.name + "/LICENSE")
    ctx.actions.write(manifest, json.encode({
        "metadata": [file.path for file in packages.metadata.to_list()],
        "policy": ctx.file.policy.path,
        "program": ctx.attr.package,
        "title": ctx.attr.title,
    }))
    ctx.actions.run(
        executable = ctx.executable._notices,
        arguments = ["--manifest", manifest.path, "--notices", notices.path, "--license", license.path],
        inputs = depset([manifest, ctx.file.policy], transitive = [packages.metadata, packages.files]),
        outputs = [notices, license],
        mnemonic = "ThirdPartyNotices",
        progress_message = "Writing the third-party notices of %{label}",
    )
    return [
        DefaultInfo(files = depset([license, notices])),
        OutputGroupInfo(license = depset([license]), notices = depset([notices])),
    ]

third_party_notices = rule(
    implementation = _third_party_notices_impl,
    doc = """Writes a Rust program's own licence and its third-party notices.

    The notices list every package the program links, with the licence the
    supply-chain policy accepts for it, followed by the licence texts packages
    ship. Build-script dependencies are left out. The `license` and `notices`
    output groups select each file.
    """,
    attrs = {
        "package": attr.string(doc = "The package whose licence is the program's own.", mandatory = True),
        "policy": attr.label(doc = "The rendered supply-chain policy, for its licence overrides.", allow_single_file = [".json"], mandatory = True),
        "program": attr.label(
            doc = "The Rust program, in the configuration it ships in.",
            mandatory = True,
            aspects = [_crates_aspect],
        ),
        "title": attr.string(doc = "The program's name in the notices.", mandatory = True),
        "_notices": attr.label(default = Label("//supply_chain/private/tools/write-notices"), executable = True, cfg = "exec"),
    },
)
