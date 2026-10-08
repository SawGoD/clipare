package update

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrPackage = errors.New("Пакет обновления повреждён или не соответствует ожидаемой версии")

const MaxExtracted int64 = 512 << 20

func safeRelative(name string) bool {
	if name == "" || len(name) > 1024 || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return false
	}
	for _, p := range strings.Split(name, "/") {
		if p == "" || p == "." || p == ".." || strings.HasSuffix(p, ".") || strings.HasSuffix(p, " ") {
			return false
		}
		upper := strings.ToUpper(strings.SplitN(p, ".", 2)[0])
		switch upper {
		case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
			return false
		}
	}
	return true
}
func Extract(ctx context.Context, archive, dest string) error {
	z, e := zip.OpenReader(archive)
	if e != nil {
		return ErrPackage
	}
	defer z.Close()
	if len(z.File) == 0 || len(z.File) > 4096 {
		return ErrPackage
	}
	// Validate the entire directory before any extraction, including case aliases.
	seen := map[string]bool{}
	total := uint64(0)
	for _, f := range z.File {
		name := strings.TrimSuffix(f.Name, "/")
		if !safeRelative(name) {
			return ErrPackage
		}
		mode := f.Mode()
		if !mode.IsRegular() && !mode.IsDir() {
			return ErrPackage
		}
		k := strings.ToLower(name)
		if seen[k] {
			return ErrPackage
		}
		seen[k] = true
		if f.UncompressedSize64 > uint64(MaxExtracted) || total > uint64(MaxExtracted)-f.UncompressedSize64 {
			return ErrPackage
		}
		total += f.UncompressedSize64
	}
	if e = os.Mkdir(dest, 0700); e != nil {
		return e
	}
	for _, f := range z.File {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := strings.TrimSuffix(f.Name, "/")
		path := filepath.Join(dest, filepath.FromSlash(name))
		if f.Mode().IsDir() {
			if e = os.MkdirAll(path, 0755); e != nil {
				return ErrPackage
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			return ErrPackage
		}
		mode := os.FileMode(0644)
		if f.Mode()&0111 != 0 {
			mode = 0755
		}
		out, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if e != nil {
			return ErrPackage
		}
		in, e := f.Open()
		if e != nil {
			out.Close()
			return ErrPackage
		}
		n, e := io.Copy(out, io.LimitReader(in, int64(f.UncompressedSize64)+1))
		in.Close()
		ce := out.Close()
		if e != nil || ce != nil || n != int64(f.UncompressedSize64) {
			return ErrPackage
		}
	}
	return nil
}
