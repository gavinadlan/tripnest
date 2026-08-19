package config

import (
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port        string
	MongoURI    string
	RedisAddr   string
	JWTSecret   string
	CORSOrigins []string
}

func Load() *Config {
	return &Config{
		Port:      env.GetString("PORT", "8083"),
		MongoURI:  env.GetString("MONGO_URI", "mongodb://localhost:27017/tripnest_search"),
		RedisAddr: env.GetString("REDIS_ADDR", "localhost:6379"),
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
