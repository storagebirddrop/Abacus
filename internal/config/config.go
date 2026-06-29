package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DBPath         string
	Env            string
	Version        string
	FrontendDir    string
	MigrationsPath string

	// Blockchain sync
	BlockchainBackend string // esplora | electrum (default: esplora)
	EsploraURL        string // default: https://mempool.space/api
	EsploraRateMS     int    // ms between requests (default: 100)
	ElectrumHost      string // default: electrum.blockstream.info
	ElectrumPort      int    // default: 50002
	ElectrumTLS       bool   // default: true
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DBPath:         getEnv("DB_PATH", "./abacus.db"),
		Env:            getEnv("ENV", "production"),
		Version:        getEnv("VERSION", "0.1.0"),
		FrontendDir:    getEnv("FRONTEND_DIR", "./web/dist"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "./migrations"),

		BlockchainBackend: getEnv("BLOCKCHAIN_BACKEND", "esplora"),
		EsploraURL:        getEnv("ESPLORA_URL", "https://mempool.space/api"),
		EsploraRateMS:     getEnvInt("ESPLORA_RATE_MS", 100),
		ElectrumHost:      getEnv("ELECTRUM_HOST", "electrum.blockstream.info"),
		ElectrumPort:      getEnvInt("ELECTRUM_PORT", 50002),
		ElectrumTLS:       getEnv("ELECTRUM_TLS", "true") != "false",
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

