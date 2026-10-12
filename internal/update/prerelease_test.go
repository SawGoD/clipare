package update

import "testing"

func TestBetaRemainsOutsideStableUpdateChannel(t *testing.T) {
	for _, v := range []string{"0.5.0-beta.1", "0.5.0-preview.1"} {
		if _, err := StableVersion(v); err == nil {
			t.Fatal("prerelease accepted as stable")
		}
		if eligibleRelease(Release{Tag: "v" + v, Prerelease: true}, ChannelStable).Version != "" {
			t.Fatal("prerelease entered stable channel")
		}
	}
}
