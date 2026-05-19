package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lidar-platform/backend/internal/domain"
)

type ExperimentRepository struct {
	db *sql.DB
}

func NewExperimentRepository(db *sql.DB) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

func (r *ExperimentRepository) Create(ctx context.Context, exp *domain.Experiment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO experiments (id, user_id, title, comments, status)
		 VALUES (?, ?, ?, ?, ?)`,
		exp.ID, exp.UserID, exp.Title, exp.Comments, string(exp.Status),
	)
	return err
}

func (r *ExperimentRepository) FindByID(ctx context.Context, id string) (*domain.Experiment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, comments,
		        start_date_time, stop_date_time,
		        bgr_file_path, zip_file_path, meteo_profile_path,
		        status, error_message
		 FROM experiments WHERE id = ?`, id,
	)
	return scanExperiment(row)
}

func (r *ExperimentRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Experiment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, comments,
		        start_date_time, stop_date_time,
		        bgr_file_path, zip_file_path, meteo_profile_path,
		        status, error_message
		 FROM experiments WHERE user_id = ? ORDER BY start_date_time DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Experiment
	for rows.Next() {
		exp, err := scanExperimentRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, exp)
	}
	return result, rows.Err()
}

func (r *ExperimentRepository) Update(ctx context.Context, exp *domain.Experiment) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE experiments SET
			title = ?, comments = ?,
			start_date_time = ?, stop_date_time = ?,
			bgr_file_path = ?, zip_file_path = ?, meteo_profile_path = ?,
			status = ?, error_message = ?
		 WHERE id = ?`,
		exp.Title, exp.Comments,
		exp.StartDateTime.Format(time.RFC3339), exp.StopDateTime.Format(time.RFC3339),
		exp.BgrFilePath, exp.ZipFilePath, exp.MeteoProfilePath,
		string(exp.Status), exp.ErrorMessage,
		exp.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil
	}
	return nil
}

func scanExperiment(row *sql.Row) (*domain.Experiment, error) {
	var e domain.Experiment
	var startDateTime, stopDateTime string
	err := row.Scan(&e.ID, &e.UserID, &e.Title, &e.Comments,
		&startDateTime, &stopDateTime,
		&e.BgrFilePath, &e.ZipFilePath, &e.MeteoProfilePath,
		&e.Status, &e.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.StartDateTime, _ = time.Parse(time.RFC3339, startDateTime)
	e.StopDateTime, _ = time.Parse(time.RFC3339, stopDateTime)
	return &e, nil
}

func scanExperimentRows(rows *sql.Rows) (*domain.Experiment, error) {
	var e domain.Experiment
	var startDateTime, stopDateTime string
	err := rows.Scan(&e.ID, &e.UserID, &e.Title, &e.Comments,
		&startDateTime, &stopDateTime,
		&e.BgrFilePath, &e.ZipFilePath, &e.MeteoProfilePath,
		&e.Status, &e.ErrorMessage,
	)
	if err != nil {
		return nil, err
	}
	e.StartDateTime, _ = time.Parse(time.RFC3339, startDateTime)
	e.StopDateTime, _ = time.Parse(time.RFC3339, stopDateTime)
	return &e, nil
}
