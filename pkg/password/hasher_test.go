package password_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"task-management/pkg/password"
)

func TestBcryptHasher_Hash_and_Verify(t *testing.T) {
	h, err := password.NewBcryptHasher(12)
	require.NoError(t, err)

	plain := "my-secret-password"
	hashed, err := h.Hash(plain)
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, plain, hashed)

	err = h.Verify(plain, hashed)
	assert.NoError(t, err)
}

func TestBcryptHasher_Verify_WrongPassword(t *testing.T) {
	h, err := password.NewBcryptHasher(12)
	require.NoError(t, err)

	hashed, err := h.Hash("correct")
	require.NoError(t, err)

	err = h.Verify("wrong", hashed)
	assert.Error(t, err)
}

func TestBcryptHasher_Verify_TamperedHash(t *testing.T) {
	h, err := password.NewBcryptHasher(12)
	require.NoError(t, err)

	err = h.Verify("anything", "not-a-valid-bcrypt-hash")
	assert.Error(t, err)
}

func TestBcryptHasher_InvalidCost(t *testing.T) {
	_, err := password.NewBcryptHasher(0)
	assert.Error(t, err)

	_, err = password.NewBcryptHasher(32)
	assert.Error(t, err)
}
