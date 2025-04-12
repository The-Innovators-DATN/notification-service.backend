// cmd/main.go
package main

import (
	"context"
	"log"
	"notification-service/internal/api"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/kafka"
	"notification-service/internal/logging"
	"notification-service/internal/notification"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Config load failed:", err)
	}

	// Initialize logger
	logger, err := logging.New()
	if err != nil {
		log.Fatal("Logger init failed:", err)
	}
	defer logger.Close()

	// Connect to DB
	dbConn, err := db.New(cfg.DB.DSN)
	if err != nil {
		logger.Error("", "DB connect failed: %v", err)
		log.Fatal("DB connect failed:", err)
	}
	defer func(dbConn *db.DB) {
		err := dbConn.Close()
		if err != nil {
			logger.Error("", "DB close failed: %v", err)
		} else {
			logger.Info("", "DB connection closed")
		}
	}(dbConn)

	// Initialize notification service
	svc := notification.New(dbConn, logger, cfg)
	var wg sync.WaitGroup
	svc.Start(&wg)

	// Start Kafka consumer
	consumer, err := kafka.NewConsumer(cfg.Kafka, svc)
	if err != nil {
		logger.Error("", "Kafka connect failed: %v", err)
		log.Fatal("Kafka connect failed:", err)
	}
	go consumer.Start(&wg)

	// Start API server
	r := api.NewRouter(dbConn, logger, cfg)
	go func() {
		logger.Info("", "API started on :8080")
		if err := r.Run(":8080"); err != nil {
			logger.Error("", "API run failed: %v", err)
		}
	}()

	// Handle graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Info("", "Shutting down...")
	cancel()
	consumer.Close()
	wg.Wait()
	logger.Info("", "Service stopped")
}
