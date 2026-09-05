package documentos

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioStorage es el adaptador S3-compatible ([ADR-0008]) que usa el SDK de
// MinIO, funcionando tanto contra MinIO local como contra S3 en producción
// (ambos S3-compatibles).
type minioStorage struct {
	client *minio.Client
	bucket string
}

// NewMinioStorage construye el adaptador y asegura que el bucket exista.
func NewMinioStorage(endpoint, accessKey, secretKey, region, bucket string, useSSL bool) (*minioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("crear cliente storage: %w", err)
	}
	s := &minioStorage{client: client, bucket: bucket}
	if err := s.ensureBucket(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *minioStorage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("verificar bucket %q: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("crear bucket %q: %w", s.bucket, err)
	}
	return nil
}

func (s *minioStorage) PresignPut(ctx context.Context, key string, size int64, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("%w: presign put: %v", ErrStorageUnavailable, err)
	}
	return u.String(), nil
}

func (s *minioStorage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("%w: presign get: %v", ErrStorageUnavailable, err)
	}
	return u.String(), nil
}

func (s *minioStorage) Stat(ctx context.Context, key string) (int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (s *minioStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}