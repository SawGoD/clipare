package update

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeArchive(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	files := []string{"Clipare.app/Contents/MacOS/Clipare", "Clipare.app/Contents/Info.plist", "README.md"}
	p := Package{Version: "0.4.0", OS: "darwin", Arch: "arm64", Files: files}
	b, _ := json.Marshal(p)
	h := make([]byte, 32)
	binary.LittleEndian.PutUint32(h, 0xfeedfacf)
	binary.LittleEndian.PutUint32(h[4:], 0x100000c)
	binary.LittleEndian.PutUint32(h[12:], 2)
	for name, b := range map[string][]byte{PackageFile: b, files[0]: h, files[1]: []byte("<key>CFBundleShortVersionString</key><string>0.4.0</string>"), files[2]: []byte("test")} {
		f, e := w.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		f.Write(b)
	}
	w.Close()
	return out.Bytes()
}
func TestStageVerifiedArchive(t *testing.T) {
	archive := fakeArchive(t)
	h := sha256.Sum256(archive)
	sum := hex.EncodeToString(h[:])
	name := "clipare-v0.4.0-darwin-arm64.zip"
	for _, scenario := range []string{"success", "wrong checksum", "broken ZIP", "interrupted"} {
		t.Run(scenario, func(t *testing.T) {
			data := archive
			checksum := sum
			if scenario == "wrong checksum" {
				checksum = strings.Repeat("0", 64)
			}
			if scenario == "broken ZIP" {
				data = []byte("broken")
				h := sha256.Sum256(data)
				checksum = hex.EncodeToString(h[:])
			}
			sums := checksum + "  " + name + "\n"
			prefix := "https://github.com/SawGoD/clipare/releases/download/v0.4.0/"
			r := Release{Version: "0.4.0", Tag: "v0.4.0", Assets: []Asset{{Name: name, URL: prefix + name, Size: int64(len(data))}, {Name: "SHA256SUMS", URL: prefix + "SHA256SUMS", Size: int64(len(sums))}}}
			g := GitHub{Client: httpFunc(func(req *http.Request) (*http.Response, error) {
				if strings.HasSuffix(req.URL.Path, "SHA256SUMS") {
					return response(200, sums), nil
				}
				if scenario == "interrupted" {
					return response(200, string(data[:len(data)/2])), nil
				}
				return response(200, string(data)), nil
			})}
			cache := t.TempDir()
			s, e := (Downloader{GitHub: g}).stageFor(context.Background(), r, cache, "darwin", "arm64")
			if scenario == "success" {
				if e != nil {
					t.Fatal(e)
				}
				if s.Package.Version != "0.4.0" {
					t.Fatal("version")
				}
				if _, e = ValidatePackage(s.Root, "0.5.0", "darwin", "arm64"); e == nil {
					t.Fatal("wrong version")
				}
				return
			}
			if e == nil {
				t.Fatal("failure not detected")
			}
			entries, _ := os.ReadDir(cache)
			if len(entries) != 0 {
				t.Fatal("failed staging residue")
			}
		})
	}
}
func TestBundleTransaction(t *testing.T) {
	old, new := t.TempDir(), t.TempDir()
	for _, dir := range []string{old, new} {
		os.MkdirAll(filepath.Join(dir, "Clipare.app", "Contents", "Resources"), 0755)
	}
	os.WriteFile(filepath.Join(old, "Clipare.app", "Contents", "Resources", "old"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(new, "Clipare.app", "Contents", "Resources", "new"), []byte("new"), 0644)
	if e := replaceTransaction(new, old, []string{"Clipare.app"}, os.Rename, func() error { return nil }); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(old, "Clipare.app", "Contents", "Resources", "old")); !os.IsNotExist(e) {
		t.Fatal("bundle not replaced")
	}
	if _, e := os.Stat(filepath.Join(old, "Clipare.app", "Contents", "Resources", "new")); e != nil {
		t.Fatal("new resource")
	}
}
