package domain

import "time"

type ExperimentStatus string

const (
	StatusStarted   ExperimentStatus = "started"
	StatusUploading ExperimentStatus = "uploading"
	StatusSuccess   ExperimentStatus = "success"
	StatusFailed    ExperimentStatus = "failed"
)

type Experiment struct {
	ID               string
	UserID           string
	Title            string
	Comments         string
	StartDateTime    time.Time
	StopDateTime     time.Time
	BgrFilePath      string
	ZipFilePath      string
	MeteoProfilePath string
	Status           ExperimentStatus
	ErrorMessage     string
}
