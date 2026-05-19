package usecases

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"sort"
	"time"

	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type GenerateImageInput struct {
	ExperimentID string
	Wavelength   float64
	Polarization string
	ChannelType  string // "photon", "analog", "glued"
	PlotType     string // "RangeCorrected", "LogRangeCorrected"
}

type GenerateImageResult struct {
	Path string
}

const (
	imgWidth      = 1024
	imgHeight     = 768
	marginLeft    = 80
	marginBottom  = 60
	marginRight   = 100
	marginTop     = 20
	heatW         = imgWidth - marginLeft - marginRight
	heatH         = imgHeight - marginTop - marginBottom
	glueThreshold = 10.0
)

func (uc *ExperimentUseCase) GenerateImage(ctx context.Context, input GenerateImageInput) (*GenerateImageResult, error) {
	var profiles []*domain.ExperimentProfile

	switch input.ChannelType {
	case "photon":
		t := true
		list, err := uc.profileRepo.FindByParams(ctx, ports.ExperimentProfileParams{
			ExperimentID: input.ExperimentID,
			Wavelength:   input.Wavelength,
			Polarization: input.Polarization,
			Photon:       &t,
		})
		if err != nil {
			return nil, fmt.Errorf("find photon profiles: %w", err)
		}
		profiles = list
	case "analog":
		f := false
		list, err := uc.profileRepo.FindByParams(ctx, ports.ExperimentProfileParams{
			ExperimentID: input.ExperimentID,
			Wavelength:   input.Wavelength,
			Polarization: input.Polarization,
			Photon:       &f,
		})
		if err != nil {
			return nil, fmt.Errorf("find analog profiles: %w", err)
		}
		profiles = list
	case "glued":
		t := true
		photonList, err := uc.profileRepo.FindByParams(ctx, ports.ExperimentProfileParams{
			ExperimentID: input.ExperimentID,
			Wavelength:   input.Wavelength,
			Polarization: input.Polarization,
			Photon:       &t,
		})
		if err != nil {
			return nil, fmt.Errorf("find photon profiles: %w", err)
		}
		f := false
		analogList, err := uc.profileRepo.FindByParams(ctx, ports.ExperimentProfileParams{
			ExperimentID: input.ExperimentID,
			Wavelength:   input.Wavelength,
			Polarization: input.Polarization,
			Photon:       &f,
		})
		if err != nil {
			return nil, fmt.Errorf("find analog profiles: %w", err)
		}
		profiles = gluePairs(photonList, analogList)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found for given parameters")
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].MeasurementStopTime.Before(profiles[j].MeasurementStopTime)
	})

	altitudes := extractCommonAltitudes(profiles)
	if len(altitudes) == 0 {
		return nil, fmt.Errorf("profiles have empty altitude grids")
	}

	matrix := buildDataMatrix(profiles, altitudes, input.PlotType)

	pngBytes, err := renderHeatmap(matrix, altitudes, profiles, input)
	if err != nil {
		return nil, fmt.Errorf("render heatmap: %w", err)
	}

	fileName := fmt.Sprintf("time-height_%s_%.1f_%s_%s_%s.png",
		input.ExperimentID, input.Wavelength, input.Polarization,
		input.ChannelType, input.PlotType)
	objectPath := fmt.Sprintf("experiments/%s/imgs/%s", input.ExperimentID, fileName)

	if err := uc.storage.Upload(ctx, objectPath, bytes.NewReader(pngBytes), int64(len(pngBytes)), "image/png"); err != nil {
		return nil, fmt.Errorf("upload image: %w", err)
	}

	return &GenerateImageResult{Path: objectPath}, nil
}

func gluePairs(photonList, analogList []*domain.ExperimentProfile) []*domain.ExperimentProfile {
	photonByTime := make(map[int64]*domain.ExperimentProfile)
	analogByTime := make(map[int64]*domain.ExperimentProfile)
	for _, p := range photonList {
		photonByTime[p.MeasurementStopTime.UnixNano()] = p
	}
	for _, p := range analogList {
		analogByTime[p.MeasurementStopTime.UnixNano()] = p
	}

	var glued []*domain.ExperimentProfile
	for ts, photon := range photonByTime {
		analog, ok := analogByTime[ts]
		if !ok {
			continue
		}
		if len(photon.Altitudes) != len(analog.Altitudes) {
			continue
		}
		g := &domain.ExperimentProfile{
			ExperimentID:         photon.ExperimentID,
			Wavelength:           photon.Wavelength,
			Polarization:         photon.Polarization,
			MeasurementStartTime: photon.MeasurementStartTime,
			MeasurementStopTime:  photon.MeasurementStopTime,
			Altitudes:            photon.Altitudes,
			Data:                 make([]float64, len(photon.Data)),
			Hmin:                 photon.Hmin,
			Hmax:                 photon.Hmax,
			BgrType:              photon.BgrType,
			NDataPoints:          photon.NDataPoints,
		}
		for i := range photon.Data {
			if photon.Data[i] < glueThreshold {
				g.Data[i] = photon.Data[i]
			} else {
				g.Data[i] = analog.Data[i]
			}
		}
		glued = append(glued, g)
	}
	return glued
}

func extractCommonAltitudes(profiles []*domain.ExperimentProfile) []float64 {
	if len(profiles) == 0 {
		return nil
	}
	ref := profiles[0].Altitudes
	altSet := make(map[float64]bool)
	for _, a := range ref {
		altSet[a] = true
	}
	for _, p := range profiles[1:] {
		for _, a := range ref {
			found := false
			for _, pa := range p.Altitudes {
				if math.Abs(pa-a) < 1e-9 {
					found = true
					break
				}
			}
			if !found {
				delete(altSet, a)
			}
		}
	}
	result := make([]float64, 0, len(altSet))
	for a := range altSet {
		result = append(result, a)
	}
	sort.Float64s(result)
	return result
}

func buildDataMatrix(profiles []*domain.ExperimentProfile, altitudes []float64, plotType string) [][]float64 {
	nTimes := len(profiles)
	nAlts := len(altitudes)
	matrix := make([][]float64, nTimes)
	for t := 0; t < nTimes; t++ {
		matrix[t] = make([]float64, nAlts)
		for a := 0; a < nAlts; a++ {
			alt := altitudes[a]
			val := interpolateProfile(profiles[t], alt)
			switch plotType {
			case "RangeCorrected":
				val = val * alt * alt
			case "LogRangeCorrected":
				rc := val * alt * alt
				if rc > 0 {
					val = math.Log10(rc)
				} else {
					val = math.NaN()
				}
			}
			matrix[t][a] = val
		}
	}
	return matrix
}

func interpolateProfile(p *domain.ExperimentProfile, alt float64) float64 {
	if len(p.Altitudes) == 0 {
		return math.NaN()
	}
	idx := sort.SearchFloat64s(p.Altitudes, alt)
	if idx == 0 {
		return p.Data[0]
	}
	if idx >= len(p.Altitudes) {
		return p.Data[len(p.Data)-1]
	}
	a0 := p.Altitudes[idx-1]
	a1 := p.Altitudes[idx]
	t := (alt - a0) / (a1 - a0)
	return p.Data[idx-1]*(1-t) + p.Data[idx]*t
}

func renderHeatmap(matrix [][]float64, altitudes []float64, profiles []*domain.ExperimentProfile, input GenerateImageInput) ([]byte, error) {
	nTimes := len(matrix)
	if nTimes == 0 {
		return nil, fmt.Errorf("empty matrix")
	}
	nAlts := len(altitudes)

	vmin, vmax := dataRange(matrix)
	if vmin == vmax {
		vmax = vmin + 1
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// white background
	fillRect(img, 0, 0, imgWidth, imgHeight, color.White)

	// heatmap area
	for py := 0; py < heatH; py++ {
		for px := 0; px < heatW; px++ {
			tf := float64(px) / float64(heatW-1) * float64(nTimes-1)
			af := (1.0 - float64(py)/float64(heatH-1)) * float64(nAlts-1)

			val := bilinearInterp(matrix, tf, af)
			c := jetColormap(val, vmin, vmax)
			img.Set(marginLeft+px, marginTop+py, c)
		}
	}

	// colorbar
	cbX := imgWidth - marginRight + 20
	for py := 0; py < heatH; py++ {
		frac := 1.0 - float64(py)/float64(heatH-1)
		val := vmin + frac*(vmax-vmin)
		c := jetColormap(val, vmin, vmax)
		for dx := 0; dx < 20; dx++ {
			img.Set(cbX+dx, marginTop+py, c)
		}
	}
	drawColorbarTicks(img, cbX, marginTop, heatH, vmin, vmax)

	// Y-axis labels (altitudes)
	yTickCount := 6
	for i := 0; i <= yTickCount; i++ {
		frac := float64(i) / float64(yTickCount)
		alt := altitudes[0] + frac*(altitudes[len(altitudes)-1]-altitudes[0])
		y := marginTop + heatH - int(frac*float64(heatH))
		label := fmt.Sprintf("%.0f", alt)
		drawString(img, marginLeft-10, y+4, label, textAnchorRight)
	}

	// X-axis labels (time)
	times := make([]time.Time, len(profiles))
	for i, p := range profiles {
		times[i] = p.MeasurementStopTime
	}
	xTickCount := 8
	for i := 0; i <= xTickCount; i++ {
		frac := float64(i) / float64(xTickCount)
		tm := interpolateTime(times, frac)
		x := marginLeft + int(frac*float64(heatW-1))
		label := tm.Format("15:04")
		drawString(img, x, imgHeight-marginBottom/2+4, label, textAnchorCenter)
	}

	// Y-axis title
	drawVerticalString(img, 12, marginTop+heatH/2+40, "Altitude, m")

	// X-axis title
	drawString(img, marginLeft+heatW/2, imgHeight-8, "Time", textAnchorCenter)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dataRange(matrix [][]float64) (float64, float64) {
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

func bilinearInterp(matrix [][]float64, tf, af float64) float64 {
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

	v00 := matrix[t0][a0]
	v10 := matrix[t1][a0]
	v01 := matrix[t0][a1]
	v11 := matrix[t1][a1]

	if math.IsNaN(v00) {
		v00 = 0
	}
	if math.IsNaN(v10) {
		v10 = 0
	}
	if math.IsNaN(v01) {
		v01 = 0
	}
	if math.IsNaN(v11) {
		v11 = 0
	}

	v0 := v00*(1-ft) + v10*ft
	v1 := v01*(1-ft) + v11*ft
	return v0*(1-fa) + v1*fa
}

func jetColormap(val, vmin, vmax float64) color.Color {
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
	// jet: blue->cyan->green->yellow->red
	r, g, b := jetRGB(t)
	return color.RGBA{r, g, b, 255}
}

func jetRGB(t float64) (uint8, uint8, uint8) {
	// piecewise linear approximation of jet colormap
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

func drawColorbarTicks(img *image.RGBA, cbX, topY, cbH int, vmin, vmax float64) {
	tickCount := 5
	for i := 0; i <= tickCount; i++ {
		frac := float64(i) / float64(tickCount)
		y := topY + cbH - int(frac*float64(cbH))
		val := vmin + frac*(vmax-vmin)
		label := formatTick(val)
		drawString(img, cbX+24, y+4, label, textAnchorLeft)
	}
}

func formatTick(v float64) string {
	if v == 0 || (math.Abs(v) >= 0.01 && math.Abs(v) < 10000) {
		return fmt.Sprintf("%.2g", v)
	}
	return fmt.Sprintf("%.1e", v)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

type textAnchor int

const (
	textAnchorLeft   textAnchor = 0
	textAnchorCenter textAnchor = 1
	textAnchorRight  textAnchor = 2
)

func drawString(img *image.RGBA, x, y int, s string, anchor textAnchor) {
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	width := font.MeasureString(face, s).Ceil()
	switch anchor {
	case textAnchorCenter:
		d.Dot = fixed.P(x-width/2, y)
	case textAnchorRight:
		d.Dot = fixed.P(x-width, y)
	default:
		d.Dot = fixed.P(x, y)
	}
	d.DrawString(s)
}

func drawVerticalString(img *image.RGBA, x, y int, s string) {
	for i, r := range s {
		drawString(img, x, y+i*13, string(r), textAnchorLeft)
	}
}

func interpolateTime(times []time.Time, frac float64) time.Time {
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
