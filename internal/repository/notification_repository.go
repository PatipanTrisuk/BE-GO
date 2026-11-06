package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type NotificationRepository struct {
	DB *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

func scanNotification(
	scanner interface {
		Scan(dest ...any) error
	},
	item *models.Notification,
) error {
	var refType sql.NullString
	var refID *uuid.UUID
	var readAt sql.NullTime

	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.Type,
		&refType,
		&refID,
		&item.MetaJSON,
		&readAt,
		&item.CreatedAt,
	); err != nil {
		return err
	}

	if refType.Valid {
		v := refType.String
		item.RefType = &v
	}
	item.RefID = refID
	if readAt.Valid {
		t := readAt.Time
		item.ReadAt = &t
	}
	return nil
}

const notificationSelect = `
	id, user_id, type, ref_type, ref_id, meta_json, read_at, created_at
`

func (r *NotificationRepository) Create(
	ctx context.Context,
	userID, notifType string,
	refType *string,
	refID *uuid.UUID,
	metaJSON string,
) (*models.Notification, error) {
	item := &models.Notification{}
	err := scanNotification(r.DB.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, ref_type, ref_id, meta_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+notificationSelect,
		userID, notifType, refType, refID, metaJSON,
	), item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *NotificationRepository) ListByUser(ctx context.Context, userID string, limit int) ([]models.Notification, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.DB.Query(ctx, `
		SELECT `+notificationSelect+`
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Notification, 0)
	for rows.Next() {
		var item models.Notification
		if err := scanNotification(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	tag, err := r.DB.Exec(ctx, `
		UPDATE notifications
		SET read_at = $3
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL
	`, id, userID, time.Now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Already read or not found — treat missing as ErrNoRows only when not owned.
		var exists bool
		_ = r.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND user_id = $2)`, id, userID).Scan(&exists)
		if !exists {
			return pgx.ErrNoRows
		}
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE notifications
		SET read_at = $2
		WHERE user_id = $1 AND read_at IS NULL
	`, userID, time.Now())
	return err
}
