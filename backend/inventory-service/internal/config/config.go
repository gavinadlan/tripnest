package config

import (
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port         string
	DatabaseURL  string
	KafkaBrokers []string
	JWTSecret    string
	CORSOrigins  []string
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")
	return &Config{
		Port:         env.GetString("PORT", "8084"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/tripnest_inventory?sslmode=disable"),
		KafkaBrokers: strings.Split(brokers, ","),
		// Tanpa default: secret kosong ditolak saat startup.
		JWTSecret:   env.GetString("JWT_SECRET", ""),
		CORSOrigins: splitAndTrim(env.GetString("CORS_ORIGINS", "http://localhost:3000")),
	}
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
