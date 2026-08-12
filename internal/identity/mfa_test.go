package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/brandall2021/consorcioabierto/internal/config"
	"github.com/pquerna/otp/totp"
)

func TestTOTPValidateVectoresRFC6238(t *testing.T) {
	// Vectores de prueba del RFC 6238 (SHA1, 8 dígitos) re-validados con la
	// librería: 8 dígitos para los vectores, 6 para producción.
	cases := []struct {
		secret string
		time   int64
	}{
		{"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", 59},
		{"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", 1111111109},
		{"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", 1111111111},
	}
	for _, c := range cases {
		code, err := totp.GenerateCodeCustom(c.secret, time.Unix(c.time, 0), totp.ValidateOpts{Period: 30, Digits: 8})
		if err != nil {
			t.Fatalf("GenerateCodeCustom: %v", err)
		}
		if ok, _ := totp.ValidateCustom(code, c.secret, time.Unix(c.time, 0), totp.ValidateOpts{Period: 30, Digits: 8}); !ok {
			t.Fatalf("código %s no válido para t=%d", code, c.time)
		}
	}
}

func TestTOTPCodeValidoEnVentana(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "iss", AccountName: "a@b.com", SecretSize: 20})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}
	secret := key.Secret()
	now := time.Now()

	code, err := totp.GenerateCode(secret, now)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if !totp.Validate(code, secret) {
		t.Fatal("código actual debería validar")
	}
	if totp.Validate("000000", secret) {
		t.Fatal("código incorrecto no debería validar")
	}
}

func TestSignMfaToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	am := &AuthManager{
		cfg: &config.Config{BaseURL: "https://api.consorcioabierto.local", MFATokenTTL: 5 * time.Minute},
		key: key,
		now: time.Now,
	}

	token, err := am.signMfaToken("11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("signMfaToken: %v", err)
	}
	claims, err := parseAndVerifyJWT(am.PublicKey(), token, am.cfg.BaseURL)
	if err != nil {
		t.Fatalf("parseAndVerifyJWT: %v", err)
	}
	if claims.Purpose != mfaTokenPurpose {
		t.Fatalf("purpose = %q, se esperaba %q", claims.Purpose, mfaTokenPurpose)
	}
	if claims.Subject != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("subject = %q", claims.Subject)
	}
}

func TestVerifyMfaRechazaAccessToken(t *testing.T) {
	// Un access token normal (sin purpose mfa) no debe poder usarse como mfa_token.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	am := &AuthManager{
		cfg: &config.Config{BaseURL: "https://api.consorcioabierto.local", MFATokenTTL: 5 * time.Minute},
		key: key,
		now: time.Now,
	}

	now := am.now()
	access, err := signJWT(key, Claims{
		Issuer:    am.cfg.BaseURL,
		Subject:   "11111111-1111-4111-8111-111111111111",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}

	if _, err := am.VerifyMfa(t.Context(), access, "000000", "127.0.0.1"); !errors.Is(err, ErrInvalidMFAToken) {
		t.Fatalf("VerifyMfa con access token = %v, se esperaba ErrInvalidMFAToken", err)
	}
}

func TestVerifyMfaRechazaTokenExpirado(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	am := &AuthManager{
		cfg: &config.Config{BaseURL: "https://api.consorcioabierto.local", MFATokenTTL: -time.Minute},
		key: key,
		now: time.Now,
	}
	token, err := am.signMfaToken("11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("signMfaToken: %v", err)
	}
	if _, err := am.VerifyMfa(t.Context(), token, "000000", "127.0.0.1"); !errors.Is(err, ErrInvalidMFAToken) {
		t.Fatalf("VerifyMfa con token expirado = %v, se esperaba ErrInvalidMFAToken", err)
	}
}
