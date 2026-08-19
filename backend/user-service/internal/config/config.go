package config

import (
	"strings"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiresIn time.Duration
	CORSOrigins  []string

	// Bootstrap admin opsional. Bila keduanya diisi, akun admin dibuat
	// atau diperbarui saat startup.
	AdminEmail    string
	AdminPassword string
}

func Load() *Config {
	return &Config{
		Port:        env.GetString("PORT", "8080"),
		DatabaseURL: env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/tripnest_users?sslmode=disable"),
		// Sengaja tanpa nilai default. Secret yang kosong akan ditolak
		// auth.NewVerifier saat startup, jadi service tidak pernah jalan
		// dengan secret yang bisa ditebak.
		JWTSecret:    env.GetString("JWT_SECRET", ""),
		JWTExpiresIn: time.Duration(env.GetInt("JWT_EXPIRES_HOURS", 24)) * time.Hour,
		CORSOrigins:  splitAndTrim(env.GetString("CORS_ORIGINS", "http://localhost:3000")),

		AdminEmail:    env.GetString("ADMIN_EMAIL", ""),
		AdminPassword: env.GetString("ADMIN_PASSWORD", ""),
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
