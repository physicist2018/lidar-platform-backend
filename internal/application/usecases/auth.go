package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthUseCase struct {
	repo  ports.UserRepository
	hash  ports.PasswordHasher
	token ports.TokenProvider
}

func NewAuthUseCase(repo ports.UserRepository, hash ports.PasswordHasher, token ports.TokenProvider) *AuthUseCase {
	return &AuthUseCase{repo: repo, hash: hash, token: token}
}

func (uc *AuthUseCase) Register(ctx context.Context, email, password string) (*domain.User, error) {
	existing, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := uc.hash.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}

	if err := uc.hash.Compare(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := uc.token.Generate(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

func (uc *AuthUseCase) Validate(ctx context.Context, token string) (string, error) {
	userID, err := uc.token.Validate(ctx, token)
	if err != nil {
		return "", ErrInvalidToken
	}

	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidToken
	}

	return user.ID, nil
}
