package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/metadata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/osv"
	policydata "github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/policy"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestUnusedIgnoresAreWarnings(t *testing.T) {
	policy := policydata.Policy{Advisories: policydata.Advisories{Ignore: []policydata.Ignore{{ID: "UNUSED", Reason: "reviewed"}}}}
	var result findings
	if err := checkAdvisories(policy, osv.Index{}, nil, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.advisories) != 0 || !slices.Equal(result.warnings, []string{"ignored advisory UNUSED matched no package (reviewed)"}) {
		t.Fatalf("%+v", result)
	}
	verdict := result.verdict(0, nil, nil)
	if !verdict.Passed || !strings.Contains(verdict.Report, "warnings:\n  ignored advisory UNUSED matched no package (reviewed)\n") {
		t.Fatalf("%+v", verdict)
	}
}

func TestOverridesSourcesAndIgnores(t *testing.T) {
	url, _ := purl.Parse("pkg:cargo/a@1.0.0?repository_url=https%3A%2F%2Fcrates.io%2F&vcs_url=git%2Bhttps%3A%2F%2Fgithub.com%2Fexample%2Frepo.git%40abc")
	p := metadata.Package{URL: url, Licence: new("NOASSERTION")}
	policy := policydata.Policy{Licenses: policydata.Licenses{Allow: []string{"MIT"}}, Sources: policydata.Sources{AllowRegistry: []string{"https://crates.io"}, AllowGit: []string{"https://github.com/example/repo"}}}
	var result findings
	if err := checkLicences(policy, []metadata.Package{p}, &result); err != nil || len(result.licences) != 1 {
		t.Fatalf("missing licence: %+v %v", result, err)
	}
	policy.Licenses.Overrides = []policydata.Override{{Package: "pkg:cargo/a", Expression: "MIT", Reason: "reviewed", SHA256: new("expected")}, {Package: "pkg:cargo/unused", Expression: "MIT", Reason: "unused"}}
	result = findings{}
	if err := checkLicences(policy, []metadata.Package{p}, &result); err != nil || len(result.licences) != 1 || len(result.warnings) != 1 {
		t.Fatalf("override: %+v %v", result, err)
	}
	text := testfs.Write(t, filepath.Join(t.TempDir(), "LICENSE"), "text\n")
	p.Text = &text
	problem, err := textMismatch(policy.Licenses.Overrides[0], p)
	if err != nil || !strings.Contains(problem, "sha256") {
		t.Fatalf("hash mismatch: %s %v", problem, err)
	}
	result = findings{}
	checkSources(policy, []metadata.Package{p}, &result)
	if len(result.sources) != 0 {
		t.Fatal(result.sources)
	}
	p.URL.Qualifiers["vcs_url"] = "git+https://github.com/other/repo.git@sha"
	checkSources(policy, []metadata.Package{p}, &result)
	if len(result.sources) != 1 {
		t.Fatal(result.sources)
	}
	p.URL.Qualifiers = nil
	policy.Advisories.Ignore = []policydata.Ignore{{ID: "alias", Reason: "reviewed"}}
	idx := osv.Index{Advisories: []osv.Advisory{{ID: "ID", Aliases: []string{"alias"}, Affected: []osv.Affected{{Package: osv.AdvisoryPackage{Ecosystem: "crates.io", Name: "a"}, Versions: []string{"1.0.0"}}}}}}
	result = findings{}
	if err := checkAdvisories(policy, idx, []metadata.Package{p}, &result); err != nil || len(result.advisories) != 0 || len(result.warnings) != 0 {
		t.Fatalf("alias ignore: %+v %v", result, err)
	}
}
