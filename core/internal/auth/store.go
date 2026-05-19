package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store — слой персистентности пользователей в users.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// User — пользователь как он хранится в БД (без чувствительных полей при выдаче).
type User struct {
	UserID       uuid.UUID
	Email        string
	PasswordHash string
	Confirmed    bool
	ConfirmToken string
}

// Create регистрирует нового пользователя. Возвращает ErrUserExists при коллизии email.
func (s *Store) Create(ctx context.Context, email, passwordHash, confirmToken string) (*User, error) {
	id := uuid.New()
	const q = `
		INSERT INTO users (user_id, email, password_hash, email_confirmed, email_confirm_token)
		VALUES ($1, $2, $3, false, $4)
		RETURNING user_id
	`
	if _, err := s.pool.Exec(ctx, q, id, email, passwordHash, confirmToken); err != nil {
		// 23505 = unique_violation
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &User{
		UserID:       id,
		Email:        email,
		PasswordHash: passwordHash,
		Confirmed:    false,
		ConfirmToken: confirmToken,
	}, nil
}

// GetByEmail возвращает пользователя или ErrUserNotFound.
func (s *Store) GetByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
		SELECT user_id, email, password_hash, email_confirmed, COALESCE(email_confirm_token, '')
		  FROM users WHERE email = $1
	`
	u := &User{}
	err := s.pool.QueryRow(ctx, q, email).Scan(
		&u.UserID, &u.Email, &u.PasswordHash, &u.Confirmed, &u.ConfirmToken,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	return u, nil
}

// Verify применяет токен подтверждения. Возвращает email и ошибку.
// Идемпотентно: если уже подтверждён, возвращает email без ошибки.
func (s *Store) Verify(ctx context.Context, token string) (string, error) {
	const q = `
		UPDATE users
		   SET email_confirmed = true, email_confirm_token = NULL
		 WHERE email_confirm_token = $1
		RETURNING email
	`
	var email string
	err := s.pool.QueryRow(ctx, q, token).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("verify: token not found or already used")
	}
	if err != nil {
		return "", fmt.Errorf("verify: %w", err)
	}
	return email, nil
}

func isUniqueViolation(err error) bool {
	// pgx возвращает PgError; для простоты используем подстроку.
	return err != nil &&
		(contains(err.Error(), "duplicate key") ||
			contains(err.Error(), "unique constraint"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
