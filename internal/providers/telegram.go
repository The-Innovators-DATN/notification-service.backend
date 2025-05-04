package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-telegram/bot"
	"notification-service/internal/models"
)

// telegramConfig holds bot token and chat ID for a Telegram contact point.
type telegramConfig struct {
	BotToken string `json:"bot_token"` // Telegram Bot API token
	ChatID   int64  `json:"chat_id"`   // Destination chat ID
}

// SendTelegram sends a Notification via the go-telegram/bot library, using parameters defined in the contact point's Configuration JSON.
func SendTelegram(ctx context.Context, notif models.Notification, cp models.ContactPoint) error {
	// Parse configuration from ContactPoint.Configuration
	var cfg telegramConfig
	configBytes, err := json.Marshal(cp.Configuration) // Convert map to JSON bytes
	if err != nil {
		return fmt.Errorf("failed to marshal configuration for contact point %s: %w", cp.ID, err)
	}
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return fmt.Errorf("invalid Telegram configuration for contact point %s: %w", cp.ID, err)
	}
	if cfg.BotToken == "" {
		return fmt.Errorf("missing bot_token in Telegram configuration for contact point %s", cp.ID)
	}
	if cfg.ChatID == 0 {
		return fmt.Errorf("missing chat_id in Telegram configuration for contact point %s", cp.ID)
	}

	// Initialize bot
	b, err := bot.New(cfg.BotToken)
	if err != nil {
		return fmt.Errorf("failed to initialize Telegram bot for contact point %s: %w", cp.ID, err)
	}

	// Compose message with contextual details (Markdown support)
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

	// Send the message
	params := &bot.SendMessageParams{
		ChatID:    cfg.ChatID,
		Text:      text,
		ParseMode: "Markdown",
	}
	if _, err := b.SendMessage(ctx, params); err != nil {
		return fmt.Errorf("failed to send Telegram message to chat_id %d: %w", cfg.ChatID, err)
	}
	return nil
}
