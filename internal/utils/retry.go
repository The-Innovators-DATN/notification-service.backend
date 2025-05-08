package utils

import (
	"fmt"
	"notification-service/internal/logging"
	"time"
)

func Retry(logger *logging.Logger, maxAttempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			logger.Errorf("Attempt %d/%d failed: %v", attempt, maxAttempts, err)
			if attempt < maxAttempts {
				time.Sleep(delay)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
