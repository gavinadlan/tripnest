package config

import (
	"github.com/gavinadlan/tripnest/backend/common/env"
)

type Config struct {
	Port      string
	MongoURI  string
	RedisAddr string
}

func Load() *Config {
	return &Config{
		Port:      env.GetString("PORT", "8083"),
		MongoURI:  env.GetString("MONGO_URI", "mongodb://localhost:27017/tripnest_search"),
		RedisAddr: env.GetString("REDIS_ADDR", "localhost:6379"),
	}
}
