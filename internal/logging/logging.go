package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps a logrus.Logger to provide structured logging with file rotation.
type Logger struct {
	instance *logrus.Logger
}

// New creates a Logger that writes JSON-formatted logs to stdout and a rotating file.
// logDir is the directory for log files; level is the minimum log level (e.g., "info", "debug").
func New(logDir, level string) (*Logger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Configure file rotation
	fileLogger := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("2006-01-02")),
		MaxSize:    100,  // megabytes
		MaxBackups: 7,    // number of old files to keep
		MaxAge:     30,   // days
		Compress:   true, // gzip compress rotated files
	}

	// Set up primary logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	log.SetOutput(io.MultiWriter(os.Stdout, fileLogger))

	// Set log level
	enabledLevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.Warnf("invalid log level '%s', defaulting to 'info'", level)
		enabledLevel = logrus.InfoLevel
	}
	log.SetLevel(enabledLevel)

	// Include caller information
	log.SetReportCaller(true)

	return &Logger{instance: log}, nil
}

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.instance.Debugf(format, args...)
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.instance.Infof(format, args...)
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.instance.Warnf(format, args...)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.instance.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message then exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.instance.Fatalf(format, args...)
}

// WithFields returns a log entry with the provided fields.
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.instance.WithFields(fields)
}
