package ui

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
