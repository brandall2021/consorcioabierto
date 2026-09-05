package documentos

import (
	"context"
	"io"
)

// Estados de antivirus del documento ([ADR-0008], §5.4).
const (
	AntivirusPendiente  = "pendiente"
	AntivirusLimpio     = "limpio"
	AntivirusCuarentena = "en_cuarentena"
)

// ScanResult es el resultado del escaneo antivirus.
type ScanResult struct {
	Limpio bool
	Mime   string
}

// Scanner es el puerto de antivirus. En dev/test el adaptador mock escanea por
// contenido de forma determinista; en staging/prod el adaptador none lo
// deshabilita (feature flag, §7).
type Scanner interface {
	Scan(ctx context.Context, r io.Reader) (ScanResult, error)
}