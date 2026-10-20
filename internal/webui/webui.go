// Package webui serves the admin UI embedded in the binary.
//
// The UI is built into static/ by `make ui` (npm run build in gui/) before
// `go build`; without it the binary has no UI and Register does nothing.
// The UI calls the API on the same origin.
package webui

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:static
var embedded embed.FS

// files of the built UI (static/), nil when the binary has none.
var files = func() fs.FS {
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}()

// Available tells whether the binary includes the admin UI.
func Available() bool { return files != nil }

// Paths of the API, never answered with the UI.
var apiPrefixes = []string{"/api/", "/admin/", "/auth/", "/uploads/", "/agents/webhook/", "/_"}

// configJS points the UI to the API of the page's own origin.
const configJS = "window.__GOHEAD_CONFIG__ = { apiBaseUrl: window.location.origin };\n"

// Register serves the UI on router: its files, and index.html for the
// UI's own routes (/login, /contrib/…). It returns false when the binary
// has no UI.
func Register(router *gin.Engine) bool {
	if files == nil {
		return false
	}
	router.GET("/config.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(configJS))
	})
	router.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if (c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) || isAPI(p) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		serve(c, p)
	})
	return true
}

func isAPI(p string) bool {
	for _, prefix := range apiPrefixes {
		if strings.HasPrefix(p, prefix) || p+"/" == prefix {
			return true
		}
	}
	return false
}

func serve(c *gin.Context, p string) {
	name := strings.TrimPrefix(path.Clean("/"+p), "/")
	if name != "" && name != "index.html" {
		if st, err := fs.Stat(files, name); err == nil && !st.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				// Vite names assets after their content.
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "public, max-age=3600")
			}
			sendFile(c, name)
			return
		}
		// A missing file (with an extension) is a 404, not the UI.
		if path.Ext(name) != "" {
			c.Status(http.StatusNotFound)
			return
		}
	}
	c.Header("Cache-Control", "no-cache")
	sendFile(c, "index.html")
}

// sendFile writes a file, gzipped when a .gz version exists and the client
// accepts it.
func sendFile(c *gin.Context, name string) {
	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	c.Header("Vary", "Accept-Encoding")
	c.Header("X-Content-Type-Options", "nosniff")
	// The admin UI is never framed by other sites (clickjacking).
	c.Header("X-Frame-Options", "DENY")
	c.Header("Content-Security-Policy", "frame-ancestors 'none'")
	if strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		if b, err := fs.ReadFile(files, name+".gz"); err == nil {
			c.Header("Content-Encoding", "gzip")
			c.Data(http.StatusOK, ctype, b)
			return
		}
	}
	b, err := fs.ReadFile(files, name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, ctype, b)
}
