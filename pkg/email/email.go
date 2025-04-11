package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

func Send(server string, port int, username, password, to, subject, body string) error {
	if !strings.Contains(to, "@") {
		return fmt.Errorf("invalid email address: %s", to)
	}

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body))
	auth := smtp.PlainAuth("", username, password, server)
	addr := fmt.Sprintf("%s:%d", server, port)
	return smtp.SendMail(addr, auth, username, []string{to}, msg)
}
