package domain

import "time"

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
}
