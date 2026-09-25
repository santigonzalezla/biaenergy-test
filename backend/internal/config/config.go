package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env            string
	Port           string
	DatabaseUrl    string
	AllowedOrigins []string
	Location       *time.Location
}

func Load() (Config, error) {
	var errs []error

	cfg := Config{
		Env:            getEnv("APP_ENV", "development"),
		Port:           getEnv("API_PORT", "8080"),
		DatabaseUrl:    getEnv("DATABASE_URL", ""),
		AllowedOrigins: splitAndTrim(getEnv("ALLOWED_ORIGINS", "")),
	}

	if cfg.DatabaseUrl == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	if cfg.IsProduction() && len(cfg.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("ALLOWED_ORIGINS is required in production"))
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		errs = append(errs, fmt.Errorf("API_PORT must be a valid integer: %q", cfg.Port))
	}

	location, err := time.LoadLocation(getEnv("APP_TIMEZONE", "America/Bogota"))

	if err != nil {
		errs = append(errs, fmt.Errorf("APP_TIMEZONE is invalid: %w", err))
	}
	cfg.Location = location

	return cfg, errors.Join(errs...)
}

func (config Config) IsProduction() bool {
	return config.Env == "production"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	return fallback
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
