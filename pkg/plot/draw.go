package plot

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextAnchor controls horizontal alignment of rendered text.
type TextAnchor int

const (
	TextAnchorLeft   TextAnchor = 0
	TextAnchorCenter TextAnchor = 1
	TextAnchorRight  TextAnchor = 2
)

// DrawLine draws a solid line using Bresenham's algorithm.
func DrawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		if x0 >= 0 && x0 < img.Bounds().Dx() && y0 >= 0 && y0 < img.Bounds().Dy() {
			img.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawDashedLine draws a dashed line using Bresenham's algorithm.
// dashLen and gapLen define the dash pattern in pixels.
func DrawDashedLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color, dashLen, gapLen int) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	step := 0
	period := dashLen + gapLen
	for {
		if step%period < dashLen {
			if x0 >= 0 && x0 < w && y0 >= 0 && y0 < h {
				img.Set(x0, y0, c)
			}
		}
		step++
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// FillRect fills a rectangular area with the given color.
func FillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

// DrawString renders a string at (x, y) with the specified horizontal anchor.
func DrawString(img *image.RGBA, x, y int, s string, anchor TextAnchor) {
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	width := font.MeasureString(face, s).Ceil()
	switch anchor {
	case TextAnchorCenter:
		d.Dot = fixed.P(x-width/2, y)
	case TextAnchorRight:
		d.Dot = fixed.P(x-width, y)
	default:
		d.Dot = fixed.P(x, y)
	}
	d.DrawString(s)
}

// DrawVerticalString renders a string vertically, one character per line.
func DrawVerticalString(img *image.RGBA, x, y int, s string) {
	for i, r := range s {
		DrawString(img, x, y+i*13, string(r), TextAnchorLeft)
	}
}

// DrawColorbarTicks draws tick labels next to the colorbar.
func DrawColorbarTicks(img *image.RGBA, cbX, topY, cbH int, vmin, vmax float64) {
	tickCount := 5
	for i := 0; i <= tickCount; i++ {
		frac := float64(i) / float64(tickCount)
		y := topY + cbH - int(frac*float64(cbH))
		val := vmin + frac*(vmax-vmin)
		label := FormatTick(val)
		DrawString(img, cbX+24, y+4, label, TextAnchorLeft)
	}
}

// FormatTick formats a numeric tick label.
func FormatTick(v float64) string {
	if v == 0 || (math.Abs(v) >= 0.01 && math.Abs(v) < 10000) {
		return fmt.Sprintf("%.2g", v)
	}
	return fmt.Sprintf("%.1e", v)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
