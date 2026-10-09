// Produce curated release notes from the matching CHANGELOG section.
package main

import (
	"fmt"
	"os"
	"strings"

	"clipare/internal/releaseversion"
)

func notes(changelog, version, previous string) (string, error) {
	if _, err := releaseversion.Parse(version); err != nil {
		return "", err
	}
	heading := "## " + version
	lines := strings.Split(changelog, "\n")
	start := -1
	end := len(lines)
	for i, line := range lines {
		if line == heading {
			if start >= 0 {
				return "", fmt.Errorf("duplicate changelog section")
			}
			start = i + 1
			continue
		}
		if start >= 0 && strings.HasPrefix(line, "## ") {
			end = i
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("missing changelog section for %s", version)
	}
	body := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if body == "" || (!strings.Contains(body, "### Нововведения") && !strings.Contains(body, "### Исправления")) {
		return "", fmt.Errorf("missing curated release summary")
	}
	body = strings.ReplaceAll(body, "### ", "## ")
	if previous != "" {
		if _, err := releaseversion.Parse(previous); err != nil {
			return "", err
		}
		body += "\n\nИзменения относительно v" + previous + ".\n\n[Сравнение версий](https://github.com/SawGoD/clipare/compare/v" + previous + "...v" + version + ")"
	}
	return body + "\n", nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: release-notes VERSION PREVIOUS_VERSION")
		os.Exit(1)
	}
	b, err := os.ReadFile("CHANGELOG.md")
	if err == nil {
		var n string
		n, err = notes(string(b), os.Args[1], os.Args[2])
		if err == nil {
			fmt.Print(n)
			return
		}
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
