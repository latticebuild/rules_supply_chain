package main

import "os"

type Result struct {
	Passed bool
	Report string
}

func (v Result) Write(report, status string) error {
	if err := os.WriteFile(report, []byte(v.Report), 0o644); err != nil {
		return err
	}
	code := "1\n"
	if v.Passed {
		code = "0\n"
	}
	return os.WriteFile(status, []byte(code), 0o644)
}
