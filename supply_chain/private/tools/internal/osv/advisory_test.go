package osv

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestAdvisoryBoundaries(t *testing.T) {
	p, _ := purl.Parse("pkg:npm/a@1.2.3")
	current, _ := parseVersion(*p.Version)
	for _, tc := range []struct {
		events []event
		want   bool
	}{{[]event{{Introduced: new("0")}, {Fixed: new("1.2.3")}}, false}, {[]event{{Introduced: new("0")}, {LastAffected: new("1.2.3")}}, true}, {[]event{{Introduced: new("0")}, {Limit: new("1.2.3")}}, false}, {[]event{{Introduced: new("1.2.3")}, {Fixed: new("1.2.3")}}, false}, {[]event{{Fixed: new("1.2.3")}, {Introduced: new("1.2.3")}}, true}} {
		r := versionRange{Kind: "SEMVER", Events: tc.events}
		got, err := r.includes("ID", p, current)
		if err != nil || got != tc.want {
			t.Fatalf("%+v=%t %v", tc.events, got, err)
		}
	}
	affected := Affected{Package: AdvisoryPackage{"npm", "a"}, Ranges: []versionRange{{Kind: "GIT"}}}
	if _, err := affected.includes("ID", p, current); err == nil {
		t.Fatal("unknown range passed")
	}
	affected.Versions = []string{"1.2.3"}
	if got, err := affected.includes("ID", p, current); err != nil || !got {
		t.Fatal("explicit versions lost precedence")
	}
	affected.Versions = nil
	affected.Ranges = []versionRange{{Kind: "SEMVER", Events: []event{{Introduced: new("0")}}}}
	affected.Specific = &databaseSpecific{LastKnown: new("< 1.2.3")}
	if got, err := affected.includes("ID", p, current); err != nil || got {
		t.Fatalf("last known bound %t %v", got, err)
	}
}

func TestAdvisoryPackageRanges(t *testing.T) {
	const data = `{"advisories": [
        {"id": "FIXED", "affected": [{"package": {"ecosystem": "npm", "name": "@scope/pkg"},
            "ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "1.2.0"}]}]}]},
        {"id": "LAST", "affected": [{"package": {"ecosystem": "crates.io", "name": "demo"},
            "ranges": [{"type": "SEMVER", "events": [{"introduced": "1.0.0"}, {"last_affected": "1.4.0"}]}]}]},
        {"id": "LIMITED", "affected": [{"package": {"ecosystem": "crates.io", "name": "limited"},
            "ranges": [{"type": "SEMVER", "events": [{"introduced": "1.0.0"}, {"limit": "2.0.0"}]}]}]},
        {"id": "LISTED", "affected": [{"package": {"ecosystem": "npm", "name": "listed"}, "versions": ["2.0.0"]}]},
        {"id": "UNFIXED", "affected": [{"package": {"ecosystem": "npm", "name": "unfixed"},
            "ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}]}],
            "database_specific": {"last_known_affected_version_range": "< 0.57"}}]}
    ]}`
	var idx Index
	if err := jsondata.Read(testfs.Write(t, filepath.Join(t.TempDir(), "index.json"), data), &idx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ purl, advisory string }{
		{"pkg:npm/%40scope/pkg@1.2.0-rc.1", "FIXED"},
		{"pkg:npm/%40scope/pkg@1.2.0", ""},
		{"pkg:npm/%40scope/pkg@0.0.0-next.1", "FIXED"},
		{"pkg:cargo/demo@1.4.0", "LAST"},
		{"pkg:cargo/demo@1.4.1", ""},
		{"pkg:cargo/limited@1.9.9", "LIMITED"},
		{"pkg:cargo/limited@2.0.0", ""},
		{"pkg:npm/listed@2.0.0", "LISTED"},
		{"pkg:npm/listed@2.0.1", ""},
		{"pkg:npm/unfixed@0.56.9", "UNFIXED"},
		{"pkg:npm/unfixed@0.57.0", ""},
	} {
		t.Run(tc.purl, func(t *testing.T) {
			p, err := purl.Parse(tc.purl)
			if err != nil {
				t.Fatal(err)
			}
			matched, err := idx.Affecting(p)
			if err != nil {
				t.Fatal(err)
			}
			var ids []string
			for _, a := range matched {
				ids = append(ids, a.ID)
			}
			var expected []string
			if tc.advisory != "" {
				expected = append(expected, tc.advisory)
			}
			if !slices.Equal(ids, expected) {
				t.Fatalf("advisories %v, want %v", ids, expected)
			}
		})
	}
}
