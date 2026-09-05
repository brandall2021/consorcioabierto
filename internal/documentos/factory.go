package documentos

import (
	"fmt"
	"time"
)

// NewStorage construye el adaptador de storage según STORAGE_DRIVER
// ([ADR-0008]): "minio" (MinIO local + S3 en prod, ambos S3-compatible) y
// "memory" (solo tests/entornos efímeros).
func NewStorage(driver, endpoint, accessKey, secretKey, region, bucket string, useSSL bool) (Storage, error) {
	switch driver {
	case "minio":
		if endpoint == "" {
			return nil, fmt.Errorf("NewStorage: S3_ENDPOINT requerido para driver %q", driver)
		}
		return NewMinioStorage(endpoint, accessKey, secretKey, region, bucket, useSSL)
	case "memory":
		return NewMemoryStorage(), nil
	default:
		return nil, fmt.Errorf("NewStorage: driver %q desconocido", driver)
	}
}

// NewScanner construye el adaptador de antivirus según SCAN_DRIVER:
// "mock" (dev/test, determinista) o "none" (no-op, staging/production).
func NewScanner(driver string) (Scanner, error) {
	switch driver {
	case "mock":
		return NewMockScanner(), nil
	case "none":
		return NewNoopScanner(), nil
	default:
		return nil, fmt.Errorf("NewScanner: driver %q desconocido", driver)
	}
}

// DocsEnvFromConfig arma las dependencias ambientales del subsistema de
// documentos desde la configuración del proceso.
func DocsEnvFromConfig(
	storageDriver, scanDriver, endpoint, accessKey, secretKey, region, bucket string,
	useSSL bool,
	signedTTL time.Duration,
	maxUpload int64,
) (DocsEnv, error) {
	storage, err := NewStorage(storageDriver, endpoint, accessKey, secretKey, region, bucket, useSSL)
	if err != nil {
		return DocsEnv{}, err
	}
	scanner, err := NewScanner(scanDriver)
	if err != nil {
		return DocsEnv{}, err
	}
	return DocsEnv{
		Storage:   storage,
		Scanner:   scanner,
		MaxUpload: maxUpload,
		SignedTTL: signedTTL,
	}, nil
}