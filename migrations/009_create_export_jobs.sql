CREATE TABLE IF NOT EXISTS export_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL DEFAULT 'users_csv',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    filters_json TEXT NOT NULL DEFAULT '{}',
    file_name VARCHAR(255),
    file_path TEXT,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS export_jobs_user_id_idx ON export_jobs (user_id);
CREATE INDEX IF NOT EXISTS export_jobs_status_idx ON export_jobs (status);
