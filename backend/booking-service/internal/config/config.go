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
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")
	return &Config{
		Port:         env.GetString("PORT", "8081"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/tripnest_bookings?sslmode=disable"),
		KafkaBrokers: strings.Split(brokers, ","),
		JWTSecret:    env.GetString("JWT_SECRET", "super-secret-key"),
	}
}
