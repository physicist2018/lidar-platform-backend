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
	key := imageKey(img.ExperimentID, img.Wavelength, img.Polarization, img.ChannelType, img.PlotType)
	r.items[key] = img
	return nil
}

func (r *InMemoryGeneratedImageRepository) FindByParams(_ context.Context, params ports.GeneratedImageParams) (*domain.GeneratedImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := imageKey(params.ExperimentID, params.Wavelength, params.Polarization, params.ChannelType, params.PlotType)
	img, ok := r.items[key]
	if !ok {
		return nil, nil
	}
	return img, nil
}

func imageKey(experimentID string, wavelength float64, polarization, channelType, plotType string) string {
	return experimentID + "|" +
		strconv.FormatFloat(wavelength, 'f', 1, 64) + "|" +
		polarization + "|" +
		channelType + "|" +
		plotType
}
