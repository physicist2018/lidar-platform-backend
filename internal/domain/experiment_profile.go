package domain

import "time"

type ExperimentProfile struct {
	ID           string
	ExperimentID string
	FileName     string

	// Profile metadata (from LicelProfile)
	Active               bool
	Photon               bool
	LaserType            int
	NDataPoints          int // original number of data points (before trim)
	HighVoltage          int
	BinWidth             float64
	Wavelength           float64
	Polarization         string
	BinShift             int
	DecBinShift          int
	AdcBits              int
	NShots               int
	DiscrLevel           float64
	DeviceID             string
	NCrate               int
	MeasurementStartTime time.Time
	MeasurementStopTime  time.Time

	// Processed data
	Altitudes []float64
	Data      []float64
	Hmin      float64
	Hmax      float64
	BgrType   string
}
