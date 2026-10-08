package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/cli"
)

type verdictData struct {
	report, status []byte
}

var embeddedVerdict *verdictData

func run(args []string) (int, error) {
	var text, report []byte
	var reportPath string
	if embeddedVerdict != nil {
		if len(args) != 0 {
			return 1, fmt.Errorf("an embedded verdict does not accept arguments")
		}
		text, report = embeddedVerdict.status, embeddedVerdict.report
	} else {
		a, err := cli.Parse(args, []string{"--report", "--status"})
		if err != nil {
			return 1, err
		}
		text, err = os.ReadFile(a["--status"][0])
		if err != nil {
			return 1, err
		}
		reportPath = a["--report"][0]
	}
	recorded := strings.TrimRightFunc(string(text), unicode.IsSpace)
	if recorded != "0" && recorded != "1" {
		return 1, fmt.Errorf("invalid recorded status %q; expected 0 or 1", string(text))
	}
	if embeddedVerdict == nil {
		var err error
		report, err = os.ReadFile(reportPath)
		if err != nil {
			return 1, err
		}
	}
	if _, err := os.Stdout.Write(report); err != nil {
		return 1, err
	}
	if recorded == "1" {
		return 1, nil
	}
	return 0, nil
}
