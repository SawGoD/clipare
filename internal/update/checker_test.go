package update

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type httpFunc func(*http.Request) (*http.Response, error)

func (f httpFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, s string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(s)), ContentLength: int64(len(s))}
}
func TestVersions(t *testing.T) {
	for _, v := range []struct {
		a, b string
		want bool
	}{{"0.10.0", "0.9.9", true}, {"v1.0.0", "0.9.0", true}, {"1.0.0", "1.0.0", false}, {"1.0.0", "2.0.0", false}, {"0.4.0", "0.4.0-dev", false}, {"0.4.0-beta.1", "0.3.0", false}, {"01.0.0", "0.3.0", false}, {"1.0.0+build", "0.3.0", true}} {
		if Newer(v.a, v.b) != v.want {
			t.Fatal(v)
		}
	}
}
func TestGitHubAndScheduling(t *testing.T) {
	calls := 0
	body := `{"tag_name":"v0.10.0","assets":[]}`
	code := 200
	g := GitHub{Client: httpFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != API {
			t.Fatal(r.URL)
		}
		return response(code, body), nil
	})}
	c := Checker{GitHub: g, Current: "0.9.0", StatePath: filepath.Join(t.TempDir(), "state.json")}
	if r, e := c.Check(context.Background(), false, false); r != nil || e != nil || calls != 0 {
		t.Fatal("disabled")
	}
	if r, e := c.Check(context.Background(), true, false); r == nil || e != nil {
		t.Fatal(r, e)
	}
	c.Check(context.Background(), true, false)
	if calls != 1 {
		t.Fatal("interval")
	}
	if c.Due(time.Now().Add(Interval+time.Second)) == false {
		t.Fatal("due")
	}
	c.Check(context.Background(), false, true)
	if calls != 2 {
		t.Fatal("manual")
	}
	for _, s := range []string{`{"tag_name":"v1.0.0","draft":true}`, `{"tag_name":"v1.0.0","prerelease":true}`, `{"tag_name":"v1.0.0-beta.1"}`} {
		body = s
		r, e := g.Latest(context.Background())
		if e != nil || r.Version != "" {
			t.Fatal(r, e)
		}
	}
	for _, status := range []int{403, 429, 500} {
		code = status
		if _, e := g.Latest(context.Background()); e == nil {
			t.Fatal(status)
		}
	}
	code = 200
	body = "invalid"
	if _, e := g.Latest(context.Background()); e == nil {
		t.Fatal("invalid JSON")
	}
	c.Current = "1.0.0-dev"
	before := calls
	c.Check(context.Background(), true, true)
	if calls != before {
		t.Fatal("dev")
	}
}
func TestPlatformAndHosts(t *testing.T) {
	r := Release{Version: "0.4.0", Tag: "v0.4.0"}
	for _, os := range []string{"darwin", "windows"} {
		for _, arch := range []string{"arm64", "amd64"} {
			name := "clipare-v0.4.0-" + os + "-" + arch + ".zip"
			r.Assets = append(r.Assets, Asset{Name: name, Size: 42, URL: "https://github.com/SawGoD/clipare/releases/download/v0.4.0/" + name})
		}
	}
	for _, os := range []string{"darwin", "windows"} {
		for _, arch := range []string{"arm64", "amd64"} {
			if _, e := r.Platform(os, arch); e != nil {
				t.Fatal(e)
			}
		}
	}
	if _, e := r.Platform("linux", "amd64"); e == nil {
		t.Fatal("unsupported")
	}
	r.Assets = nil
	if _, e := r.Platform("darwin", "arm64"); e == nil {
		t.Fatal("missing")
	}
	for _, s := range []string{"http://github.com/x", "https://github.com.evil/x", "https://evil/x", "https://user@github.com/x", "https://github.com:444/x"} {
		if AllowedURL(s) {
			t.Fatal(s)
		}
	}
	if !AllowedURL("https://release-assets.githubusercontent.com/x") {
		t.Fatal("asset CDN")
	}
	client := NewHTTPClient()
	req, _ := http.NewRequest("GET", "https://evil/x", nil)
	if client.CheckRedirect(req, nil) == nil {
		t.Fatal("redirect")
	}
}
