package purl

import (
	"testing"
)

func TestPURLIdentityAndLossyDecoding(t *testing.T) {
	p, err := Parse("pkg:NPM/%40scope/name@1.0.0%2Bbuild?a=b%20c&a=last#ignored")
	if err != nil || p.Kind != "npm" || p.FullName() != "@scope/name" || *p.Version != "1.0.0+build" || p.Qualifiers["a"] != "last" {
		t.Fatalf("%+v %v", p, err)
	}
	for text, want := range map[string]string{"%FF": "�", "%FF%FF": "��", "%E1%80": "�", "%E1%80%FF": "��", "%ED%A0%80": "���", "a%zz": "a%zz"} {
		if got := percentDecode(text); got != want {
			t.Fatalf("%s=%q want=%q", text, got, want)
		}
	}
	specific, _ := Parse("pkg:npm/a@1.0.0")
	other, _ := Parse("pkg:npm/a@2.0.0")
	general, _ := Parse("pkg:npm/a")
	if specific.Covers(other) || !general.Covers(other) {
		t.Fatal("wrong version matching")
	}
}
