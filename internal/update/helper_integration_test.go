package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// The helper integration uses the test executable as a synthetic Clipare.
// It never initializes native UI, clipboard, identity, autostart or real config.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--apply-update" {
		if e := Apply(context.Background(), os.Args[2]); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "--fake-old" {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if len(os.Args) > 2 {
				if _, e := os.Stat(os.Args[2]); e == nil {
					os.Exit(0)
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "--config" {
		if len(os.Args) != 7 || os.Args[3] != "--update-ready" {
			os.Exit(2)
		}
		path, token := os.Args[4], os.Args[6]
		if Acknowledge(path, token) != nil {
			os.Exit(3)
		}
		work := filepath.Dir(path)
		os.WriteFile(filepath.Join(work, "fake-new-pid"), []byte(strconv.Itoa(os.Getpid())), 0600)
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, e := os.Stat(filepath.Join(work, "test-stop")); e == nil {
				os.Exit(0)
			}
			time.Sleep(50 * time.Millisecond)
		}
		os.Exit(4)
	}
	os.Exit(m.Run())
}

func TestHelperReplacesFakeInstallationAndRestarts(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("desktop helper")
	}
	cache, e := CacheDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(cache, 0700); e != nil {
		t.Fatal(e)
	}
	work, e := os.MkdirTemp(cache, "stage-helper-test-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(work)
	targetDir := t.TempDir()
	targetDir, e = filepath.EvalSymlinks(targetDir)
	if e != nil {
		t.Fatal(e)
	}
	root := filepath.Join(work, "package")
	os.Mkdir(root, 0700)
	exe, _ := os.Executable()
	var target, binary string
	pkg := Package{Version: "0.4.0", OS: runtime.GOOS, Arch: runtime.GOARCH}
	if runtime.GOOS == "darwin" {
		target = filepath.Join(targetDir, "Clipare.app")
		binary = "Clipare.app/Contents/MacOS/Clipare"
		pkg.Files = []string{binary, "Clipare.app/Contents/Info.plist"}
		plist := []byte("<key>CFBundleShortVersionString</key><string>0.4.0</string>")
		os.MkdirAll(filepath.Join(root, "Clipare.app", "Contents", "MacOS"), 0755)
		os.WriteFile(filepath.Join(root, "Clipare.app", "Contents", "Info.plist"), plist, 0644)
	} else {
		target = targetDir
		binary = "Clipare.exe"
		pkg.Files = []string{binary, "clipare-console.exe"}
		if e = copyRegular(exe, filepath.Join(root, "clipare-console.exe"), 0755); e != nil {
			t.Fatal(e)
		}
	}
	if e = copyRegular(exe, filepath.Join(root, filepath.FromSlash(binary)), 0755); e != nil {
		t.Fatal(e)
	}
	metadata, _ := json.Marshal(pkg)
	os.WriteFile(filepath.Join(root, PackageFile), metadata, 0644)
	installedExe := filepath.Join(targetDir, filepath.FromSlash(binary))
	os.MkdirAll(filepath.Dir(installedExe), 0755)
	if e = copyRegular(exe, installedExe, 0755); e != nil {
		t.Fatal(e)
	}
	// An unrelated file represents existing settings: the installer must preserve it.
	configPath := filepath.Join(targetDir, "config.yaml")
	os.WriteFile(configPath, []byte("synthetic unchanged settings"), 0600)
	old := exec.Command(exe, "--fake-old", filepath.Join(work, "old-stop"))
	if e = old.Start(); e != nil {
		t.Fatal(e)
	}
	oldDone := make(chan error, 1)
	go func() { oldDone <- old.Wait() }()
	defer old.Process.Kill()
	p := Plan{Work: work, Target: target, Executable: installedExe, ConfigPath: configPath, Version: pkg.Version, Token: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ParentPID: old.Process.Pid, Created: time.Now().UTC(), Hashes: map[string]string{}}
	for _, f := range append(pkg.Files, PackageFile) {
		p.Hashes[f], e = FileSHA256(filepath.Join(root, filepath.FromSlash(f)))
		if e != nil {
			t.Fatal(e)
		}
	}
	b, _ := json.Marshal(p)
	os.WriteFile(filepath.Join(work, "plan.json"), b, 0600)
	p.Arm()
	helper := exec.Command(exe, "--apply-update", filepath.Join(work, "plan.json"))
	helper.Stderr = os.Stderr
	if e = helper.Start(); e != nil {
		t.Fatal(e)
	}
	helperDone := make(chan error, 1)
	go func() { helperDone <- helper.Wait() }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	ready := false
	for !ready {
		select {
		case e := <-helperDone:
			t.Fatal("helper exited before readiness", e)
		case <-timer.C:
			helper.Process.Kill()
			<-helperDone
			t.Fatal("helper readiness timeout")
		case <-tick.C:
			b, e := os.ReadFile(filepath.Join(work, "helper-ready"))
			ready = e == nil && string(b) == p.Token
		}
	}
	os.WriteFile(filepath.Join(work, "old-stop"), []byte("stop"), 0600)
	if e = <-helperDone; e != nil {
		t.Fatal(e)
	}
	if e = <-oldDone; e != nil {
		t.Fatal(e)
	}
	b, e = os.ReadFile(filepath.Join(work, "completed"))
	if e != nil || string(b) != p.Token {
		t.Fatal("not applied", e)
	}
	b, e = os.ReadFile(configPath)
	if e != nil || string(b) != "synthetic unchanged settings" {
		t.Fatal("config changed")
	}
	b, e = os.ReadFile(filepath.Join(work, "fake-new-pid"))
	if e != nil {
		t.Fatal(e)
	}
	pid, e := strconv.Atoi(string(b))
	if e != nil {
		t.Fatal(e)
	}
	wait, close, e := exitWaiter(pid)
	if e != nil {
		t.Fatal(e)
	}
	defer close()
	os.WriteFile(filepath.Join(work, "test-stop"), []byte("stop"), 0600)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e = wait(ctx); e != nil {
		t.Fatal("new fake app did not exit", e)
	}
	entries, _ := os.ReadDir(targetDir)
	for _, entry := range entries {
		if len(entry.Name()) > 15 && entry.Name()[:15] == ".clipare-update" {
			t.Fatal("backup residue")
		}
	}
}
