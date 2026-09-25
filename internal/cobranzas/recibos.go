package cobranzas

import (
	"fmt"
	"strings"
)

// GenerateReciboPDF genera un PDF mínimo con los datos principales de la cobranza.
func GenerateReciboPDF(c CobranzaDTO, asignaciones []AllocationDTO) ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	lines := []string{
		"BT",
		"/F1 18 Tf",
		"72 720 Td",
		"(Recibo de Cobranza) Tj",
		"/F1 12 Tf",
		fmt.Sprintf("0 -30 Td (Cobranza: %s) Tj", escapePDF(c.ID)),
		fmt.Sprintf("0 -20 Td (Unidad: %s) Tj", escapePDF(c.UnidadID)),
		fmt.Sprintf("0 -20 Td (Importe: %d) Tj", c.Importe.AmountCents),
		fmt.Sprintf("0 -20 Td (Saldo a favor: %d) Tj", c.SaldoAFavorCents),
		fmt.Sprintf("0 -20 Td (Referencia: %s) Tj", escapePDF(stringOrDash(c.Referencia))),
		fmt.Sprintf("0 -20 Td (Asignaciones: %d) Tj", len(asignaciones)),
		"ET",
	}
	stream := strings.Join(lines, "\n")
	fmt.Fprintf(&buf, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(stream), stream)
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 0; i < 5; i++ {
		buf.WriteString("0000000000 00000 n \n")
	}
	buf.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n0\n%%EOF\n")
	return []byte(buf.String()), nil
}

func stringOrDash(s *string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return "-"
	}
	return *s
}

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
