package repository

import (
	"context"
	"sync"

	"github.com/lidar-platform/backend/internal/domain"
)

type InMemoryExperimentRepository struct {
	mu          sync.RWMutex
	experiments map[string]*domain.Experiment
}

func NewInMemoryExperimentRepository() *InMemoryExperimentRepository {
	return &InMemoryExperimentRepository{
		experiments: make(map[string]*domain.Experiment),
	}
}

func (r *InMemoryExperimentRepository) Create(_ context.Context, exp *domain.Experiment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.experiments[exp.ID] = exp
	return nil
}

func (r *InMemoryExperimentRepository) FindByID(_ context.Context, id string) (*domain.Experiment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exp, ok := r.experiments[id]
	if !ok {
		return nil, nil
	}
	return exp, nil
}

func (r *InMemoryExperimentRepository) Update(_ context.Context, exp *domain.Experiment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.experiments[exp.ID]; !ok {
		return nil
	}
	r.experiments[exp.ID] = exp
	return nil
}
