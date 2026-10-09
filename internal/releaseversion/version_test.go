package releaseversion

import "testing"

func TestReleaseVersions(t *testing.T) {
	for _, s := range []string{"0.5.0", "0.5.0-beta.1", "0.6.0-preview.2", "1.0.0-rc.0", "1.0.0-beta-x.1"} {
		v, err := Parse(s)
		if err != nil || v.Core == "" {
			t.Fatalf("%s: %+v %v", s, v, err)
		}
		if v.Prerelease != (s != v.Core) {
			t.Fatal("wrong prerelease classification")
		}
	}
	for _, s := range []string{"", "v0.5.0", "01.2.3", "1.2", "1.2.3-", "1.2.3-beta..1", "1.2.3-beta.01", "1.2.3+meta", "1.2.3; command", "1.2.3-beta/evil", "1.2.3\n", "18446744073709551616.0.0"} {
		if _, err := Parse(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
}
