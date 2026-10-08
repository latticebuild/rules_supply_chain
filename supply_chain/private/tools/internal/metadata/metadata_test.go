package metadata

import (
	"path/filepath"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestUniqueRetainsDistinctLossyURLs(t *testing.T) {
	one, _ := purl.Parse("pkg:npm/a%FF@1.0.0")
	two, _ := purl.Parse("pkg:npm/a%FF%FF@1.0.0")
	if got := Unique([]Package{{URL: one}, {URL: two}}); len(got) != 2 {
		t.Fatal("invalid bytes merged packages")
	}
}

func TestJSONFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	for _, text := range []string{`{"purl":"pkg:npm/a","extra":"\ud800"}`, `{"purl":"pkg:npm/a","extra":{"\ud800":"ignored"}}`, `{"purl":"pkg:npm/a","Purl":"invalid","extra":"` + string([]byte{255}) + `"}`, `["pkg:npm/a"]`} {
		testfs.Write(t, path, text)
		if _, err := Read(path); err != nil {
			t.Fatalf("unknown/positional fields: %s: %v", text, err)
		}
	}
	for _, text := range []string{`{"purl":"pkg:npm/\ud800"}`, `{"purl":"pkg:npm/a","\ud800":true}`, `{"purl":"pkg:npm/a","attributes":{"unused":12,"unused":"valid"}}`, `{"purl":"pkg:npm/a","attributes":{"unused":null,"unused":"valid"}}`} {
		testfs.Write(t, path, text)
		if _, err := Read(path); err == nil {
			t.Fatalf("accepted invalid known field: %s", text)
		}
	}
}
