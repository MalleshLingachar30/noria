package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// A small, legible native tray icon with no external asset or font dependency.
// The centered N stays recognizable at 16x16 and scales to HiDPI displays.
func iconPNG(running bool) []byte {
	const size = 48
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	background := color.NRGBA{R: 24, G: 33, B: 43, A: 255}
	if !running {
		background = color.NRGBA{R: 115, G: 122, B: 130, A: 255}
	}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-24, y-24
			if dx*dx+dy*dy <= 23*23 {
				img.SetNRGBA(x, y, background)
			}
			// Parallel vertical strokes and a diagonal center form the N.
			if y >= 13 && y <= 35 && ((x >= 13 && x <= 18) || (x >= 30 && x <= 35) ||
				(x >= 18 && x <= 30 && abs(x-(18+(y-13)*12/22)) <= 3)) {
				img.SetNRGBA(x, y, white)
			}
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, img)
	return out.Bytes()
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
