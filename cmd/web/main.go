package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lidar-platform/backend/internal/application/usecases"
	"github.com/lidar-platform/backend/internal/infrastructure/auth"
	"github.com/lidar-platform/backend/internal/infrastructure/config"
	repo "github.com/lidar-platform/backend/internal/infrastructure/repository/sqlite"
	"github.com/lidar-platform/backend/internal/infrastructure/storage"
	chiRouter "github.com/lidar-platform/backend/internal/interfaces/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := repo.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Auth
	userRepo := repo.NewUserRepository(db)
	bcryptHasher := auth.NewBcryptHasher(0)
	jwtProvider := auth.NewJWTProvider(cfg.JWTSecret, cfg.JWTExpiry)
	authUseCase := usecases.NewAuthUseCase(userRepo, bcryptHasher, jwtProvider)

	// Experiments
	minioStorage, err := storage.NewMinioStorage(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioUseSSL)
	if err != nil {
		log.Fatalf("failed to init minio storage: %v", err)
	}
	expRepo := repo.NewExperimentRepository(db)
	profileRepo := repo.NewExperimentProfileRepository(db)
	expUseCase := usecases.NewExperimentUseCase(expRepo, profileRepo, minioStorage)

	router := chiRouter.NewRouter(authUseCase, expUseCase)

	srv := &http.Server{
		Addr:           cfg.ServerAddr,
		Handler:        router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderBytes,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("received signal %s, shutting down...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("starting server on %s", cfg.ServerAddr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}

	<-idleConnsClosed
	log.Println("server stopped gracefully")
}
