package audit

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// EncodeCursor serializa la clave de paginación keyset.
func EncodeCursor(c *Cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor deserializa la clave de paginación keyset.
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("cursor inválido")
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, errors.New("cursor inválido")
	}
	return &c, nil
}
