package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrTokenInvalid = errors.New("token is invalid")

type JWTProvider struct {
	secret []byte
	expiry time.Duration
}

func NewJWTProvider(secret string, expiry time.Duration) *JWTProvider {
	return &JWTProvider{
		secret: []byte(secret),
		expiry: expiry,
	}
}

func (p *JWTProvider) Generate(_ context.Context, userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(p.expiry).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.secret)
}

func (p *JWTProvider) Validate(_ context.Context, tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return p.secret, nil
	})
	if err != nil {
		return "", ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", ErrTokenInvalid
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", ErrTokenInvalid
	}

	return userID, nil
}
