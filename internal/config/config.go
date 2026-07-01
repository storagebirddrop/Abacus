package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DBPath      string
	Env         string
	FrontendDir string

	// Security
	APIToken     string // when set, /api/v1 requires Bearer auth (default: off)
	RateLimitRPM int    // per-IP requests/min on /api/v1; <= 0 disables
	TrustProxy   bool   // when true, derive client IP from X-Forwarded-For/X-Real-IP
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DBPath:      getEnv("DB_PATH", "./abacus.db"),
		Env:         getEnv("ENV", "production"),
		FrontendDir: getEnv("FRONTEND_DIR", "./web/dist"),

		APIToken:     getEnv("API_TOKEN", ""),
		RateLimitRPM: getEnvInt("RATE_LIMIT_RPM", 600),
		TrustProxy:   getEnvBool("TRUST_PROXY", false),
	}
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
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
