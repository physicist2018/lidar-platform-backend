package repository

import (
	"context"
	"sync"

	"github.com/lidar-platform/backend/internal/domain"
)

type InMemoryExperimentProfileRepository struct {
	mu    sync.RWMutex
	data  map[string]*domain.ExperimentProfile // profileID -> profile
	index map[string][]string                  // experimentID -> []profileID
}

func NewInMemoryExperimentProfileRepository() *InMemoryExperimentProfileRepository {
	return &InMemoryExperimentProfileRepository{
		data:  make(map[string]*domain.ExperimentProfile),
		index: make(map[string][]string),
	}
}

func (r *InMemoryExperimentProfileRepository) Create(_ context.Context, p *domain.ExperimentProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[p.ID] = p
	r.index[p.ExperimentID] = append(r.index[p.ExperimentID], p.ID)
	return nil
}

func (r *InMemoryExperimentProfileRepository) FindByExperimentID(_ context.Context, experimentID string) ([]*domain.ExperimentProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.index[experimentID]
	result := make([]*domain.ExperimentProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.data[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *InMemoryExperimentProfileRepository) DeleteByExperimentID(_ context.Context, experimentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range r.index[experimentID] {
		delete(r.data, id)
	}
	delete(r.index, experimentID)
	return nil
}
