package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"notification-service/internal/config"
	"notification-service/internal/models"
)

type telegramConfig struct {
	ChatID int64 `json:"chat_id"`
}

func SendTelegram(task models.Task, cfg config.Config, cp models.ContactPoint) error {
	botToken := cfg.Telegram.BotToken
	if botToken == "" {
		return fmt.Errorf("missing Telegram configuration: botToken is empty")
	}

	var tConfig telegramConfig
	if err := json.Unmarshal([]byte(cp.Configuration), &tConfig); err != nil {
		return fmt.Errorf("failed to parse Telegram configuration for user_id=%d: %w", task.RecipientID, err)
	}

	if tConfig.ChatID == 0 {
		return fmt.Errorf("chat_id not set in configuration for user_id=%d", task.RecipientID)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id": tConfig.ChatID,
		"text":    fmt.Sprintf("%s\n%s", task.Subject, task.Body),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram payload for chat_id=%d: %w", tConfig.ChatID, err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send Telegram message to chat_id=%d: %w", tConfig.ChatID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned status %d for chat_id=%d", resp.StatusCode, tConfig.ChatID)
	}

	return nil
}
