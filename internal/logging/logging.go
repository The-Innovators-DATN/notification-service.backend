package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	logger *logrus.Logger
	file   *os.File
}

func New() (*Logger, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	logFileName := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger.SetOutput(io.MultiWriter(os.Stdout, file))
	return &Logger{logger: logger, file: file}, nil
}

func (c *Logger) Close() error {
	if err := c.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}
	return nil
}

func (c *Logger) Infof(format string, args ...interface{}) {
	c.logger.Infof(format, args...)
}

func (c *Logger) Errorf(format string, args ...interface{}) {
	c.logger.Errorf(format, args...)
}
