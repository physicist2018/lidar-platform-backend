package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lidar-platform/backend/internal/application/usecases"
	"github.com/lidar-platform/backend/internal/interfaces/http/middleware"
)

type ExperimentHandler struct {
	uc *usecases.ExperimentUseCase
}

func NewExperimentHandler(uc *usecases.ExperimentUseCase) *ExperimentHandler {
	return &ExperimentHandler{uc: uc}
}

func (h *ExperimentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	comments := r.FormValue("comments")
	if title == "" || comments == "" {
		http.Error(w, "title and comments are required", http.StatusBadRequest)
		return
	}

	zipFile, _, err := r.FormFile("ZipFile")
	if err != nil {
		http.Error(w, "ZipFile is required", http.StatusBadRequest)
		return
	}
	defer zipFile.Close()

	bgrFile, bgrHeader, _ := r.FormFile("BgrFile")
	if bgrFile != nil {
		defer bgrFile.Close()
	}

	meteoFile, meteoHeader, _ := r.FormFile("MeteoFile")
	if meteoFile != nil {
		defer meteoFile.Close()
	}

	input := usecases.CreateExperimentInput{
		UserID:   userID,
		Title:    title,
		Comments: comments,
		ZipFile:  zipFile,
	}

	if bgrFile != nil {
		input.BgrFile = bgrFile
		input.BgrContentType = bgrHeader.Header.Get("Content-Type")
	}

	if meteoFile != nil {
		input.MeteoFile = meteoFile
		input.MeteoContentType = meteoHeader.Header.Get("Content-Type")
	}

	id, err := h.uc.Create(r.Context(), input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":     id,
		"status": "started",
	})
}

func (h *ExperimentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	exp, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecases.ErrExperimentNotFound) {
			http.Error(w, "experiment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exp)
}

func (h *ExperimentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	exps, err := h.uc.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exps)
}

func (h *ExperimentHandler) Prepare(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	var body struct {
		Hmin    float64 `json:"Hmin"`
		Hmax    float64 `json:"Hmax"`
		BgrType string  `json:"BgrType"`
		BgrAlt  float64 `json:"BgrAlt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.BgrType != "file" && body.BgrType != "avgtail" {
		http.Error(w, "BgrType must be 'file' or 'avgtail'", http.StatusBadRequest)
		return
	}

	if body.BgrType == "avgtail" && body.BgrAlt == 0 {
		http.Error(w, "BgrAlt is required for avgtail", http.StatusBadRequest)
		return
	}

	result, err := h.uc.Prepare(r.Context(), id, usecases.PrepareExperimentInput{
		Hmin:    body.Hmin,
		Hmax:    body.Hmax,
		BgrType: body.BgrType,
		BgrAlt:  body.BgrAlt,
	})
	if err != nil {
		if errors.Is(err, usecases.ErrExperimentNotFound) {
			http.Error(w, "experiment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ExperimentHandler) GenerateImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	var body struct {
		Wavelength   float64 `json:"wavelength"`
		Polarization string  `json:"polarization"`
		ChannelType  string  `json:"channelType"`
		PlotType     string  `json:"plottype"`
		GlueHmin     float64 `json:"glueHmin"`
		GlueHmax     float64 `json:"glueHmax"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.Polarization != "o" && body.Polarization != "p" && body.Polarization != "s" {
		http.Error(w, "polarization must be 'o', 'p' or 's'", http.StatusBadRequest)
		return
	}
	if body.ChannelType != "photon" && body.ChannelType != "analog" && body.ChannelType != "glued" {
		http.Error(w, "channelType must be 'photon', 'analog' or 'glued'", http.StatusBadRequest)
		return
	}
	if body.PlotType != "Raw" && body.PlotType != "RangeCorrected" && body.PlotType != "LogRangeCorrected" {
		http.Error(w, "plottype must be 'Raw', 'RangeCorrected' or 'LogRangeCorrected'", http.StatusBadRequest)
		return
	}

	if body.ChannelType == "glued" && body.GlueHmin == 0 && body.GlueHmax == 0 {
		http.Error(w, "glueHmin and glueHmax are required for channelType=glued", http.StatusBadRequest)
		return
	}
	if body.ChannelType != "glued" && (body.GlueHmin != 0 || body.GlueHmax != 0) {
		http.Error(w, "glueHmin/glueHmax require channelType=glued", http.StatusBadRequest)
		return
	}

	result, err := h.uc.GenerateImage(r.Context(), usecases.GenerateImageInput{
		ExperimentID: id,
		Wavelength:   body.Wavelength,
		Polarization: body.Polarization,
		ChannelType:  body.ChannelType,
		PlotType:     body.PlotType,
		GlueHmin:     body.GlueHmin,
		GlueHmax:     body.GlueHmax,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"path": result.Path,
	})
}

func (h *ExperimentHandler) DownloadImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wavelength := chi.URLParam(r, "wavelength")
	polarization := chi.URLParam(r, "polarization")
	channelType := chi.URLParam(r, "channelType")
	plotType := chi.URLParam(r, "plotType")

	if id == "" || wavelength == "" || polarization == "" || channelType == "" || plotType == "" {
		http.Error(w, "missing path parameters", http.StatusBadRequest)
		return
	}

	rc, err := h.uc.GetImage(r.Context(), id, wavelength, polarization, channelType, plotType)
	if err != nil {
		if errors.Is(err, usecases.ErrImageNotGenerated) {
			http.Error(w, "image not generated", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("failed to stream image: %v", err)
	}
}

func (h *ExperimentHandler) DownloadImageJSON(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wavelength := chi.URLParam(r, "wavelength")
	polarization := chi.URLParam(r, "polarization")
	channelType := chi.URLParam(r, "channelType")
	plotType := chi.URLParam(r, "plotType")

	if id == "" || wavelength == "" || polarization == "" || channelType == "" || plotType == "" {
		http.Error(w, "missing path parameters", http.StatusBadRequest)
		return
	}

	rc, err := h.uc.GetImageJSON(r.Context(), id, wavelength, polarization, channelType, plotType)
	if err != nil {
		if errors.Is(err, usecases.ErrImageNotGenerated) {
			http.Error(w, "json not generated", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		log.Printf("failed to stream json: %v", err)
	}
}

func (h *ExperimentHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	profiles, err := h.uc.ListProfiles(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecases.ErrExperimentNotFound) {
			http.Error(w, "experiment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (h *ExperimentHandler) GenerateProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	var body struct {
		Wavelength   float64 `json:"wavelength"`
		Polarization string  `json:"polarization"`
		ChannelType  string  `json:"channelType"`
		PlotType     string  `json:"plottype"`
		GlueHmin     float64 `json:"glueHmin"`
		GlueHmax     float64 `json:"glueHmax"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.Polarization != "o" && body.Polarization != "p" && body.Polarization != "s" {
		http.Error(w, "polarization must be 'o', 'p' or 's'", http.StatusBadRequest)
		return
	}
	if body.ChannelType != "photon" && body.ChannelType != "analog" && body.ChannelType != "glued" {
		http.Error(w, "channelType must be 'photon', 'analog' or 'glued'", http.StatusBadRequest)
		return
	}
	if body.PlotType != "Raw" && body.PlotType != "RangeCorrected" && body.PlotType != "LogRangeCorrected" {
		http.Error(w, "plottype must be 'Raw', 'RangeCorrected' or 'LogRangeCorrected'", http.StatusBadRequest)
		return
	}

	if body.ChannelType == "glued" && body.GlueHmin == 0 && body.GlueHmax == 0 {
		http.Error(w, "glueHmin and glueHmax are required for channelType=glued", http.StatusBadRequest)
		return
	}
	if body.ChannelType != "glued" && (body.GlueHmin != 0 || body.GlueHmax != 0) {
		http.Error(w, "glueHmin/glueHmax require channelType=glued", http.StatusBadRequest)
		return
	}

	result, err := h.uc.GenerateProfile(r.Context(), usecases.GenerateProfileInput{
		ExperimentID: id,
		Wavelength:   body.Wavelength,
		Polarization: body.Polarization,
		ChannelType:  body.ChannelType,
		PlotType:     body.PlotType,
		GlueHmin:     body.GlueHmin,
		GlueHmax:     body.GlueHmax,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"path": result.Path,
	})
}

func (h *ExperimentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing experiment id", http.StatusBadRequest)
		return
	}

	exp, err := h.uc.GetStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecases.ErrExperimentNotFound) {
			http.Error(w, "experiment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":     exp.ID,
		"status": string(exp.Status),
		"error":  exp.ErrorMessage,
	})
}
