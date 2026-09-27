package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	GetByEmail(ctx context.Context, email string) (db.AppUser, error)
	GetById(ctx context.Context, id uuid.UUID) (db.AppUser, error)
}

type PostgresRepository struct {
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: db.New(pool)}
}

func (repository *PostgresRepository) GetByEmail(ctx context.Context, email string) (db.AppUser, error) {
	user, err := repository.queries.GetUserByEmail(ctx, email)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.AppUser{}, ErrUserNotFound
	}

	if err != nil {
		return db.AppUser{}, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (repository *PostgresRepository) Upsert(ctx context.Context, params db.UpsertUserParams) (db.AppUser, error) {
	user, err := repository.queries.UpsertUser(ctx, params)

	if err != nil {
		return db.AppUser{}, fmt.Errorf("failed to upsert user: %w", err)
	}

	return user, nil
}

func (repository *PostgresRepository) GetById(ctx context.Context, id uuid.UUID) (db.AppUser, error) {
	user, err := repository.queries.GetUser(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.AppUser{}, ErrUserNotFound
	}

	if err != nil {
		return db.AppUser{}, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
