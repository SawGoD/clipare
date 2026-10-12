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
	return fakeArchiveVersion(t, "0.4.0")
}

func fakeArchiveVersion(t *testing.T, version string) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	files := []string{"Clipare.app/Contents/MacOS/Clipare", "Clipare.app/Contents/Info.plist", "README.md"}
	p := Package{Version: version, OS: "darwin", Arch: "arm64", Files: files}
	b, _ := json.Marshal(p)
	h := make([]byte, 32)
	binary.LittleEndian.PutUint32(h, 0xfeedfacf)
	binary.LittleEndian.PutUint32(h[4:], 0x100000c)
	binary.LittleEndian.PutUint32(h[12:], 2)
	parsed, err := packageVersion(version)
	if err != nil {
		t.Fatal(err)
	}
	plist := "<key>CFBundleShortVersionString</key><string>" + parsed.Core + "</string><key>CFBundleGetInfoString</key><string>Clipare " + version + "</string>"
	for name, b := range map[string][]byte{PackageFile: b, files[0]: h, files[1]: []byte(plist), files[2]: []byte("test")} {
		f, e := w.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		f.Write(b)
	}
	w.Close()
	return out.Bytes()
}

func TestStageBetaPackageChecksFullBundleVersion(t *testing.T) {
	version := "0.6.0-beta.2"
	archive := fakeArchiveVersion(t, version)
	h := sha256.Sum256(archive)
	name := "clipare-v" + version + "-darwin-arm64.zip"
	sums := hex.EncodeToString(h[:]) + "  " + name + "\n"
	prefix := "https://github.com/SawGoD/clipare/releases/download/v" + version + "/"
	r := Release{Version: version, Tag: "v" + version, Prerelease: true, Assets: []Asset{{Name: name, URL: prefix + name, Size: int64(len(archive))}, {Name: "SHA256SUMS", URL: prefix + "SHA256SUMS", Size: int64(len(sums))}}}
	g := GitHub{Client: httpFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "SHA256SUMS") {
			return response(200, sums), nil
		}
		return response(200, string(archive)), nil
	})}
	s, err := (Downloader{GitHub: g}).stageFor(context.Background(), r, t.TempDir(), "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(s.Root, "Clipare.app", "Contents", "Info.plist")
	if err := os.WriteFile(plist, []byte("<key>CFBundleShortVersionString</key><string>0.6.0</string><key>CFBundleGetInfoString</key><string>Clipare 0.6.0-beta.1</string>"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePackage(s.Root, version, "darwin", "arm64"); err == nil {
		t.Fatal("wrong beta bundle accepted")
	}
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
