package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Kafka struct {
		Broker  string
		Topic   string
		GroupID string
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
	API struct {
		Port     string
		BasePath string
	}
	Notification struct {
		QueueSize  int
		MaxWorkers int
	}
}

func Load() (Config, error) {
	if err := godotenv.Load("../.env"); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("failed to load .env file: %w", err)
	}

	var cfg Config

	// Kafka
	cfg.Kafka.Broker = os.Getenv("KAFKA_BROKER")
	cfg.Kafka.Topic = os.Getenv("KAFKA_TOPIC")
	cfg.Kafka.GroupID = os.Getenv("KAFKA_GROUP_ID")

	// Database
	cfg.DB.DSN = os.Getenv("DB_DSN")

	// Email
	cfg.Email.SMTPServer = os.Getenv("EMAIL_SMTP_SERVER")
	if port, err := strconv.Atoi(os.Getenv("EMAIL_SMTP_PORT")); err == nil {
		cfg.Email.SMTPPort = port
	}
	cfg.Email.Username = os.Getenv("EMAIL_USERNAME")
	cfg.Email.Password = os.Getenv("EMAIL_PASSWORD")

	// Telegram
	cfg.Telegram.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	// SMS (Twilio)
	cfg.SMS.AccountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	cfg.SMS.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	cfg.SMS.FromNumber = os.Getenv("TWILIO_FROM_NUMBER")

	// API
	cfg.API.Port = os.Getenv("API_PORT")
	cfg.API.BasePath = os.Getenv("API_BASE_PATH")

	// Notification
	if queueSize, err := strconv.Atoi(os.Getenv("QUEUE_SIZE")); err == nil {
		cfg.Notification.QueueSize = queueSize
	}
	if maxWorkers, err := strconv.Atoi(os.Getenv("MAX_WORKERS")); err == nil {
		cfg.Notification.MaxWorkers = maxWorkers
	}

	if cfg.Kafka.Broker == "" || cfg.DB.DSN == "" {
		return Config{}, fmt.Errorf("missing required configurations: KAFKA_BROKER and DB_DSN are required")
	}

	if cfg.API.Port == "" {
		cfg.API.Port = ":8080"
	}
	if cfg.API.BasePath == "" {
		cfg.API.BasePath = "/api/v0"
	}
	if cfg.Notification.QueueSize == 0 {
		cfg.Notification.QueueSize = 500
	}
	if cfg.Notification.MaxWorkers == 0 {
		cfg.Notification.MaxWorkers = 10
	}

	return cfg, nil
}
