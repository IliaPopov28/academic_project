package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailTaken = errors.New("email already registered")
var ErrNotFound = errors.New("user not found")

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u User) error {
	_, err := r.db.Exec(ctx, `
	INSERT INTO users (id, email, password_hash, name)
	VALUES ($1, $2, $3, $4)
	`, u.ID, u.Email, u.PasswordHash, u.Name)
	if isUniqueViolation(err) {
		return ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User

	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("select user: %w", err)
	}

	return u, nil
}
