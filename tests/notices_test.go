package checks

import (
	"bytes"
	"github.com/bazelbuild/rules_go/go/runfiles"
	"os"
	"testing"
)

func TestRustNoticeGraph(t *testing.T) {
	read := func(variable string) []byte {
		t.Helper()
		path, err := runfiles.Rlocation(os.Getenv(variable))
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if got, want := read("OWN_OUTPUT"), read("OWN_SOURCE"); !bytes.Equal(got, want) {
		t.Errorf("program license = %q, want exact bytes %q", got, want)
	}
	want := "Third-party notices for Fixture\n\nFixture includes the following packages, each under the licence shown.\n\na-linked 1.0.0: MIT\nz-macro 1.0.0: MIT\n\n--- a-linked 1.0.0 ---\n\nDependency license text\n\n--- z-macro 1.0.0 ---\n\nDependency license text\n"
	if got := read("NOTICE_OUTPUT"); string(got) != want {
		t.Errorf("notices = %s, want %s", got, want)
	}
}
