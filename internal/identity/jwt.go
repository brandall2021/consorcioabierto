package identity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims son las declaraciones del access token (JWT RS256).
type Claims struct {
	Issuer     string   `json:"iss"`
	Subject    string   `json:"sub"`
	IssuedAt   int64    `json:"iat"`
	ExpiresAt  int64    `json:"exp"`
	Purpose    string   `json:"purpose,omitempty"`
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	Membership string   `json:"membership_id,omitempty"`
	Tenant     string   `json:"tenant_id,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	ScopeType  string   `json:"scope_type,omitempty"`
	ScopeID    string   `json:"scope_id,omitempty"`
	SessionKey string   `json:"session_key,omitempty"`
}

func b64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// signJWT firma las claims con RS256 usando la clave privada RSA.
func signJWT(key *rsa.PrivateKey, claims Claims) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := b64url(header) + "." + b64url(payload)
	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + b64url(sig), nil
}

// parseAndVerifyJWT valida firma (RS256), exp y iss; devuelve las claims.
func parseAndVerifyJWT(pub *rsa.PublicKey, token, issuer string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("JWT mal formado")
	}
	unsigned := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("firma ilegible")
	}
	digest := sha256.Sum256([]byte(unsigned))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return nil, errors.New("firma inválida")
	}

	var claims Claims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("payload ilegible")
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New("claims inválidas")
	}

	now := time.Now().Unix()
	if claims.ExpiresAt <= now {
		return nil, errors.New("token expirado")
	}
	if issuer != "" && claims.Issuer != issuer {
		return nil, errors.New("issuer inválido")
	}
	return &claims, nil
}

// EncodePublicKeyPEM exporta la clave pública en formato PKIX PEM.
func EncodePublicKeyPEM(key *rsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----\n",
		base64.StdEncoding.EncodeToString(der))), nil
}

// ParseRSAPrivateKeyFromPEM decodifica una clave privada PEM PKCS8 a
// *rsa.PrivateKey.
func ParseRSAPrivateKeyFromPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("clave PEM privada inválida")
	}
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("clave PEM no es RSA privada")
	}
	return key, nil
}
