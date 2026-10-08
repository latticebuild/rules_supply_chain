package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestNoticesDeterministicAndOwnLicenceBytes(t *testing.T) {
	dir := t.TempDir()
	metadata := []string{}
	own := testfs.Write(t, filepath.Join(dir, "own.LICENSE"), "own\r\n\x00")
	third := testfs.Write(t, filepath.Join(dir, "third.LICENSE"), "Third party\n\n")
	for i, item := range []struct {
		url, licence string
		text         *string
	}{{"pkg:npm/z@1.0.0", "MIT", &third}, {"pkg:cargo/program@1.0.0", "MIT", &own}, {"pkg:npm/a@2.0.0", "NOASSERTION", nil}} {
		attribute := map[string]any{"kind": map[string]string{"identifier": item.licence}}
		if item.text != nil {
			attribute["text"] = *item.text
		}
		data, err := json.Marshal(attribute)
		if err != nil {
			t.Fatal(err)
		}
		attr := testfs.Write(t, filepath.Join(dir, fmt.Sprintf("attribute-%d.json", i)), string(data))
		data, err = json.Marshal(map[string]any{"purl": item.url, "attributes": map[string]string{"build.bazel.attribute.license": attr}})
		if err != nil {
			t.Fatal(err)
		}
		metadata = append(metadata, testfs.Write(t, filepath.Join(dir, fmt.Sprintf("metadata-%d.json", i)), string(data)))
	}
	policy := testfs.Write(t, filepath.Join(dir, "policy.json"), `{"licenses":{"allow":["MIT"],"overrides":[{"package":"pkg:npm/a","expression":"ISC AND MIT","reason":"reviewed"}]}}`)
	notices, licence := filepath.Join(dir, "notices"), filepath.Join(dir, "LICENSE")
	var first []byte
	for i := range 2 {
		if i == 1 {
			slices.Reverse(metadata)
			metadata = append(metadata, metadata[0])
		}
		data, err := json.Marshal(map[string]any{"title": "Tool", "program": "program", "policy": policy, "metadata": metadata})
		if err != nil {
			t.Fatal(err)
		}
		manifest := testfs.Write(t, filepath.Join(dir, "manifest.json"), string(data))
		code, err := run([]string{"--manifest", manifest, "--notices", notices, "--license", licence})
		if code != 0 || err != nil {
			t.Fatalf("%d %v", code, err)
		}
		body, err := os.ReadFile(notices)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = body
		} else if string(first) != string(body) {
			t.Fatalf("order/duplicate changed notices:\n%s\n%s", first, body)
		}
		data, err = os.ReadFile(licence)
		if err != nil || string(data) != "own\r\n\x00" {
			t.Fatalf("own licence changed: %q %v", data, err)
		}
	}
	if !strings.Contains(string(first), "a 2.0.0: ISC AND MIT\nz 1.0.0: MIT\n\n--- z 1.0.0 ---\n\nThird party\n") {
		t.Fatalf("%s", first)
	}
}
