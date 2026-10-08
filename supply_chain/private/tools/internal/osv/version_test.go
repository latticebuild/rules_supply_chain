package osv

import (
	"testing"
)

func TestRustVersionOrdering(t *testing.T) {
	for _, pair := range [][2]string{{"0", "0.0.0"}, {"1.0.0-999999999999999999999999", "1.0.0-1000000000000000000000000"}, {"1.0.0-1", "1.0.0-a"}, {"1.0.0", "1.0.0+0"}, {"1.0.0+0", "1.0.0+00"}, {"1.0.0+00", "1.0.0+1"}, {"1.0.0+1", "1.0.0+01"}, {"18446744073709551614.0.0", "18446744073709551615.0.0"}} {
		a, err := parseVersion(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		b, err := parseVersion(pair[1])
		if err != nil || compareVersion(a, b) >= 0 {
			t.Fatalf("%v: %v", pair, err)
		}
	}
	for _, value := range []string{"1.02.3", "1.2.3-01", "1.2.3+", "1.2.3-alpha..one", "18446744073709551616.0.0", "1.2.3.4", "1.2.3-é"} {
		if _, err := parseVersion(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	a, err := parseVersion(" v=1.2 ")
	b, _ := parseVersion("1.2.0")
	if err != nil || compareVersion(a, b) != 0 {
		t.Fatalf("normalization: %+v %v", a, err)
	}
}
