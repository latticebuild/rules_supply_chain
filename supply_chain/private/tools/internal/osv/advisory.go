package osv

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
)

type Index struct {
	Advisories []Advisory `json:"advisories"`
}
type Advisory struct {
	ID        string     `json:"id"`
	Aliases   []string   `json:"aliases,omitempty" optional:"true"`
	Summary   string     `json:"summary,omitempty" optional:"true"`
	Withdrawn *string    `json:"withdrawn,omitempty"`
	Affected  []Affected `json:"affected" optional:"true"`
}
type Affected struct {
	Package  AdvisoryPackage   `json:"package"`
	Ranges   []versionRange    `json:"ranges,omitempty" optional:"true"`
	Versions []string          `json:"versions,omitempty" optional:"true"`
	Specific *databaseSpecific `json:"database_specific,omitempty"`
}
type AdvisoryPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}
type versionRange struct {
	Kind   string  `json:"type"`
	Events []event `json:"events"`
}
type event struct {
	Introduced   *string `json:"introduced,omitempty"`
	Fixed        *string `json:"fixed,omitempty"`
	LastAffected *string `json:"last_affected,omitempty"`
	Limit        *string `json:"limit,omitempty"`
}
type databaseSpecific struct {
	Informational *string `json:"informational,omitempty"`
	LastKnown     *string `json:"last_known_affected_version_range,omitempty"`
}

func (i Index) Counts() []string {
	counts := map[string]int{}
	for _, a := range i.Advisories {
		if len(a.Affected) > 0 {
			counts[a.Affected[0].Package.Ecosystem]++
		}
	}
	var result []string
	for _, key := range slices.Sorted(maps.Keys(counts)) {
		result = append(result, fmt.Sprintf("%s: %d", key, counts[key]))
	}
	return result
}
func (i Index) Affecting(p purl.URL) ([]Advisory, error) {
	if p.Ecosystem() == "" || p.Version == nil {
		return nil, nil
	}
	var result []Advisory
	for _, a := range i.Advisories {
		for _, affected := range a.Affected {
			if affected.Package.Ecosystem != p.Ecosystem() || affected.Package.Name != p.FullName() {
				continue
			}
			version, err := parseVersion(*p.Version)
			if err != nil {
				return nil, versionError(a.ID, *p.Version, p)
			}
			included, err := affected.includes(a.ID, p, version)
			if err != nil {
				return nil, err
			}
			if included {
				result = append(result, a)
				break
			}
		}
	}
	return result, nil
}
func (a Advisory) IDs() []string { return append([]string{a.ID}, a.Aliases...) }
func (a Advisory) Informational() *string {
	for _, item := range a.Affected {
		if item.Specific != nil && item.Specific.Informational != nil {
			return item.Specific.Informational
		}
	}
	return nil
}
func (a Affected) includes(id string, p purl.URL, version version) (bool, error) {
	if slices.Contains(a.Versions, *p.Version) {
		return true, nil
	}
	included, bounded := false, false
	for _, interval := range a.Ranges {
		if interval.Kind != "SEMVER" && interval.Kind != "ECOSYSTEM" {
			return false, fmt.Errorf("%s: unsupported %s range for %s", id, interval.Kind, p)
		}
		for _, event := range interval.Events {
			if event.Fixed != nil || event.LastAffected != nil || event.Limit != nil {
				bounded = true
			}
		}
		matched, err := interval.includes(id, p, version)
		if err != nil {
			return false, err
		}
		included = included || matched
	}
	if included && !bounded && a.Specific != nil && a.Specific.LastKnown != nil {
		limit := *a.Specific.LastKnown
		operator, text, ok := strings.Cut(strings.TrimSpace(limit), " ")
		bound, err := parseVersion(text)
		if !ok || err != nil || (operator != "<" && operator != "<=") {
			return false, versionError(id, limit, p)
		}
		order := compareVersion(version, bound)
		return order < 0 || operator == "<=" && order == 0, nil
	}
	return included, nil
}
func (r versionRange) includes(id string, p purl.URL, current version) (bool, error) {
	type boundary struct {
		at   version
		kind string
	}
	var events []boundary
	for _, event := range r.Events {
		var text *string
		kind := "fixed"
		switch {
		case event.Introduced != nil:
			text, kind = event.Introduced, "introduced"
		case event.Fixed != nil:
			text = event.Fixed
		case event.LastAffected != nil:
			text, kind = event.LastAffected, "last"
		case event.Limit != nil:
			text = event.Limit
		default:
			continue
		}
		parsed, err := parseVersion(*text)
		if err != nil {
			return false, versionError(id, *text, p)
		}
		events = append(events, boundary{parsed, kind})
	}
	slices.SortStableFunc(events, func(a, b boundary) int { return compareVersion(a.at, b.at) })
	affected := false
	for _, event := range events {
		order := compareVersion(current, event.at)
		switch event.kind {
		case "introduced":
			if order >= 0 {
				affected = true
			}
		case "fixed":
			if order >= 0 {
				affected = false
			}
		case "last":
			if order > 0 {
				affected = false
			}
		}
	}
	return affected, nil
}
func versionError(advisory, text string, p purl.URL) error {
	return fmt.Errorf("%s: unreadable version %q for %s", advisory, text, p)
}
