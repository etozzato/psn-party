package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Name                     string
	Env                      string
	PublicBaseURL            string
	BindAddress              string
	Port                     int
	DatabaseURL              string
	CORSAllowOrigin          string
	LogLevel                 slog.Level
	DBMaxConns               int32
	DBMinConns               int32
	DBMaxConnIdleSeconds     int
	DBMaxConnLifetimeSeconds int
	ProfileTimeoutSeconds    int
	AdminToken               string
}

func Load() (Config, error) {
	loadEnvFiles(".env", "config/.env")

	cfg := Config{
		Name:            env("NAME", "PSN Add"),
		Env:             strings.ToLower(env("APP_ENV", "development")),
		PublicBaseURL:   strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8890"), "/"),
		BindAddress:     env("BIND_ADDRESS", "127.0.0.1"),
		DatabaseURL:     env("DATABASE_URL", ""),
		CORSAllowOrigin: env("CORS_ALLOW_ORIGIN", "*"),
		AdminToken:      env("ADMIN_TOKEN", ""),
	}

	var errs []error
	var err error

	cfg.Port, err = intEnv("PORT", 8890)
	if err != nil {
		errs = append(errs, err)
	}
	maxConns, err := intEnv("DB_MAX_CONNS", 15)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.DBMaxConns = int32(maxConns)
	minConns, err := intEnv("DB_MIN_CONNS", 2)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.DBMinConns = int32(minConns)
	cfg.DBMaxConnIdleSeconds, err = intEnv("DB_MAX_CONN_IDLE_SECONDS", 120)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.DBMaxConnLifetimeSeconds, err = intEnv("DB_MAX_CONN_LIFETIME_SECONDS", 1800)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.ProfileTimeoutSeconds, err = intEnv("PROFILE_TIMEOUT_SECONDS", 4)
	if err != nil {
		errs = append(errs, err)
	}
	cfg.LogLevel, err = logLevel(env("LOG_LEVEL", "info"))
	if err != nil {
		errs = append(errs, err)
	}

	if cfg.BindAddress == "" {
		errs = append(errs, errors.New("BIND_ADDRESS cannot be empty"))
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, fmt.Errorf("PORT out of range: %d", cfg.Port))
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if cfg.DBMaxConns < 1 {
		errs = append(errs, errors.New("DB_MAX_CONNS must be >= 1"))
	}
	if cfg.DBMinConns < 0 || cfg.DBMinConns > cfg.DBMaxConns {
		errs = append(errs, errors.New("DB_MIN_CONNS must be between 0 and DB_MAX_CONNS"))
	}
	if cfg.ProfileTimeoutSeconds < 1 {
		errs = append(errs, errors.New("PROFILE_TIMEOUT_SECONDS must be >= 1"))
	}

	if len(errs) > 0 {
		return cfg, errors.Join(errs...)
	}
	return cfg, nil
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
		}
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback, fmt.Errorf("%s must be an integer, got %q", key, raw)
	}
	return value, nil
}

func logLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid LOG_LEVEL: %q", value)
	}
}
