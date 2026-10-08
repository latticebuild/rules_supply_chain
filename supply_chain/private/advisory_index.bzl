"""Build the deterministic OSV index with index-advisories."""

visibility("//...")

def _advisory_index_impl(ctx):
    output = ctx.actions.declare_file(ctx.label.name + ".json")
    arguments = ctx.actions.args()
    arguments.add("--output", output)
    listings = []
    for target, ecosystem in sorted(ctx.attr.sources.items(), key = lambda item: str(item[0].label)):
        listing = ctx.actions.args()
        listing.add_all(target[DefaultInfo].files)
        listing.use_param_file("--source=" + ecosystem + "=%s", use_always = True)
        listing.set_param_file_format("multiline")
        listings.append(listing)
    ctx.actions.run(
        executable = ctx.executable._index,
        arguments = [arguments] + listings,
        inputs = depset(transitive = [target[DefaultInfo].files for target in ctx.attr.sources]),
        outputs = [output],
        mnemonic = "SupplyChainIndex",
        progress_message = "Indexing advisories for %{label}",
    )
    return [DefaultInfo(files = depset([output]))]

advisory_index = rule(
    implementation = _advisory_index_impl,
    doc = """Reduces OSV advisory databases to the index `supply_chain_test` reads.

    Each source keeps one ecosystem's advisories, and withdrawn advisories are
    dropped.
    """,
    attrs = {
        "sources": attr.label_keyed_string_dict(
            doc = "OSV advisory files, each keyed to the ecosystem kept from them: `crates.io` or `npm`.",
            allow_files = [".json"],
            mandatory = True,
        ),
        "_index": attr.label(default = Label("//supply_chain/private/tools/index-advisories"), executable = True, cfg = "exec"),
    },
)
