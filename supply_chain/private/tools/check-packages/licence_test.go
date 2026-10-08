package main

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestSPDXInventoryAndDigests(t *testing.T) {
	r, err := registry()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.licences) != 734 || len(r.exceptions) != 84 || len(r.gnu) != 64 {
		t.Fatalf("inventory %d/%d/%d", len(r.licences), len(r.exceptions), len(r.gnu))
	}
	for id := range r.licences {
		e, err := Parse(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		allow, err := Allowlist([]string{e.root.requirement.licence})
		if err != nil || len(e.Refused(allow)) != 0 {
			t.Fatalf("%s did not satisfy itself: %v", id, err)
		}
	}
	for id := range r.exceptions {
		text := "MIT WITH " + id
		e, err := Parse(text)
		if err != nil {
			t.Fatal(err)
		}
		allow, err := Allowlist([]string{text})
		if err != nil || len(e.Refused(allow)) != 0 {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for file, digest := range map[string]string{"licenses.json": "f728c534d8bd1044fc515a2ddb2292be99559021d830bfa3281be0bcd36302ee", "exceptions.json": "bd145bb558f44432fcd6f0d7e956ed0124dff72af7641a7cfcb1b557dc390a5b"} {
		data, err := spdxFiles.ReadFile("assets/generated/spdx/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			t.Fatalf("changed pinned %s", file)
		}
	}
}
func TestSPDXSemantics(t *testing.T) {
	for _, tc := range []struct {
		expression     string
		allow, refused []string
	}{
		{"MIT OR GPL-3.0-only", []string{"MIT"}, nil}, {"MIT AND GPL-3.0-only", []string{"MIT"}, []string{"GPL-3.0-only"}},
		{"GPL-3.0-only OR (MIT AND GPL-3.0-only)", []string{"MIT"}, []string{"GPL-3.0-only", "GPL-3.0-only"}},
		{"Apache-2.0 WITH LLVM-exception", []string{"Apache-2.0"}, []string{"Apache-2.0 WITH LLVM-exception"}},
		{"Apache-2.0 WITH LLVM-exception", []string{"Apache-2.0 WITH LLVM-exception"}, nil},
		{"Apache-1.0+", []string{"Apache-2.0"}, nil}, {"Apache-2.0+", []string{"Apache-1.0"}, []string{"Apache-2.0+"}},
		{"GPL-2.0+", []string{"GPL-2.0-or-later"}, nil}, {"GPL-2.0-or-later", []string{"GPL-3.0-only"}, []string{"GPL-2.0-or-later"}},
		{"mit / Apache 2.0", []string{"MIT"}, nil}, {"DocumentRef-doc:LicenseRef-local WITH AdditionRef-extra", []string{"DocumentRef-doc:LicenseRef-local WITH AdditionRef-extra"}, nil},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			e, err := Parse(tc.expression)
			if err != nil {
				t.Fatal(err)
			}
			allow, err := Allowlist(tc.allow)
			if err != nil {
				t.Fatal(err)
			}
			if got := e.Refused(allow); !slices.Equal(got, tc.refused) {
				t.Fatalf("refused=%v want=%v", got, tc.refused)
			}
		})
	}
	for _, text := range []string{"MIT +", "MIT And Apache-2.0", "LicenseRef-x+", "GPL-2.0-or-later+", "MIT WITH MIT", "MIT AND", "(MIT", "MIT)"} {
		if _, err := Parse(text); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
	if _, err := Allowlist([]string{"Apache-1.0+"}); err == nil {
		t.Fatal("accepted + on allowlist")
	}
	for text, expected := range map[string]string{"MIT MIT": "`AND`, `OR`, `WITH`, `)`, `+`", "(MIT MIT)": "`AND`, `OR`, `WITH`, `)`, `+`", "MIT++": "`AND`, `OR`, `WITH`, `)`", "(MIT) MIT": "`AND`, `OR`", "MIT WITH LLVM-exception MIT": "`AND`, `OR`, `)`"} {
		_, err := Parse(text)
		if err == nil || !strings.HasSuffix(err.Error(), "expected one of "+expected+" here") {
			t.Fatalf("%s: %v", text, err)
		}
	}
}
