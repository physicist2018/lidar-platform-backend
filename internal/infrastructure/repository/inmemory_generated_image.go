package repository

import (
	"context"
	"strconv"
	"sync"

	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
)

type InMemoryGeneratedImageRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.GeneratedImage
}

func NewInMemoryGeneratedImageRepository() *InMemoryGeneratedImageRepository {
	return &InMemoryGeneratedImageRepository{
		items: make(map[string]*domain.GeneratedImage),
	}
}

func (r *InMemoryGeneratedImageRepository) Create(_ context.Context, img *domain.GeneratedImage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	imageType := img.ImageType
	if imageType == "" {
		imageType = "heatmap"
	}
	key := imageKey(img.ExperimentID, img.Wavelength, img.Polarization, img.ChannelType, img.PlotType, imageType)
	r.items[key] = img
	return nil
}

func (r *InMemoryGeneratedImageRepository) FindByParams(_ context.Context, params ports.GeneratedImageParams) (*domain.GeneratedImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := imageKey(params.ExperimentID, params.Wavelength, params.Polarization, params.ChannelType, params.PlotType, params.ImageType)
	img, ok := r.items[key]
	if !ok {
		return nil, nil
	}
	return img, nil
}

func imageKey(experimentID string, wavelength float64, polarization, channelType, plotType, imageType string) string {
	return experimentID + "|" +
		strconv.FormatFloat(wavelength, 'f', 1, 64) + "|" +
		polarization + "|" +
		channelType + "|" +
		plotType + "|" +
		imageType
}
