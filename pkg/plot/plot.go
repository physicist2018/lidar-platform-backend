package plot

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"time"
)

// Image dimensions and layout constants.
const (
	ImgWidth     = 1024
	ImgHeight    = 768
	MarginLeft   = 80
	MarginBottom = 60
	MarginRight  = 100
	MarginTop    = 20
	HeatW        = ImgWidth - MarginLeft - MarginRight
	HeatH        = ImgHeight - MarginTop - MarginBottom
)

// DataRange returns the min and max of a 2D float64 matrix, ignoring NaN and Inf.
func DataRange(matrix [][]float64) (float64, float64) {
	vmin := math.MaxFloat64
	vmax := -math.MaxFloat64
	for _, row := range matrix {
		for _, v := range row {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if v < vmin {
				vmin = v
			}
			if v > vmax {
				vmax = v
			}
		}
	}
	if vmin == math.MaxFloat64 {
		return 0, 1
	}
	return vmin, vmax
}

// BilinearInterp performs bilinear interpolation on a 2D matrix at fractional indices (tf, af).
func BilinearInterp(matrix [][]float64, tf, af float64) float64 {
	nTimes := len(matrix)
	nAlts := len(matrix[0])

	t0 := int(math.Floor(tf))
	t1 := t0 + 1
	if t0 < 0 {
		t0 = 0
	}
	if t1 >= nTimes {
		t1 = nTimes - 1
	}
	a0 := int(math.Floor(af))
	a1 := a0 + 1
	if a0 < 0 {
		a0 = 0
	}
	if a1 >= nAlts {
		a1 = nAlts - 1
	}

	ft := tf - float64(t0)
	fa := af - float64(a0)

	v00 := nanToZero(matrix[t0][a0])
	v10 := nanToZero(matrix[t1][a0])
	v01 := nanToZero(matrix[t0][a1])
	v11 := nanToZero(matrix[t1][a1])

	v0 := v00*(1-ft) + v10*ft
	v1 := v01*(1-ft) + v11*ft
	return v0*(1-fa) + v1*fa
}

func nanToZero(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return v
}

// InterpolateTime returns a time linearly interpolated from a slice at fractional position frac ∈ [0,1].
func InterpolateTime(times []time.Time, frac float64) time.Time {
	if len(times) == 0 {
		return time.Time{}
	}
	f := frac * float64(len(times)-1)
	idx := int(math.Floor(f))
	if idx >= len(times)-1 {
		return times[len(times)-1]
	}
	rem := f - float64(idx)
	d := times[idx+1].Sub(times[idx])
	return times[idx].Add(time.Duration(float64(d) * rem))
}

// RenderHeatmap renders a time-height heatmap PNG.
func RenderHeatmap(matrix [][]float64, altitudes []float64, times []time.Time) ([]byte, error) {
	nTimes := len(matrix)
	if nTimes == 0 {
		return nil, fmt.Errorf("empty matrix")
	}
	nAlts := len(altitudes)

	vmin, vmax := DataRange(matrix)
	if vmin == vmax {
		vmax = vmin + 1
	}

	img := image.NewRGBA(image.Rect(0, 0, ImgWidth, ImgHeight))
	FillRect(img, 0, 0, ImgWidth, ImgHeight, color.White)

	// heatmap area
	for py := 0; py < HeatH; py++ {
		for px := 0; px < HeatW; px++ {
			tf := float64(px) / float64(HeatW-1) * float64(nTimes-1)
			af := (1.0 - float64(py)/float64(HeatH-1)) * float64(nAlts-1)
			val := BilinearInterp(matrix, tf, af)
			c := JetColormap(val, vmin, vmax)
			img.Set(MarginLeft+px, MarginTop+py, c)
		}
	}

	// colorbar
	cbX := ImgWidth - MarginRight + 20
	for py := 0; py < HeatH; py++ {
		frac := 1.0 - float64(py)/float64(HeatH-1)
		val := vmin + frac*(vmax-vmin)
		c := JetColormap(val, vmin, vmax)
		for dx := 0; dx < 20; dx++ {
			img.Set(cbX+dx, MarginTop+py, c)
		}
	}
	DrawColorbarTicks(img, cbX, MarginTop, HeatH, vmin, vmax)

	// grid and axes
	drawPlotFrame(img, 8, 6)

	// Y-axis labels (altitudes)
	for i := 0; i <= 6; i++ {
		frac := float64(i) / 6.0
		alt := altitudes[0] + frac*(altitudes[len(altitudes)-1]-altitudes[0])
		y := MarginTop + HeatH - int(frac*float64(HeatH))
		DrawString(img, MarginLeft-10, y+4, fmt.Sprintf("%.0f", alt), TextAnchorRight)
	}

	// X-axis labels (time)
	for i := 0; i <= 8; i++ {
		frac := float64(i) / 8.0
		tm := InterpolateTime(times, frac)
		x := MarginLeft + int(frac*float64(HeatW-1))
		DrawString(img, x, ImgHeight-MarginBottom/2+4, tm.Format("15:04"), TextAnchorCenter)
	}

	// Titles
	DrawVerticalString(img, 12, MarginTop+HeatH/2+40, "Altitude, m")
	DrawString(img, MarginLeft+HeatW/2, ImgHeight-8, "Time", TextAnchorCenter)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderProfile renders a vertical profile PNG (signal vs altitude).
func RenderProfile(altitudes []float64, values []float64) ([]byte, error) {
	n := len(altitudes)
	if n == 0 {
		return nil, fmt.Errorf("empty profile")
	}

	valMin, valMax := DataRange([][]float64{values})
	if valMin == valMax {
		valMax = valMin + 1
	}

	altMin, altMax := altitudes[0], altitudes[n-1]
	img := image.NewRGBA(image.Rect(0, 0, ImgWidth, ImgHeight))
	FillRect(img, 0, 0, ImgWidth, ImgHeight, color.White)

	// profile polyline
	for i := 1; i < n; i++ {
		if math.IsNaN(values[i-1]) || math.IsNaN(values[i]) {
			continue
		}
		x0 := MarginLeft + int((values[i-1]-valMin)/(valMax-valMin)*float64(HeatW-1))
		y0 := MarginTop + HeatH - int((altitudes[i-1]-altMin)/(altMax-altMin)*float64(HeatH-1)) - 1
		x1 := MarginLeft + int((values[i]-valMin)/(valMax-valMin)*float64(HeatW-1))
		y1 := MarginTop + HeatH - int((altitudes[i]-altMin)/(altMax-altMin)*float64(HeatH-1)) - 1
		DrawLine(img, x0, y0, x1, y1, color.Black)
	}

	// grid and axes
	drawPlotFrame(img, 6, 6)

	// X-axis labels (Signal)
	for i := 0; i <= 6; i++ {
		frac := float64(i) / 6.0
		val := valMin + frac*(valMax-valMin)
		x := MarginLeft + int(frac*float64(HeatW-1))
		DrawString(img, x, ImgHeight-MarginBottom/2+4, FormatTick(val), TextAnchorCenter)
	}

	// Y-axis labels (Altitude)
	for i := 0; i <= 6; i++ {
		frac := float64(i) / 6.0
		alt := altMin + frac*(altMax-altMin)
		y := MarginTop + HeatH - int(frac*float64(HeatH))
		DrawString(img, MarginLeft-10, y+4, fmt.Sprintf("%.0f", alt), TextAnchorRight)
	}

	// Titles
	DrawString(img, MarginLeft+HeatW/2, ImgHeight-8, "Signal", TextAnchorCenter)
	DrawVerticalString(img, 12, MarginTop+HeatH/2+40, "Altitude, m")

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawPlotFrame renders dashed grid lines and a solid axis border around the plot area.
func drawPlotFrame(img *image.RGBA, xTickCount, yTickCount int) {
	gridColor := color.RGBA{200, 200, 200, 255}
	axisColor := color.RGBA{0, 0, 0, 255}

	// horizontal dashed grid lines
	for i := 0; i <= yTickCount; i++ {
		frac := float64(i) / float64(yTickCount)
		y := MarginTop + HeatH - int(frac*float64(HeatH))
		DrawDashedLine(img, MarginLeft, y, MarginLeft+HeatW-1, y, gridColor, 6, 4)
	}

	// vertical dashed grid lines
	for i := 0; i <= xTickCount; i++ {
		frac := float64(i) / float64(xTickCount)
		x := MarginLeft + int(frac*float64(HeatW-1))
		DrawDashedLine(img, x, MarginTop, x, MarginTop+HeatH-1, gridColor, 6, 4)
	}

	// solid axis border
	DrawLine(img, MarginLeft, MarginTop, MarginLeft+HeatW-1, MarginTop, axisColor)                 // top
	DrawLine(img, MarginLeft, MarginTop+HeatH-1, MarginLeft+HeatW-1, MarginTop+HeatH-1, axisColor) // bottom
	DrawLine(img, MarginLeft, MarginTop, MarginLeft, MarginTop+HeatH-1, axisColor)                 // left
	DrawLine(img, MarginLeft+HeatW-1, MarginTop, MarginLeft+HeatW-1, MarginTop+HeatH-1, axisColor) // right
}
