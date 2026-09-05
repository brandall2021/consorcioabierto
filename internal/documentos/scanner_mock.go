package documentos

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// eicarSignature es la firma EICAR estándar usada por el mock para marcar un
// archivo como infectado de forma determinista.
const eicarSignature = "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"

var _ Scanner = (*mockScanner)(nil)

// mockScanner es el adaptador de antivirus determinista para dev/test: detecta
// la firma EICAR y el MIME por contenido ([ADR-0008]; §7 mocks solo dev/test).
type mockScanner struct{}

// NewMockScanner construye el escáner mock.
func NewMockScanner() *mockScanner { return &mockScanner{} }

func (s *mockScanner) Scan(_ context.Context, r io.Reader) (ScanResult, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return ScanResult{}, err
	}
	if bytes.Contains(content, []byte(eicarSignature)) {
		return ScanResult{Limpio: false}, nil
	}
	return ScanResult{Limpio: true, Mime: http.DetectContentType(content)}, nil
}