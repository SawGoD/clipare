//go:build darwin

package autostart

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
)

func Set(enabled bool, executable, configPath string) error {
	home, e := os.UserHomeDir()
	if e != nil {
		return e
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	path := filepath.Join(dir, "io.clipare.agent.plist")
	if !enabled {
		e = os.Remove(path)
		if os.IsNotExist(e) {
			return nil
		}
		return e
	}
	escape := func(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
	body := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>io.clipare.agent</string><key>ProgramArguments</key><array><string>` + escape(executable) + `</string><string>--config</string><string>` + escape(configPath) + `</string></array><key>RunAtLoad</key><true/></dict></plist>`
	if e = os.MkdirAll(dir, 0700); e != nil {
		return errors.New("Не удалось создать LaunchAgents")
	}
	return os.WriteFile(path, []byte(body), 0600)
}
