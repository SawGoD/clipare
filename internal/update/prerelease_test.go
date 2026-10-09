package update

import "testing"

func TestBetaRemainsOutsideStableUpdateChannel(t *testing.T) {
	for _, v := range []string{"0.5.0-beta.1", "0.5.0-preview.1"} {
		if _, err := StableVersion(v); err == nil {
			t.Fatal("prerelease accepted as stable")
		}
		if Newer(v, "0.4.1") || Newer("0.6.0", v) {
			t.Fatal("prerelease entered stable update channel")
		}
	}
}
