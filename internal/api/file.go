package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/render"
)

type FileAPI struct {
	client          *minio.Client
	bucket          string
	publicURLPrefix string
}

func NewFileAPI(cfg config.StorageConfig) (*FileAPI, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	return &FileAPI{
		client:          client,
		bucket:          cfg.Bucket,
		publicURLPrefix: cfg.PublicURLPrefix,
	}, nil
}

type PresignUploadReq struct {
	FileName    string `json:"file_name"    validate:"required"`
	ContentType string `json:"content_type"`
}

type PresignUploadResp struct {
	UploadURL   string `json:"upload_url"`
	DownloadURL string `json:"download_url"`
	FileKey     string `json:"file_key"`
}

func (a *FileAPI) PresignUpload(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[PresignUploadReq](w, r)
	if err != nil {
		return
	}

	ext := strings.ToLower(filepath.Ext(req.FileName))
	fileKey := fmt.Sprintf("uploads/%s%s", uuid.NewV7(), ext)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	expiry := 15 * time.Minute
	presignedURL, err := a.client.PresignedPutObject(ctx, a.bucket, fileKey, expiry)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, "generate presigned url failed: "+err.Error())
		return
	}

	downloadURL := presignedURL.String()
	if a.publicURLPrefix != "" {
		downloadURL = fmt.Sprintf("%s/%s/%s", strings.TrimRight(a.publicURLPrefix, "/"), a.bucket, fileKey)
	}

	render.Success(w, "获取上传预签名链接成功", PresignUploadResp{
		UploadURL:   presignedURL.String(),
		DownloadURL: downloadURL,
		FileKey:     fileKey,
	})
}
