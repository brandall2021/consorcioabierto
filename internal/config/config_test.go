package config

import (
	"encoding/base64"
	"testing"
)

func baseConfig() *Config {
	return &Config{
		Env:                      "production",
		DatabaseURL:              "postgres://user:pass@localhost:5432/db",
		StorageDriver:            "s3",
		MailDriver:               "smtp",
		PSPDriver:                "mercadopago",
		MercadoPagoWebhookSecret: "secret",
	}
}

func TestValidateExigeDatabaseURL(t *testing.T) {
	cfg := baseConfig()
	cfg.DatabaseURL = ""
	cfg.Env = "local"
	if err := cfg.Validate(); err == nil {
		t.Fatal("se esperaba error por DATABASE_URL vacía")
	}
}

func TestValidateProductionRechazaDriversSimulados(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"psp mock", func(c *Config) { c.PSPDriver = "mock" }},
		{"storage mock", func(c *Config) { c.StorageDriver = "mock" }},
		{"mail mailpit", func(c *Config) { c.MailDriver = "mailpit" }},
		{"mercado pago sin secret", func(c *Config) { c.MercadoPagoWebhookSecret = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseConfig()
			tc.mut(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("se esperaba error por driver simulado en production")
			}
		})
	}
}

func TestValidateProductionAceptaDriversReales(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"todo real", func(*Config) {}},
		// MinIO es S3-compatible (servidor real), no un mock: se permite.
		{"storage minio", func(c *Config) { c.StorageDriver = "minio" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseConfig()
			tc.mut(cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("no se esperaba error: %v", err)
			}
		})
	}
}

func TestJWTPrivateKeyPrecedenciaYBase64(t *testing.T) {
	pem := "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"
	b64 := base64.StdEncoding.EncodeToString([]byte(pem))

	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PRIVATE_KEY_B64", b64)
	if got := jwtPrivateKey(); got != pem {
		t.Fatalf("JWT_PRIVATE_KEY_B64: se esperaba el PEM decodificado, got %q", got)
	}

	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PRIVATE_KEY_B64", "no-es-base64-!")
	if got := jwtPrivateKey(); got != "" {
		t.Fatalf("B64 inválido: se esperaba vacío, got %q", got)
	}

	t.Setenv("JWT_PRIVATE_KEY_B64", b64)
	t.Setenv("JWT_PRIVATE_KEY", "lineal")
	if got := jwtPrivateKey(); got != "lineal" {
		t.Fatalf("JWT_PRIVATE_KEY debe tener prioridad, got %q", got)
	}
}
