package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
)

// UploadHandler exposes file uploads for training resources.
type UploadHandler struct{}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadStoredPath is the local directory that stores uploaded files
// and is served under /uploads.
const UploadStoredPath = "uploads"

// Upload saves an uploaded PDF and returns a public URL.
func (h *UploadHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		badRequest(c, "file is required")
		return
	}
	defer file.Close()

	// Only allow PDF uploads.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" {
		fail(c, http.StatusBadRequest, "BAD_EXTENSION", "Only PDF files are allowed", nil)
		return
	}

	if err := os.MkdirAll(UploadStoredPath, 0o755); err != nil {
		internal(c)
		return
	}

	// Unique filename to avoid collisions and path traversal.
	name := uuid.NewString() + ".pdf"
	dst := filepath.Join(UploadStoredPath, name)
	out, err := os.Create(dst)
	if err != nil {
		internal(c)
		return
	}
	defer out.Close()
	if _, err := out.ReadFrom(file); err != nil {
		internal(c)
		return
	}

	ok(c, gin.H{
		"url":       fmt.Sprintf("/uploads/%s", name),
		"file_name": header.Filename,
	})
}

// RegisterRoutes wires the upload and static file endpoints.
func (h *UploadHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	// Manager and admin can upload; employees cannot.
	r.POST("/uploads", auth, role(model.RoleManager, model.RoleAdmin), h.Upload)
}

// RegisterStatic mounts the uploads directory for public file access.
func RegisterStatic(r *gin.Engine) {
	if _, err := os.Stat(UploadStoredPath); os.IsNotExist(err) {
		_ = os.MkdirAll(UploadStoredPath, 0o755)
	}
	r.Static("/uploads", UploadStoredPath)
}