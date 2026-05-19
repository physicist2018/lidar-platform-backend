package ports

import (
	"context"

	"github.com/lidar-platform/backend/internal/domain"
)

type ExperimentProfileRepository interface {
	Create(ctx context.Context, p *domain.ExperimentProfile) error
	FindByExperimentID(ctx context.Context, experimentID string) ([]*domain.ExperimentProfile, error)
	DeleteByExperimentID(ctx context.Context, experimentID string) error
}
