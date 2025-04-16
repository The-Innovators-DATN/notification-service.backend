package main

import (
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
		log.Fatalf("Failed to load config: %v", err)
	}

	// Khởi tạo logger
	logger, err := logging.New()
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// Kết nối database
	dbConn, err := db.New(cfg.DB.DSN)
	if err != nil {
		logger.Errorf("Failed to connect to database: %v", err)
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbConn.Close()

	// Khởi tạo notification service
	svc := notification.New(dbConn, logger, cfg)
	var wg sync.WaitGroup
	svc.Start(&wg)

	// Khởi chạy Kafka consumer
	consumer, err := kafka.NewConsumer(cfg.Kafka.Broker, svc)
	if err != nil {
		logger.Errorf("Failed to initialize Kafka consumer: %v", err)
		log.Fatalf("Kafka consumer initialization failed: %v", err)
	}
	go consumer.Start(&wg)

	// Khởi chạy API server
	router := api.NewRouter(dbConn, logger, cfg)
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
