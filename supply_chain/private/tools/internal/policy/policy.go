package policy

import "github.com/latticebuild/rules_supply_chain/supply_chain/private/tools/internal/jsondata"

type Policy struct {
	Licenses   Licenses   `json:"licenses"`
	Advisories Advisories `json:"advisories" optional:"true"`
	Sources    Sources    `json:"sources" optional:"true"`
}
type Licenses struct {
	Allow     []string   `json:"allow"`
	Overrides []Override `json:"overrides" optional:"true"`
}
type Override struct {
	Package    string  `json:"package"`
	Expression string  `json:"expression"`
	SHA256     *string `json:"sha256"`
	Reason     string  `json:"reason"`
}
type Advisories struct {
	Ignore []Ignore `json:"ignore" optional:"true"`
}
type Ignore struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}
type Sources struct {
	AllowRegistry []string `json:"allow-registry" optional:"true"`
	AllowGit      []string `json:"allow-git" optional:"true"`
}

func Read(path string) (Policy, error) {
	var value Policy
	err := jsondata.ReadMode(path, &value, true)
	return value, err
}
