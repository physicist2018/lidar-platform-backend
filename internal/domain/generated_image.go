package domain

import "time"

type GeneratedImage struct {
	ID           string
	ExperimentID string
	FileName     string
	ObjectPath   string
	Wavelength   float64
	Polarization string
	ChannelType  string
	PlotType     string
	CreatedAt    time.Time
}
