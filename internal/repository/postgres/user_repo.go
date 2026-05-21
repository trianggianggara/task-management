package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"task-management/internal/domain"
	"task-management/internal/repository"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) repository.UserRepository {
	return &userRepo{db: db}
}

var ErrNotFound = errors.New("record not found")

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, team_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.TeamID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}
	query := `
		SELECT id, email, password_hash, name, team_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.TeamID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	return user, nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user := &domain.User{}
	query := `
		SELECT id, email, password_hash, name, team_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.TeamID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	return user, nil
}

func (r *userRepo) UpdateTeamID(ctx context.Context, userID string, teamID *string) error {
	query := `
		UPDATE users SET team_id = $1, updated_at = NOW()
		WHERE id = $2
	`
	result, err := r.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		return fmt.Errorf("userRepo.UpdateTeamID: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *userRepo) FindTeamByCode(ctx context.Context, code string) (*domain.Team, error) {
	team := &domain.Team{}
	query := `SELECT id, code, name, created_at FROM teams WHERE code = $1`
	err := r.db.QueryRowContext(ctx, query, code).Scan(&team.ID, &team.Code, &team.Name, &team.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindTeamByCode: %w", err)
	}
	return team, nil
}
