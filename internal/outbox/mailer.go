// Mailer del outbox worker (H3.6, §5.4). Interfaz mínima con drivers mailpit
// y mock; se elige por la variable MAIL_DRIVER al armar el Worker.
package outbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// mailpitPayload arma el body de POST /api/v1/send según el schema de Mailpit:
// from y to son objetos {name, email} / array de ellos, y el texto va en "text".
func mailpitPayload(to, subject, body string) ([]byte, error) {
	payload := map[string]any{
		"from":    map[string]string{"email": "consorcioabierto@local"},
		"to":      []map[string]string{{"email": to}},
		"subject": subject,
		"text":    body,
	}
	return json.Marshal(payload)
}

func (m *MailpitDriver) Send(to, subject, body string) error {
	data, err := mailpitPayload(to, subject, body)
	if err != nil {
		return fmt.Errorf("mailpit marshal: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(m.BaseURL+"/api/v1/send", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("mailpit send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("mailpit %d: read body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("mailpit %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (m *MockDriver) Send(to, subject, body string) error {
	m.Log.Info("email mock", "to", to, "subject", subject, "body_len", len(body))
	return nil
}
