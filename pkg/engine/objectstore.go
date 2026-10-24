package engine

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
// S3-compatible endpoint, including Cloudflare R2.
type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64) error
	Get(ctx context.Context, key string) ([]byte, error)
}

type MinioObjectStore struct {
	client *minio.Client
	bucket string
}

// ObjectStoreConfig describes an S3-compatible endpoint. PathStyle selects
// path-style addressing (MinIO, rustfs, local endpoints); the default is
// virtual-hosted-style, which R2 and AWS S3 support.
type ObjectStoreConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
	PathStyle bool
}

func NewMinioObjectStore(cfg ObjectStoreConfig) (*MinioObjectStore, error) {
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	lookup := minio.BucketLookupDNS
	if cfg.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &MinioObjectStore{client: client, bucket: cfg.Bucket}, nil
}

// PresignPut generates a time-limited upload URL for direct client-to-storage
// transfers, used by the API layer for media uploads.
func (m *MinioObjectStore) PresignPut(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	return m.client.PresignedPutObject(ctx, m.bucket, key, expiry)
}

// EnsureBucket creates the archive bucket when missing.
func (m *MinioObjectStore) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", m.bucket, err)
	}
	if exists {
		return nil
	}
	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", m.bucket, err)
	}
	return nil
}

func (m *MinioObjectStore) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	if _, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (m *MinioObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	object, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
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
