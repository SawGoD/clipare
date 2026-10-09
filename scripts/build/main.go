// Build desktop release archives using only Go and the platform SDK.
package main

import (
	"archive/zip"
	"clipare"
	"clipare/internal/releaseversion"
	"clipare/internal/update"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	targetOS := flag.String("os", runtime.GOOS, "darwin or windows")
	arch := flag.String("arch", runtime.GOARCH, "amd64 or arm64")
	out := flag.String("out", "dist", "output directory")
	flag.Parse()
	version := clipare.Version()
	parsedVersion, err := releaseversion.Parse(version)
	if err != nil {
		return err
	}
	if (*targetOS != "darwin" && *targetOS != "windows") || (*arch != "amd64" && *arch != "arm64") {
		return errors.New("unsupported target")
	}
	if *targetOS == "darwin" && runtime.GOOS != "darwin" {
		return errors.New("macOS builds require macOS and Xcode Command Line Tools")
	}
	revision, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil {
		return errors.New("cannot determine Git commit")
	}
	commit := strings.TrimSpace(string(revision))
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(commit) {
		return errors.New("invalid Git commit")
	}
	name := fmt.Sprintf("clipare-v%s-%s-%s", version, *targetOS, *arch)
	stage := filepath.Join(*out, name)
	if e = os.MkdirAll(stage, 0755); e != nil {
		return e
	}
	flags := "-s -w -X clipare.Commit=" + commit
	env := append(os.Environ(), "GOOS="+*targetOS, "GOARCH="+*arch, "CGO_ENABLED=0")
	build := func(path, ldflags string) error {
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", path, "./cmd/clipare")
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	if *targetOS == "darwin" {
		env = append(env, "CGO_ENABLED=1")
		bundle := filepath.Join(stage, "Clipare.app", "Contents")
		if e = os.MkdirAll(filepath.Join(bundle, "MacOS"), 0755); e != nil {
			return e
		}
		if e = build(filepath.Join(bundle, "MacOS", "Clipare"), flags); e != nil {
			return e
		}
		b, err := os.ReadFile("assets/Info.plist")
		if err != nil {
			return err
		}
		// Apple's bundle version fields require numeric components. The binary,
		// release metadata and human-readable bundle info retain the full SemVer.
		plist := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(b), "@VERSION@", parsedVersion.Core), "@BUILD@", parsedVersion.Core), "@RELEASE_VERSION@", version)
		if e = os.WriteFile(filepath.Join(bundle, "Info.plist"), []byte(plist), 0644); e != nil {
			return e
		}
		// Ad-hoc signing seals the bundle (required for Apple Silicon). Developer ID
		// signing/notarization is a separate process that requires Apple credentials.
		cmd := exec.Command("codesign", "--force", "--sign", "-", filepath.Join(stage, "Clipare.app"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e = cmd.Run(); e != nil {
			return e
		}
	} else {
		if e = build(filepath.Join(stage, "Clipare.exe"), flags+" -H=windowsgui"); e != nil {
			return e
		}
		if e = build(filepath.Join(stage, "clipare-console.exe"), flags); e != nil {
			return e
		}
	}
	if e = copyFile("README.md", filepath.Join(stage, "README.md")); e != nil {
		return e
	}
	// Version/platform metadata is covered by the archive's release checksum.
	pkg := update.Package{Version: version, OS: *targetOS, Arch: *arch}
	if e = filepath.Walk(stage, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) != update.PackageFile {
			pkg.Files = append(pkg.Files, filepath.ToSlash(rel))
		}
		return nil
	}); e != nil {
		return e
	}
	metadata, e := json.Marshal(pkg)
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(stage, update.PackageFile), metadata, 0644); e != nil {
		return e
	}
	archive := filepath.Join(*out, name+".zip")
	if e = archiveDir(stage, archive); e != nil {
		return e
	}
	f, e := os.Open(archive)
	if e != nil {
		return e
	}
	h := sha256.New()
	_, e = io.Copy(h, f)
	f.Close()
	if e != nil {
		return e
	}
	checksum := hex.EncodeToString(h.Sum(nil)) + "  " + filepath.Base(archive) + "\n"
	if e = os.WriteFile(archive+".sha256", []byte(checksum), 0644); e != nil {
		return e
	}
	fmt.Println(archive)
	return nil
}
func copyFile(from, to string) error {
	b, e := os.ReadFile(from)
	if e != nil {
		return e
	}
	return os.WriteFile(to, b, 0644)
}
func archiveDir(dir, archive string) (result error) {
	f, e := os.Create(archive)
	if e != nil {
		return e
	}
	defer func() {
		if e := f.Close(); result == nil {
			result = e
		}
	}()
	w := zip.NewWriter(f)
	defer func() {
		if e := w.Close(); result == nil {
			result = e
		}
	}()
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("unexpected non-regular file in archive")
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		header.Method = zip.Deflate
		dst, err := w.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(dst, src)
		return err
	})
}
