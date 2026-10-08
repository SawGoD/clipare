package ui

import "testing"

func TestScrollBounds(t *testing.T) {
	for _, dpi := range []int32{96, 120, 144, 192} {
		extent, page := 688*dpi/96, 500*dpi/96
		for _, p := range []int32{-100, 0, 20, 10000} {
			got := clampScroll(p, extent, page)
			if got < 0 || got > extent-page {
				t.Fatalf("DPI %d, position %d", dpi, got)
			}
		}
		if clampScroll(100, 200, 500) != 0 {
			t.Fatal("scrollbar with fitting content")
		}
	}
}
