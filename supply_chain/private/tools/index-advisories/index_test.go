package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestIndexRejectsSourcesFilteredToEmpty(t *testing.T) {
	for _, tc := range []struct{ name, contents string }{
		{"different ecosystem", `{"id":"ID","affected":[{"package":{"ecosystem":"npm","name":"a"},"versions":["1.0.0"]}]}`},
		{"withdrawn", `{"id":"ID","withdrawn":"2026-01-01","affected":[{"package":{"ecosystem":"crates.io","name":"a"},"versions":["1.0.0"]}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			advisory := testfs.Write(t, filepath.Join(dir, "advisory.json"), tc.contents)
			listing := testfs.Write(t, filepath.Join(dir, "list"), advisory+"\n")
			output := filepath.Join(dir, "index.json")
			code, err := run([]string{"--source", "crates.io=" + listing, "--output", output})
			if code == 0 || err == nil || !strings.Contains(err.Error(), "no crates.io advisory") {
				t.Fatalf("code %d: %v", code, err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("failed index published: %v", err)
			}
		})
	}
}

func TestIndexOutputAndSourceRefusals(t *testing.T) {
	text := map[string]string{"bound": "< 1 & > 0\u2028\u2029", "literal": `\u2028`}
	data, err := indexJSON(text)
	if err != nil || strings.Contains(string(data), `\u003c`) {
		t.Fatalf("%s %v", data, err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(decoded, text) {
		t.Fatalf("%s %v", data, err)
	}
	if !strings.Contains(string(data), "\u2028\u2029") {
		t.Fatalf("escaped line separators: %s", data)
	}
	dir := t.TempDir()
	advisory := testfs.Write(t, filepath.Join(dir, "advisory.json"), `{"id":"ID","affected":[{"package":{"ecosystem":"npm","name":"a"}}]}`)
	listing := filepath.Join(dir, "list")
	output := filepath.Join(dir, "index")
	for _, body := range []string{"", advisory + "\r", string([]byte{255})} {
		testfs.Write(t, listing, body)
		if _, err := run([]string{"--source", "npm=" + listing, "--output", output}); err == nil {
			t.Fatalf("accepted list %q", body)
		}
	}
	testfs.Write(t, listing, advisory+"\r\n")
	if code, err := run([]string{"--source", "npm=" + listing, "--output", output}); code != 0 || err != nil {
		t.Fatalf("CRLF: %d %v", code, err)
	}
}
