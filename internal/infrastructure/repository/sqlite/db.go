package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const initSQL = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS experiments (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    comments TEXT NOT NULL DEFAULT '',
    start_date_time TEXT NOT NULL DEFAULT '',
    stop_date_time TEXT NOT NULL DEFAULT '',
    bgr_file_path TEXT NOT NULL DEFAULT '',
    zip_file_path TEXT NOT NULL DEFAULT '',
    meteo_profile_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'started',
    error_message TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_experiments_user_id ON experiments(user_id);

CREATE TABLE IF NOT EXISTS experiment_profiles (
    id TEXT PRIMARY KEY,
    experiment_id TEXT NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    photon INTEGER NOT NULL DEFAULT 0,
    laser_type INTEGER NOT NULL DEFAULT 0,
    n_data_points INTEGER NOT NULL DEFAULT 0,
    high_voltage INTEGER NOT NULL DEFAULT 0,
    bin_width REAL NOT NULL DEFAULT 0,
    wavelength REAL NOT NULL DEFAULT 0,
    polarization TEXT NOT NULL DEFAULT '',
    bin_shift INTEGER NOT NULL DEFAULT 0,
    dec_bin_shift INTEGER NOT NULL DEFAULT 0,
    adc_bits INTEGER NOT NULL DEFAULT 0,
    n_shots INTEGER NOT NULL DEFAULT 0,
    discr_level REAL NOT NULL DEFAULT 0,
    device_id TEXT NOT NULL DEFAULT '',
    n_crate INTEGER NOT NULL DEFAULT 0,
    measurement_start_time TEXT NOT NULL DEFAULT '',
    measurement_stop_time TEXT NOT NULL DEFAULT '',
    altitudes TEXT NOT NULL DEFAULT '[]',
    data TEXT NOT NULL DEFAULT '[]',
    hmin REAL NOT NULL DEFAULT 0,
    hmax REAL NOT NULL DEFAULT 0,
    bgr_type TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_experiment_profiles_experiment_id ON experiment_profiles(experiment_id);

CREATE TABLE IF NOT EXISTS generated_images (
    id TEXT PRIMARY KEY,
    experiment_id TEXT NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    object_path TEXT NOT NULL,
    json_data TEXT NOT NULL DEFAULT '',
    wavelength REAL NOT NULL,
    polarization TEXT NOT NULL,
    channel_type TEXT NOT NULL,
    plot_type TEXT NOT NULL,
    image_type TEXT NOT NULL DEFAULT 'heatmap',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_generated_images_experiment_id ON generated_images(experiment_id);
`

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(initSQL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}
