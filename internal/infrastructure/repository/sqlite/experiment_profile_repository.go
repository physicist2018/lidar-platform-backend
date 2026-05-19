package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lidar-platform/backend/internal/domain"
)

type ExperimentProfileRepository struct {
	db *sql.DB
}

func NewExperimentProfileRepository(db *sql.DB) *ExperimentProfileRepository {
	return &ExperimentProfileRepository{db: db}
}

func (r *ExperimentProfileRepository) Create(ctx context.Context, p *domain.ExperimentProfile) error {
	altitudesJSON, _ := json.Marshal(p.Altitudes)
	dataJSON, _ := json.Marshal(p.Data)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO experiment_profiles (
			id, experiment_id, file_name,
			active, photon, laser_type, n_data_points, high_voltage,
			bin_width, wavelength, polarization, bin_shift, dec_bin_shift,
			adc_bits, n_shots, discr_level, device_id, n_crate,
			altitudes, data, hmin, hmax, bgr_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.ExperimentID, p.FileName,
		boolToInt(p.Active), boolToInt(p.Photon), p.LaserType, p.NDataPoints, p.HighVoltage,
		p.BinWidth, p.Wavelength, p.Polarization, p.BinShift, p.DecBinShift,
		p.AdcBits, p.NShots, p.DiscrLevel, p.DeviceID, p.NCrate,
		string(altitudesJSON), string(dataJSON), p.Hmin, p.Hmax, p.BgrType,
	)
	return err
}

func (r *ExperimentProfileRepository) FindByExperimentID(ctx context.Context, experimentID string) ([]*domain.ExperimentProfile, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, experiment_id, file_name,
		        active, photon, laser_type, n_data_points, high_voltage,
		        bin_width, wavelength, polarization, bin_shift, dec_bin_shift,
		        adc_bits, n_shots, discr_level, device_id, n_crate,
		        altitudes, data, hmin, hmax, bgr_type
		 FROM experiment_profiles WHERE experiment_id = ?`, experimentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.ExperimentProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *ExperimentProfileRepository) DeleteByExperimentID(ctx context.Context, experimentID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM experiment_profiles WHERE experiment_id = ?`, experimentID,
	)
	return err
}

func scanProfile(rows *sql.Rows) (*domain.ExperimentProfile, error) {
	var p domain.ExperimentProfile
	var active, photon int
	var altitudesJSON, dataJSON string

	err := rows.Scan(&p.ID, &p.ExperimentID, &p.FileName,
		&active, &photon, &p.LaserType, &p.NDataPoints, &p.HighVoltage,
		&p.BinWidth, &p.Wavelength, &p.Polarization, &p.BinShift, &p.DecBinShift,
		&p.AdcBits, &p.NShots, &p.DiscrLevel, &p.DeviceID, &p.NCrate,
		&altitudesJSON, &dataJSON, &p.Hmin, &p.Hmax, &p.BgrType,
	)
	if err != nil {
		return nil, err
	}

	p.Active = active == 1
	p.Photon = photon == 1

	if altitudesJSON != "" {
		json.Unmarshal([]byte(altitudesJSON), &p.Altitudes)
	}
	if dataJSON != "" {
		json.Unmarshal([]byte(dataJSON), &p.Data)
	}
	if p.Altitudes == nil {
		p.Altitudes = []float64{}
	}
	if p.Data == nil {
		p.Data = []float64{}
	}

	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
