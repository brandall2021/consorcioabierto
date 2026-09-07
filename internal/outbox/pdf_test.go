package outbox

import (
	"strings"
	"testing"
)

func TestSimplePDFGenerator(t *testing.T) {
	gen := &SimplePDFGenerator{}
	pdf, err := gen.Generate(LiquidacionPayload{
		LiquidacionID: "test-123",
		Periodo:       "202609",
		ConsorcioID:   "cons-abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(pdf)
	if !strings.HasPrefix(s, "%PDF-1.4") {
		t.Error("missing PDF header")
	}
	if !strings.Contains(s, "%%EOF") {
		t.Error("missing PDF EOF")
	}
	if !strings.Contains(s, "202609") {
		t.Error("missing periodo in PDF")
	}
	if !strings.Contains(s, "test-123") {
		t.Error("missing liquidacion_id in PDF")
	}
	if len(pdf) < 100 {
		t.Error("PDF too small, likely invalid")
	}
}
