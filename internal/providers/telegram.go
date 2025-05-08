package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"golang.org/x/time/rate"
	"notification-service/internal/config"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/internal/utils"
)

// telegramConfig holds bot token and chat ID for a Telegram contact point.
type telegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   int64  `json:"chat_id"`
}

// telegramLimiter is the global rate limiter for Telegram messages
var telegramLimiter *rate.Limiter

// initTelegramLimiter initializes the Telegram rate limiter
func initTelegramLimiter(ratePerSecond int) {
	telegramLimiter = rate.NewLimiter(rate.Limit(float64(ratePerSecond)), ratePerSecond)
}

// SendTelegram sends a Notification via the go-telegram/bot library
func SendTelegram(ctx context.Context, notif models.Notification, cp models.ContactPoint, logger *logging.Logger, cfg config.Config) error {
	// Initialize rate limiter if not set
	if telegramLimiter == nil {
		initTelegramLimiter(cfg.RateLimit.TelegramRateLimiter)
	}

	// Check rate limit
	if err := telegramLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("telegram rate limit exceeded: %w", err)
	}

	// Parse configuration
	var tCfg telegramConfig
	configBytes, err := json.Marshal(cp.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration for contact point %s: %w", cp.ID, err)
	}
	if err := json.Unmarshal(configBytes, &tCfg); err != nil {
		return fmt.Errorf("invalid Telegram configuration for contact point %s: %w", cp.ID, err)
	}
	if tCfg.BotToken == "" {
		return fmt.Errorf("missing bot_token in Telegram configuration for contact point %s", cp.ID)
	}
	if tCfg.ChatID == 0 {
		return fmt.Errorf("missing chat_id in Telegram configuration for contact point %s", cp.ID)
	}

	// Compose message
	text := fmt.Sprintf(
		"*%s*\n%s\n\n"+
			"*Station ID:* %d\n"+
			"*Metric:* %s (ID %d)\n"+
			"*Operator:* %s\n"+
			"*Threshold:* %.2f (min %.2f, max %.2f)\n"+
			"*Value:* %.2f",
		notif.Subject,
		notif.Body,
		notif.Context.StationID,
		notif.Context.MetricName,
		notif.Context.MetricID,
		notif.Context.Operator,
		notif.Context.Threshold,
		notif.Context.ThresholdMin,
		notif.Context.ThresholdMax,
		notif.Context.Value,
	)

	// Retry sending message
	return utils.Retry(logger, 3, time.Second, func() error {
		b, err := bot.New(tCfg.BotToken)
		if err != nil {
			return fmt.Errorf("failed to initialize Telegram bot for contact point %s: %w", cp.ID, err)
		}

		params := &bot.SendMessageParams{
			ChatID:    tCfg.ChatID,
			Text:      text,
			ParseMode: "Markdown",
		}
		if _, err := b.SendMessage(ctx, params); err != nil {
			return fmt.Errorf("failed to send Telegram message to chat_id %d: %w", tCfg.ChatID, err)
		}
		return nil
	})
}
