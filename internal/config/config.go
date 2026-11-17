package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	HTTPAddr string
	BaseURL  string
	LogLevel slog.Level
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getRequiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", errors.New(key + " is not set")
	}

	if value == "" {
		return "", errors.New(key + " is empty")
	}

	return value, nil
}

func getLogLevel() (slog.Level, error) {
	var lvl slog.Level
	key := "LOG_LEVEL"

	value := getEnv(key, "info")

	switch value {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return lvl, fmt.Errorf("%s: unsupported value %q, want debug, info, warn or error", key, value)
	}

	return lvl, nil
}

func GetConfig() (*Config, error) {
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	baseURL, err := getRequiredEnv("BASE_URL")
	if err != nil {
		return nil, err
	}

	lvl, err := getLogLevel()
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPAddr: httpAddr,
		BaseURL:  baseURL,
		LogLevel: lvl,
	}, nil
}
