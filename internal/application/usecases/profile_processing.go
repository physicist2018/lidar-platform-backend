package usecases

import (
	"math"
	"sort"

	"github.com/lidar-platform/backend/internal/domain"
)

// extractCommonAltitudes returns altitudes present in all profiles, sorted ascending.
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

// buildDataMatrix creates a 2D slice times × altitudes with transformed values.
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
			case "Raw":
				// no transformation
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

// interpolateProfile returns a linearly interpolated data value at the given altitude.
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
