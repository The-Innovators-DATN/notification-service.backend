package main

import (
	"log"
	"notification-service/internal/api"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/kafka"
	"notification-service/internal/logging"
	"notification-service/internal/services"
	"sync"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger, err := logging.New(cfg.Logging.Dir, cfg.Logging.Level)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	// Connect to database
	dbConn, err := db.New(cfg.DB.DSN)
	if err != nil {
		logger.Errorf("Failed to connect to database: %v", err)
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbConn.Close()

	// Initialize notification service
	svc := services.New(dbConn, logger, cfg)
	var wg sync.WaitGroup
	svc.Start(&wg)

	// Initialize Kafka consumer
	consumer := kafka.NewConsumer([]string{cfg.Kafka.Broker}, cfg.Kafka.Topic, cfg.Kafka.GroupID, svc)
	log.Printf("Kafka consumer initialized with topic: %s", cfg.Kafka.Topic)
	go consumer.Start(&wg)

	// Start API server
	handler := api.NewHandler(dbConn, logger, svc)
	router := api.NewRouter(logger, cfg, handler)
	logger.Infof("Starting API server on :9191")
	if err := router.Run(":9191"); err != nil {
		logger.Errorf("API server failed: %v", err)
	}
}
