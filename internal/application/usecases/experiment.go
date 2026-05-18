package usecases

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
	"github.com/physicist2018/licelfile/licelformat"
)

type ExperimentUseCase struct {
	repo    ports.ExperimentRepository
	storage ports.FileStorage
}

func NewExperimentUseCase(repo ports.ExperimentRepository, storage ports.FileStorage) *ExperimentUseCase {
	return &ExperimentUseCase{repo: repo, storage: storage}
}

type CreateExperimentInput struct {
	UserID           string
	Title            string
	Comments         string
	ZipFile          io.Reader
	ZipSize          int64
	BgrFile          io.Reader
	BgrContentType   string
	MeteoFile        io.Reader
	MeteoContentType string
}

func (uc *ExperimentUseCase) Create(ctx context.Context, input CreateExperimentInput) (string, error) {
	experimentID := uuid.New().String()

	// 1. Save zip to a temp file and extract licel metadata
	startTime, stopTime, zipData, err := extractLicelTimes(input.ZipFile, input.ZipSize)
	if err != nil {
		return "", fmt.Errorf("extract licel times: %w", err)
	}

	// 2. Upload zip to MinIO
	zipPath := fmt.Sprintf("experiments/%s/data.zip", experimentID)
	if err := uc.storage.Upload(ctx, zipPath, bytes.NewReader(zipData), int64(len(zipData)), "application/zip"); err != nil {
		return "", fmt.Errorf("upload zip: %w", err)
	}

	// 3. Upload BgrFile if present
	var bgrPath string
	if input.BgrFile != nil {
		contentType := input.BgrContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		bgrPath = fmt.Sprintf("experiments/%s/bgr.licel", experimentID)
		if err := uc.storage.Upload(ctx, bgrPath, input.BgrFile, -1, contentType); err != nil {
			return "", fmt.Errorf("upload bgr file: %w", err)
		}
	}

	// 4. Upload MeteoFile
	var meteoPath string
	if input.MeteoFile != nil {
		contentType := input.MeteoContentType
		if contentType == "" {
			contentType = "text/plain"
		}
		meteoPath = fmt.Sprintf("experiments/%s/meteo.txt", experimentID)
		if err := uc.storage.Upload(ctx, meteoPath, input.MeteoFile, -1, contentType); err != nil {
			return "", fmt.Errorf("upload meteo file: %w", err)
		}
	}

	// 5. Persist experiment
	exp := &domain.Experiment{
		ID:               experimentID,
		UserID:           input.UserID,
		Title:            input.Title,
		Comments:         input.Comments,
		StartDateTime:    startTime,
		StopDateTime:     stopTime,
		BgrFilePath:      bgrPath,
		ZipFilePath:      zipPath,
		MeteoProfilePath: meteoPath,
	}

	if err := uc.repo.Create(ctx, exp); err != nil {
		return "", fmt.Errorf("create experiment: %w", err)
	}

	return experimentID, nil
}

func extractLicelTimes(r io.Reader, size int64) (start, stop time.Time, zipData []byte, err error) {
	zipData, err = io.ReadAll(r)
	if err != nil {
		return time.Time{}, time.Time{}, nil, fmt.Errorf("read zip: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "licel-*.zip")
	if err != nil {
		return time.Time{}, time.Time{}, nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(zipData); err != nil {
		tmpFile.Close()
		return time.Time{}, time.Time{}, nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	pack := licelformat.NewLicelPackFromZip(tmpFile.Name())
	if pack == nil || len(pack.Data) == 0 {
		return time.Time{}, time.Time{}, nil, fmt.Errorf("no licel files found in zip")
	}

	start = pack.StartTime

	// Find max MeasurementStopTime across all files in the pack
	for _, lf := range pack.Data {
		if lf.MeasurementStopTime.After(stop) {
			stop = lf.MeasurementStopTime
		}
	}

	return start, stop, zipData, nil
}
