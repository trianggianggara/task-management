package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"task-management/pkg/utils/response"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/pkg/utils/password"
)

type AuthUsecase interface {
	Register(ctx context.Context, name, email, password string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (string, error)
	JoinTeam(ctx context.Context, userID, code string) error
	LeaveTeam(ctx context.Context, userID string) error
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
		return nil, "", response.Internal("failed to check email", err)
	}
	if existing != nil {
		return nil, "", response.Conflict("email already registered")
	}

	hashed, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, "", response.Internal("failed to hash password", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hashed,
		Name:         name,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, "", response.Internal("failed to create user", err)
	}

	token, err := uc.generateJWT(user.ID, user.TeamID)
	if err != nil {
		return nil, "", response.Internal("failed to generate token", err)
	}

	return user, token, nil
}

func (uc *authUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", response.Internal("failed to find user", err)
	}
	if user == nil {
		return "", response.Unauthorized("invalid email or password")
	}

	if err := uc.hasher.Verify(password, user.PasswordHash); err != nil {
		return "", response.Unauthorized("invalid email or password")
	}

	token, err := uc.generateJWT(user.ID, user.TeamID)
	if err != nil {
		return "", response.Internal("failed to generate token", err)
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

func (uc *authUsecase) JoinTeam(ctx context.Context, userID, code string) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return response.Internal("failed to find user", err)
	}
	if user == nil {
		return response.NotFound("user not found")
	}

	team, err := uc.userRepo.FindTeamByCode(ctx, code)
	if err != nil {
		return response.Internal("failed to find team", err)
	}
	if team == nil {
		return response.NotFound("team not found")
	}

	if err := uc.userRepo.UpdateTeamID(ctx, userID, &team.ID); err != nil {
		return response.Internal("failed to join team", err)
	}

	return nil
}

func (uc *authUsecase) LeaveTeam(ctx context.Context, userID string) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return response.Internal("failed to find user", err)
	}
	if user == nil {
		return response.NotFound("user not found")
	}

	if err := uc.userRepo.UpdateTeamID(ctx, userID, nil); err != nil {
		return response.Internal("failed to leave team", err)
	}

	return nil
}
