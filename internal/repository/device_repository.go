package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type DeviceRepository struct {
	DB *pgxpool.Pool
}

func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{DB: db}
}

type DeviceUpsertInput struct {
	UserID    string
	DeviceKey string
	Name      string
	UserAgent string
	IPAddress string
}

func (r *DeviceRepository) Upsert(
	ctx context.Context,
	input DeviceUpsertInput,
) (*models.Device, error) {
	query := `
		INSERT INTO devices (
			user_id,
			device_key,
			name,
			user_agent,
			ip_address,
			last_seen_at
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NOW())
		ON CONFLICT (user_id, device_key)
		DO UPDATE SET
			name = COALESCE(EXCLUDED.name, devices.name),
			user_agent = COALESCE(EXCLUDED.user_agent, devices.user_agent),
			ip_address = COALESCE(EXCLUDED.ip_address, devices.ip_address),
			last_seen_at = NOW()
		RETURNING
			id::text,
			user_id::text,
			device_key,
			COALESCE(name, ''),
			COALESCE(user_agent, ''),
			COALESCE(ip_address, ''),
			last_seen_at,
			created_at
	`

	device := &models.Device{}
	err := r.DB.QueryRow(
		ctx,
		query,
		input.UserID,
		input.DeviceKey,
		input.Name,
		input.UserAgent,
		input.IPAddress,
	).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceKey,
		&device.Name,
		&device.UserAgent,
		&device.IPAddress,
		&device.LastSeenAt,
		&device.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (r *DeviceRepository) Touch(
	ctx context.Context,
	deviceID string,
	userAgent string,
	ipAddress string,
) error {
	query := `
		UPDATE devices
		SET
			user_agent = COALESCE(NULLIF($2, ''), user_agent),
			ip_address = COALESCE(NULLIF($3, ''), ip_address),
			last_seen_at = NOW()
		WHERE id = $1
	`

	_, err := r.DB.Exec(ctx, query, deviceID, userAgent, ipAddress)
	return err
}

func (r *DeviceRepository) FindByUserAndID(
	ctx context.Context,
	userID string,
	deviceID string,
) (*models.Device, error) {
	query := `
		SELECT
			id::text,
			user_id::text,
			device_key,
			COALESCE(name, ''),
			COALESCE(user_agent, ''),
			COALESCE(ip_address, ''),
			last_seen_at,
			created_at
		FROM devices
		WHERE user_id = $1 AND id = $2
	`

	device := &models.Device{}
	err := r.DB.QueryRow(ctx, query, userID, deviceID).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceKey,
		&device.Name,
		&device.UserAgent,
		&device.IPAddress,
		&device.LastSeenAt,
		&device.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (r *DeviceRepository) DeleteByUserAndID(
	ctx context.Context,
	userID string,
	deviceID string,
) error {
	tag, err := r.DB.Exec(
		ctx,
		`DELETE FROM devices WHERE user_id = $1 AND id = $2`,
		userID,
		deviceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *DeviceRepository) ListByUserID(
	ctx context.Context,
	userID string,
) ([]models.Device, error) {
	query := `
		SELECT
			id::text,
			user_id::text,
			device_key,
			COALESCE(name, ''),
			COALESCE(user_agent, ''),
			COALESCE(ip_address, ''),
			last_seen_at,
			created_at
		FROM devices
		WHERE user_id = $1
		ORDER BY last_seen_at DESC
	`

	rows, err := r.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.DeviceKey,
			&device.Name,
			&device.UserAgent,
			&device.IPAddress,
			&device.LastSeenAt,
			&device.CreatedAt,
		); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}

	return devices, rows.Err()
}
