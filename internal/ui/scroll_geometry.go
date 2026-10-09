package ui

// Resolve mutually dependent scrollbar visibility without resizing the window
// to measure each intermediate state.
func scrollBars(width, height, contentWidth, contentHeight, barWidth, barHeight int32) (horizontal, vertical bool) {
	for i := 0; i < 3; i++ {
		w, h := width, height
		if vertical {
			w -= barWidth
		}
		if horizontal {
			h -= barHeight
		}
		horizontal = horizontal || contentWidth > w
		vertical = vertical || contentHeight > h
	}
	return
}

func clampScroll(pos, extent, page int32) int32 {
	max := extent - page
	if max < 0 {
		max = 0
	}
	if pos < 0 {
		return 0
	}
	if pos > max {
		return max
	}
	return pos
}
