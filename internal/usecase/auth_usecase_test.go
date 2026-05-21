package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"task-management/internal/domain"
	"task-management/internal/usecase"
)

type mockUserRepo struct {
	createFunc     func(ctx context.Context, user *domain.User) error
	findByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	findByIDFunc   func(ctx context.Context, id string) (*domain.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.createFunc(ctx, user)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.findByEmailFunc(ctx, email)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return m.findByIDFunc(ctx, id)
}

type mockPasswordHasher struct {
	hashFunc   func(plain string) (string, error)
	verifyFunc func(plain, hashed string) error
}

func (m *mockPasswordHasher) Hash(plain string) (string, error) {
	return m.hashFunc(plain)
}

func (m *mockPasswordHasher) Verify(plain, hashed string) error {
	return m.verifyFunc(plain, hashed)
}

func TestRegister_Success(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, user *domain.User) error {
			user.ID = "user-1"
			return nil
		},
	}

	passwordSvc := &mockPasswordHasher{
		hashFunc: func(plain string) (string, error) {
			return "$2a$12$hashed", nil
		},
	}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	user, token, err := uc.Register(context.Background(), "Test User", "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "user-1", user.ID)
	assert.NotEmpty(t, token)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: "existing", Email: "test@example.com"}, nil
		},
	}

	passwordSvc := &mockPasswordHasher{}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	_, _, err := uc.Register(context.Background(), "Test", "test@example.com", "password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email already registered")
}

func TestRegister_FindEmailError(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("db connection error")
		},
	}

	passwordSvc := &mockPasswordHasher{}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	_, _, err := uc.Register(context.Background(), "Test", "test@example.com", "password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check email")
}

func TestLogin_Success(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-1",
				Email:        "test@example.com",
				PasswordHash: "$2a$12$hashed",
			}, nil
		},
	}

	passwordSvc := &mockPasswordHasher{
		verifyFunc: func(plain, hashed string) error {
			return nil
		},
	}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	token, err := uc.Login(context.Background(), "test@example.com", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-1",
				Email:        "test@example.com",
				PasswordHash: "$2a$12$hashed",
			}, nil
		},
	}

	passwordSvc := &mockPasswordHasher{
		verifyFunc: func(plain, hashed string) error {
			return errors.New("hash mismatch")
		},
	}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	_, err := uc.Login(context.Background(), "test@example.com", "wrongpassword")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		},
	}

	passwordSvc := &mockPasswordHasher{}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	_, err := uc.Login(context.Background(), "nonexistent@example.com", "password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestLogin_DBError(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, errors.New("db connection error")
		},
	}

	passwordSvc := &mockPasswordHasher{}

	uc := usecase.NewAuthUsecase(userRepo, passwordSvc, "my-secret-key", 24)

	_, err := uc.Login(context.Background(), "test@example.com", "password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find user")
}
