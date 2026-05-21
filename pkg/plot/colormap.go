package plot

import (
	"image/color"
	"math"
)

// JetColormap returns a color from the jet colormap for a given value.
// Values outside [vmin, vmax] are clamped. NaN/Inf yields transparent.
func JetColormap(val, vmin, vmax float64) color.Color {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return color.Transparent
	}
	t := (val - vmin) / (vmax - vmin)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	r, g, b := jetRGB(t)
	return color.RGBA{r, g, b, 255}
}

// jetRGB returns RGB components for the jet colormap at normalised value t ∈ [0,1].
func jetRGB(t float64) (uint8, uint8, uint8) {
	var r, g, b float64
	switch {
	case t < 0.125:
		r = 0
		g = 0
		b = 0.5 + t/0.125*0.5
	case t < 0.375:
		s := (t - 0.125) / 0.25
		r = 0
		g = s
		b = 1
	case t < 0.625:
		s := (t - 0.375) / 0.25
		r = s
		g = 1
		b = 1 - s
	case t < 0.875:
		s := (t - 0.625) / 0.25
		r = 1
		g = 1 - s
		b = 0
	default:
		s := (t - 0.875) / 0.125
		r = 1 - 0.5*s
		g = 0
		b = 0
	}
	return uint8(r*255 + 0.5), uint8(g*255 + 0.5), uint8(b*255 + 0.5)
}
