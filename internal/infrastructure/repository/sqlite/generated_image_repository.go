package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
)

type GeneratedImageRepository struct {
	db *sql.DB
}

func NewGeneratedImageRepository(db *sql.DB) *GeneratedImageRepository {
	return &GeneratedImageRepository{db: db}
}

func (r *GeneratedImageRepository) Create(ctx context.Context, img *domain.GeneratedImage) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO generated_images (id, experiment_id, file_name, object_path, wavelength, polarization, channel_type, plot_type, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		img.ID, img.ExperimentID, img.FileName, img.ObjectPath,
		img.Wavelength, img.Polarization, img.ChannelType, img.PlotType,
		img.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *GeneratedImageRepository) FindByParams(ctx context.Context, params ports.GeneratedImageParams) (*domain.GeneratedImage, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, experiment_id, file_name, object_path, wavelength, polarization, channel_type, plot_type, created_at
		 FROM generated_images
		 WHERE experiment_id = ? AND wavelength = ? AND polarization = ? AND channel_type = ? AND plot_type = ?`,
		params.ExperimentID, params.Wavelength, params.Polarization, params.ChannelType, params.PlotType,
	)
	return scanGeneratedImage(row)
}

func scanGeneratedImage(row *sql.Row) (*domain.GeneratedImage, error) {
	var img domain.GeneratedImage
	var createdAt string
	err := row.Scan(
		&img.ID, &img.ExperimentID, &img.FileName, &img.ObjectPath,
		&img.Wavelength, &img.Polarization, &img.ChannelType, &img.PlotType,
		&createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	img.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &img, nil
}
