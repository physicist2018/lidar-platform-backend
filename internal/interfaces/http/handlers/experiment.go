package handlers

import (
	"encoding/json"
	"net/http"

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

	// ZipFile — required
	zipFile, zipHeader, err := r.FormFile("ZipFile")
	if err != nil {
		http.Error(w, "ZipFile is required", http.StatusBadRequest)
		return
	}
	defer zipFile.Close()

	// BgrFile — optional
	bgrFile, bgrHeader, _ := r.FormFile("BgrFile")
	if bgrFile != nil {
		defer bgrFile.Close()
	}

	// MeteoFile — optional
	meteoFile, meteoHeader, _ := r.FormFile("MeteoFile")
	if meteoFile != nil {
		defer meteoFile.Close()
	}

	input := usecases.CreateExperimentInput{
		UserID:   userID,
		Title:    title,
		Comments: comments,
		ZipFile:  zipFile,
		ZipSize:  zipHeader.Size,
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
		"id": id,
	})
}
