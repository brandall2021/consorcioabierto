package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func pemEncode(typ string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der})
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	return key
}

func TestSignAndVerifyJWT(t *testing.T) {
	key := testRSAKey(t)
	now := time.Now()
	claims := Claims{
		Issuer:     "https://api.consorcioabierto.local",
		Subject:    "11111111-1111-4111-8111-111111111111",
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(10 * time.Minute).Unix(),
		Email:      "a@b.com",
		Membership: "22222222-2222-4222-8222-222222222222",
		Tenant:     "33333333-3333-4333-8333-333333333333",
		Roles:      []string{"tenant_admin"},
	}

	token, err := signJWT(key, claims)
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}

	got, err := parseAndVerifyJWT(&key.PublicKey, token, claims.Issuer)
	if err != nil {
		t.Fatalf("parseAndVerifyJWT: %v", err)
	}
	if got.Subject != claims.Subject || got.Email != claims.Email || got.Membership != claims.Membership {
		t.Fatalf("claims no coinciden: %+v", got)
	}
	if len(got.Roles) != 1 || got.Roles[0] != "tenant_admin" {
		t.Fatalf("roles no coinciden: %v", got.Roles)
	}
}

func TestVerifyJWTRechazaFirmaAlterada(t *testing.T) {
	key := testRSAKey(t)
	other := testRSAKey(t)
	now := time.Now()
	claims := Claims{
		Issuer:    "iss",
		Subject:   "sub",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Minute).Unix(),
	}
	token, err := signJWT(key, claims)
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}
	if _, err := parseAndVerifyJWT(&other.PublicKey, token, "iss"); err == nil {
		t.Fatal("firma con otra clave debería fallar")
	}
}

func TestVerifyJWTRechazaExpirado(t *testing.T) {
	key := testRSAKey(t)
	now := time.Now()
	claims := Claims{
		Issuer:    "iss",
		Subject:   "sub",
		IssuedAt:  now.Add(-2 * time.Minute).Unix(),
		ExpiresAt: now.Add(-1 * time.Minute).Unix(),
	}
	token, _ := signJWT(key, claims)
	if _, err := parseAndVerifyJWT(&key.PublicKey, token, "iss"); err == nil {
		t.Fatal("token expirado debería fallar")
	}
}

func TestVerifyJWTRechazaIssuerDistinto(t *testing.T) {
	key := testRSAKey(t)
	now := time.Now()
	claims := Claims{
		Issuer:    "iss-a",
		Subject:   "sub",
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Minute).Unix(),
	}
	token, _ := signJWT(key, claims)
	if _, err := parseAndVerifyJWT(&key.PublicKey, token, "iss-b"); err == nil {
		t.Fatal("issuer distinto debería fallar")
	}
}

func TestParseRSAPrivateKeyFromPEM(t *testing.T) {
	key := testRSAKey(t)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey: %v", err)
	}
	pemBytes := pemEncode("PRIVATE KEY", der)
	parsed, err := ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyFromPEM: %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 {
		t.Fatal("claves no coinciden")
	}
}

func TestParseRSAPrivateKeyFromPEMRechazaBasura(t *testing.T) {
	if _, err := ParseRSAPrivateKeyFromPEM([]byte("not-a-key")); err == nil {
		t.Fatal("basura no debería parsear")
	}
}
