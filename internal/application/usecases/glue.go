package usecases

import (
	"github.com/lidar-platform/backend/internal/domain"
)

// computeGlueCoefficient calculates the average ratio of photon to analog
// signals within the given altitude region.
func computeGlueCoefficient(photon, analog *domain.ExperimentProfile, glueHmin, glueHmax float64) float64 {
	n := len(photon.Data)
	if len(analog.Data) < n {
		n = len(analog.Data)
	}

	indices := findGlueRegionByAltitudes(photon.Altitudes[:n], glueHmin, glueHmax)
	if len(indices) == 0 {
		return 0
	}

	var sumRatio float64
	count := 0
	for _, i := range indices {
		if analog.Data[i] == 0 {
			continue
		}
		sumRatio += photon.Data[i] / analog.Data[i]
		count++
	}
	if count == 0 {
		return 0
	}
	return sumRatio / float64(count)
}

// findGlueRegionByAltitudes returns indices of altitudes within [hmin, hmax].
func findGlueRegionByAltitudes(altitudes []float64, hmin, hmax float64) []int {
	var indices []int
	for i, alt := range altitudes {
		if alt >= hmin && alt <= hmax {
			indices = append(indices, i)
		}
	}
	return indices
}

// gluePairs merges paired photon/analog profiles into glued profiles.
func gluePairs(photonList, analogList []*domain.ExperimentProfile, glueHmin, glueHmax float64) []*domain.ExperimentProfile {
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
		k := computeGlueCoefficient(photon, analog, glueHmin, glueHmax)
		for i := range photon.Data {
			pVal := photon.Data[i]
			aVal := analog.Data[i]
			alt := photon.Altitudes[i]
			switch {
			case alt > glueHmax:
				g.Data[i] = pVal
			case alt < glueHmin:
				g.Data[i] = k * aVal
			default:
				g.Data[i] = (pVal + k*aVal) / 2.0
			}
		}
		glued = append(glued, g)
	}
	return glued
}
