package sms

import (
	"fmt"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"strings"
)

func Send(accountSID, authToken, fromNumber, toNumber, body string) error {
	if !strings.HasPrefix(toNumber, "+") {
		return fmt.Errorf("invalid phone number: %s", toNumber)
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	params := &twilioApi.CreateMessageParams{
		To:   &toNumber,
		From: &fromNumber,
		Body: &body,
	}

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS to %s: %v", toNumber, err)
	}
	return nil
}
