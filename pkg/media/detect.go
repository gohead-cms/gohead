package media

import (
	"image"
	_ "image/gif"  // register decoders for DecodeConfig
	_ "image/jpeg" //
	_ "image/png"  //
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp" //
)

// DefaultAllowedTypes are accepted when media.allowed_types is not set:
// common images, video, audio and documents. SVG, HTML and scripts are left
// out on purpose: served from your domain they can run scripts.
var DefaultAllowedTypes = []string{
	"image/jpeg", "image/png", "image/gif", "image/webp", "image/avif",
	"video/mp4", "video/webm", "video/quicktime",
	"audio/mpeg", "audio/ogg", "audio/wav", "audio/webm", "audio/aac", "audio/mp4",
	"application/pdf",
	"text/plain", "text/csv", "text/markdown",
	"application/zip", "application/json",
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.ms-excel",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.ms-powerpoint",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.oasis.opendocument.text",
	"application/vnd.oasis.opendocument.spreadsheet",
}

// extTypes completes mime.TypeByExtension, which depends on the system's
// mime.types (absent from minimal container images).
var extTypes = map[string]string{
	".avif": "image/avif", ".webp": "image/webp", ".svg": "image/svg+xml",
	".mp4": "video/mp4", ".m4v": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
	".mp3": "audio/mpeg", ".ogg": "audio/ogg", ".oga": "audio/ogg", ".wav": "audio/wav",
	".aac": "audio/aac", ".m4a": "audio/mp4",
	".pdf": "application/pdf", ".txt": "text/plain", ".csv": "text/csv", ".md": "text/markdown",
	".json": "application/json", ".zip": "application/zip",
	".doc": "application/msword", ".xls": "application/vnd.ms-excel", ".ppt": "application/vnd.ms-powerpoint",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".html": "text/html", ".htm": "text/html", ".js": "text/javascript", ".xml": "application/xml",
}

// genericSniffs are content types too vague to decide on; the file
// extension then refines them (a .docx sniffs as a zip, a .csv as text).
var genericSniffs = map[string]bool{
	"application/octet-stream": true,
	"application/zip":          true,
	"text/plain":               true,
	"text/xml":                 true,
	"application/xml":          true,
}

// DetectType returns the MIME type of a file from its first bytes, using the
// file name only when the content alone is ambiguous. The client's declared
// Content-Type is never trusted.
func DetectType(head []byte, fileName string) string {
	sniffed := baseType(http.DetectContentType(head))
	if !genericSniffs[sniffed] {
		return sniffed
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	byExt := extTypes[ext]
	if byExt == "" {
		byExt = baseType(mime.TypeByExtension(ext))
	}
	if byExt == "" {
		return sniffed
	}
	// Binary content never becomes a text type because of its name, and
	// text never becomes a binary format.
	textual := strings.HasPrefix(sniffed, "text/") || strings.HasSuffix(sniffed, "/xml")
	byExtTextual := strings.HasPrefix(byExt, "text/") || strings.HasSuffix(byExt, "+xml") ||
		strings.HasSuffix(byExt, "/xml") || byExt == "application/json"
	if textual != byExtTextual {
		return sniffed
	}
	return byExt
}

func baseType(t string) string {
	t, _, _ = strings.Cut(t, ";")
	return strings.TrimSpace(strings.ToLower(t))
}

// Allowed reports whether mimeType matches one of the patterns, which are
// exact types ("image/png"), wildcards ("image/*") or "*/*".
func Allowed(mimeType string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		switch {
		case p == "*/*" || p == "*":
			return true
		case strings.HasSuffix(p, "/*"):
			if strings.HasPrefix(mimeType, strings.TrimSuffix(p, "*")) {
				return true
			}
		case p == mimeType:
			return true
		}
	}
	return false
}

// InlineSafe reports whether browsers may display a type inline. Everything
// else is served as an attachment.
func InlineSafe(mimeType string) bool {
	switch {
	case mimeType == "image/svg+xml":
		return false
	case strings.HasPrefix(mimeType, "image/"),
		strings.HasPrefix(mimeType, "video/"),
		strings.HasPrefix(mimeType, "audio/"),
		mimeType == "application/pdf",
		mimeType == "text/plain":
		return true
	}
	return false
}

// ImageSize returns the pixel dimensions of a JPEG, PNG, GIF or WebP image,
// reading only its header.
func ImageSize(r io.Reader) (width, height int, ok bool) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
