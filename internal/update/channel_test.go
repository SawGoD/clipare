package update

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestSemVerPrereleaseOrdering(t *testing.T) {
	ordered := []string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.1.0-beta.1"}
	for i := range ordered {
		for j := range ordered {
			if Newer(ordered[i], ordered[j]) != (i > j) {
				t.Fatal(ordered[i], ordered[j])
			}
		}
	}
	for _, invalid := range []string{"1.0.0-beta.01", "1.0.0-", "1.0.0-beta/1", "1.0.0+", "v1.00.0"} {
		if _, err := parseVersion(invalid); err == nil {
			t.Fatal("invalid version accepted", invalid)
		}
	}
	if Newer("1.0.0+second", "1.0.0+first") {
		t.Fatal("metadata affected precedence")
	}
}

func TestChannelDefaultsAndOverrides(t *testing.T) {
	for _, tc := range []struct{ preference, current, want string }{
		{"", "0.5.0", ChannelStable}, {"", "0.6.0-beta.1", ChannelBeta}, {"stable", "0.6.0-preview.1", ChannelStable}, {"beta", "0.5.0", ChannelBeta},
	} {
		if got := ResolveChannel(tc.preference, tc.current); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	if DisplayVersion("0.6.0-beta.1") != "0.6.0-beta.1 (бета)" || DisplayVersion("0.5.0") != "0.5.0" {
		t.Fatal("display label")
	}
}

func TestBetaSelectionUsesSemVerAndIgnoresDrafts(t *testing.T) {
	releases := []Release{
		{Tag: "v9.0.0", Draft: true}, {Tag: "v8.0.0-dev", Prerelease: true}, {Tag: "v0.6.0-beta.2", Prerelease: true},
		{Tag: "v0.6.0-beta.11", Prerelease: true}, {Tag: "v0.5.0"}, {Tag: "v7.0.0-beta.1"},
	}
	body, _ := json.Marshal(releases)
	g := GitHub{Client: httpFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatal(r.URL)
		}
		return response(200, string(body)), nil
	})}
	r, err := g.LatestForChannel(context.Background(), ChannelBeta)
	if err != nil || r.Version != "0.6.0-beta.11" {
		t.Fatal(r, err)
	}
	releases = append(releases, Release{Tag: "v0.6.0"})
	body, _ = json.Marshal(releases)
	r, err = g.LatestForChannel(context.Background(), ChannelBeta)
	if err != nil || r.Version != "0.6.0" {
		t.Fatal("stable promotion", r, err)
	}
}

func TestChannelCheckAndSwitchBypassesPreviousInterval(t *testing.T) {
	calls := 0
	g := GitHub{Client: httpFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() == API {
			return response(200, `{"tag_name":"v0.5.0"}`), nil
		}
		return response(200, `[{"tag_name":"v0.6.0-beta.2","prerelease":true}]`), nil
	})}
	c := Checker{GitHub: g, Current: "0.5.0", StatePath: filepath.Join(t.TempDir(), "state.json")}
	if r, err := c.Check(context.Background(), true, false); err != nil || r != nil {
		t.Fatal(r, err)
	}
	c.Channel = ChannelBeta
	if r, err := c.Check(context.Background(), true, false); err != nil || r == nil || r.Version != "0.6.0-beta.2" {
		t.Fatal(r, err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	c.Current = "0.6.0-beta.2"
	c.Channel = ChannelStable
	if r, err := c.Check(context.Background(), true, true); err != nil || r != nil {
		t.Fatal("downgrade offered", r, err)
	}
	c.Current = "0.6.0-dev"
	before := calls
	c.Check(context.Background(), true, true)
	if calls != before {
		t.Fatal("development check")
	}
}

func TestBetaPaginationAndErrors(t *testing.T) {
	calls := 0
	g := GitHub{Client: httpFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			res := response(200, `[{"tag_name":"v0.5.0"}]`)
			res.Header = http.Header{"Link": []string{`<https://evil.invalid>; rel="next"`}}
			return res, nil
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Host != "api.github.com" {
			t.Fatal(r.URL)
		}
		return response(200, `[{"tag_name":"v0.6.0-beta.1","prerelease":true}]`), nil
	})}
	if r, err := g.LatestForChannel(context.Background(), ChannelBeta); err != nil || r.Version != "0.6.0-beta.1" {
		t.Fatal(r, err)
	}
	for _, body := range []string{"bad", `{}`, `null`} {
		g.Client = httpFunc(func(*http.Request) (*http.Response, error) { return response(200, body), nil })
		if _, err := g.LatestForChannel(context.Background(), ChannelBeta); err == nil {
			t.Fatal("malformed API", body)
		}
	}
	for _, code := range []int{403, 429, 500} {
		g.Client = httpFunc(func(*http.Request) (*http.Response, error) { return response(code, "unavailable"), nil })
		if _, err := g.LatestForChannel(context.Background(), ChannelBeta); err == nil {
			t.Fatal("API failure ignored", code)
		}
	}
}

func TestBetaDefaultSchedulingDisabledAndManual(t *testing.T) {
	calls := 0
	g := GitHub{Client: httpFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return response(200, `[{"tag_name":"v0.6.0-beta.2","prerelease":true}]`), nil
	})}
	c := Checker{GitHub: g, Current: "0.6.0-beta.1", StatePath: filepath.Join(t.TempDir(), "state.json")}
	if r, err := c.Check(context.Background(), false, false); err != nil || r != nil || calls != 0 {
		t.Fatal("disabled beta check", r, err, calls)
	}
	if r, err := c.Check(context.Background(), true, false); err != nil || r == nil {
		t.Fatal(r, err)
	}
	c.Check(context.Background(), true, false)
	if calls != 1 {
		t.Fatal("beta interval ignored", calls)
	}
	if r, err := c.Check(context.Background(), false, true); err != nil || r == nil || calls != 2 {
		t.Fatal("manual bypass", r, err, calls)
	}
}
