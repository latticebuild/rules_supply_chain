package checks

import (
	"bytes"
	"github.com/bazelbuild/rules_go/go/runfiles"
	"os"
	"os/exec"
	"path/filepath"
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
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("expected policy refusal exit 1: %v", err)
			}
			data, err := os.ReadFile(executable)
			if err != nil {
				t.Fatal(err)
			}
			copied := filepath.Join(t.TempDir(), "standalone verdict")
			if err := os.WriteFile(copied, data, 0700); err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(t.Context(), copied)
			command.Dir = filepath.Dir(copied)
			for _, entry := range os.Environ() {
				name, _, _ := strings.Cut(entry, "=")
				switch name {
				case "RUNFILES_DIR", "RUNFILES_MANIFEST_FILE", "JAVA_RUNFILES", "TEST_SRCDIR", "TEST_WORKSPACE", "TEST_TMPDIR":
					continue
				}
				command.Env = append(command.Env, entry)
			}
			copiedOutput, err := command.CombinedOutput()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || !bytes.Equal(copiedOutput, output) {
				t.Fatalf("copied verdict differs: %v %q; original=%q", err, copiedOutput, output)
			}
			command = exec.CommandContext(t.Context(), copied, "--report", "override", "--status", "override")
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Run(); err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "does not accept arguments") {
				t.Fatalf("copied verdict accepted overrides: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
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
