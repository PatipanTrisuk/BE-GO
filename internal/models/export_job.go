package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ExportJobPending    = "pending"
	ExportJobProcessing = "processing"
	ExportJobCompleted  = "completed"
	ExportJobFailed     = "failed"

	ExportTypeUsersCSV = "users_csv"
)

type ExportJob struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	FiltersJSON  string     `json:"-"`
	FileName     *string    `json:"file_name,omitempty"`
	FilePath     *string    `json:"-"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}
