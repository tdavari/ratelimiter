package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Redis struct {
		Addr     string
		DB       int
		PoolSize int
	}
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	getInt := func(key string, def int) int {
		if val := os.Getenv(key); val != "" {
			if v, err := strconv.Atoi(val); err == nil {
				return v
			}
		}
		return def
	}

	cfg := &Config{}

	cfg.Redis.Addr = os.Getenv("REDIS_ADDR")
	cfg.Redis.DB = getInt("REDIS_DB", 0)
	cfg.Redis.PoolSize = getInt("REDIS_POOLSIZE", 100)

	return cfg, nil
}
