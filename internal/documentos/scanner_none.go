package documentos

import (
	"context"
	"io"
)

var _ Scanner = (*noopScanner)(nil)

// noopScanner es el adaptador de antivirus para staging/production: no-op,
// deshabilitado según §7 ("mock deshabilitado"). El MIME no se detecta;
// queda a cargo de un escáner real futuro habilitado por feature flag.
type noopScanner struct{}

// NewNoopScanner construye el escáner none (sin-op).
func NewNoopScanner() *noopScanner { return &noopScanner{} }

func (s *noopScanner) Scan(_ context.Context, _ io.Reader) (ScanResult, error) {
	return ScanResult{Limpio: true}, nil
}