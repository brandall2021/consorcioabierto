package http

import "testing"

func TestValidateMercadoPagoWebhookSecret(t *testing.T) {
	if !validateMercadoPagoWebhookSecret("", "") {
		t.Fatal("empty secret should allow local mode")
	}
	if !validateMercadoPagoWebhookSecret("secret", "secret") {
		t.Fatal("matching secrets should validate")
	}
	if validateMercadoPagoWebhookSecret("secret", "other") {
		t.Fatal("mismatched secrets should fail")
	}
}
