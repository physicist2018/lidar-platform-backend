package auth

import (
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *BcryptHasher) Compare(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
