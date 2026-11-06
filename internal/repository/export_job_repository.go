package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type ExportJobRepository struct {
	DB *pgxpool.Pool
}

func NewExportJobRepository(db *pgxpool.Pool) *ExportJobRepository {
	return &ExportJobRepository{DB: db}
}

func scanExportJob(
	scanner interface {
		Scan(dest ...any) error
	},
	job *models.ExportJob,
) error {
	var fileName sql.NullString
	var filePath sql.NullString
	var errorMessage sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	if err := scanner.Scan(
		&job.ID,
		&job.UserID,
		&job.Type,
		&job.Status,
		&job.FiltersJSON,
		&fileName,
		&filePath,
		&errorMessage,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
	); err != nil {
		return err
	}

	if fileName.Valid {
		v := fileName.String
		job.FileName = &v
	}
	if filePath.Valid {
		v := filePath.String
		job.FilePath = &v
	}
	if errorMessage.Valid {
		v := errorMessage.String
		job.ErrorMessage = &v
	}
	if startedAt.Valid {
		t := startedAt.Time
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		job.CompletedAt = &t
	}
	return nil
}

const exportJobSelect = `
	id, user_id, type, status, filters_json, file_name, file_path, error_message, created_at, started_at, completed_at
`

func (r *ExportJobRepository) Create(ctx context.Context, userID, jobType, filtersJSON string) (*models.ExportJob, error) {
	job := &models.ExportJob{}
	err := scanExportJob(r.DB.QueryRow(ctx, `
		INSERT INTO export_jobs (user_id, type, status, filters_json)
		VALUES ($1, $2, $3, $4)
		RETURNING `+exportJobSelect, userID, jobType, models.ExportJobPending, filtersJSON), job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *ExportJobRepository) FindByID(ctx context.Context, id string) (*models.ExportJob, error) {
	job := &models.ExportJob{}
	err := scanExportJob(r.DB.QueryRow(ctx, `
		SELECT `+exportJobSelect+`
		FROM export_jobs
		WHERE id = $1
	`, id), job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *ExportJobRepository) MarkProcessing(ctx context.Context, id string) error {
	tag, err := r.DB.Exec(ctx, `
		UPDATE export_jobs
		SET status = $2, started_at = $3
		WHERE id = $1 AND status = $4
	`, id, models.ExportJobProcessing, time.Now(), models.ExportJobPending)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ExportJobRepository) MarkCompleted(ctx context.Context, id, fileName, filePath string) error {
	tag, err := r.DB.Exec(ctx, `
		UPDATE export_jobs
		SET status = $2, file_name = $3, file_path = $4, completed_at = $5, error_message = NULL
		WHERE id = $1
	`, id, models.ExportJobCompleted, fileName, filePath, time.Now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ExportJobRepository) MarkFailed(ctx context.Context, id, message string) error {
	tag, err := r.DB.Exec(ctx, `
		UPDATE export_jobs
		SET status = $2, error_message = $3, completed_at = $4
		WHERE id = $1
	`, id, models.ExportJobFailed, message, time.Now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
