package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Hasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hashed string) error
}

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) (Hasher, error) {
	if cost < 4 || cost > 31 {
		return nil, fmt.Errorf("bcrypt cost must be between 4 and 31, got %d", cost)
	}
	return &BcryptHasher{cost: cost}, nil
}

func (h *BcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *BcryptHasher) Verify(plain, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
