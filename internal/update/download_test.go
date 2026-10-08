package update

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksums(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x")
	os.WriteFile(path, []byte("hello"), 0600)
	sum, _ := FileSHA256(path)
	m, e := Checksums([]byte(sum + "  x.zip\n"))
	if e != nil || m["x.zip"] != sum {
		t.Fatal(e)
	}
	if e = VerifyFile(path, sum); e != nil {
		t.Fatal(e)
	}
	if VerifyFile(path, strings.Repeat("0", 64)) == nil {
		t.Fatal("wrong checksum")
	}
	for _, s := range []string{"", sum + " x.zip", sum + "  ../evil", sum + "  x.zip\n" + sum + "  x.zip\n", strings.Repeat("g", 64) + "  x.zip\n"} {
		if _, e = Checksums([]byte(s)); e == nil {
			t.Fatal(s)
		}
	}
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	v := Ed25519Verifier{pub}
	msg := []byte("manifest")
	if v.Verify(msg, ed25519.Sign(priv, msg)) != nil || v.Verify([]byte("bad"), ed25519.Sign(priv, msg)) == nil {
		t.Fatal("signature")
	}
}
func testZIP(t *testing.T, name string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.zip")
	f, _ := os.Create(p)
	w := zip.NewWriter(f)
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(mode)
	out, e := w.CreateHeader(h)
	if e != nil {
		t.Fatal(e)
	}
	out.Write([]byte("data"))
	w.Close()
	f.Close()
	return p
}
func TestExtractSecurity(t *testing.T) {
	for _, name := range []string{"../evil", "../../file", "/absolute", "C:/evil", "a\\evil", "a/../b", "CON", "safe/COM1.exe", "a/./b"} {
		t.Run(name, func(t *testing.T) {
			if Extract(context.Background(), testZIP(t, name, 0644), filepath.Join(t.TempDir(), "pkg")) == nil {
				t.Fatal("unsafe path")
			}
		})
	}
	if Extract(context.Background(), testZIP(t, "link", os.ModeSymlink|0777), filepath.Join(t.TempDir(), "pkg")) == nil {
		t.Fatal("symlink")
	}
	dest := filepath.Join(t.TempDir(), "pkg")
	if e := Extract(context.Background(), testZIP(t, "dir/file", 0755), dest); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(dest, "dir", "file"))
	if e != nil || string(b) != "data" {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if Extract(ctx, testZIP(t, "file", 0644), filepath.Join(t.TempDir(), "pkg")) == nil {
		t.Fatal("cancel")
	}
}
func TestDownloadStreamingLimits(t *testing.T) {
	data := "archive"
	a := Asset{Name: "x.zip", Size: int64(len(data)), URL: "https://github.com/SawGoD/clipare/releases/download/v1.0.0/x.zip"}
	d := Downloader{GitHub: GitHub{Client: httpFunc(func(r *http.Request) (*http.Response, error) { return response(200, data), nil })}}
	path := filepath.Join(t.TempDir(), "x.zip")
	if e := d.download(context.Background(), a, path, 10); e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256([]byte(data))
	if VerifyFile(path, hex.EncodeToString(h[:])) != nil {
		t.Fatal("download hash")
	}
	for _, size := range []int64{1, 100} {
		a.Size = size
		p := filepath.Join(t.TempDir(), "bad")
		if d.download(context.Background(), a, p, 10) == nil {
			t.Fatal(fmt.Sprint(size))
		}
		if _, e := os.Stat(p); !os.IsNotExist(e) {
			t.Fatal("failed file remains")
		}
	}
}
