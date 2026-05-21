package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
	"github.com/lidar-platform/backend/pkg/plot"
)

type GenerateImageInput struct {
	ExperimentID string
	Wavelength   float64
	Polarization string
	ChannelType  string // "photon", "analog", "glued"
	PlotType     string // "Raw", "RangeCorrected", "LogRangeCorrected"
	GlueHmin     float64
	GlueHmax     float64
}

type GenerateImageResult struct {
	Path string
}

type GenerateProfileInput struct {
	ExperimentID string
	Wavelength   float64
	Polarization string
	ChannelType  string
	PlotType     string
	GlueHmin     float64
	GlueHmax     float64
}

type GenerateProfileResult struct {
	Path string
}

var ErrImageNotGenerated = errors.New("image not generated")

func (uc *ExperimentUseCase) GetImage(ctx context.Context, experimentID, wavelength, polarization, channelType, plotType string) (io.ReadCloser, error) {
	wavelengthFloat, err := strconv.ParseFloat(wavelength, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid wavelength: %w", err)
	}

	img, err := uc.generatedImage.FindByParams(ctx, ports.GeneratedImageParams{
		ExperimentID: experimentID,
		Wavelength:   wavelengthFloat,
		Polarization: polarization,
		ChannelType:  channelType,
		PlotType:     plotType,
		ImageType:    "heatmap",
	})
	if err != nil {
		return nil, fmt.Errorf("find generated image: %w", err)
	}
	if img == nil {
		return nil, ErrImageNotGenerated
	}

	rc, _, err := uc.storage.Download(ctx, img.ObjectPath)
	if err != nil {
		return nil, fmt.Errorf("image not found in storage: %w", err)
	}
	return rc, nil
}

func (uc *ExperimentUseCase) GenerateImage(ctx context.Context, input GenerateImageInput) (*GenerateImageResult, error) {
	profiles, err := uc.loadProfilesForRendering(ctx, input)
	if err != nil {
		return nil, err
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

	times := make([]time.Time, len(profiles))
	for i, p := range profiles {
		times[i] = p.MeasurementStopTime
	}

	pngBytes, err := plot.RenderHeatmap(matrix, altitudes, times)
	if err != nil {
		return nil, fmt.Errorf("render heatmap: %w", err)
	}

	fileName := fmt.Sprintf("time-height_%.1f_%s_%s_%s",
		input.Wavelength, input.Polarization,
		input.ChannelType, input.PlotType)
	objectPath := fmt.Sprintf("experiments/%s/imgs/%s.png", input.ExperimentID, fileName)

	jsonBytes, err := json.Marshal(plotlyHeatmapData{
		Z:    matrix,
		Y:    altitudes,
		X:    formatTimesISO(times),
		Type: "heatmap",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}
	jsonPath := fmt.Sprintf("experiments/%s/json/%s.json", input.ExperimentID, fileName)

	if err := uc.uploadAndSaveImage(ctx, input.ExperimentID, fileName, objectPath, jsonPath, input, "heatmap", pngBytes, jsonBytes); err != nil {
		return nil, err
	}

	return &GenerateImageResult{Path: objectPath}, nil
}

func (uc *ExperimentUseCase) GenerateProfile(ctx context.Context, input GenerateProfileInput) (*GenerateProfileResult, error) {
	profiles, err := uc.loadProfilesForRendering(ctx, GenerateImageInput{
		ExperimentID: input.ExperimentID,
		Wavelength:   input.Wavelength,
		Polarization: input.Polarization,
		ChannelType:  input.ChannelType,
		GlueHmin:     input.GlueHmin,
		GlueHmax:     input.GlueHmax,
	})
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles found for given parameters")
	}

	altitudes := extractCommonAltitudes(profiles)
	if len(altitudes) == 0 {
		return nil, fmt.Errorf("profiles have empty altitude grids")
	}

	// average over time
	averaged := make([]float64, len(altitudes))
	counts := make([]int, len(altitudes))
	for _, p := range profiles {
		for a, alt := range altitudes {
			val := interpolateProfile(p, alt)
			if !math.IsNaN(val) {
				averaged[a] += val
				counts[a]++
			}
		}
	}
	for a := range altitudes {
		if counts[a] > 0 {
			averaged[a] /= float64(counts[a])
		} else {
			averaged[a] = math.NaN()
		}
	}

	// apply plot type transformation
	for a, alt := range altitudes {
		v := averaged[a]
		if math.IsNaN(v) {
			continue
		}
		switch input.PlotType {
		case "Raw":
			// no transformation
		case "RangeCorrected":
			averaged[a] = v * alt * alt
		case "LogRangeCorrected":
			rc := v * alt * alt
			if rc > 0 {
				averaged[a] = math.Log10(rc)
			} else {
				averaged[a] = math.NaN()
			}
		}
	}

	pngBytes, err := plot.RenderProfile(altitudes, averaged)
	if err != nil {
		return nil, fmt.Errorf("render profile: %w", err)
	}

	fileName := fmt.Sprintf("profile_%.1f_%s_%s_%s",
		input.Wavelength, input.Polarization,
		input.ChannelType, input.PlotType)
	objectPath := fmt.Sprintf("experiments/%s/imgs/%s.png", input.ExperimentID, fileName)

	jsonBytes, err := json.Marshal(plotlyProfileData{
		X:    altitudes,
		Y:    averaged,
		Type: "scatter",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}
	jsonPath := fmt.Sprintf("experiments/%s/json/%s.json", input.ExperimentID, fileName)

	if err := uc.uploadAndSaveImage(ctx, input.ExperimentID, fileName, objectPath, jsonPath, GenerateImageInput{
		Wavelength:   input.Wavelength,
		Polarization: input.Polarization,
		ChannelType:  input.ChannelType,
		PlotType:     input.PlotType,
	}, "profile", pngBytes, jsonBytes); err != nil {
		return nil, err
	}

	return &GenerateProfileResult{Path: objectPath}, nil
}

func (uc *ExperimentUseCase) uploadAndSaveImage(ctx context.Context, experimentID, fileName, objectPath, jsonPath string, input GenerateImageInput, imageType string, pngBytes, jsonBytes []byte) error {
	if err := uc.storage.Upload(ctx, objectPath, bytes.NewReader(pngBytes), int64(len(pngBytes)), "image/png"); err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	if err := uc.storage.Upload(ctx, jsonPath, bytes.NewReader(jsonBytes), int64(len(jsonBytes)), "application/json"); err != nil {
		return fmt.Errorf("upload json: %w", err)
	}

	generatedImg := &domain.GeneratedImage{
		ID:           uuid.New().String(),
		ExperimentID: experimentID,
		FileName:     fileName,
		ObjectPath:   objectPath,
		JsonData:     jsonPath,
		Wavelength:   input.Wavelength,
		Polarization: input.Polarization,
		ChannelType:  input.ChannelType,
		PlotType:     input.PlotType,
		ImageType:    imageType,
		CreatedAt:    time.Now(),
	}
	if err := uc.generatedImage.Create(ctx, generatedImg); err != nil {
		return fmt.Errorf("save generated image: %w", err)
	}
	return nil
}

func (uc *ExperimentUseCase) loadProfilesForRendering(ctx context.Context, input GenerateImageInput) ([]*domain.ExperimentProfile, error) {
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
		profiles = gluePairs(photonList, analogList, input.GlueHmin, input.GlueHmax)
	}
	return profiles, nil
}

type plotlyHeatmapData struct {
	Z    [][]float64 `json:"z"`
	Y    []float64   `json:"y"`
	X    []string    `json:"x"`
	Type string      `json:"type"`
}

type plotlyProfileData struct {
	X    []float64 `json:"x"`
	Y    []float64 `json:"y"`
	Type string    `json:"type"`
}

func formatTimesISO(times []time.Time) []string {
	out := make([]string, len(times))
	for i, t := range times {
		out[i] = t.Format(time.RFC3339)
	}
	return out
}
