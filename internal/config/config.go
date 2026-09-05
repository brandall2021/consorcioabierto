// Package config carga y valida la configuración desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config concentra las variables de entorno del proceso (apps/api y apps/worker).
type Config struct {
	Env              string
	LogFormat        string
	HTTPAddr         string
	BaseURL          string
	RequestTimeout   time.Duration
	DatabaseURL      string
	DatabaseURLAdmin string

	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	MFATokenTTL        time.Duration
	LoginMaxAttempts   int
	LoginAttemptWindow time.Duration
	MembershipCacheTTL time.Duration
	JWTPrivateKey      string

	StorageDriver string
	MailDriver    string
	PSPDriver     string

	S3Endpoint    string
	S3Bucket      string
	S3AccessKey   string
	S3SecretKey   string
	S3Region      string
	S3UseSSL      bool
	S3SignedTTL   time.Duration

	ScanDriver      string
	MaxUploadBytes  int64
}

// Load lee la configuración y la valida.
func Load() (*Config, error) {
	cfg := &Config{
		Env:              getenv("APP_ENV", "local"),
		LogFormat:        getenv("LOG_FORMAT", "json"),
		HTTPAddr:         getenv("APP_HTTP_ADDR", ":8080"),
		BaseURL:          getenv("APP_BASE_URL", "http://localhost:8080"),
		RequestTimeout:   getDuration("APP_REQUEST_TIMEOUT", 30*time.Second),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseURLAdmin: os.Getenv("DATABASE_URL_ADMIN"),
		AccessTokenTTL:     getDuration("ACCESS_TOKEN_TTL", 10*time.Minute),
		RefreshTokenTTL:    getDuration("REFRESH_TOKEN_TTL", 720*time.Hour),
		MFATokenTTL:        getDuration("MFA_TOKEN_TTL", 5*time.Minute),
		LoginMaxAttempts:   getInt("LOGIN_MAX_ATTEMPTS", 5),
		LoginAttemptWindow: getDuration("LOGIN_ATTEMPT_WINDOW", 15*time.Minute),
		MembershipCacheTTL: getDuration("MEMBERSHIP_CACHE_TTL", time.Minute),
		JWTPrivateKey:      os.Getenv("JWT_PRIVATE_KEY"),
		StorageDriver:    getenv("STORAGE_DRIVER", "minio"),
		MailDriver:       getenv("MAIL_DRIVER", "mailpit"),
		PSPDriver:        getenv("PSP_DRIVER", "mock"),
		S3Endpoint:       getenv("S3_ENDPOINT", "http://localhost:9000"),
		S3Bucket:         getenv("S3_BUCKET", "consorcio-docs"),
		S3AccessKey:      getenv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getenv("S3_SECRET_KEY", ""),
		S3Region:         getenv("S3_REGION", "us-east-1"),
		S3UseSSL:         os.Getenv("S3_USE_SSL") == "true",
		S3SignedTTL:      getDuration("S3_SIGNED_URL_TTL", 5*time.Minute),
		ScanDriver:       getenv("SCAN_DRIVER", "mock"),
		MaxUploadBytes:   getInt64("MAX_UPLOAD_BYTES", 10<<20),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate aplica las reglas mínimas de arranque. En production se prohíben
// adaptadores simulados ([ADR-0008], docs/product-decisions.md).
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL es obligatoria")
	}
	if c.Env == "production" {
		for _, d := range []struct{ name, val string }{
			{"STORAGE_DRIVER", c.StorageDriver},
			{"MAIL_DRIVER", c.MailDriver},
			{"PSP_DRIVER", c.PSPDriver},
			{"SCAN_DRIVER", c.ScanDriver},
		} {
			if d.val == "mock" || d.val == "mailpit" || strings.Contains(d.val, "mock") {
				return fmt.Errorf("%s=%q está prohibido en production", d.name, d.val)
			}
		}
	}
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}
