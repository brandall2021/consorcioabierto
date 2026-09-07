package outbox

import (
	"fmt"
	"strings"
)

// SimplePDFGenerator genera PDFs mínimos sin dependencias externas.
type SimplePDFGenerator struct{}

var _ PDFGenerator = (*SimplePDFGenerator)(nil)

func (g *SimplePDFGenerator) Generate(p LiquidacionPayload) ([]byte, error) {
	var buf strings.Builder
	objects := make([]int64, 0)
	offsets := make([]int64, 0)

	// Header
	buf.WriteString("%PDF-1.4\n")
	headerOffset := int64(buf.Len())
	_ = headerOffset

	// Obj 1: Catalog
	offsets = append(offsets, int64(buf.Len()))
	objects = append(objects, 1)
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Obj 2: Pages
	offsets = append(offsets, int64(buf.Len()))
	objects = append(objects, 2)
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Obj 3: Page
	offsets = append(offsets, int64(buf.Len()))
	objects = append(objects, 3)
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// Obj 4: Content stream
	lines := []string{
		"BT",
		"/F1 18 Tf",
		"72 720 Td",
		"(Liquidacion de Expensas) Tj",
		"/F1 12 Tf",
		fmt.Sprintf("0 -30 Td (Periodo: %s) Tj", escapePDF(p.Periodo)),
		fmt.Sprintf("0 -20 Td (Liquidacion ID: %s) Tj", escapePDF(p.LiquidacionID)),
		fmt.Sprintf("0 -20 Td (Consorcio ID: %s) Tj", escapePDF(p.ConsorcioID)),
		"0 -40 Td (Documento generado por ConsorcioAbierto) Tj",
		"ET",
	}
	stream := strings.Join(lines, "\n")
	offsets = append(offsets, int64(buf.Len()))
	objects = append(objects, 4)
	fmt.Fprintf(&buf, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(stream), stream)

	// Obj 5: Font
	offsets = append(offsets, int64(buf.Len()))
	objects = append(objects, 5)
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// xref
	xrefOffset := int64(buf.Len())
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	// trailer
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\n", len(objects)+1)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return []byte(buf.String()), nil
}

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
