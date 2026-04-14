package config

import (
	"strconv"
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port         string
	DatabaseURL  string
	KafkaBrokers []string
	BookingURL   string
	Midtrans     MidtransConfig
}

type MidtransConfig struct {
	MerchantID   string
	ClientKey    string
	ServerKey    string
	IsProduction bool
}

func Load() *Config {
	brokers := env.GetString("KAFKA_BROKERS", "localhost:9092")
	isProduction, _ := strconv.ParseBool(env.GetString("MIDTRANS_IS_PRODUCTION", "false"))
	return &Config{
		Port:         env.GetString("PORT", "8082"),
		DatabaseURL:  env.GetString("DATABASE_URL", "postgres://postgres:postgres@localhost:5434/tripnest_payments?sslmode=disable"),
		KafkaBrokers: strings.Split(brokers, ","),
		BookingURL:   env.GetString("BOOKING_SERVICE_URL", "http://booking-service:8081"),
		Midtrans: MidtransConfig{
			MerchantID:   env.GetString("MIDTRANS_MERCHANT_ID", ""),
			ClientKey:    env.GetString("MIDTRANS_CLIENT_KEY", ""),
			ServerKey:    env.GetString("MIDTRANS_SERVER_KEY", ""),
			IsProduction: isProduction,
		},
	}
}
