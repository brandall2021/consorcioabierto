package documentos

import (
	"context"
	"errors"
	"io"
	"time"
)

// Errores de dominio del subsistema de documentos.
var (
	ErrDocumentoInvalid    = errors.New("documento inválido")
	ErrDocumentoNotFound   = errors.New("documento no encontrado")
	ErrDocumentoTooLarge   = errors.New("documento excede el tamaño máximo")
	ErrDocumentoCuarentena = errors.New("documento en cuarentena")
	ErrStorageUnavailable  = errors.New("storage no disponible")
)

// Storage es el puerto de almacenamiento S3-compatible ([ADR-0008]). El
// backend genera la storage key; el cliente nunca la envía. Los adaptadores
// reales son MinIO (local/test) y S3 (producción); memory es para pruebas.
type Storage interface {
	// PresignPut devuelve una URL firmada breve para subir el objeto.
	PresignPut(ctx context.Context, key string, size int64, ttl time.Duration) (string, error)
	// PresignGet devuelve una URL firmada breve para leer el objeto.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Stat verifica que el objeto exista y devuelve su tamaño.
	Stat(ctx context.Context, key string) (int64, error)
	// Get abre el objeto para lectura (usado por el antivirus).
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}