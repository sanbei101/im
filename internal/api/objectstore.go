package api

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

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

// MinioObjectStore serves the business-side object storage needs: presigned
// media uploads and durability shipping of the engine's exported archive
// files. The message engine itself never talks to it.
type MinioObjectStore struct {
	client *minio.Client
	bucket string
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

// EnsureBucket creates the bucket when missing.
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

// PutFile uploads one local file, used to mirror the engine's exported
// archive directory into durable storage.
func (m *MinioObjectStore) PutFile(ctx context.Context, key, path string) error {
	if _, err := m.client.FPutObject(ctx, m.bucket, key, path, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}
