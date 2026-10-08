<!-- Generated with Stardoc: http://skydoc.bazel.build -->

Public supply-chain policy and notice rules.

<a id="advisory_index"></a>

## advisory_index

<pre>
load("@latticebuild_supply_chain//supply_chain:defs.bzl", "advisory_index")

advisory_index(<a href="#advisory_index-name">name</a>, <a href="#advisory_index-sources">sources</a>)
</pre>

Reduces OSV advisory databases to the index `supply_chain_test` reads.

Each source keeps one ecosystem's advisories, and withdrawn advisories are
dropped.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="advisory_index-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="advisory_index-sources"></a>sources |  OSV advisory files, each keyed to the ecosystem kept from them: `crates.io` or `npm`.   | <a href="https://bazel.build/rules/lib/core/dict">Dictionary: Label -> String</a> | required |  |


<a id="supply_chain_test"></a>

## supply_chain_test

<pre>
load("@latticebuild_supply_chain//supply_chain:defs.bzl", "supply_chain_test")

supply_chain_test(<a href="#supply_chain_test-name">name</a>, <a href="#supply_chain_test-advisories">advisories</a>, <a href="#supply_chain_test-packages">packages</a>, <a href="#supply_chain_test-policy">policy</a>)
</pre>

Checks the licence, advisories and source of third-party packages.

Every package must have an allowed licence, no advisory the policy does not
ignore, and, for Cargo, an allowed registry or Git repository. Each
`packages` label must contribute at least one package.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="supply_chain_test-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="supply_chain_test-advisories"></a>advisories |  The `advisory_index` to check against.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="supply_chain_test-packages"></a>packages |  Targets providing `PackageMetadataInfo`, directly or through a filegroup's `srcs`.   | <a href="https://bazel.build/concepts/labels">List of labels</a> | required |  |
| <a id="supply_chain_test-policy"></a>policy |  The caller-owned policy JSON.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |


<a id="third_party_notices"></a>

## third_party_notices

<pre>
load("@latticebuild_supply_chain//supply_chain:defs.bzl", "third_party_notices")

third_party_notices(<a href="#third_party_notices-name">name</a>, <a href="#third_party_notices-package">package</a>, <a href="#third_party_notices-policy">policy</a>, <a href="#third_party_notices-program">program</a>, <a href="#third_party_notices-title">title</a>)
</pre>

Writes a Rust program's own licence and its third-party notices.

The notices list every package the program links, with the licence the
supply-chain policy accepts for it, followed by the licence texts packages
ship. Build-script dependencies are left out. The `license` and `notices`
output groups select each file.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="third_party_notices-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="third_party_notices-package"></a>package |  The package whose licence is the program's own.   | String | required |  |
| <a id="third_party_notices-policy"></a>policy |  The rendered supply-chain policy, for its licence overrides.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="third_party_notices-program"></a>program |  The Rust program, in the configuration it ships in.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="third_party_notices-title"></a>title |  The program's name in the notices.   | String | required |  |
