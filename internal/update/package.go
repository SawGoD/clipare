package update

import (
	"debug/macho"
	"debug/pe"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const PackageFile = "update-package.json"

type Package struct {
	Version string   `json:"version"`
	OS      string   `json:"os"`
	Arch    string   `json:"arch"`
	Files   []string `json:"files"`
}
type Staged struct {
	Work, Root string
	Package    Package
}

func ValidatePackage(root, version, targetOS, arch string) (Package, error) {
	var p Package
	b, e := os.ReadFile(filepath.Join(root, PackageFile))
	if e != nil || len(b) > 128<<10 || json.Unmarshal(b, &p) != nil {
		return p, ErrPackage
	}
	if p.Version != version || p.OS != targetOS || p.Arch != arch || len(p.Files) == 0 || len(p.Files) > 4096 {
		return p, ErrPackage
	}
	if _, e = StableVersion(p.Version); e != nil {
		return p, ErrPackage
	}
	wanted := map[string]bool{PackageFile: true}
	for _, f := range p.Files {
		if !safeRelative(f) || wanted[f] {
			return p, ErrPackage
		}
		wanted[f] = true
	}
	e = filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return ErrPackage
		}
		rel, e := filepath.Rel(root, path)
		if e != nil || !wanted[filepath.ToSlash(rel)] {
			return ErrPackage
		}
		delete(wanted, filepath.ToSlash(rel))
		return nil
	})
	if e != nil || len(wanted) != 0 {
		return p, ErrPackage
	}
	if targetOS == "darwin" {
		for _, f := range p.Files {
			if f != "README.md" && !strings.HasPrefix(f, "Clipare.app/") {
				return p, ErrPackage
			}
		}
		b, e = os.ReadFile(filepath.Join(root, "Clipare.app", "Contents", "Info.plist"))
		if e != nil || len(b) > 64<<10 || !strings.Contains(string(b), "<key>CFBundleShortVersionString</key><string>"+version+"</string>") {
			return p, ErrPackage
		}
		f, e := macho.Open(filepath.Join(root, "Clipare.app", "Contents", "MacOS", "Clipare"))
		if e != nil {
			return p, ErrPackage
		}
		defer f.Close()
		if arch == "arm64" && f.Cpu != macho.CpuArm64 || arch == "amd64" && f.Cpu != macho.CpuAmd64 {
			return p, ErrPackage
		}
	} else if targetOS == "windows" {
		for _, name := range []string{"Clipare.exe", "clipare-console.exe"} {
			f, e := pe.Open(filepath.Join(root, name))
			if e != nil {
				return p, ErrPackage
			}
			valid := arch == "arm64" && f.Machine == pe.IMAGE_FILE_MACHINE_ARM64 || arch == "amd64" && f.Machine == pe.IMAGE_FILE_MACHINE_AMD64
			f.Close()
			if !valid {
				return p, ErrPackage
			}
		}
	} else {
		return p, ErrPackage
	}
	return p, nil
}
