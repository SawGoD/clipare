package clipare

import (
	"embed"
	"strings"
)

// Version is the single source of the release version for binaries and bundles.
//
//go:embed VERSION
var versionFiles embed.FS

func Version() string { b, _ := versionFiles.ReadFile("VERSION"); return strings.TrimSpace(string(b)) }

// Commit is populated by release builds with -ldflags.
var Commit = "unknown"
