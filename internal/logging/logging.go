package logging

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

type Logger struct {
	logger *logrus.Logger
}

func New() (*Logger, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Create logs directory if it doesn't exist
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	logger.SetOutput(file)
	logger.AddHook(&consoleHook{logger: logrus.New()})
	return &Logger{logger: logger}, nil
}

func (l *Logger) Close() {
	// Logrus file is closed automatically when program exits
}

func (l *Logger) Info(requestID, msg string, args ...interface{}) {
	l.logger.WithField("request_id", requestID).Infof(msg, args...)
}

func (l *Logger) Warn(requestID, msg string, args ...interface{}) {
	l.logger.WithField("request_id", requestID).Warnf(msg, args...)
}

func (l *Logger) Error(requestID, msg string, args ...interface{}) {
	l.logger.WithField("request_id", requestID).Errorf(msg, args...)
}

func (l *Logger) Debug(requestID, msg string, args ...interface{}) {
	l.logger.WithField("request_id", requestID).Debugf(msg, args...)
}

type consoleHook struct {
	logger *logrus.Logger
}

func (h *consoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *consoleHook) Fire(entry *logrus.Entry) error {
	h.logger.SetFormatter(&logrus.TextFormatter{})
	h.logger.SetOutput(os.Stdout)
	switch entry.Level {
	case logrus.InfoLevel:
		h.logger.Info(entry.Message)
	case logrus.WarnLevel:
		h.logger.Warn(entry.Message)
	case logrus.ErrorLevel:
		h.logger.Error(entry.Message)
	case logrus.DebugLevel:
		h.logger.Debug(entry.Message)
	default:
		panic("unhandled default case")
	}
	return nil
}
