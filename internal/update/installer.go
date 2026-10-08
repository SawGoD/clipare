package update

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var ErrInstall = errors.New("Clipare не может обновить себя в текущей папке. Переместите приложение в доступное для записи место или обновите вручную")
var ErrRollback = errors.New("update rollback failed; backup preserved")

type Plan struct {
	Work       string            `json:"work"`
	Target     string            `json:"target"`
	Executable string            `json:"executable"`
	ConfigPath string            `json:"config_path"`
	Version    string            `json:"version"`
	Token      string            `json:"token"`
	ParentPID  int               `json:"parent_pid"`
	Created    time.Time         `json:"created"`
	Hashes     map[string]string `json:"hashes"`
}

func contains(root, path string) bool {
	r, e := filepath.Rel(root, path)
	return e == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}
func copyRegular(src, dst string, mode os.FileMode) error {
	i, e := os.Lstat(src)
	if e != nil || !i.Mode().IsRegular() {
		return ErrPackage
	}
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if e != nil {
		return e
	}
	_, e = io.Copy(out, in)
	if e == nil {
		e = out.Sync()
	}
	ce := out.Close()
	if e == nil {
		e = ce
	}
	return e
}
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		r, e := filepath.Rel(src, path)
		if e != nil {
			return e
		}
		to := filepath.Join(dst, r)
		if i.IsDir() {
			return os.MkdirAll(to, 0755)
		}
		if !i.Mode().IsRegular() {
			return ErrPackage
		}
		return copyRegular(path, to, i.Mode())
	})
}
func installLocation(exe string) (string, error) {
	exe, e := filepath.EvalSymlinks(exe)
	if e != nil {
		return "", ErrInstall
	}
	if runtime.GOOS == "darwin" {
		bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
		if filepath.Base(bundle) != "Clipare.app" || exe != filepath.Join(bundle, "Contents", "MacOS", "Clipare") {
			return "", ErrInstall
		}
		return bundle, nil
	}
	if runtime.GOOS == "windows" && filepath.Base(exe) == "Clipare.exe" {
		return filepath.Dir(exe), nil
	}
	return "", ErrInstall
}
func (s *Staged) Prepare(ctx context.Context, exe, configPath string) (Plan, error) {
	target, e := installLocation(exe)
	if e != nil {
		return Plan{}, e
	}
	parent := target
	if runtime.GOOS == "darwin" {
		parent = filepath.Dir(target)
	}
	// Test writable sibling staging before asking the running app to stop.
	probe, e := os.MkdirTemp(parent, ".clipare-permission-")
	if e != nil {
		return Plan{}, ErrInstall
	}
	os.Remove(probe)
	pkg, e := ValidatePackage(s.Root, s.Package.Version, runtime.GOOS, runtime.GOARCH)
	if e != nil {
		return Plan{}, e
	}
	newExe := filepath.Join(s.Root, "Clipare.exe")
	if runtime.GOOS == "darwin" {
		newExe = filepath.Join(s.Root, "Clipare.app", "Contents", "MacOS", "Clipare")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if e = validatePlatformPackage(ctx, s.Root); e != nil {
		return Plan{}, e
	}
	cmd := exec.CommandContext(ctx, newExe, "--version")
	configureCommand(cmd)
	out := &limitedOutput{}
	cmd.Stdout = out
	e = cmd.Run()
	if e != nil || !strings.HasPrefix(out.String(), "Clipare "+pkg.Version+" (") {
		return Plan{}, ErrPackage
	}
	configPath, e = filepath.Abs(configPath)
	if e != nil {
		return Plan{}, e
	}
	if runtime.GOOS == "darwin" && contains(target, configPath) {
		return Plan{}, ErrInstall
	}
	token := make([]byte, 32)
	if _, e = rand.Read(token); e != nil {
		return Plan{}, e
	}
	p := Plan{Work: s.Work, Target: target, Executable: exe, ConfigPath: configPath, Version: pkg.Version, Token: hex.EncodeToString(token), ParentPID: os.Getpid(), Created: time.Now().UTC()}
	p.Hashes = map[string]string{}
	for _, name := range append(append([]string(nil), pkg.Files...), PackageFile) {
		hash, e := FileSHA256(filepath.Join(s.Root, filepath.FromSlash(name)))
		if e != nil {
			return Plan{}, e
		}
		p.Hashes[name] = hash
	}
	if e = p.validate(); e != nil {
		return Plan{}, e
	}
	if runtime.GOOS == "windows" {
		for _, name := range packageItems(pkg) {
			if contains(filepath.Join(target, name), configPath) {
				return Plan{}, ErrInstall
			}
		}
	}
	b, e := json.Marshal(p)
	if e != nil {
		return Plan{}, e
	}
	if e = atomicWrite(filepath.Join(s.Work, "plan.json"), b); e != nil {
		return Plan{}, e
	}
	return p, nil
}

type limitedOutput struct{ bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4096 {
		return 0, ErrPackage
	}
	return b.Buffer.Write(p)
}
func (p Plan) validate() error {
	cache, e := CacheDir()
	if e != nil {
		return e
	}
	if !filepath.IsAbs(p.Work) || filepath.Dir(p.Work) != cache || !strings.HasPrefix(filepath.Base(p.Work), "stage-") || !filepath.IsAbs(p.Target) || !filepath.IsAbs(p.ConfigPath) || p.ParentPID <= 0 || len(p.Token) != 64 || time.Since(p.Created) > time.Hour || p.Created.After(time.Now().Add(time.Minute)) {
		return ErrInstall
	}
	if _, e = hex.DecodeString(p.Token); e != nil {
		return ErrInstall
	}
	if _, e = StableVersion(p.Version); e != nil {
		return ErrInstall
	}
	target, e := installLocation(p.Executable)
	if e != nil || target != p.Target {
		return ErrInstall
	}
	i, e := os.Lstat(p.Work)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return ErrInstall
	}
	return nil
}
func (p Plan) LaunchHelper(ctx context.Context) error {
	path := filepath.Join(p.Work, "helper")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if e := copyRegular(p.Executable, path, 0700); e != nil {
		return e
	}
	cmd := exec.Command(path, "--apply-update", filepath.Join(p.Work, "plan.json"))
	configureHelper(cmd)
	if e := cmd.Start(); e != nil {
		return e
	}
	success := false
	defer func() {
		if success {
			cmd.Process.Release()
		} else {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ErrInstall
		case <-tick.C:
			b, e := os.ReadFile(filepath.Join(p.Work, "helper-ready"))
			if e == nil && string(b) == p.Token {
				success = true
				return nil
			}
		}
	}
}
func (p Plan) Arm() error { return os.WriteFile(filepath.Join(p.Work, "armed"), []byte(p.Token), 0600) }
func packageItems(p Package) []string {
	if p.OS == "darwin" {
		return []string{"Clipare.app"}
	}
	m := map[string]bool{PackageFile: true}
	for _, f := range p.Files {
		m[strings.SplitN(f, "/", 2)[0]] = true
	}
	var a []string
	for k := range m {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}

// replaceTransaction prepares a sibling copy and preserves every old item until
// the new process acknowledges startup. Unknown files in the install directory
// are never touched. rename/start are injected for failure tests.
func replaceTransaction(root, target string, items []string, rename func(string, string) error, start func() error) error {
	tx, e := os.MkdirTemp(target, ".clipare-update-")
	if e != nil {
		return ErrInstall
	}
	defer func() {
		if tx != "" {
			os.RemoveAll(tx)
		}
	}()
	fresh, backup := filepath.Join(tx, "new"), filepath.Join(tx, "backup")
	if e = os.Mkdir(fresh, 0700); e != nil {
		return e
	}
	if e = os.Mkdir(backup, 0700); e != nil {
		return e
	}
	for _, name := range items {
		if !safeRelative(name) || strings.Contains(name, "/") {
			return ErrPackage
		}
		src := filepath.Join(root, name)
		info, e := os.Lstat(src)
		if e != nil {
			return e
		}
		if info.IsDir() {
			e = copyTree(src, filepath.Join(fresh, name))
		} else {
			e = copyRegular(src, filepath.Join(fresh, name), info.Mode())
		}
		if e != nil {
			return e
		}
	}
	old, newNames := []string{}, []string{}
	rollback := func() error {
		var failure error
		for i := len(newNames) - 1; i >= 0; i-- {
			name := newNames[i]
			if e := rename(filepath.Join(target, name), filepath.Join(fresh, name)); e != nil {
				failure = e
			}
		}
		for i := len(old) - 1; i >= 0; i-- {
			name := old[i]
			if e := rename(filepath.Join(backup, name), filepath.Join(target, name)); e != nil {
				failure = e
			}
		}
		if failure != nil { // Preserve the only recoverable copy; do not delete tx.
			tx = ""
			return ErrRollback
		}
		return nil
	}
	for _, name := range items {
		dest := filepath.Join(target, name)
		if info, e := os.Lstat(dest); e == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				if r := rollback(); r != nil {
					return r
				}
				return ErrInstall
			}
			if e = rename(dest, filepath.Join(backup, name)); e != nil {
				if r := rollback(); r != nil {
					return r
				}
				return e
			}
			old = append(old, name)
		} else if !os.IsNotExist(e) {
			if r := rollback(); r != nil {
				return r
			}
			return e
		}
		if e = rename(filepath.Join(fresh, name), dest); e != nil {
			if r := rollback(); r != nil {
				return r
			}
			return e
		}
		newNames = append(newNames, name)
	}
	if e = start(); e != nil {
		if r := rollback(); r != nil {
			return r
		}
		return e
	}
	return nil
}

// Apply is entered only by the temporary helper, before any clipboard/config UI.
func Apply(ctx context.Context, planPath string) error {
	b, e := os.ReadFile(planPath)
	var p Plan
	if e != nil || len(b) > 512<<10 || json.Unmarshal(b, &p) != nil || planPath != filepath.Join(p.Work, "plan.json") {
		return ErrInstall
	}
	if e = p.validate(); e != nil {
		return e
	}
	logFile, e := os.OpenFile(filepath.Join(p.Work, "helper.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		return e
	}
	defer logFile.Close()
	log := slog.New(slog.NewJSONHandler(logFile, nil))
	log.Info("update helper started", "version", p.Version)
	// Revalidate package immediately before replacement; no partial files accepted.
	root := filepath.Join(p.Work, "package")
	pkg, e := ValidatePackage(root, p.Version, runtime.GOOS, runtime.GOARCH)
	if e != nil {
		return e
	}
	if len(p.Hashes) != len(pkg.Files)+1 {
		return ErrVerify
	}
	for _, name := range append(append([]string(nil), pkg.Files...), PackageFile) {
		if e = VerifyFile(filepath.Join(root, filepath.FromSlash(name)), p.Hashes[name]); e != nil {
			return e
		}
	}
	wait, closeWait, e := exitWaiter(p.ParentPID)
	if e != nil {
		return e
	}
	defer closeWait()
	if e = os.WriteFile(filepath.Join(p.Work, "helper-ready"), []byte(p.Token), 0600); e != nil {
		return e
	}
	waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	e = wait(waitCtx)
	cancel()
	if e != nil {
		return e
	}
	b, e = os.ReadFile(filepath.Join(p.Work, "armed"))
	if e != nil || string(b) != p.Token {
		return ErrInstall
	}
	target := p.Target
	if runtime.GOOS == "darwin" {
		target = filepath.Dir(target)
	}
	e = replaceTransaction(root, target, packageItems(pkg), os.Rename, func() error { return p.startNew(ctx) })
	if e != nil {
		log.Warn("update failed", "rollback_available", !errors.Is(e, ErrRollback))
		if errors.Is(e, ErrRollback) {
			return e
		}
		// Old paths are restored before relaunch. Retain recovery data on failure.
		cmd := exec.Command(p.Executable, "--config", p.ConfigPath, "--update-failed")
		configureHelper(cmd)
		if cmd.Start() == nil {
			cmd.Process.Release()
		}
		return e
	}
	os.WriteFile(filepath.Join(p.Work, "completed"), []byte(p.Token), 0600)
	log.Info("update applied", "version", p.Version)
	return nil
}
func (p Plan) startNew(ctx context.Context) error {
	cmd := exec.Command(p.Executable, "--config", p.ConfigPath, "--update-ready", filepath.Join(p.Work, "startup-ready"), "--update-token", p.Token)
	configureHelper(cmd)
	if e := cmd.Start(); e != nil {
		return e
	}
	success := false
	defer func() {
		if !success {
			cmd.Process.Kill()
			cmd.Wait()
		} else {
			cmd.Process.Release()
		}
	}()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ErrInstall
		case <-tick.C:
			b, e := os.ReadFile(filepath.Join(p.Work, "startup-ready"))
			if e == nil && string(b) == p.Token {
				success = true
				return nil
			}
		}
	}
}
func Acknowledge(path, token string) error {
	cache, e := CacheDir()
	if e != nil {
		return e
	}
	work := filepath.Dir(path)
	if filepath.Dir(work) != cache || !strings.HasPrefix(filepath.Base(work), "stage-") || filepath.Base(path) != "startup-ready" || len(token) != 64 {
		return ErrInstall
	}
	b, e := os.ReadFile(filepath.Join(work, "plan.json"))
	var p Plan
	if e != nil || json.Unmarshal(b, &p) != nil || p.Token != token || p.Work != work {
		return ErrInstall
	}
	if e = os.WriteFile(path, []byte(token), 0600); e != nil {
		return e
	}
	// The new app removes its own completed staging only after the helper exits.
	return nil
}
