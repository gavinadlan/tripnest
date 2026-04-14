package config

import (
	"strconv"
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	KafkaBrokers          []string
	JWTSecret             string
	BookingExpiryMinutes  int
	BookingExpiryInterval int
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")
	expiryMinutes, err := strconv.Atoi(env.GetString("BOOKING_EXPIRY_MINUTES", "15"))
	if err != nil || expiryMinutes <= 0 {
		expiryMinutes = 15
	}
	expiryInterval, err := strconv.Atoi(env.GetString("BOOKING_EXPIRY_INTERVAL_SECONDS", "30"))
	if err != nil || expiryInterval <= 0 {
		expiryInterval = 30
	}
	return &Config{
		Port:                  env.GetString("PORT", "8081"),
		DatabaseURL:           env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/tripnest_booking?sslmode=disable"),
		KafkaBrokers:          strings.Split(brokers, ","),
		JWTSecret:             env.GetString("JWT_SECRET", "super-secret-key"),
		BookingExpiryMinutes:  expiryMinutes,
		BookingExpiryInterval: expiryInterval,
	}
}
