package config

import "testing"

func baseConfig() *Config {
	return &Config{
		Env:           "production",
		DatabaseURL:   "postgres://user:pass@localhost:5432/db",
		StorageDriver: "s3",
		MailDriver:    "smtp",
		PSPDriver:     "mercadopago",
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
