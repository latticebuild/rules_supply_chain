package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/cli"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/osv"
)

func run(args []string) (int, error) {
	a, err := cli.Parse(args, []string{"--output"}, "--source")
	if err != nil {
		return 1, err
	}
	if len(a["--source"]) == 0 {
		return 1, fmt.Errorf("at least one --source is required")
	}
	found := map[string]osv.Advisory{}
	for _, source := range a["--source"] {
		ecosystem, list, ok := strings.Cut(source, "=")
		if !ok {
			return 1, fmt.Errorf("expected ECOSYSTEM=LIST, got %s", source)
		}
		before := len(found)
		listing, err := os.ReadFile(list)
		if err != nil {
			return 1, err
		}
		if !utf8.Valid(listing) {
			return 1, fmt.Errorf("%s: invalid UTF-8", list)
		}
		lines := strings.Split(string(listing), "\n")
		for i, path := range lines {
			if i < len(lines)-1 {
				path = strings.TrimSuffix(path, "\r")
			}
			if path == "" {
				continue
			}
			var entry osv.Advisory
			if err := jsondata.Read(path, &entry); err != nil {
				return 1, err
			}
			if entry.Withdrawn != nil {
				continue
			}
			kept := make([]osv.Affected, 0, len(entry.Affected))
			for _, item := range entry.Affected {
				if item.Package.Ecosystem == ecosystem {
					kept = append(kept, item)
				}
			}
			entry.Affected = kept
			if len(kept) > 0 {
				if _, exists := found[entry.ID]; !exists {
					found[entry.ID] = entry
				}
			}
		}
		if len(found) == before {
			return 1, fmt.Errorf("%s: no %s advisory in the listed files", list, ecosystem)
		}
	}
	result := osv.Index{Advisories: make([]osv.Advisory, 0, len(found))}
	for _, id := range slices.Sorted(maps.Keys(found)) {
		result.Advisories = append(result.Advisories, found[id])
	}
	data, err := indexJSON(result)
	if err != nil {
		return 1, err
	}
	return 0, os.WriteFile(a["--output"][0], data, 0o644)
}

// Match serde_json's compact, UTF-8 output, including HTML and line separators.
func indexJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	data := bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == '\\' && i+1 < len(data) {
			if i+6 <= len(data) && (string(data[i:i+6]) == `\u2028` || string(data[i:i+6]) == `\u2029`) {
				if data[i+5] == '8' {
					out = append(out, "\u2028"...)
				} else {
					out = append(out, "\u2029"...)
				}
				i += 5
				continue
			}
			out = append(out, data[i], data[i+1])
			i++
			continue
		}
		out = append(out, data[i])
	}
	return out, nil
}
