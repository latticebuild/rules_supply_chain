package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/cli"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/metadata"
	policydata "github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/policy"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
)

func run(args []string) (int, error) {
	a, err := cli.Parse(args, []string{"--manifest", "--notices", "--license"})
	if err != nil {
		return 1, err
	}
	var manifest struct {
		Title    string   `json:"title"`
		Program  string   `json:"program"`
		Policy   string   `json:"policy"`
		Metadata []string `json:"metadata"`
	}
	if err := jsondata.Read(a["--manifest"][0], &manifest); err != nil {
		return 1, err
	}
	policy, err := policydata.Read(manifest.Policy)
	if err != nil {
		return 1, err
	}
	var overrides []noticeOverride
	for _, entry := range policy.Licenses.Overrides {
		pattern, err := purl.Parse(entry.Package)
		if err != nil {
			return 1, err
		}
		overrides = append(overrides, noticeOverride{pattern, entry.Expression})
	}
	var packages []metadata.Package
	for _, path := range manifest.Metadata {
		p, err := metadata.Read(path)
		if err != nil {
			return 1, err
		}
		packages = append(packages, p)
	}
	packages = metadata.Unique(packages)
	var program, others []metadata.Package
	for _, p := range packages {
		if p.URL.FullName() == manifest.Program {
			program = append(program, p)
		} else {
			others = append(others, p)
		}
	}
	if len(program) == 0 || program[0].Text == nil {
		return 1, fmt.Errorf("the program's package %s is not in the build graph or ships no licence text", manifest.Program)
	}
	own, err := os.ReadFile(*program[0].Text)
	if err != nil {
		return 1, err
	}
	if err := os.WriteFile(a["--license"][0], own, 0o644); err != nil {
		return 1, err
	}
	text, err := renderNotices(manifest.Title, others, overrides)
	if err != nil {
		return 1, err
	}
	return 0, os.WriteFile(a["--notices"][0], []byte(text), 0o644)
}

type noticeOverride struct {
	pattern    purl.URL
	expression string
}

func renderNotices(title string, packages []metadata.Package, overrides []noticeOverride) (string, error) {
	var list, texts strings.Builder
	fmt.Fprintf(&list, "Third-party notices for %s\n\n%s includes the following packages, each under the licence shown.\n\n", title, title)
	for _, p := range packages {
		version := "unversioned"
		if p.URL.Version != nil {
			version = *p.URL.Version
		}
		licence := "NOASSERTION"
		if p.Licence != nil {
			licence = *p.Licence
		}
		for _, entry := range overrides {
			if entry.pattern.Covers(p.URL) {
				licence = entry.expression
				break
			}
		}
		fmt.Fprintf(&list, "%s %s: %s\n", p.URL.FullName(), version, licence)
		if p.Text != nil {
			body, err := os.ReadFile(*p.Text)
			if err != nil {
				return "", err
			}
			if !utf8.Valid(body) {
				return "", fmt.Errorf("%s: invalid UTF-8 licence text", *p.Text)
			}
			fmt.Fprintf(&texts, "\n--- %s %s ---\n\n%s\n", p.URL.FullName(), version, strings.TrimRightFunc(string(body), unicode.IsSpace))
		}
	}
	return list.String() + texts.String(), nil
}
