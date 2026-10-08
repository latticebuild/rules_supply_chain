package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type licenceRegistry struct {
	licences, exceptions, gnu map[string]bool
	aliases                   [][2]string
}

var registry = sync.OnceValues(func() (*licenceRegistry, error) {
	read := func(name string, target any) error {
		path := "assets/spdx/" + name
		if name == "licenses.json" || name == "exceptions.json" {
			path = "assets/generated/spdx/" + name
		}
		data, err := spdxFiles.ReadFile(path)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var licences struct {
		Licenses []struct {
			ID string `json:"licenseId"`
		} `json:"licenses"`
	}
	var exceptions struct {
		Exceptions []struct {
			ID string `json:"licenseExceptionId"`
		} `json:"exceptions"`
	}
	var compatibility struct {
		Licenses []string    `json:"licenses"`
		GNU      []string    `json:"gnuLicenses"`
		Aliases  [][2]string `json:"aliases"`
	}
	if err := read("licenses.json", &licences); err != nil {
		return nil, err
	}
	if err := read("exceptions.json", &exceptions); err != nil {
		return nil, err
	}
	if err := read("compatibility.json", &compatibility); err != nil {
		return nil, err
	}
	r := &licenceRegistry{licences: map[string]bool{}, exceptions: map[string]bool{}, gnu: map[string]bool{}, aliases: compatibility.Aliases}
	for _, item := range licences.Licenses {
		r.licences[item.ID] = true
	}
	for _, item := range compatibility.Licenses {
		r.licences[item] = true
	}
	for _, item := range exceptions.Exceptions {
		r.exceptions[item.ID] = true
	}
	for _, item := range compatibility.GNU {
		r.gnu[item] = true
	}
	return r, nil
})

type licenceToken struct {
	kind, text string
	start, end int
}
type licenceLexer struct {
	text     string
	offset   int
	registry *licenceRegistry
}

func refChar(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.'
}
func reference(text, prefix string) int {
	offset := 0
	if strings.HasPrefix(text, "DocumentRef-") {
		offset = len("DocumentRef-")
		for offset < len(text) && refChar(text[offset]) {
			offset++
		}
		if offset == len(text) || text[offset] != ':' {
			return 0
		}
		offset++
	}
	if !strings.HasPrefix(text[offset:], prefix) {
		return 0
	}
	offset += len(prefix)
	for offset < len(text) && refChar(text[offset]) {
		offset++
	}
	return offset
}
func (l *licenceLexer) error(start, end int, reason string) error {
	marker := strings.Repeat("^", max(0, end-start))
	if reason == "unclosed parens" {
		marker = "-"
	}
	return fmt.Errorf("%s\n%s%s %s", l.text, strings.Repeat(" ", start), marker, reason)
}
func (l *licenceLexer) next() (licenceToken, error) {
	before := l.offset
	for l.offset < len(l.text) {
		c, size := utf8.DecodeRuneInString(l.text[l.offset:])
		if !unicode.IsSpace(c) {
			break
		}
		l.offset += size
	}
	start := l.offset
	if start == len(l.text) {
		return licenceToken{kind: "end", start: start, end: start}, nil
	}
	rest := l.text[start:]
	length := 1
	token := licenceToken{start: start}
	switch rest[0] {
	case '+':
		if before != start {
			return token, l.error(before, start, "`+` must not follow whitespace")
		}
		token.kind = "+"
	case '(':
		token.kind = "("
	case ')':
		token.kind = ")"
	case '/':
		token.kind = "OR"
	default:
		length = 0
		for length < len(rest) && (refChar(rest[length]) || rest[length] == ':') {
			length++
		}
		if length == 0 {
			return token, l.error(start, len(l.text), "invalid character(s)")
		}
		word := rest[:length]
		token.text = word
		switch {
		case word == "AND" || word == "and":
			token.kind = "AND"
		case word == "OR" || word == "or":
			token.kind = "OR"
		case word == "WITH" || word == "with":
			token.kind = "WITH"
		case l.registry.licences[word]:
			token.kind = "licence"
		case l.registry.exceptions[word]:
			token.kind = "exception"
		case reference(word, "LicenseRef-") > 0:
			token.kind = "licref"
			length = reference(word, "LicenseRef-")
			token.text = rest[:length]
		case reference(word, "AdditionRef-") > 0:
			token.kind = "addref"
			length = reference(word, "AdditionRef-")
			token.text = rest[:length]
		default:
			for _, alias := range l.registry.aliases {
				if len(rest) >= len(alias[0]) && strings.EqualFold(rest[:len(alias[0])], alias[0]) {
					token.kind, token.text = "licence", alias[1]
					length = len(alias[0])
					break
				}
			}
			if token.kind == "" {
				return token, l.error(start, start+length, "unknown term")
			}
		}
	}
	l.offset += length
	token.end = l.offset
	return token, nil
}

type requirement struct {
	licence, addition string
	later, other      bool
}

func (r requirement) String() string {
	result := r.licence
	if r.later {
		result += "+"
	}
	if r.addition != "" {
		result += " WITH " + r.addition
	}
	return result
}
func (a requirement) satisfies(r requirement) bool {
	if a.other != r.other || a.addition != r.addition {
		return false
	}
	if a.licence == r.licence {
		return true
	}
	if a.other || !r.later {
		return false
	}
	aa, bb := strings.Split(a.licence, "-"), strings.Split(r.licence, "-")
	if len(aa) != len(bb) {
		return false
	}
	numeric := func(s string) bool {
		for _, c := range s {
			if c != '.' && (c < '0' || c > '9') {
				return false
			}
		}
		return true
	}
	for i, x := range aa {
		y := bb[i]
		if x != y && (!numeric(x) || !numeric(y) || x <= y) {
			return false
		}
	}
	return true
}

type Expression struct {
	Text string
	root *expressionNode
}
type expressionNode struct {
	requirement *requirement
	operator    string
	left, right *expressionNode
}
type licenceParser struct {
	lexer  licenceLexer
	cached *licenceToken
	last   string
}

func (p *licenceParser) peek() (licenceToken, error) {
	if p.cached != nil {
		return *p.cached, nil
	}
	t, err := p.lexer.next()
	if err == nil {
		p.cached = &t
	}
	return t, err
}
func (p *licenceParser) take() (licenceToken, error) {
	t, err := p.peek()
	p.cached = nil
	if err == nil {
		p.last = t.kind
	}
	return t, err
}
func (p *licenceParser) unexpected(t licenceToken, expected ...string) error {
	reason := "the term was not expected here"
	if len(expected) == 1 {
		reason = "expected a `" + expected[0] + "` here"
	} else if len(expected) > 1 {
		reason = "expected one of `" + strings.Join(expected, "`, `") + "` here"
	}
	return p.lexer.error(t.start, t.end, reason)
}
func Parse(text string) (Expression, error) {
	r, err := registry()
	if err != nil {
		return Expression{}, err
	}
	p := licenceParser{lexer: licenceLexer{text: text, registry: r}}
	first, err := p.peek()
	if err != nil {
		return Expression{}, err
	}
	if first.kind == "end" {
		return Expression{}, p.lexer.error(0, len(text), "empty expression")
	}
	root, err := p.binary(1)
	if err != nil {
		return Expression{}, err
	}
	next, err := p.peek()
	if err != nil {
		return Expression{}, err
	}
	if next.kind != "end" {
		if next.kind == ")" {
			return Expression{}, p.lexer.error(next.start, next.end, "unopened parens")
		}
		return Expression{}, p.unexpected(next, p.expected()...)
	}
	return Expression{text, root}, nil
}
func (p *licenceParser) binary(minimum int) (*expressionNode, error) {
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		token, err := p.peek()
		if err != nil {
			return nil, err
		}
		precedence := 0
		if token.kind == "OR" {
			precedence = 1
		}
		if token.kind == "AND" {
			precedence = 2
		}
		if precedence < minimum {
			return left, nil
		}
		_, _ = p.take()
		right, err := p.binary(precedence + 1)
		if err != nil {
			return nil, err
		}
		left = &expressionNode{operator: token.kind, left: left, right: right}
	}
}
func (p *licenceParser) primary() (*expressionNode, error) {
	token, err := p.take()
	if err != nil {
		return nil, err
	}
	if token.kind == "(" {
		node, err := p.binary(1)
		if err != nil {
			return nil, err
		}
		close, err := p.peek()
		if err != nil {
			return nil, err
		}
		if close.kind == "end" {
			return nil, p.lexer.error(token.start, token.end, "unclosed parens")
		}
		if close.kind != ")" {
			return nil, p.unexpected(close, p.expected()...)
		}
		_, _ = p.take()
		return node, nil
	}
	if token.kind != "licence" && token.kind != "licref" {
		return nil, p.unexpected(token, "<license>", "(")
	}
	req := requirement{licence: token.text, other: token.kind == "licref"}
	next, err := p.peek()
	if err != nil {
		return nil, err
	}
	if next.kind == "+" {
		if req.other {
			return nil, p.unexpected(next, "AND", "OR", "WITH", ")")
		}
		_, _ = p.take()
		if p.lexer.registry.gnu[req.licence] {
			if strings.HasSuffix(req.licence, "-or-later") {
				return nil, p.lexer.error(next.start, next.end, "a GNU license was followed by a `+` even though it ended in `-only` or `-or-later`")
			}
			req.licence = strings.TrimSuffix(req.licence, "-only") + "-or-later"
			if !p.lexer.registry.licences[req.licence] {
				return nil, p.lexer.error(next.start, next.end, "unknown license id")
			}
		} else {
			req.later = true
		}
		next, err = p.peek()
		if err != nil {
			return nil, err
		}
	}
	if next.kind == "WITH" {
		_, _ = p.take()
		addition, err := p.take()
		if err != nil {
			return nil, err
		}
		if addition.kind != "exception" && addition.kind != "addref" {
			return nil, p.unexpected(addition, "<addition>")
		}
		req.addition = addition.text
	}
	return &expressionNode{requirement: &req}, nil
}
func parseLicensee(text string) (requirement, error) {
	r, err := registry()
	if err != nil {
		return requirement{}, err
	}
	l := licenceLexer{text: text, registry: r}
	token, err := l.next()
	if err != nil {
		return requirement{}, err
	}
	if token.kind != "licence" && token.kind != "licref" {
		return requirement{}, l.error(token.start, token.end, "expected a `<license>` here")
	}
	req := requirement{licence: token.text, other: token.kind == "licref"}
	next, err := l.next()
	if err != nil {
		return req, err
	}
	if next.kind == "end" {
		return req, nil
	}
	if next.kind != "WITH" {
		return req, l.error(next.start, next.end, "expected a `WITH` here")
	}
	addition, err := l.next()
	if err != nil {
		return req, err
	}
	if addition.kind != "exception" && addition.kind != "addref" {
		return req, l.error(addition.start, addition.end, "expected a `<addition>` here")
	}
	req.addition = addition.text
	// Licensee's previous parser stops after its exception. Keep that acceptance
	// behavior distinct from the full expression grammar.
	return req, nil
}
func Allowlist(entries []string) ([]requirement, error) {
	result := make([]requirement, 0, len(entries))
	for _, entry := range entries {
		parsed, err := parseLicensee(entry)
		if err != nil {
			return nil, fmt.Errorf("allowed licence %q: %w", entry, err)
		}
		result = append(result, parsed)
	}
	return result, nil
}
func (e Expression) Refused(allow []requirement) []string {
	var failures []string
	var evaluate func(*expressionNode) bool
	evaluate = func(node *expressionNode) bool {
		if node.requirement != nil {
			for _, accepted := range allow {
				if accepted.satisfies(*node.requirement) {
					return true
				}
			}
			failures = append(failures, node.requirement.String())
			return false
		}
		left, right := evaluate(node.left), evaluate(node.right)
		if node.operator == "AND" {
			return left && right
		}
		return left || right
	}
	if evaluate(e.root) {
		return nil
	}
	return failures
}

func (p *licenceParser) expected() []string {
	switch p.last {
	case ")":
		return []string{"AND", "OR"}
	case "exception", "addref":
		return []string{"AND", "OR", ")"}
	case "licence":
		return []string{"AND", "OR", "WITH", ")", "+"}
	default:
		return []string{"AND", "OR", "WITH", ")"}
	}
}
