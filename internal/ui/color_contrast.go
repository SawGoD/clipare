package ui

import "math"

// COLORREF uses BGR ordering. Used to retain contrast with any Windows accent.
func contrastRatio(a, b uint32) float64 {
	lum := func(c uint32) float64 {
		component := func(v uint32) float64 {
			f := float64(v) / 255
			if f <= .04045 {
				return f / 12.92
			}
			return math.Pow((f+.055)/1.055, 2.4)
		}
		return .2126*component(c&255) + .7152*component((c>>8)&255) + .0722*component((c>>16)&255)
	}
	x, y := lum(a), lum(b)
	if x < y {
		x, y = y, x
	}
	return (x + .05) / (y + .05)
}
