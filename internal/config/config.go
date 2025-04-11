package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
)

type Config struct {
	Kafka struct {
		Broker string
	}
	DB struct {
		DSN string
	}
	Email struct {
		SMTPServer string
		SMTPPort   int
		Username   string
		Password   string
	}
	Telegram struct {
		BotToken string
	}
	SMS struct {
		AccountSID string
		AuthToken  string
		FromNumber string
	}
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, err
	}

	var cfg Config
	cfg.Kafka.Broker = os.Getenv("KAFKA_BROKER")
	cfg.DB.DSN = os.Getenv("DB_DSN")
	cfg.Email.SMTPServer = os.Getenv("EMAIL_SMTP_SERVER")
	cfg.Email.SMTPPort, _ = strconv.Atoi(os.Getenv("EMAIL_SMTP_PORT"))
	cfg.Email.Username = os.Getenv("EMAIL_USERNAME")
	cfg.Email.Password = os.Getenv("EMAIL_PASSWORD")
	cfg.Telegram.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	cfg.SMS.AccountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	cfg.SMS.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	cfg.SMS.FromNumber = os.Getenv("TWILIO_FROM_NUMBER")

	// Basic validation
	if cfg.Kafka.Broker == "" || cfg.DB.DSN == "" {
		return Config{}, fmt.Errorf("missing required config: KAFKA_BROKER or DB_DSN")
	}
	return cfg, nil
}
