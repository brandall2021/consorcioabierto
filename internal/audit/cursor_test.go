package audit

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	c := &Cursor{CreatedAt: time.Date(2026, 8, 12, 15, 30, 0, 0, time.UTC), ID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc"}
	got, err := DecodeCursor(EncodeCursor(c))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.CreatedAt.Equal(c.CreatedAt) {
		t.Errorf("created_at = %v, se esperaba %v", got.CreatedAt, c.CreatedAt)
	}
	if got.ID != c.ID {
		t.Errorf("id = %q, se esperaba %q", got.ID, c.ID)
	}
}

func TestDecodeCursorVacio(t *testing.T) {
	c, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("decode vacío no debería fallar: %v", err)
	}
	if c != nil {
		t.Fatalf("decode vacío debería devolver nil, fue %+v", c)
	}
}

func TestDecodeCursorInvalido(t *testing.T) {
	for _, s := range []string{"!!!no-base64!!!", "AAAA", "bm90LWpzb24"} {
		if _, err := DecodeCursor(s); err == nil {
			t.Errorf("cursor %q debería fallar", s)
		}
	}
}
