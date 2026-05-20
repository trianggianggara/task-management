package postgres

import (
	"context"

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

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, team_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.TeamID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
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
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	return user, nil
}
