// Mailer del outbox worker (H3.6, §5.4). Interfaz mínima con drivers mailpit
// y mock; se elige por la variable MAIL_DRIVER al armar el Worker.
package outbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (m *MailpitDriver) Send(to, subject, body string) error {
	payload := map[string]string{
		"from":    "consorcioabierto@local",
		"to":      to,
		"subject": subject,
		"body":    body,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailpit marshal: %w", err)
	}
	resp, err := http.Post(m.BaseURL+"/api/v1/send", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("mailpit send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailpit %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (m *MockDriver) Send(to, subject, body string) error {
	m.Log.Info("email mock", "to", to, "subject", subject, "body_len", len(body))
	return nil
}
