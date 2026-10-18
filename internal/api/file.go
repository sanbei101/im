package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

type FileAPI struct {
	objects       *store.MinioObjectStore
	publicBaseURL string
}

func NewFileAPI(objects *store.MinioObjectStore, publicBaseURL string) *FileAPI {
	return &FileAPI{objects: objects, publicBaseURL: publicBaseURL}
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

	uploadURL, err := a.objects.PresignPut(ctx, fileKey, 15*time.Minute)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, "generate upload url failed: "+err.Error())
		return
	}
	downloadURL := strings.TrimRight(a.publicBaseURL, "/") + "/" + fileKey

	render.Success(w, "获取上传预签名链接成功", PresignUploadResp{
		UploadURL:   uploadURL.String(),
		DownloadURL: downloadURL,
		FileKey:     fileKey,
	})
}
