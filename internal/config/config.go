package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	Kafka struct {
		Broker string
	}
	DB struct {
		DSN string
	}
}

func Load() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, err
	} // Load .env file
	var cfg Config
	cfg.Kafka.Broker = os.Getenv("KAFKA_BROKER")
	cfg.DB.DSN = os.Getenv("DB_DSN")
	return cfg, nil
}
