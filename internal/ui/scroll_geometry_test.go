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

func TestScrollBarVisibility(t *testing.T) {
	for _, tc := range []struct {
		w, h, cw, ch         int32
		horizontal, vertical bool
	}{
		{500, 500, 400, 400, false, false}, {500, 500, 500, 500, false, false},
		{500, 500, 400, 600, false, true}, {500, 500, 600, 400, true, false},
		{500, 500, 490, 600, true, true}, {500, 500, 600, 490, true, true},
	} {
		h, v := scrollBars(tc.w, tc.h, tc.cw, tc.ch, 17, 17)
		if h != tc.horizontal || v != tc.vertical {
			t.Fatalf("%+v: got horizontal=%v vertical=%v", tc, h, v)
		}
	}
}
