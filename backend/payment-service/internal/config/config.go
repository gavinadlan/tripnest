package config

import (
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port         string
	DatabaseURL  string
	KafkaBrokers []string
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")
	return &Config{
		Port:         env.GetString("PORT", "8082"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/tripnest_payments?sslmode=disable"),
		KafkaBrokers: strings.Split(brokers, ","),
	}
}
