package checks

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/runfiles"
)

func TestStandaloneRecordedVerdicts(t *testing.T) {
	for _, fixture := range []struct {
		variable, report string
		status           int
	}{
		{"PASSING_VERDICT", "recorded verdict: accepted\n", 0},
		{"FAILING_VERDICT", "recorded verdict: refused\n", 1},
	} {
		t.Run(fixture.variable, func(t *testing.T) {
			executable, err := runfiles.Rlocation(os.Getenv(fixture.variable))
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(executable)
			if err != nil {
				t.Fatal(err)
			}
			copied := filepath.Join(t.TempDir(), "copied verdict.exe")
			if err := os.WriteFile(copied, data, 0700); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{executable, copied} {
				command := exec.CommandContext(t.Context(), path)
				command.Dir = filepath.Dir(copied)
				for _, entry := range os.Environ() {
					name, _, _ := strings.Cut(entry, "=")
					switch name {
					case "RUNFILES_DIR", "RUNFILES_MANIFEST_FILE", "JAVA_RUNFILES", "TEST_SRCDIR", "TEST_WORKSPACE", "TEST_TMPDIR":
						continue
					}
					command.Env = append(command.Env, entry)
				}
				var output, errors bytes.Buffer
				command.Stdout, command.Stderr = &output, &errors
				err := command.Run()
				status := 0
				if err != nil {
					exit, ok := err.(*exec.ExitError)
					if !ok {
						t.Fatal(err)
					}
					status = exit.ExitCode()
				}
				if status != fixture.status || output.String() != fixture.report || errors.Len() != 0 {
					t.Fatalf("%s: status=%d stdout=%q stderr=%q", path, status, output.String(), errors.String())
				}
				command = exec.CommandContext(t.Context(), path, "--report", "replacement", "--status", "replacement")
				output.Reset()
				errors.Reset()
				command.Stdout, command.Stderr = &output, &errors
				err = command.Run()
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || output.Len() != 0 || !strings.Contains(errors.String(), "does not accept arguments") {
					t.Fatalf("override accepted: %v stdout=%q stderr=%q", err, output.String(), errors.String())
				}
			}
		})
	}
}
