package ports

import (
	"context"

	"github.com/lidar-platform/backend/internal/domain"
)

type GeneratedImageParams struct {
	ExperimentID string
	Wavelength   float64
	Polarization string
	ChannelType  string
	PlotType     string
}

type GeneratedImageRepository interface {
	Create(ctx context.Context, img *domain.GeneratedImage) error
	FindByParams(ctx context.Context, params GeneratedImageParams) (*domain.GeneratedImage, error)
}
