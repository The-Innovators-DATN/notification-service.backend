package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment.
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
		FromName   string
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

// Load reads environment variables, applies defaults, and returns a Config.
func Load() (Config, error) {
	// Load .env if present
	if err := godotenv.Load("../.env"); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("failed to load .env file: %w", err)
	}

	var cfg Config

	// Kafka settings
	cfg.Kafka.Broker = os.Getenv("KAFKA_BROKER")
	cfg.Kafka.Topic = os.Getenv("KAFKA_TOPIC")
	cfg.Kafka.GroupID = os.Getenv("KAFKA_GROUP_ID")

	// Database DSN
	cfg.DB.DSN = os.Getenv("DB_DSN")

	// Email settings
	cfg.Email.SMTPServer = os.Getenv("EMAIL_SMTP_SERVER")
	if p, err := strconv.Atoi(os.Getenv("EMAIL_SMTP_PORT")); err == nil {
		cfg.Email.SMTPPort = p
	}
	cfg.Email.Username = os.Getenv("EMAIL_USERNAME")
	cfg.Email.Password = os.Getenv("EMAIL_PASSWORD")
	cfg.Email.FromName = os.Getenv("EMAIL_FROM_NAME")

	// API settings
	cfg.API.Port = os.Getenv("API_PORT")
	cfg.API.BasePath = os.Getenv("API_BASE_PATH")

	// Notification worker settings
	if qs, err := strconv.Atoi(os.Getenv("QUEUE_SIZE")); err == nil {
		cfg.Notification.QueueSize = qs
	}
	if mw, err := strconv.Atoi(os.Getenv("MAX_WORKERS")); err == nil {
		cfg.Notification.MaxWorkers = mw
	}

	// Validate required settings
	missing := []string{}
	if cfg.Kafka.Broker == "" {
		missing = append(missing, "KAFKA_BROKER")
	}
	if cfg.DB.DSN == "" {
		missing = append(missing, "DB_DSN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required configurations: %v", missing)
	}

	// Apply defaults
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
