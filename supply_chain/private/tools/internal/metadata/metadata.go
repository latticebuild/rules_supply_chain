package metadata

import (
	"slices"

	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"
	"github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/purl"
)

type Package struct {
	URL           purl.URL
	Licence, Text *string
}

func Read(path string) (Package, error) {
	var metadata struct {
		Purl       string            `json:"purl"`
		Attributes map[string]string `json:"attributes" optional:"true"`
	}
	if err := jsondata.Read(path, &metadata); err != nil {
		return Package{}, err
	}
	url, err := purl.Parse(metadata.Purl)
	if err != nil {
		return Package{}, err
	}
	result := Package{URL: url}
	if path, ok := metadata.Attributes["build.bazel.attribute.license"]; ok {
		var attribute struct {
			Kind struct {
				Identifier string `json:"identifier"`
			} `json:"kind"`
			Text *string `json:"text"`
		}
		if err := jsondata.Read(path, &attribute); err != nil {
			return result, err
		}
		result.Licence, result.Text = &attribute.Kind.Identifier, attribute.Text
	}
	return result, nil
}
func Unique(packages []Package) []Package {
	slices.SortStableFunc(packages, func(a, b Package) int { return purl.Compare(a.URL, b.URL) })
	return slices.CompactFunc(packages, func(a, b Package) bool { return purl.Compare(a.URL, b.URL) == 0 })
}
