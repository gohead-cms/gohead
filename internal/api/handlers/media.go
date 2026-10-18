package handlers

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gohead-cms/gohead/internal/models"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/media"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
)

// MediaPolicy limits uploads; set at startup from the media configuration.
var MediaPolicy = storage.UploadPolicy{MaxSize: 50 << 20}

// maxFilesPerUpload bounds one multipart request.
const maxFilesPerUpload = 20

// multipartMemory is how much of an upload is kept in memory before
// spilling to a temporary file.
const multipartMemory = 8 << 20

func mediaUnavailable(c *gin.Context) bool {
	if storage.MediaStore() != nil {
		return false
	}
	c.Set("response", "Media uploads are disabled on this server")
	c.Set("status", http.StatusServiceUnavailable)
	return true
}

func mediaJSON(c *gin.Context, m models.Media) map[string]any {
	return models.MediaResponse(m, storage.MediaURL(c.Request.Context(), m))
}

// UploadMedia stores the files of a multipart/form-data request (fields
// "files" or "file"; optional "alt" and "caption"). Requires the create
// permission. All files are stored, or none.
func UploadMedia(c *gin.Context) {
	if !hasPermission(c.GetString("role"), "create") {
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	if mediaUnavailable(c) {
		return
	}
	if ct, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type")); ct != "multipart/form-data" {
		c.Set("response", "Send files as multipart/form-data in a field named 'files'")
		c.Set("status", http.StatusBadRequest)
		return
	}
	if MediaPolicy.MaxSize > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MediaPolicy.MaxSize*maxFilesPerUpload+(1<<20))
	}
	if err := c.Request.ParseMultipartForm(multipartMemory); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			c.Set("response", "Upload too large")
			c.Set("status", http.StatusRequestEntityTooLarge)
			return
		}
		c.Set("response", "Invalid multipart form: "+err.Error())
		c.Set("status", http.StatusBadRequest)
		return
	}
	defer func() { _ = c.Request.MultipartForm.RemoveAll() }()

	form := c.Request.MultipartForm
	headers := append(append([]*multipart.FileHeader{}, form.File["files"]...), form.File["file"]...)
	if len(headers) == 0 {
		c.Set("response", "No file found: send them in a field named 'files'")
		c.Set("status", http.StatusBadRequest)
		return
	}
	if len(headers) > maxFilesPerUpload {
		c.Set("response", fmt.Sprintf("Too many files (max %d per request)", maxFilesPerUpload))
		c.Set("status", http.StatusBadRequest)
		return
	}

	saved := make([]*models.Media, 0, len(headers))
	rollback := func() {
		// The request may have been cancelled (client gone): clean up anyway.
		ctx := context.WithoutCancel(c.Request.Context())
		for _, m := range saved {
			if err := storage.DeleteMedia(ctx, m.ID); err != nil {
				logger.Log.WithError(err).WithField("media_id", m.ID).Warn("Failed to roll back upload")
			}
		}
	}
	for _, fh := range headers {
		m, err := saveUploadedFile(c, fh, c.PostForm("alt"), c.PostForm("caption"))
		if err != nil {
			rollback()
			var merr *storage.MediaError
			if errors.As(err, &merr) {
				c.Set("response", merr.Msg)
				c.Set("status", merr.Status)
				return
			}
			logger.Log.WithError(err).Error("Upload failed")
			c.Set("response", "Failed to store the upload")
			c.Set("status", http.StatusInternalServerError)
			return
		}
		saved = append(saved, m)
	}

	out := make([]map[string]any, 0, len(saved))
	for _, m := range saved {
		out = append(out, mediaJSON(c, *m))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"count": len(out)})
	c.Set("status", http.StatusCreated)
}

func saveUploadedFile(c *gin.Context, fh *multipart.FileHeader, alt, caption string) (*models.Media, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return storage.SaveMedia(c.Request.Context(), storage.UploadFile{
		Name: fh.Filename, Size: fh.Size, Content: f,
		Alt: alt, Caption: caption, User: c.GetString("username"),
	}, MediaPolicy)
}

// ListMedia lists the media library: page, pageSize, mime (type prefix such
// as "image/") and search (file name). Browsing the library needs the
// create permission: it includes files only drafts use, so readers only
// reach files through the content that references them.
func ListMedia(c *gin.Context) {
	if !hasPermission(c.GetString("role"), "create") {
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	q := storage.MediaQuery{Page: 1, PageSize: 25, MimePrefix: c.Query("mime"), Search: c.Query("search")}
	if v := c.Query("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			validationError(c, "invalid page")
			return
		}
		q.Page = p
	}
	if v := c.Query("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			validationError(c, "pageSize must be between 1 and 100")
			return
		}
		q.PageSize = ps
	}
	list, total, err := storage.ListMedia(q)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to list media")
		c.Set("response", "Failed to list media")
		c.Set("status", http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, mediaJSON(c, m))
	}
	c.Set("response", out)
	c.Set("meta", gin.H{"pagination": gin.H{
		"page": q.Page, "pageSize": q.PageSize, "total": total,
		"pageCount": (total + q.PageSize - 1) / q.PageSize,
	}})
	c.Set("status", http.StatusOK)
}

func mediaID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		validationError(c, "Invalid media ID")
		return 0, false
	}
	return uint(id), true
}

func mediaLookupError(c *gin.Context, err error) {
	if errors.Is(err, storage.ErrMediaNotFound) {
		c.Set("response", "Media not found")
		c.Set("status", http.StatusNotFound)
		return
	}
	logger.Log.WithError(err).Error("Media operation failed")
	c.Set("response", "Media operation failed")
	c.Set("status", http.StatusInternalServerError)
}

// GetMedia returns one media file. Requires the create permission, like
// ListMedia.
func GetMedia(c *gin.Context) {
	if !hasPermission(c.GetString("role"), "create") {
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	id, ok := mediaID(c)
	if !ok {
		return
	}
	m, err := storage.GetMedia(id)
	if err != nil {
		mediaLookupError(c, err)
		return
	}
	c.Set("response", mediaJSON(c, *m))
	c.Set("status", http.StatusOK)
}

// UpdateMedia changes a file's name, alt text or caption:
// {"data": {"alt": "…"}}. Requires the update permission.
func UpdateMedia(c *gin.Context) {
	if !hasPermission(c.GetString("role"), "update") {
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	id, ok := mediaID(c)
	if !ok {
		return
	}
	var body struct {
		Data *struct {
			Name    *string `json:"name"`
			Alt     *string `json:"alt"`
			Caption *string `json:"caption"`
		} `json:"data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Data == nil {
		validationError(c, "Invalid request body. Expecting {\"data\": {\"name\"|\"alt\"|\"caption\": …}}")
		return
	}
	m, err := storage.UpdateMedia(id, storage.MediaInfo{Name: body.Data.Name, Alt: body.Data.Alt, Caption: body.Data.Caption})
	if err != nil {
		mediaLookupError(c, err)
		return
	}
	c.Set("response", mediaJSON(c, *m))
	c.Set("status", http.StatusOK)
}

// DeleteMedia removes a file and its content. Requires the delete permission.
func DeleteMedia(c *gin.Context) {
	if !hasPermission(c.GetString("role"), "delete") {
		c.Set("response", "Access denied")
		c.Set("status", http.StatusForbidden)
		return
	}
	if mediaUnavailable(c) {
		return // without a store the file itself could not be removed
	}
	id, ok := mediaID(c)
	if !ok {
		return
	}
	if err := storage.DeleteMedia(c.Request.Context(), id); err != nil {
		mediaLookupError(c, err)
		return
	}
	c.Set("response", nil)
	c.Set("status", http.StatusOK)
}

// ServeLocalMedia serves files of the local driver under /uploads/<key>.
// Only files recorded in the media library are served, with their detected
// type, and with headers that stop browsers from running anything in them.
func ServeLocalMedia(store *media.Local) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimPrefix(c.Param("key"), "/")
		path, err := store.Path(key)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		var m models.Media
		if err := database.DB.Where(&models.Media{Key: key}).First(&m).Error; err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", m.MimeType)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
		h.Set("Cache-Control", "public, max-age=31536000, immutable") // keys are never reused
		if !media.InlineSafe(m.MimeType) {
			h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": m.Name}))
		}
		// ServeContent handles Range requests (video seeking) and
		// conditional requests; the type is already set, so it does not sniff.
		http.ServeContent(c.Writer, c.Request, "", info.ModTime(), f)
	}
}
