package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

const uniqueViolationCode = "23505"

// UserRepository adapts the sqlc-generated Queries to domain.UserRepository.
type UserRepository struct {
	q *Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{q: New(pool)}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	row, err := r.q.CreateUser(ctx, CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.ErrUserAlreadyExists
		}
		return err
	}
	*user = *toDomainUser(row)
	return nil
}

func toDomainUser(row User) *domain.User {
	return &domain.User{
		ID:           fromUUID(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		CreatedAt:    fromTimestamptz(row.CreatedAt),
		DeletedAt:    fromNullTimestamptz(row.DeletedAt),
		DeletedBy:    fromNullUUID(row.DeletedBy),
	}
}
