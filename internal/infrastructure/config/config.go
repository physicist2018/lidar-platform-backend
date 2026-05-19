package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddr     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
	JWTSecret      string
	JWTExpiry      time.Duration
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
}

func Load() (*Config, error) {
	port := envOrDefault("PORT", "8080")
	host := envOrDefault("HOST", "0.0.0.0")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET env is required")
	}

	minioBucket := envOrDefault("MINIO_BUCKET", "lidar-experiments")

	return &Config{
		ServerAddr:     fmt.Sprintf("%s:%s", host, port),
		ReadTimeout:    parseDuration(envOrDefault("READ_TIMEOUT", "10s"), 10*time.Second),
		WriteTimeout:   parseDuration(envOrDefault("WRITE_TIMEOUT", "60s"), 60*time.Second),
		IdleTimeout:    parseDuration(envOrDefault("IDLE_TIMEOUT", "60s"), 60*time.Second),
		MaxHeaderBytes: intEnv("MAX_HEADER_BYTES", 1<<20), // 1MB
		JWTSecret:      jwtSecret,
		JWTExpiry:      parseDuration(envOrDefault("JWT_EXPIRY", "24h"), 24*time.Hour),
		MinioEndpoint:  envOrDefault("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: envOrDefault("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: envOrDefault("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    minioBucket,
		MinioUseSSL:    envOrDefault("MINIO_USE_SSL", "false") == "true",
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
