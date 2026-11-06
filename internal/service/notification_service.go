package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"go-be/internal/models"
	"go-be/internal/repository"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationDTO struct {
	ID        uuid.UUID      `json:"id"`
	Type      string         `json:"type"`
	RefType   *string        `json:"ref_type,omitempty"`
	RefID     *uuid.UUID     `json:"ref_id,omitempty"`
	Meta      map[string]any `json:"meta"`
	ReadAt    *string        `json:"read_at,omitempty"`
	CreatedAt string         `json:"created_at"`
}

type NotificationService struct {
	Repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{Repo: repo}
}

func toNotificationDTO(item models.Notification) NotificationDTO {
	meta := map[string]any{}
	if item.MetaJSON != "" {
		_ = json.Unmarshal([]byte(item.MetaJSON), &meta)
	}

	dto := NotificationDTO{
		ID:        item.ID,
		Type:      item.Type,
		RefType:   item.RefType,
		RefID:     item.RefID,
		Meta:      meta,
		CreatedAt: item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if item.ReadAt != nil {
		s := item.ReadAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		dto.ReadAt = &s
	}
	return dto
}

func (s *NotificationService) List(ctx context.Context, userID string, limit int) ([]NotificationDTO, int, error) {
	items, err := s.Repo.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.Repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]NotificationDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toNotificationDTO(item))
	}
	return out, unread, nil
}

func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.Repo.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	err := s.Repo.MarkRead(ctx, id, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotificationNotFound
	}
	return err
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	return s.Repo.MarkAllRead(ctx, userID)
}

func (s *NotificationService) Create(
	ctx context.Context,
	userID, notifType string,
	refType *string,
	refID *uuid.UUID,
	meta map[string]any,
) error {
	raw := "{}"
	if meta != nil {
		b, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		raw = string(b)
	}
	_, err := s.Repo.Create(ctx, userID, notifType, refType, refID, raw)
	return err
}
