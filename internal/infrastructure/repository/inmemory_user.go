package repository

import (
	"context"
	"sync"

	"github.com/lidar-platform/backend/internal/domain"
)

type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (r *InMemoryUserRepository) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (r *InMemoryUserRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}
