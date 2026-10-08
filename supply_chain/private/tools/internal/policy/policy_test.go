package policy

import (
	"path/filepath"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestJSONParity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	for _, text := range []string{`{}`, `{"licenses":null}`, `{"Licenses":{"allow":[]}}`, `{"licenses":{"allow":[],"extra":true}}`, `{"licenses":{"allow":[],"allow":[]}}`, `{"licenses":{"allow":null}}`, `{"licenses":{"allow":[1]}}`, `{"licenses":{"allow":[]}} false`, `{"licenses":{"allow":[],"overrides":[{"package":"pkg:npm/a","expression":"MIT","reason":"\ud800"}]}}`} {
		testfs.Write(t, path, text)
		if _, err := Read(path); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
	for _, text := range []string{`{"licenses":{"allow":["MIT"]}}`, `[[["MIT"]]]`, `{"licenses":{"allow":[],"overrides":[{"package":"pkg:npm/a","expression":"MIT","reason":"\ud83d\ude00","sha256":null}]}}`} {
		testfs.Write(t, path, text)
		if _, err := Read(path); err != nil {
			t.Fatalf("refused %s: %v", text, err)
		}
	}

}
