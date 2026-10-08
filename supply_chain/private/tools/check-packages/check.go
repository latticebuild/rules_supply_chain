package main

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/cli"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/metadata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/osv"
	policydata "github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/policy"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
)

type findings struct{ licences, advisories, sources, warnings []string }

func run(args []string) (int, error) {
	a, err := cli.Parse(args, []string{"--manifest", "--report", "--status"})
	if err != nil {
		return 1, err
	}
	result, err := checkPackages(a["--manifest"][0])
	if err != nil {
		return 1, err
	}
	return 0, result.Write(a["--report"][0], a["--status"][0])
}
func checkPackages(path string) (Result, error) {
	var manifest struct {
		Policy     string `json:"policy"`
		Advisories string `json:"advisories"`
		Sources    []struct {
			Label    string   `json:"label"`
			Metadata []string `json:"metadata"`
		} `json:"sources"`
	}
	if err := jsondata.Read(path, &manifest); err != nil {
		return Result{}, err
	}
	policy, err := policydata.Read(manifest.Policy)
	if err != nil {
		return Result{}, err
	}
	var index osv.Index
	if err := jsondata.Read(manifest.Advisories, &index); err != nil {
		return Result{}, err
	}
	var result findings
	var counts []string
	var read []metadata.Package
	for _, source := range manifest.Sources {
		if len(source.Metadata) == 0 {
			result.sources = append(result.sources, source.Label+" contributed no packages")
		}
		counts = append(counts, fmt.Sprintf("%s: %d", source.Label, len(source.Metadata)))
		for _, path := range source.Metadata {
			p, err := metadata.Read(path)
			if err != nil {
				return Result{}, err
			}
			read = append(read, p)
		}
	}
	packages := metadata.Unique(read)
	if err := checkLicences(policy, packages, &result); err != nil {
		return Result{}, err
	}
	if err := checkAdvisories(policy, index, packages, &result); err != nil {
		return Result{}, err
	}
	checkSources(policy, packages, &result)
	return result.verdict(len(packages), counts, index.Counts()), nil
}
func checkLicences(policy policydata.Policy, packages []metadata.Package, findings *findings) error {
	allow, err := Allowlist(policy.Licenses.Allow)
	if err != nil {
		return err
	}
	patterns := make([]purl.URL, len(policy.Licenses.Overrides))
	for i, entry := range policy.Licenses.Overrides {
		parsed, err := purl.Parse(entry.Package)
		if err != nil {
			return err
		}
		patterns[i] = parsed
	}
	used := map[int]bool{}
	for _, p := range packages {
		position := -1
		for i, pattern := range patterns {
			if pattern.Covers(p.URL) {
				position = i
				break
			}
		}
		var expression Expression
		if position >= 0 {
			used[position] = true
			entry := policy.Licenses.Overrides[position]
			problem, err := textMismatch(entry, p)
			if err != nil {
				return err
			}
			if problem != "" {
				findings.licences = append(findings.licences, fmt.Sprintf("%s: %s", p.URL, problem))
				continue
			}
			expression, err = Parse(entry.Expression)
			if err != nil {
				return fmt.Errorf("licence override for %s: expression %q: %w", entry.Package, entry.Expression, err)
			}
		} else {
			if p.Licence == nil || *p.Licence == "NOASSERTION" {
				findings.licences = append(findings.licences, p.URL.String()+": no licence stated")
				continue
			}
			expression, err = Parse(*p.Licence)
			if err != nil {
				findings.licences = append(findings.licences, fmt.Sprintf("%s: unreadable licence %q: %s", p.URL, *p.Licence, err))
				continue
			}
		}
		if refused := expression.Refused(allow); len(refused) > 0 {
			findings.licences = append(findings.licences, fmt.Sprintf("%s: %s is not allowed (%s not on the allowlist)", p.URL, expression.Text, strings.Join(refused, ", ")))
		}
	}
	for i, entry := range policy.Licenses.Overrides {
		if !used[i] {
			findings.warnings = append(findings.warnings, fmt.Sprintf("licence override %s matched no package (%s)", entry.Package, entry.Reason))
		}
	}
	return nil
}
func textMismatch(entry policydata.Override, p metadata.Package) (string, error) {
	if entry.SHA256 == nil {
		return "", nil
	}
	if p.Text == nil {
		return fmt.Sprintf("override %s expects a licence text", entry.Package), nil
	}
	contents, err := os.ReadFile(*p.Text)
	if err != nil {
		return "", err
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(contents))
	if actual != *entry.SHA256 {
		return fmt.Sprintf("licence text sha256 %s differs from the reviewed %s of override %s", actual, *entry.SHA256, entry.Package), nil
	}
	return "", nil
}
func checkAdvisories(policy policydata.Policy, index osv.Index, packages []metadata.Package, findings *findings) error {
	ignored := map[string]string{}
	for _, entry := range policy.Advisories.Ignore {
		ignored[entry.ID] = entry.Reason
	}
	used := map[string]bool{}
	for _, p := range packages {
		if _, vcs := p.URL.Qualifiers["vcs_url"]; vcs {
			continue
		}
		advisories, err := index.Affecting(p.URL)
		if err != nil {
			return err
		}
		for _, advisory := range advisories {
			matched := false
			for _, id := range advisory.IDs() {
				if _, ok := ignored[id]; ok {
					used[id] = true
					matched = true
				}
			}
			if matched {
				continue
			}
			kind := ""
			if value := advisory.Informational(); value != nil {
				kind = " [" + *value + "]"
			}
			aliases := ""
			if len(advisory.Aliases) > 0 {
				aliases = " (" + strings.Join(advisory.Aliases, ", ") + ")"
			}
			findings.advisories = append(findings.advisories, fmt.Sprintf("%s%s%s %s: %s", advisory.ID, aliases, kind, advisory.Summary, p.URL))
		}
	}
	for _, id := range slices.Sorted(maps.Keys(ignored)) {
		if !used[id] {
			findings.warnings = append(findings.warnings, fmt.Sprintf("ignored advisory %s matched no package (%s)", id, ignored[id]))
		}
	}
	return nil
}
func normalizeSource(url string) string {
	return strings.Map(func(c rune) rune {
		if c >= 'A' && c <= 'Z' {
			return c + ('a' - 'A')
		}
		return c
	}, strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(url), "/"), ".git"))
}
func checkSources(policy policydata.Policy, packages []metadata.Package, findings *findings) {
	registries, repositories := map[string]bool{}, map[string]bool{}
	for _, url := range policy.Sources.AllowRegistry {
		registries[normalizeSource(url)] = true
	}
	for _, url := range policy.Sources.AllowGit {
		repositories[normalizeSource(url)] = true
	}
	for _, p := range packages {
		if p.URL.Kind != "cargo" {
			continue
		}
		if registry, ok := p.URL.Qualifiers["repository_url"]; ok && !registries[normalizeSource(registry)] {
			findings.sources = append(findings.sources, fmt.Sprintf("%s: registry %s is not allowed", p.URL, registry))
		}
		if vcs, ok := p.URL.Qualifiers["vcs_url"]; ok {
			remote := strings.TrimPrefix(vcs, "git+")
			if at := strings.LastIndexByte(remote, '@'); at >= 0 {
				remote = remote[:at]
			}
			if !repositories[normalizeSource(remote)] {
				findings.sources = append(findings.sources, fmt.Sprintf("%s: git repository %s is not allowed", p.URL, remote))
			}
		}
	}
}
func (f findings) verdict(packages int, counts, advisories []string) Result {
	var report strings.Builder
	fmt.Fprintf(&report, "supply-chain: %d packages (%s); advisories (%s)\n", packages, strings.Join(counts, "; "), strings.Join(advisories, "; "))
	passed := true
	for _, section := range []struct {
		name  string
		items []string
	}{{"licences", f.licences}, {"advisories", f.advisories}, {"sources", f.sources}} {
		if len(section.items) == 0 {
			fmt.Fprintf(&report, "%s: ok\n", section.name)
		} else {
			passed = false
			fmt.Fprintf(&report, "%s: FAILED\n", section.name)
			for _, item := range section.items {
				fmt.Fprintf(&report, "  %s\n", item)
			}
		}
	}
	if len(f.warnings) > 0 {
		report.WriteString("warnings:\n")
		for _, item := range f.warnings {
			fmt.Fprintf(&report, "  %s\n", item)
		}
	}
	return Result{Passed: passed, Report: report.String()}
}
