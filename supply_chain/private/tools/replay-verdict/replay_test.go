package main

import (
	"path/filepath"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestReplayRejectsMalformedVerdicts(t *testing.T) {
	dir := t.TempDir()
	report := testfs.Write(t, filepath.Join(dir, "report"), "")
	status := filepath.Join(dir, "status")
	for _, value := range []string{"0", "1\n", " 0", "2", "00", "0\n1", ""} {
		testfs.Write(t, status, value)
		code, err := run([]string{"--report", report, "--status", status})
		valid := value == "0" || value == "1\n"
		if (err == nil) != valid || valid && code != int(value[0]-'0') {
			t.Fatalf("%q: %d %v", value, code, err)
		}
	}
}
