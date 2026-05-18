package ports

import "context"

type TokenProvider interface {
	Generate(ctx context.Context, userID string) (string, error)
	Validate(ctx context.Context, token string) (userID string, err error)
}
