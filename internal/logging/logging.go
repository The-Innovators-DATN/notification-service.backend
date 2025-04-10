package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type Logger struct {
	logger *log.Logger
	file   *os.File
}

func New() (*Logger, error) {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("create logs folder failed: %v", err)
	}
	logFile := fmt.Sprintf("logs/%s.log", time.Now().Format("2006-01-02"))
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file failed: %v", err)
	}
	// Output to both file and console
	logger := log.New(io.MultiWriter(file, os.Stdout), "", 0)
	return &Logger{logger: logger, file: file}, nil
}

func (l *Logger) log(level, requestID, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("%s [%s] [request_id=%s] ", timestamp, level, requestID)
	l.logger.Printf(prefix+msg, args...)
}

func (l *Logger) Debug(requestID, msg string, args ...interface{}) {
	l.log("DEBUG", requestID, msg, args...)
}

func (l *Logger) Info(requestID, msg string, args ...interface{}) {
	l.log("INFO", requestID, msg, args...)
}

func (l *Logger) Warn(requestID, msg string, args ...interface{}) {
	l.log("WARN", requestID, msg, args...)
}

func (l *Logger) Error(requestID, msg string, args ...interface{}) {
	l.log("ERROR", requestID, msg, args...)
}

func (l *Logger) Fatal(requestID, msg string, args ...interface{}) {
	l.log("FATAL", requestID, msg, args...)
	os.Exit(1)
}

func (l *Logger) Close() {
	err := l.file.Close()
	if err != nil {
		return
	}
}
