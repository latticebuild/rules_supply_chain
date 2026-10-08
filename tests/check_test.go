package checks

import (
	"github.com/bazelbuild/rules_go/go/runfiles"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRefusals(t *testing.T) {
	for _, c := range []struct {
		name             string
		required, absent []string
	}{
		{"LICENCES", []string{"disallowed@1.0.0: GPL-3.0-only is not allowed", "unlicensed@1.0.0: no licence stated", "altered@1.0.0: licence text sha256"}, nil},
		{"ADVISORIES", []string{"FIXTURE-0001 (GHSA-fixture-0001)"}, []string{"FIXTURE-0002", "FIXTURE-0003"}},
		{"SOURCES", []string{"git repository https://github.com/example/foreign.git is not allowed", ":nothing contributed no packages"}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			executable, err := runfiles.Rlocation(os.Getenv("LATTICEBUILD_TEST_" + c.name))
			if err != nil {
				t.Fatal(err)
			}
			output, err := exec.CommandContext(t.Context(), executable).CombinedOutput()
			if err == nil {
				t.Fatalf("fixture passed: %s", output)
			}
			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatal(err)
			}
			for _, want := range c.required {
				if !strings.Contains(string(output), want) {
					t.Errorf("missing %q: %s", want, output)
				}
			}
			for _, absent := range c.absent {
				if strings.Contains(string(output), absent) {
					t.Errorf("unexpected %q: %s", absent, output)
				}
			}
		})
	}
}
