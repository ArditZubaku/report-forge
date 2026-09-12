package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type ReportStore struct {
	db *sqlx.DB
}

func NewReportStore(db *sql.DB) *ReportStore {
	return &ReportStore{
		db: sqlx.NewDb(db, "postgres"),
	}
}

type Report struct {
	UserId               uuid.UUID  `db:"user_id"`
	Id                   uuid.UUID  `db:"id"`
	ReportType           string     `db:"report_type"`
	OutputFilePath       *string    `db:"output_file_path"`
	DownloadUrl          *string    `db:"download_url"`
	DownloadUrlExpiresAt *time.Time `db:"download_url_expires_at"`
	ErrorMessage         *string    `db:"error_message"`
	CreatedAt            time.Time  `db:"created_at"`
	StartedAt            *time.Time `db:"started_at"`
	CompletedAt          *time.Time `db:"completed_at"`
	FailedAt             *time.Time `db:"failed_at"`
}

func (s *ReportStore) Create(ctx context.Context, userId uuid.UUID, reportType string) (*Report, error) {
	const insertQuery = `INSERT INTO reports(user_id, report_type) VALUES ($1, $2) RETURNING *;`

	var report Report
	if err := s.db.GetContext(ctx, &report, insertQuery, userId, reportType); err != nil {
		return nil, fmt.Errorf("failed to insert report for user %s: %w", userId, err)
	}

	return &report, nil
}

func (s *ReportStore) Update(ctx context.Context, report *Report) (*Report, error) {
	const updateQuery = `UPDATE reports
							SET report_type = $1,
								output_file_path = $2,
    							download_url = $3,
    							download_url_expires_at = $4,
    							error_message = $5,
    							started_at = $6,
								completed_at = $7,
								failed_at = $8
    						WHERE user_id = $9 AND id = $10
    						RETURNING *;`

	var result Report
	if err := s.db.GetContext(ctx, &result, updateQuery,
		report.ReportType,
		report.OutputFilePath,
		report.DownloadUrl,
		report.DownloadUrlExpiresAt,
		report.ErrorMessage,
		report.StartedAt,
		report.CompletedAt,
		report.FailedAt,
		report.UserId,
		report.Id,
	); err != nil {
		return nil, fmt.Errorf("failed to update report for user %s: %w", report.UserId, err)
	}

	return &result, nil
}

func (s *ReportStore) ByPrimaryKey(ctx context.Context, userId uuid.UUID, id uuid.UUID) (*Report, error) {
	const selectQuery = `SELECT * FROM reports WHERE user_id = $1 AND id = $2`

	var report Report
	if err := s.db.GetContext(ctx, &report, selectQuery, userId, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query report %s for user %s: %w", id, userId, err)
	}

	return &report, nil
}

func (s *ReportStore) Delete(ctx context.Context, id uuid.UUID) error {
	const deleteQuery = `DELETE FROM reports WHERE id = $1;`
	if _, err := s.db.ExecContext(ctx, deleteQuery, id); err != nil {
		return fmt.Errorf("failed to delete report %s: %w", id, err)
	}
	return nil
}
