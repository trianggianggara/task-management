package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"task-management/internal/apperror"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/pkg/password"
)

type AuthUsecase interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authUsecase struct {
	userRepo  repository.UserRepository
	hasher    password.Hasher
	jwtSecret []byte
	jwtExpiry time.Duration
}

func NewAuthUsecase(userRepo repository.UserRepository, hasher password.Hasher, jwtSecret string, jwtExpiry time.Duration) AuthUsecase {
	return &authUsecase{
		userRepo:  userRepo,
		hasher:    hasher,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

func (uc *authUsecase) Register(ctx context.Context, name, email, password string) (*domain.User, string, error) {
	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", apperror.Internal("failed to check email", err)
	}
	if existing != nil {
		return nil, "", apperror.Conflict("email already registered")
	}

	hashed, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, "", apperror.Internal("failed to hash password", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hashed,
		Name:         name,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, "", apperror.Internal("failed to create user", err)
	}

	token, err := uc.generateJWT(user.ID, user.TeamID)
	if err != nil {
		return nil, "", apperror.Internal("failed to generate token", err)
	}

	return user, token, nil
}

func (uc *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", apperror.Internal("failed to find user", err)
	}
	if user == nil {
		return "", apperror.Unauthorized("invalid email or password")
	}

	if err := uc.hasher.Verify(password, user.PasswordHash); err != nil {
		return "", apperror.Unauthorized("invalid email or password")
	}

	token, err := uc.generateJWT(user.ID, user.TeamID)
	if err != nil {
		return "", apperror.Internal("failed to generate token", err)
	}

	return token, nil
}

func (uc *authUsecase) generateJWT(userID string, teamID *string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"iat":  now.Unix(),
		"exp":  now.Add(uc.jwtExpiry).Unix(),
	}
	if teamID != nil {
		claims["team_id"] = *teamID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(uc.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return tokenStr, nil
}
