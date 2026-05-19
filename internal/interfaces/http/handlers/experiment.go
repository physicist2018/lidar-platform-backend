package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	hmin, err := parseFloat64Form(r, "Hmin")
	if err != nil {
		http.Error(w, "Hmin is required and must be a number", http.StatusBadRequest)
		return
	}

	hmax, err := parseFloat64Form(r, "Hmax")
	if err != nil {
		http.Error(w, "Hmax is required and must be a number", http.StatusBadRequest)
		return
	}

	bgrType := r.FormValue("BgrType")
	if bgrType != "file" && bgrType != "avgtail" {
		http.Error(w, "BgrType must be 'file' or 'avgtail'", http.StatusBadRequest)
		return
	}

	bgrAlt := 0.0
	if bgrType == "avgtail" {
		bgrAlt, err = parseFloat64Form(r, "BgrAlt")
		if err != nil {
			http.Error(w, "BgrAlt is required for avgtail and must be a number", http.StatusBadRequest)
			return
		}
	}

	result, err := h.uc.Prepare(r.Context(), id, usecases.PrepareExperimentInput{
		Hmin:    hmin,
		Hmax:    hmax,
		BgrType: bgrType,
		BgrAlt:  bgrAlt,
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

func parseFloat64Form(r *http.Request, key string) (float64, error) {
	s := r.FormValue(key)
	if s == "" {
		return 0, errors.New("empty value")
	}
	return strconv.ParseFloat(s, 64)
}
