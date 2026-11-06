package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type EmailVerificationRepository struct {
	DB *pgxpool.Pool
}

func NewEmailVerificationRepository(db *pgxpool.Pool) *EmailVerificationRepository {
	return &EmailVerificationRepository{DB: db}
}

func (r *EmailVerificationRepository) InvalidateActiveByUserID(ctx context.Context, userID string) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE email_verification_tokens
			SET used_at = NOW()
			WHERE user_id = $1
			  AND used_at IS NULL
			  AND expires_at > NOW()
		`,
		userID,
	)
	return err
}

func (r *EmailVerificationRepository) Create(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) (*models.EmailVerificationToken, error) {
	token := &models.EmailVerificationToken{}
	err := r.DB.QueryRow(
		ctx,
		`
			INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
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

func (r *EmailVerificationRepository) FindValidByHash(
	ctx context.Context,
	tokenHash string,
) (*models.EmailVerificationToken, error) {
	token := &models.EmailVerificationToken{}
	err := r.DB.QueryRow(
		ctx,
		`
			SELECT id::text, user_id::text, token_hash, expires_at, used_at, created_at
			FROM email_verification_tokens
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

func (r *EmailVerificationRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.DB.Exec(
		ctx,
		`
			UPDATE email_verification_tokens
			SET used_at = NOW()
			WHERE id = $1
			  AND used_at IS NULL
		`,
		id,
	)
	return err
}
