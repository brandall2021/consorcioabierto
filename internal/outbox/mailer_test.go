package outbox

import (
	"encoding/json"
	"testing"
)

func TestMailpitPayloadShape(t *testing.T) {
	data, err := mailpitPayload("destinatario@example.com", "Asunto", "cuerpo del mail")
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("payload is not a JSON object: %v", err)
	}

	if _, ok := raw["body"]; ok {
		t.Error(`unexpected "body" key: Mailpit expects "text"`)
	}
	for _, key := range []string{"from", "to", "subject", "text"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing %q key", key)
		}
	}

	var from map[string]string
	if err := json.Unmarshal(raw["from"], &from); err != nil {
		t.Fatalf("from must be an object {name,email}: %v", err)
	}
	if from["email"] != "consorcioabierto@local" {
		t.Errorf("from.email = %q, want consorcioabierto@local", from["email"])
	}

	var to []map[string]string
	if err := json.Unmarshal(raw["to"], &to); err != nil {
		t.Fatalf("to must be an array of {name,email}: %v", err)
	}
	if len(to) != 1 || to[0]["email"] != "destinatario@example.com" {
		t.Errorf("to = %#v, want [{email:destinatario@example.com}]", to)
	}

	var subject, text string
	if err := json.Unmarshal(raw["subject"], &subject); err != nil {
		t.Fatalf("subject must be a string: %v", err)
	}
	if subject != "Asunto" {
		t.Errorf("subject = %q, want Asunto", subject)
	}
	if err := json.Unmarshal(raw["text"], &text); err != nil {
		t.Fatalf("text must be a string: %v", err)
	}
	if text != "cuerpo del mail" {
		t.Errorf("text = %q, want cuerpo del mail", text)
	}
}
