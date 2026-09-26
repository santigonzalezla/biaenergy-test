package config

import (
	"errors"
	"fmt"
	"net/url"
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
	AiServiceUrl   string
	AiTimeout      time.Duration
}

func Load() (Config, error) {
	var errs []error

	cfg := Config{
		Env:            getEnv("APP_ENV", "development"),
		Port:           getEnv("API_PORT", "8080"),
		DatabaseUrl:    getEnv("DATABASE_URL", ""),
		AllowedOrigins: splitAndTrim(getEnv("ALLOWED_ORIGINS", "")),
		AiServiceUrl:   getEnv("AI_SERVICE_URL", "http://localhost:8000"),
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

	if cfg.IsProduction() && os.Getenv("AI_SERVICE_URL") == "" {
		errs = append(errs, errors.New("AI_SERVICE_URL is required in production"))
	}

	if !isHTTPURL(cfg.AiServiceUrl) {
		errs = append(errs, fmt.Errorf("AI_SERVICE_URL must be an http(s) URL: %q", cfg.AiServiceUrl))
	}

	aiTimeout, err := time.ParseDuration(getEnv("AI_TIMEOUT", "2m"))

	if err != nil || aiTimeout <= 0 {
		errs = append(errs, fmt.Errorf("AI_TIMEOUT must be a positive duration like 90s or 2m: %q", getEnv("AI_TIMEOUT", "2m")))
	}
	cfg.AiTimeout = aiTimeout

	return cfg, errors.Join(errs...)
}

func isHTTPURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)

	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
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
