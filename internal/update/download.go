package update

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var ErrDownload = errors.New("Не удалось загрузить обновление. Попробуйте ещё раз")

type Downloader struct {
	GitHub   GitHub
	Verifier ManifestVerifier
}

func (d Downloader) download(ctx context.Context, a Asset, path string, max int64) error {
	if a.Size <= 0 || a.Size > max {
		return ErrDownload
	}
	res, e := d.GitHub.request(ctx, a.URL, max)
	if e != nil {
		return ErrDownload
	}
	defer res.Body.Close()
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	n, e := io.Copy(f, io.LimitReader(res.Body, max+1))
	if e == nil && n != a.Size {
		e = ErrDownload
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		os.Remove(path)
		return ErrDownload
	}
	return nil
}

// Stage owns a fresh private temporary directory. No installation is touched.
func (d Downloader) Stage(ctx context.Context, r Release, cache string) (*Staged, error) {
	a, e := r.Platform(runtime.GOOS, runtime.GOARCH)
	if e != nil {
		return nil, e
	}
	sums, e := r.Asset("SHA256SUMS")
	if e != nil {
		return nil, ErrVerify
	}
	if e = os.MkdirAll(cache, 0700); e != nil {
		return nil, e
	}
	work, e := os.MkdirTemp(cache, "stage-")
	if e != nil {
		return nil, e
	}
	mark, _ := json.Marshal(struct {
		Created time.Time `json:"created"`
	}{time.Now().UTC()})
	if e = os.WriteFile(filepath.Join(work, ".clipare-stage"), mark, 0600); e != nil {
		os.RemoveAll(work)
		return nil, e
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(work)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	checksumPath := filepath.Join(work, "SHA256SUMS")
	if e = d.download(ctx, sums, checksumPath, 64<<10); e != nil {
		return nil, e
	}
	b, e := os.ReadFile(checksumPath)
	if e != nil {
		return nil, e
	}
	verifier := d.Verifier
	if verifier == nil {
		verifier = IntegrityOnly{}
	}
	var signature []byte
	if verifier.RequiresSignature() {
		sig, e := r.Asset("SHA256SUMS.sig")
		if e != nil {
			return nil, ErrVerify
		}
		p := filepath.Join(work, "SHA256SUMS.sig")
		if e = d.download(ctx, sig, p, 1024); e != nil {
			return nil, e
		}
		signature, e = os.ReadFile(p)
		if e != nil {
			return nil, e
		}
	}
	if e = verifier.Verify(b, signature); e != nil {
		return nil, e
	}
	hashes, e := Checksums(b)
	if e != nil || hashes[a.Name] == "" {
		return nil, ErrVerify
	}
	archive := filepath.Join(work, a.Name)
	if e = d.download(ctx, a, archive, MaxArchive); e != nil {
		return nil, e
	}
	if e = VerifyFile(archive, hashes[a.Name]); e != nil {
		return nil, e
	}
	extracted := filepath.Join(work, "package")
	if e = Extract(ctx, archive, extracted); e != nil {
		return nil, e
	}
	pkg, e := ValidatePackage(extracted, r.Version, runtime.GOOS, runtime.GOARCH)
	if e != nil {
		return nil, e
	}
	ok = true
	return &Staged{Work: work, Root: extracted, Package: pkg}, nil
}
