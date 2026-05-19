package ports

import (
	"context"

	"github.com/lidar-platform/backend/internal/domain"
)

type ExperimentProfileParams struct {
	ExperimentID string
	Wavelength   float64
	Polarization string
	Photon       *bool // nil = don't filter, true = photon, false = analog
}

type ExperimentProfileRepository interface {
	Create(ctx context.Context, p *domain.ExperimentProfile) error
	FindByExperimentID(ctx context.Context, experimentID string) ([]*domain.ExperimentProfile, error)
	FindByParams(ctx context.Context, params ExperimentProfileParams) ([]*domain.ExperimentProfile, error)
	DeleteByExperimentID(ctx context.Context, experimentID string) error
}
