package usecases

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lidar-platform/backend/internal/application/ports"
	"github.com/lidar-platform/backend/internal/domain"
	"github.com/physicist2018/licelfile/licelformat"
)

var ErrExperimentNotFound = errors.New("experiment not found")

type ExperimentUseCase struct {
	repo        ports.ExperimentRepository
	profileRepo ports.ExperimentProfileRepository
	storage     ports.FileStorage
}

func NewExperimentUseCase(repo ports.ExperimentRepository, profileRepo ports.ExperimentProfileRepository, storage ports.FileStorage) *ExperimentUseCase {
	return &ExperimentUseCase{repo: repo, profileRepo: profileRepo, storage: storage}
}

type CreateExperimentInput struct {
	UserID           string
	Title            string
	Comments         string
	ZipFile          io.Reader
	BgrFile          io.Reader
	BgrContentType   string
	MeteoFile        io.Reader
	MeteoContentType string
}

// asyncUploadInput holds all file data read upfront, so the goroutine
// can work independently of the HTTP request lifecycle.
type asyncUploadInput struct {
	UserID           string
	Title            string
	Comments         string
	ZipData          []byte
	BgrData          []byte
	BgrContentType   string
	MeteoData        []byte
	MeteoContentType string
}

func (uc *ExperimentUseCase) Create(ctx context.Context, input CreateExperimentInput) (string, error) {
	experimentID := uuid.New().String()

	zipData, err := io.ReadAll(input.ZipFile)
	if err != nil {
		return "", fmt.Errorf("read zip: %w", err)
	}

	var bgrData []byte
	if input.BgrFile != nil {
		bgrData, err = io.ReadAll(input.BgrFile)
		if err != nil {
			return "", fmt.Errorf("read bgr file: %w", err)
		}
	}

	var meteoData []byte
	if input.MeteoFile != nil {
		meteoData, err = io.ReadAll(input.MeteoFile)
		if err != nil {
			return "", fmt.Errorf("read meteo file: %w", err)
		}
	}

	exp := &domain.Experiment{
		ID:       experimentID,
		UserID:   input.UserID,
		Title:    input.Title,
		Comments: input.Comments,
		Status:   domain.StatusStarted,
	}

	if err := uc.repo.Create(ctx, exp); err != nil {
		return "", fmt.Errorf("create experiment: %w", err)
	}

	asyncInput := asyncUploadInput{
		UserID:           input.UserID,
		Title:            input.Title,
		Comments:         input.Comments,
		ZipData:          zipData,
		BgrData:          bgrData,
		BgrContentType:   input.BgrContentType,
		MeteoData:        meteoData,
		MeteoContentType: input.MeteoContentType,
	}

	go uc.processUpload(experimentID, asyncInput)

	return experimentID, nil
}

func (uc *ExperimentUseCase) processUpload(experimentID string, input asyncUploadInput) {
	ctx := context.Background()

	uc.setStatus(ctx, experimentID, domain.StatusUploading, "")

	// extract licel metadata
	startTime, stopTime, err := extractLicelTimesFromBytes(input.ZipData)
	if err != nil {
		uc.setStatus(ctx, experimentID, domain.StatusFailed, fmt.Sprintf("extract licel times: %v", err))
		return
	}

	// upload zip
	zipPath := fmt.Sprintf("experiments/%s/data.zip", experimentID)
	if err := uc.storage.Upload(ctx, zipPath, bytes.NewReader(input.ZipData), int64(len(input.ZipData)), "application/zip"); err != nil {
		uc.setStatus(ctx, experimentID, domain.StatusFailed, fmt.Sprintf("upload zip: %v", err))
		return
	}

	// upload bgr
	var bgrPath string
	if len(input.BgrData) > 0 {
		contentType := input.BgrContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		bgrPath = fmt.Sprintf("experiments/%s/bgr.licel", experimentID)
		if err := uc.storage.Upload(ctx, bgrPath, bytes.NewReader(input.BgrData), int64(len(input.BgrData)), contentType); err != nil {
			uc.setStatus(ctx, experimentID, domain.StatusFailed, fmt.Sprintf("upload bgr file: %v", err))
			return
		}
	}

	// upload meteo
	var meteoPath string
	if len(input.MeteoData) > 0 {
		contentType := input.MeteoContentType
		if contentType == "" {
			contentType = "text/plain"
		}
		meteoPath = fmt.Sprintf("experiments/%s/meteo.txt", experimentID)
		if err := uc.storage.Upload(ctx, meteoPath, bytes.NewReader(input.MeteoData), int64(len(input.MeteoData)), contentType); err != nil {
			uc.setStatus(ctx, experimentID, domain.StatusFailed, fmt.Sprintf("upload meteo file: %v", err))
			return
		}
	}

	// persist final state
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
		Status:           domain.StatusSuccess,
	}

	if err := uc.repo.Update(ctx, exp); err != nil {
		log.Printf("failed to update experiment %s to success: %v", experimentID, err)
	}
}

func (uc *ExperimentUseCase) setStatus(ctx context.Context, experimentID string, status domain.ExperimentStatus, errMsg string) {
	exp, err := uc.repo.FindByID(ctx, experimentID)
	if err != nil || exp == nil {
		log.Printf("failed to find experiment %s for status update: %v", experimentID, err)
		return
	}
	exp.Status = status
	exp.ErrorMessage = errMsg
	if err := uc.repo.Update(ctx, exp); err != nil {
		log.Printf("failed to update experiment %s status to %s: %v", experimentID, status, err)
	}
}

func (uc *ExperimentUseCase) GetByID(ctx context.Context, experimentID string) (*domain.Experiment, error) {
	exp, err := uc.repo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("find experiment: %w", err)
	}
	if exp == nil {
		return nil, ErrExperimentNotFound
	}
	return exp, nil
}

func (uc *ExperimentUseCase) ListByUser(ctx context.Context, userID string) ([]*domain.Experiment, error) {
	exps, err := uc.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	if exps == nil {
		exps = []*domain.Experiment{}
	}
	return exps, nil
}

func (uc *ExperimentUseCase) GetStatus(ctx context.Context, experimentID string) (*domain.Experiment, error) {
	exp, err := uc.repo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("find experiment: %w", err)
	}
	if exp == nil {
		return nil, ErrExperimentNotFound
	}
	return exp, nil
}

type PrepareExperimentInput struct {
	Hmin    float64
	Hmax    float64
	BgrType string  // "file" or "avgtail"
	BgrAlt  float64 // used only when BgrType=avgtail
}

type PrepareExperimentResult struct {
	ProfileCount int
}

func (uc *ExperimentUseCase) Prepare(ctx context.Context, experimentID string, input PrepareExperimentInput) (*PrepareExperimentResult, error) {
	exp, err := uc.repo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("find experiment: %w", err)
	}
	if exp == nil {
		return nil, ErrExperimentNotFound
	}

	// Download zip from MinIO to temp file
	tmpDir, err := os.MkdirTemp("", "lidar-prepare-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := exp.ZipFilePath
	if zipPath == "" {
		zipPath = fmt.Sprintf("experiments/%s/data.zip", experimentID)
	}

	localZipPath := tmpDir + "/data.zip"
	if err := uc.downloadToFile(ctx, zipPath, localZipPath); err != nil {
		return nil, fmt.Errorf("download zip: %w", err)
	}

	// Load zip as LicelPack
	pack := licelformat.NewLicelPackFromZip(localZipPath)
	if pack == nil || len(pack.Data) == 0 {
		return nil, fmt.Errorf("no licel files found in zip")
	}

	// Load bgr file if BgrType=file
	var bgrLicelFile *licelformat.LicelFile
	if input.BgrType == "file" {
		bgrPath := exp.BgrFilePath
		if bgrPath == "" {
			bgrPath = fmt.Sprintf("experiments/%s/bgr.licel", experimentID)
		}
		localBgrPath := tmpDir + "/bgr.licel"
		if err := uc.downloadToFile(ctx, bgrPath, localBgrPath); err != nil {
			return nil, fmt.Errorf("download bgr file: %w", err)
		}
		lf := licelformat.LoadLicelFile(localBgrPath)
		bgrLicelFile = &lf
	}

	// Delete existing profiles for this experiment before creating new ones
	_ = uc.profileRepo.DeleteByExperimentID(ctx, experimentID)

	profileCount := 0
	for fileName, lf := range pack.Data {
		for _, prof := range lf.Profiles {
			altitudes := make([]float64, prof.NDataPoints)
			for i := 0; i < prof.NDataPoints; i++ {
				altitudes[i] = float64(i+prof.BinShift) * prof.BinWidth
			}

			// Compute background subtraction
			correctedData := make([]float64, prof.NDataPoints)
			copy(correctedData, prof.Data)

			switch input.BgrType {
			case "file":
				if bgrLicelFile != nil {
					bgrProf := bgrLicelFile.SelectCertainWavelength(prof.Photon, prof.Wavelength)
					if bgrProf.Wavelength != 0 && len(bgrProf.Data) == len(prof.Data) {
						for j := range correctedData {
							correctedData[j] -= bgrProf.Data[j]
						}
					}
				}
			case "avgtail":
				bg := computeTailAverage(altitudes, prof.Data, input.BgrAlt)
				for j := range correctedData {
					correctedData[j] -= bg
				}
			}

			// Trim to [Hmin, Hmax]
			trimmedAltitudes := make([]float64, 0)
			trimmedData := make([]float64, 0)
			for j := 0; j < prof.NDataPoints; j++ {
				if altitudes[j] >= input.Hmin && altitudes[j] <= input.Hmax {
					trimmedAltitudes = append(trimmedAltitudes, altitudes[j])
					trimmedData = append(trimmedData, correctedData[j])
				}
			}

			ep := &domain.ExperimentProfile{
				ID:                   uuid.New().String(),
				ExperimentID:         experimentID,
				FileName:             fileName,
				Active:               prof.Active,
				Photon:               prof.Photon,
				LaserType:            prof.LaserType,
				NDataPoints:          prof.NDataPoints,
				HighVoltage:          prof.HighVoltage,
				BinWidth:             prof.BinWidth,
				Wavelength:           prof.Wavelength,
				Polarization:         prof.Polarization,
				BinShift:             prof.BinShift,
				DecBinShift:          prof.DecBinShift,
				AdcBits:              prof.AdcBits,
				NShots:               prof.NShots,
				DiscrLevel:           prof.DiscrLevel,
				DeviceID:             prof.DeviceID,
				NCrate:               prof.NCrate,
				MeasurementStartTime: lf.MeasurementStartTime,
				MeasurementStopTime:  lf.MeasurementStopTime,
				Altitudes:            trimmedAltitudes,
				Data:                 trimmedData,
				Hmin:                 input.Hmin,
				Hmax:                 input.Hmax,
				BgrType:              input.BgrType,
			}

			if err := uc.profileRepo.Create(ctx, ep); err != nil {
				return nil, fmt.Errorf("create profile: %w", err)
			}
			profileCount++
		}
	}

	// Update experiment status
	exp.Status = domain.StatusSuccess
	if err := uc.repo.Update(ctx, exp); err != nil {
		log.Printf("failed to update experiment %s status after prepare: %v", experimentID, err)
	}

	return &PrepareExperimentResult{ProfileCount: profileCount}, nil
}

func (uc *ExperimentUseCase) downloadToFile(ctx context.Context, objectPath, localPath string) error {
	rc, _, err := uc.storage.Download(ctx, objectPath)
	if err != nil {
		return fmt.Errorf("minio download: %w", err)
	}
	defer rc.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return fmt.Errorf("write local file: %w", err)
	}
	return nil
}

func computeTailAverage(altitudes, data []float64, bgrAlt float64) float64 {
	var sum float64
	var count int
	for i, alt := range altitudes {
		if alt >= bgrAlt {
			sum += data[i]
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func extractLicelTimesFromBytes(zipData []byte) (start, stop time.Time, err error) {
	tmpFile, err := os.CreateTemp("", "licel-*.zip")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(zipData); err != nil {
		tmpFile.Close()
		return time.Time{}, time.Time{}, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	pack := licelformat.NewLicelPackFromZip(tmpFile.Name())
	if pack == nil || len(pack.Data) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no licel files found in zip")
	}

	start = pack.StartTime

	for _, lf := range pack.Data {
		if lf.MeasurementStopTime.After(stop) {
			stop = lf.MeasurementStopTime
		}
	}

	return start, stop, nil
}
