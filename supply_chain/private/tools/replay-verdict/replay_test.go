package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/testfs"
)

func TestReplayParity(t *testing.T) {
	report := []byte("policy verdict\x00\nπ\r\n")
	for _, status := range []string{"0", "1\n", "0\t\u2003", " 0", "2", "00", "0\n1", ""} {
		for _, embedded := range []bool{false, true} {
			t.Run(strings.ReplaceAll(status, "\n", "\\n")+"/"+map[bool]string{false: "files", true: "embedded"}[embedded], func(t *testing.T) {
				var args []string
				var verdict *verdictData
				if embedded {
					verdict = &verdictData{report: report, status: []byte(status)}
				} else {
					dir := t.TempDir()
					args = []string{"--report", testfs.Write(t, filepath.Join(dir, "report"), string(report)), "--status", testfs.Write(t, filepath.Join(dir, "status"), status)}
				}
				code, err, output := replayOutput(t, verdict, args)
				valid := status == "0" || status == "1\n" || status == "0\t\u2003"
				if valid {
					if err != nil || code != int(status[0]-'0') || !bytes.Equal(output, report) {
						t.Fatalf("code=%d err=%v output=%q", code, err, output)
					}
				} else if err == nil || code != 1 || len(output) != 0 {
					t.Fatalf("malformed verdict: code=%d err=%v output=%q", code, err, output)
				}
			})
		}
	}
}

func TestReplayRefusesOverridesAndMissingInputs(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	status := testfs.Write(t, filepath.Join(dir, "status"), "0")
	for _, args := range [][]string{nil, {"--report", missing, "--status", status}, {"--report", missing, "--status", missing}} {
		code, err, output := replayOutput(t, nil, args)
		if err == nil || code != 1 || len(output) != 0 {
			t.Fatalf("%v: code=%d err=%v output=%q", args, code, err, output)
		}
	}
	testfs.Write(t, status, "bad status")
	_, err, output := replayOutput(t, nil, []string{"--report", missing, "--status", status})
	if err == nil || !strings.Contains(err.Error(), "invalid recorded status") || len(output) != 0 {
		t.Fatalf("status validation must precede report loading: %v %q", err, output)
	}
	for _, args := range [][]string{{"--help"}, {"--report", missing, "--status", status}} {
		code, err, output := replayOutput(t, &verdictData{report: []byte("private verdict"), status: []byte("0")}, args)
		if err == nil || !strings.Contains(err.Error(), "does not accept arguments") || code != 1 || len(output) != 0 {
			t.Fatalf("override accepted: code=%d err=%v output=%q", code, err, output)
		}
	}
}

func TestReplayReportsOutputFailure(t *testing.T) {
	stdout, verdict := os.Stdout, embeddedVerdict
	t.Cleanup(func() { os.Stdout, embeddedVerdict = stdout, verdict })
	file, err := os.CreateTemp(t.TempDir(), "closed-stdout")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout, embeddedVerdict = file, &verdictData{report: []byte("report"), status: []byte("0")}
	if code, err := run(nil); code != 1 || err == nil {
		t.Fatalf("output error lost: %d %v", code, err)
	}
}

func replayOutput(t *testing.T, verdict *verdictData, args []string) (int, error, []byte) {
	t.Helper()
	stdout, previous := os.Stdout, embeddedVerdict
	defer func() { os.Stdout, embeddedVerdict = stdout, previous }()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	os.Stdout, embeddedVerdict = file, verdict
	code, runErr := run(args)
	output, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return code, runErr, output
}
