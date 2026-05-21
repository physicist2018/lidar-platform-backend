package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lidar-platform/backend/internal/application/usecases"
	httpHandlers "github.com/lidar-platform/backend/internal/interfaces/http/handlers"
	authMw "github.com/lidar-platform/backend/internal/interfaces/http/middleware"
)

func NewRouter(authUseCase *usecases.AuthUseCase, expUseCase *usecases.ExperimentUseCase) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/health"))

	authHandler := httpHandlers.NewAuthHandler(authUseCase)
	expHandler := httpHandlers.NewExperimentHandler(expUseCase)

	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Get("/health", httpHandlers.HealthCheck)

	r.Group(func(r chi.Router) {
		r.Use(authMw.AuthMiddleware(authUseCase))
		r.Post("/api/experiments", expHandler.Create)
		r.Get("/api/experiments", expHandler.List)
		r.Get("/api/experiments/{id}", expHandler.GetByID)
		r.Get("/api/experiments/{id}/status", expHandler.GetStatus)
		r.Get("/api/experiments/{id}/profiles", expHandler.ListProfiles)
		r.Post("/api/experiments/{id}/prepare", expHandler.Prepare)
		r.Post("/api/experiments/{id}/image", expHandler.GenerateImage)
		r.Post("/api/experiments/{id}/profile", expHandler.GenerateProfile)
		r.Get("/api/experiments/{id}/{wavelength}/{polarization}/{channelType}/{plotType}", expHandler.DownloadImage)
	})

	return r
}
