package telegram

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func Send(token string, chatIDs []int64, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	for _, chatID := range chatIDs {
		body := fmt.Sprintf(`{"chat_id": %d, "text": "%s"}`, chatID, message)
		resp, err := http.Post(url, "application/json", strings.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to send to chat_id %d: %v", chatID, err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("telegram API returned %d for chat_id %d", resp.StatusCode, chatID)
		}
	}
	return nil
}
