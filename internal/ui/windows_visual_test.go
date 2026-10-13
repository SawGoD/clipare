//go:build windows

package ui

import "testing"

func TestFluentVisualStates(t *testing.T) {
	for _, dark := range []bool{false, true} {
		theme := winTheme{surface: color(0xffffff), text: color(0x1b1b1b), muted: color(0x616161), border: color(0xd8d8d8), accent: color(0x005fb8), onAccent: color(0xffffff), danger: color(0xa4262c)}
		if dark {
			theme.surface, theme.text, theme.muted = color(0x2b2b2b), color(0xf5f5f5), color(0xbcbcbc)
		}
		for _, kind := range []int{0, 1, 2, 3, 5, 6} {
			idle, _, _ := theme.buttonColors(kind, false, false, false, theme.surface)
			hover, _, _ := theme.buttonColors(kind, false, true, false, theme.surface)
			pressed, _, _ := theme.buttonColors(kind, false, true, true, theme.surface)
			if idle == hover || hover == pressed {
				t.Fatalf("kind %d dark=%v has indistinguishable interaction surfaces", kind, dark)
			}
			fill, _, fg := theme.buttonColors(kind, true, true, true, theme.surface)
			if fill != theme.surface || fg != theme.muted {
				t.Fatal("disabled controls must not retain accent or destructive emphasis")
			}
		}
		if theme.rowFill(false, true) == theme.surface || theme.rowFill(true, true) != theme.rowFill(true, false) {
			t.Fatal("hover must be subtle and not override selection")
		}
		theme.contrast = true
		if theme.rowFill(true, true) != theme.accent || theme.rowFill(false, true) != theme.surface {
			t.Fatal("contrast themes must use unblended system surfaces")
		}
		fill, _, fg := theme.buttonColors(1, false, true, true, theme.surface)
		if fill != theme.accent || fg != theme.onAccent {
			t.Fatal("contrast themes must retain the system highlight pair")
		}
	}
}
