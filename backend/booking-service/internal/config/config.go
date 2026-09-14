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
	CatalogURL            string
	CORSOrigins           []string
	BookingExpiryMinutes  int
	SeatTimeoutSeconds    int
	BookingExpiryInterval int
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")

	expiryMinutes, err := strconv.Atoi(env.GetString("BOOKING_EXPIRY_MINUTES", "15"))
	if err != nil || expiryMinutes < 10 {
		expiryMinutes = 15
	}
	seatTimeout, err := strconv.Atoi(env.GetString("BOOKING_SEAT_TIMEOUT_SECONDS", "120"))
	if err != nil || seatTimeout < 30 {
		seatTimeout = 120
	}
	expiryInterval, err := strconv.Atoi(env.GetString("BOOKING_EXPIRY_INTERVAL_SECONDS", "60"))
	if err != nil || expiryInterval < 60 {
		expiryInterval = 60
	}

	return &Config{
		Port:         env.GetString("PORT", "8081"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/tripnest_booking?sslmode=disable"),
		KafkaBrokers: strings.Split(brokers, ","),
		// Tanpa default: secret kosong ditolak saat startup.
		JWTSecret: env.GetString("JWT_SECRET", ""),
		// Sumber resmi harga dan sisa kursi. Trip sekarang dimiliki
		// catalog-service (Postgres); search-service hanya read model.
		CatalogURL:            env.GetString("CATALOG_SERVICE_URL", "http://inventory-service:8084"),
		CORSOrigins:           splitAndTrim(env.GetString("CORS_ORIGINS", "http://localhost:3000")),
		BookingExpiryMinutes:  expiryMinutes,
		// Batas menunggu jawaban kursi dari catalog-service. Normalnya
		// beberapa detik; ini jaring pengaman kalau jawabannya tidak
		// pernah datang.
		SeatTimeoutSeconds:    seatTimeout,
		BookingExpiryInterval: expiryInterval,
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
