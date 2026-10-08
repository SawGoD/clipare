package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const API = "https://api.github.com/repos/SawGoD/clipare/releases/latest"
const MaxArchive int64 = 128 << 20

var ErrUnavailable = errors.New("Не удалось проверить обновления. Проверьте подключение и попробуйте позже")
var ErrPlatform = errors.New("Обновление для этой платформы недоступно")

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type Release struct {
	Version    string
	Tag        string  `json:"tag_name"`
	Notes      string  `json:"body"`
	URL        string  `json:"html_url"`
	Assets     []Asset `json:"assets"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
}
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func AllowedURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" && u.Port() != "443" {
		return false
	}
	switch u.Hostname() {
	case "api.github.com", "github.com", "objects.githubusercontent.com", "github-releases.githubusercontent.com", "release-assets.githubusercontent.com":
		return true
	}
	return false
}
func NewHTTPClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 10 * time.Second
	return &http.Client{Transport: t, Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 5 || !AllowedURL(r.URL.String()) {
			return errors.New("unexpected update redirect")
		}
		return nil
	}}
}

type GitHub struct{ Client HTTPClient }

func (g GitHub) request(ctx context.Context, raw string, max int64) (*http.Response, error) {
	if !AllowedURL(raw) {
		return nil, ErrUnavailable
	}
	r, e := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if e != nil {
		return nil, e
	}
	r.Header.Set("User-Agent", "Clipare-updater")
	r.Header.Set("Accept", "application/vnd.github+json")
	c := g.Client
	if c == nil {
		c = NewHTTPClient()
	}
	res, e := c.Do(r)
	if e != nil {
		return nil, ErrUnavailable
	}
	if res.StatusCode != 200 || res.ContentLength > max {
		res.Body.Close()
		return nil, ErrUnavailable
	}
	return res, nil
}
func (g GitHub) Latest(ctx context.Context) (Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, e := g.request(ctx, API, 1<<20)
	if e != nil {
		return Release{}, e
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if e != nil || len(b) > 1<<20 {
		return Release{}, ErrUnavailable
	}
	var r Release
	if json.Unmarshal(b, &r) != nil {
		return r, ErrUnavailable
	}
	if r.Draft || r.Prerelease {
		return Release{}, nil
	}
	if _, e = StableVersion(r.Tag); e != nil {
		return Release{}, nil
	}
	r.Version = r.Tag[1:]
	if r.Tag != "v"+r.Version {
		return Release{}, ErrUnavailable
	}
	if len(r.Assets) > 64 {
		return Release{}, ErrUnavailable
	}
	return r, nil
}
func (r Release) Asset(name string) (Asset, error) {
	var found Asset
	for _, a := range r.Assets {
		if a.Name == name {
			if found.Name != "" {
				return Asset{}, ErrPlatform
			}
			found = a
		}
	}
	if found.Name == "" || found.Size <= 0 || found.Size > MaxArchive || !AllowedURL(found.URL) {
		return Asset{}, ErrPlatform
	}
	// Initial URLs must belong to this repository/tag, not an arbitrary allowed host.
	if found.URL != "https://github.com/SawGoD/clipare/releases/download/"+r.Tag+"/"+name {
		return Asset{}, ErrPlatform
	}
	return found, nil
}
func (r Release) Platform(os, arch string) (Asset, error) {
	if (os != "darwin" && os != "windows") || (arch != "amd64" && arch != "arm64") {
		return Asset{}, ErrPlatform
	}
	if _, e := StableVersion(r.Version); e != nil || r.Tag != "v"+r.Version {
		return Asset{}, ErrPlatform
	}
	return r.Asset(fmt.Sprintf("clipare-%s-%s-%s.zip", r.Tag, os, arch))
}
