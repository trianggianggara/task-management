package password

import (
	"golang.org/x/crypto/bcrypt"
)

type Hasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hashed string) error
}

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() Hasher {
	return &BcryptHasher{cost: 12}
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
