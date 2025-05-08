package main

import (
	"log"
	"notification-service/internal/api"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/kafka"
	"notification-service/internal/logging"
	"notification-service/internal/services"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
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
	go consumer.Start(&wg)

	// Start API server
	handler := api.NewHandler(dbConn, logger, svc)
	router := api.NewRouter(logger, cfg, handler)
	go func() {
		logger.Infof("Starting API server on :8080")
		if err := router.Run(":8080"); err != nil {
			logger.Errorf("API server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logger.Infof("Shutting down service...")
	consumer.Close()
	wg.Wait()
	logger.Infof("Service stopped gracefully")
}
