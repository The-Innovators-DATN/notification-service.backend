package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"notification-service/internal/config"
	"notification-service/internal/models"
	"strings"
)

type smsConfig struct {
	PhoneNumber string `json:"phone_number"` // Quay lại một số điện thoại duy nhất
}

func SendSMS(task models.Task, cfg config.Config, cp models.ContactPoint) error {
	// Parse configuration để lấy phone_number
	var sConfig smsConfig
	if err := json.Unmarshal([]byte(cp.Configuration), &sConfig); err != nil {
		return fmt.Errorf("failed to parse SMS configuration for user_id=%d: %w", task.RecipientID, err)
	}

	if sConfig.PhoneNumber == "" {
		return fmt.Errorf("phone_number not set in configuration for user_id=%d", task.RecipientID)
	}

	// Cấu hình Twilio
	accountSID := cfg.SMS.AccountSID
	authToken := cfg.SMS.AuthToken
	fromNumber := cfg.SMS.FromNumber

	if accountSID == "" || authToken == "" || fromNumber == "" {
		return fmt.Errorf("missing SMS configuration: AccountSID, AuthToken, or FromNumber is empty")
	}

	// Tạo nội dung SMS
	urlStr := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", accountSID)
	msgData := url.Values{}
	msgData.Set("To", sConfig.PhoneNumber)
	msgData.Set("From", fromNumber)
	msgData.Set("Body", fmt.Sprintf("%s\n%s", task.Subject, task.Body))
	msgDataReader := *strings.NewReader(msgData.Encode())

	// Tạo request
	client := &http.Client{}
	req, err := http.NewRequest("POST", urlStr, &msgDataReader)
	if err != nil {
		return fmt.Errorf("failed to create SMS request for phone_number=%s: %w", sConfig.PhoneNumber, err)
	}

	req.SetBasicAuth(accountSID, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Gửi SMS
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS to %s: %w", sConfig.PhoneNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Twilio API returned status %d for phone_number=%s", resp.StatusCode, sConfig.PhoneNumber)
	}

	return nil
}
