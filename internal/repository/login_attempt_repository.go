package repository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultMaxLoginAttempts = 5
	DefaultLockDuration     = 60 * time.Minute
)

type LoginAttemptRepository struct {
	DB           *pgxpool.Pool
	MaxAttempts  int
	LockDuration time.Duration
}

func NewLoginAttemptRepository(db *pgxpool.Pool) *LoginAttemptRepository {
	return &LoginAttemptRepository{
		DB:           db,
		MaxAttempts:  envInt("LOGIN_MAX_ATTEMPTS", DefaultMaxLoginAttempts),
		LockDuration: time.Duration(envInt("LOGIN_LOCK_MINUTES", 60)) * time.Minute,
	}
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *LoginAttemptRepository) IsLocked(ctx context.Context, email string) (bool, error) {
	email = normalizeEmail(email)

	var lockedUntil *time.Time
	err := r.DB.QueryRow(
		ctx,
		`SELECT locked_until FROM login_attempts WHERE email = $1`,
		email,
	).Scan(&lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return lockedUntil != nil && lockedUntil.After(time.Now()), nil
}

func (r *LoginAttemptRepository) RegisterFailure(ctx context.Context, email string) (bool, error) {
	email = normalizeEmail(email)
	maxAttempts := r.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = DefaultMaxLoginAttempts
	}
	lockDuration := r.LockDuration
	if lockDuration <= 0 {
		lockDuration = DefaultLockDuration
	}

	query := `
		INSERT INTO login_attempts (email, failed_count, locked_until, updated_at)
		VALUES ($1, 1, NULL, NOW())
		ON CONFLICT (email) DO UPDATE SET
			failed_count = CASE
				WHEN login_attempts.locked_until IS NOT NULL
					AND login_attempts.locked_until <= NOW()
				THEN 1
				ELSE login_attempts.failed_count + 1
			END,
			locked_until = CASE
				WHEN (
					CASE
						WHEN login_attempts.locked_until IS NOT NULL
							AND login_attempts.locked_until <= NOW()
						THEN 1
						ELSE login_attempts.failed_count + 1
					END
				) >= $2
				THEN NOW() + ($3 * INTERVAL '1 minute')
				WHEN login_attempts.locked_until IS NOT NULL
					AND login_attempts.locked_until <= NOW()
				THEN NULL
				ELSE login_attempts.locked_until
			END,
			updated_at = NOW()
		RETURNING failed_count, locked_until
	`

	var failedCount int
	var lockedUntil *time.Time
	err := r.DB.QueryRow(
		ctx,
		query,
		email,
		maxAttempts,
		int(lockDuration.Minutes()),
	).Scan(&failedCount, &lockedUntil)
	if err != nil {
		return false, err
	}

	return lockedUntil != nil && lockedUntil.After(time.Now()), nil
}

func (r *LoginAttemptRepository) Reset(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	_, err := r.DB.Exec(
		ctx,
		`
		INSERT INTO login_attempts (email, failed_count, locked_until, updated_at)
		VALUES ($1, 0, NULL, NOW())
		ON CONFLICT (email) DO UPDATE SET
			failed_count = 0,
			locked_until = NULL,
			updated_at = NOW()
		`,
		email,
	)
	return err
}
