package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/internal/storage"
	"github.com/go-chi/chi/v5"
)

const maxFileSize = 500 * 1024 * 1024 // 500MB

type FileHandler struct {
	fileRepo domain.FileRepository
	storage  *storage.LocalStorage
}

func NewFileHandler(fileRepo domain.FileRepository, store *storage.LocalStorage) *FileHandler {
	return &FileHandler{
		fileRepo: fileRepo,
		storage:  store,
	}
}

// Upload handles POST /v1/files (multipart/form-data).
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+1024) // small buffer for headers

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' form field")
		return
	}
	defer func() { _ = f.Close() }()

	filename := header.Filename
	if err := validateFilename(filename); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mimeType := detectMimeType(filename, header.Header.Get("Content-Type"))

	wsID := workspaceIDFromCtx(r)
	fileID := domain.NewFileID()

	storagePath, sizeBytes, err := h.storage.Store(wsID, fileID, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	now := time.Now().UTC()
	file := &domain.File{
		ID:           fileID,
		WorkspaceID:  wsID,
		Filename:     filename,
		MimeType:     mimeType,
		SizeBytes:    sizeBytes,
		StoragePath:  storagePath,
		Downloadable: false, // user uploads are not downloadable
		CreatedAt:    now,
		Type:         "file",
	}

	if err := h.fileRepo.Create(r.Context(), file); err != nil {
		_ = h.storage.Delete(storagePath)
		writeError(w, http.StatusInternalServerError, "failed to create file record")
		return
	}

	writeJSON(w, http.StatusOK, file)
}

// List handles GET /v1/files.
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	lp := parseListParams(r)
	lp.WorkspaceID = workspaceIDFromCtx(r)
	params := domain.FileListParams{ListParams: lp}

	files, nextPage, err := h.fileRepo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list files")
		return
	}
	if files == nil {
		files = []*domain.File{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.File]{
		Data:     files,
		NextPage: nextPage,
	})
}

// GetMetadata handles GET /v1/files/{file_id}.
func (h *FileHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "file_id")
	file, err := h.fileRepo.GetByID(r.Context(), id)
	if err != nil || file.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, file)
}

// Download handles GET /v1/files/{file_id}/content.
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "file_id")
	file, err := h.fileRepo.GetByID(r.Context(), id)
	if err != nil || file.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if !file.Downloadable {
		writeError(w, http.StatusForbidden, "uploaded files cannot be downloaded; only files created by skills or code execution are downloadable")
		return
	}

	reader, err := h.storage.Read(file.StoragePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Filename))
	_, _ = io.Copy(w, reader)
}

// Delete handles DELETE /v1/files/{file_id}.
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "file_id")
	file, err := h.fileRepo.GetByID(r.Context(), id)
	if err != nil || file.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if err := h.fileRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}

	_ = h.storage.Delete(file.StoragePath)

	writeJSON(w, http.StatusOK, domain.FileDeleteResponse{
		ID:   id,
		Type: "file_deleted",
	})
}

// UploadInternal handles file uploads from sandbox runtime (agent-generated).
// Sets Downloadable=true since these are tool/skill-generated files.
func (h *FileHandler) UploadInternal(w http.ResponseWriter, r *http.Request, wsID string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+1024)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' form field")
		return
	}
	defer func() { _ = f.Close() }()

	filename := header.Filename
	if err := validateFilename(filename); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mimeType := detectMimeType(filename, header.Header.Get("Content-Type"))
	fileID := domain.NewFileID()

	storagePath, sizeBytes, err := h.storage.Store(wsID, fileID, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store file")
		return
	}

	now := time.Now().UTC()
	file := &domain.File{
		ID:           fileID,
		WorkspaceID:  wsID,
		Filename:     filename,
		MimeType:     mimeType,
		SizeBytes:    sizeBytes,
		StoragePath:  storagePath,
		Downloadable: true, // agent-generated files are downloadable
		CreatedAt:    now,
		Type:         "file",
	}

	if err := h.fileRepo.Create(r.Context(), file); err != nil {
		_ = h.storage.Delete(storagePath)
		writeError(w, http.StatusInternalServerError, "failed to create file record")
		return
	}

	writeJSON(w, http.StatusOK, file)
}

var forbiddenChars = []rune{'<', '>', ':', '"', '|', '?', '*', '\\', '/'}

func validateFilename(name string) error {
	if len(name) == 0 || len(name) > 255 {
		return fmt.Errorf("filename must be 1-255 characters")
	}
	for _, c := range name {
		if c < 32 { // unicode 0-31
			return fmt.Errorf("filename contains forbidden characters")
		}
		for _, fc := range forbiddenChars {
			if c == fc {
				return fmt.Errorf("filename contains forbidden character: %c", c)
			}
		}
	}
	return nil
}

func detectMimeType(filename, headerType string) string {
	// Try to detect from extension first
	ext := filepath.Ext(filename)
	if ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
	}
	// Fallback to Content-Type header
	if headerType != "" && headerType != "application/octet-stream" {
		return headerType
	}
	// Common fallbacks
	extLower := strings.ToLower(ext)
	switch extLower {
	case ".pdf":
		return "application/pdf"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// defaultJSONRaw returns raw if non-empty, otherwise the fallback string as RawMessage.
func defaultJSONRaw(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}
