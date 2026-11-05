package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type PasswordResetRepository struct {
	DB *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{DB: db}
}

func (r *PasswordResetRepository) InvalidateActiveByUserID(ctx context.Context, userID string) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE password_reset_tokens
			SET used_at = NOW()
			WHERE user_id = $1
			  AND used_at IS NULL
			  AND expires_at > NOW()
		`,
		userID,
	)
	return err
}

func (r *PasswordResetRepository) Create(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) (*models.PasswordResetToken, error) {
	token := &models.PasswordResetToken{}
	err := r.DB.QueryRow(
		ctx,
		`
			INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)
			RETURNING id::text, user_id::text, token_hash, expires_at, used_at, created_at
		`,
		userID,
		tokenHash,
		expiresAt,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *PasswordResetRepository) FindValidByHash(
	ctx context.Context,
	tokenHash string,
) (*models.PasswordResetToken, error) {
	token := &models.PasswordResetToken{}
	err := r.DB.QueryRow(
		ctx,
		`
			SELECT id::text, user_id::text, token_hash, expires_at, used_at, created_at
			FROM password_reset_tokens
			WHERE token_hash = $1
			  AND used_at IS NULL
			  AND expires_at > NOW()
		`,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE password_reset_tokens
			SET used_at = NOW()
			WHERE id = $1
			  AND used_at IS NULL
		`,
		id,
	)
	return err
}
