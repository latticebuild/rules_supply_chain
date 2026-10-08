package osv

import (
	"cmp"
	"errors"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

type version string

func parseVersion(text string) (version, error) {
	text = strings.TrimLeft(strings.TrimSpace(text), "v=")
	if text == "0" {
		text = "0.0.0-0"
	}
	end := strings.IndexAny(text, "-+")
	if end < 0 {
		end = len(text)
	}
	core, suffix := text[:end], text[end:]
	parts := strings.Split(core, ".")
	if len(parts) > 3 {
		return "", errors.New("invalid version core")
	}
	// Rust's core components are u64; prerelease numbers have no size limit.
	for _, part := range parts {
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return "", err
		}
	}
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	normalized := "v" + strings.Join(parts, ".") + suffix
	if !semver.IsValid(normalized) {
		return "", errors.New("invalid semantic version")
	}
	return version(normalized), nil
}
func digits(text string) bool {
	if text == "" {
		return false
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func compareVersion(a, b version) int {
	if order := semver.Compare(string(a), string(b)); order != 0 {
		return order
	}
	return compareBuild(strings.TrimPrefix(semver.Build(string(a)), "+"), strings.TrimPrefix(semver.Build(string(b)), "+"))
}
func compareBuild(a, b string) int {
	if a == b {
		return 0
	}
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < min(len(aa), len(bb)); i++ {
		x, y := aa[i], bb[i]
		xn, yn := digits(x) || x == "", digits(y) || y == ""
		order := 0
		switch {
		case xn && yn:
			xx, yy := strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
			order = cmp.Compare(len(xx), len(yy))
			if order == 0 {
				order = strings.Compare(xx, yy)
			}
			if order == 0 {
				order = cmp.Compare(len(x), len(y))
			}
		case xn:
			order = -1
		case yn:
			order = 1
		default:
			order = strings.Compare(x, y)
		}
		if order != 0 {
			return order
		}
	}
	return cmp.Compare(len(aa), len(bb))
}
