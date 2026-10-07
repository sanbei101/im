package store

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore is the remote archive destination. MinIO covers any
// S3-compatible endpoint; tests back it with a local directory.
type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64) error
	Get(ctx context.Context, key string) ([]byte, error)
}

type MinioObjectStore struct {
	client *minio.Client
	bucket string
}

func NewMinioObjectStore(
	endpoint, bucket, accessKeyID, secretAccessKey string,
	useSSL bool,
) (*MinioObjectStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
		// A fixed region keeps PresignPut purely local signing; without it
		// minio-go queries the bucket location over the network first.
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &MinioObjectStore{client: client, bucket: bucket}, nil
}

// Bucket exposes the configured bucket name for URL composition outside the store.
func (s *MinioObjectStore) Bucket() string {
	return s.bucket
}

// PresignPut generates a time-limited upload URL for direct client-to-storage
// transfers, used by the API layer for media uploads.
func (s *MinioObjectStore) PresignPut(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	return s.client.PresignedPutObject(ctx, s.bucket, key, expiry)
}

// EnsureBucket creates the archive bucket when missing.
func (s *MinioObjectStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", s.bucket, err)
	}
	return nil
}

func (s *MinioObjectStore) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	if _, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *MinioObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer object.Close()
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}
	return data, nil
}
