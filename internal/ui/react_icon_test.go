//go:build windows || (darwin && cgo)

package ui

import (
	"bytes"
	"image/png"
	"testing"
)

func TestReactTrayIconStates(t *testing.T) {
	colors := map[string][3]uint32{"active": {48, 164, 96}, "disabled": {207, 61, 67}, "degraded": {231, 153, 53}}
	for state, want := range colors {
		img, err := png.Decode(bytes.NewReader(reactTrayIcon(state, "dark")))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
			t.Fatalf("%s: wrong size %v", state, img.Bounds())
		}
		r, g, b, a := img.At(51, 51).RGBA()
		if r>>8 != want[0] || g>>8 != want[1] || b>>8 != want[2] || a>>8 != 255 {
			t.Fatalf("%s: unexpected badge", state)
		}
	}
}
