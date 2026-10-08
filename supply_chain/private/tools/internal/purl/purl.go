package purl

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

type URL struct {
	Kind       string
	namespace  *string
	name       string
	Version    *string
	Qualifiers map[string]string
}

func Parse(text string) (URL, error) {
	bad := func(reason string) (URL, error) {
		return URL{}, fmt.Errorf("%s: not a package URL: %s", text, reason)
	}
	rest, ok := strings.CutPrefix(text, "pkg:")
	if !ok {
		return bad("no pkg: scheme")
	}
	rest, _, _ = strings.Cut(rest, "#")
	rest, query, _ := strings.Cut(rest, "?")
	kind, path, ok := strings.Cut(rest, "/")
	if !ok {
		return bad("no name")
	}
	result := URL{Kind: asciiLower(kind), Qualifiers: map[string]string{}}
	for _, pair := range strings.Split(query, "&") {
		if key, value, ok := strings.Cut(pair, "="); ok {
			result.Qualifiers[asciiLower(percentDecode(key))] = percentDecode(value)
		}
	}
	if at := strings.LastIndexByte(path, '@'); at > 0 {
		version := percentDecode(path[at+1:])
		result.Version = &version
		path = path[:at]
	}
	if slash := strings.LastIndexByte(path, '/'); slash >= 0 {
		namespace := percentDecode(path[:slash])
		result.namespace = &namespace
		path = path[slash+1:]
	}
	result.name = percentDecode(path)
	if result.name == "" {
		return bad("no name")
	}
	return result, nil
}
func asciiLower(value string) string {
	return strings.Map(func(c rune) rune {
		if c >= 'A' && c <= 'Z' {
			return c + ('a' - 'A')
		}
		return c
	}, value)
}
func percentDecode(value string) string {
	hex := func(c byte) (byte, bool) {
		switch {
		case c >= '0' && c <= '9':
			return c - '0', true
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10, true
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10, true
		}
		return 0, false
	}
	var out []byte
	for i := 0; i < len(value); i++ {
		if value[i] == '%' && i+2 < len(value) {
			a, aok := hex(value[i+1])
			b, bok := hex(value[i+2])
			if aok && bok {
				out = append(out, a*16+b)
				i += 2
				continue
			}
		}
		out = append(out, value[i])
	}
	return lossyUTF8(out)
}
func (p URL) FullName() string {
	if p.namespace != nil {
		return *p.namespace + "/" + p.name
	}
	return p.name
}
func (p URL) String() string {
	name := p.FullName()
	if p.Version != nil {
		name += "@" + *p.Version
	}
	return name
}
func (p URL) Ecosystem() string {
	switch p.Kind {
	case "cargo":
		return "crates.io"
	case "npm":
		return "npm"
	}
	return ""
}
func sameOptional(a, b *string) bool { return compareOptional(a, b) == 0 }
func compareOptional(a, b *string) int {
	if a == nil {
		if b == nil {
			return 0
		}
		return -1
	}
	if b == nil {
		return 1
	}
	return strings.Compare(*a, *b)
}
func (p URL) Covers(other URL) bool {
	return p.Kind == other.Kind && sameOptional(p.namespace, other.namespace) && p.name == other.name && (p.Version == nil || sameOptional(p.Version, other.Version))
}
func Compare(a, b URL) int {
	for _, order := range []int{strings.Compare(a.Kind, b.Kind), compareOptional(a.namespace, b.namespace), strings.Compare(a.name, b.name), compareOptional(a.Version, b.Version)} {
		if order != 0 {
			return order
		}
	}
	ak, bk := slices.Sorted(maps.Keys(a.Qualifiers)), slices.Sorted(maps.Keys(b.Qualifiers))
	for i := 0; i < min(len(ak), len(bk)); i++ {
		if order := strings.Compare(ak[i], bk[i]); order != 0 {
			return order
		}
		if order := strings.Compare(a.Qualifiers[ak[i]], b.Qualifiers[bk[i]]); order != 0 {
			return order
		}
	}
	return cmp.Compare(len(ak), len(bk))
}

// Rust replaces one invalid UTF-8 sequence at a time, including a truncated
// valid prefix; collapsing an entire invalid run would merge distinct PURLs.
func lossyUTF8(data []byte) string {
	var out strings.Builder
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r != utf8.RuneError || size != 1 {
			out.Write(data[:size])
			data = data[size:]
			continue
		}
		size = 1
		width := 0
		switch {
		case data[0] >= 0xC2 && data[0] <= 0xDF:
			width = 2
		case data[0] >= 0xE0 && data[0] <= 0xEF:
			width = 3
		case data[0] >= 0xF0 && data[0] <= 0xF4:
			width = 4
		}
		for size < width && size < len(data) {
			b := data[size]
			if b < 0x80 || b > 0xBF {
				break
			}
			if size == 1 && (data[0] == 0xE0 && b < 0xA0 || data[0] == 0xED && b > 0x9F || data[0] == 0xF0 && b < 0x90 || data[0] == 0xF4 && b > 0x8F) {
				break
			}
			size++
		}
		out.WriteRune(utf8.RuneError)
		data = data[size:]
	}
	return out.String()
}
