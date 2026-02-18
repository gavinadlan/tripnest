package config

import (
	"time"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiresIn time.Duration
}

func Load() *Config {
	return &Config{
		Port:         env.GetString("PORT", "8080"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://user:password@localhost:5432/tripnest_users?sslmode=disable"),
		JWTSecret:    env.GetString("JWT_SECRET", "super-secret-key"),
		JWTExpiresIn: time.Hour * 24,
	}
}
