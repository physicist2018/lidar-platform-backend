package ports

import (
	"context"

	"github.com/lidar-platform/backend/internal/domain"
)

type ExperimentRepository interface {
	Create(ctx context.Context, exp *domain.Experiment) error
	FindByID(ctx context.Context, id string) (*domain.Experiment, error)
}
