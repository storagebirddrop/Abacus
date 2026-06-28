package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port       string
	DBPath     string
	Env        string
	Version    string
	FrontendDir string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DBPath:      getEnv("DB_PATH", "./abacus.db"),
		Env:         getEnv("ENV", "production"),
		Version:     getEnv("VERSION", "0.1.0"),
		FrontendDir: getEnv("FRONTEND_DIR", "./web/dist"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

var _ = getEnvInt // silence unused warning; used in future phases
