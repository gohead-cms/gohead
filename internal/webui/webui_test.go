package webui

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gz(s string) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return b.Bytes()
}

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	prev := files
	t.Cleanup(func() { files = prev })
	files = fstest.MapFS{
		"index.html":             {Data: []byte("<html>ui</html>")},
		"config.js":              {Data: []byte("window.__GOHEAD_CONFIG__ = { apiBaseUrl: 'http://localhost:8080' };")},
		"gohead_logo.svg":        {Data: []byte("<svg/>")},
		"assets/index-abc.js":    {Data: []byte("console.log(1)")},
		"assets/index-abc.js.gz": {Data: gz("console.log(1)")},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/collections/x", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	require.True(t, Register(r))
	return r
}

func get(r http.Handler, method, path string, hdr ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestServesTheUI(t *testing.T) {
	r := testRouter(t)

	for _, p := range []string{"/", "/login", "/contrib/edit/articles", "/settings/users", "/agents"} {
		w := get(r, http.MethodGet, p)
		assert.Equal(t, http.StatusOK, w.Code, p)
		assert.Equal(t, "<html>ui</html>", w.Body.String(), p)
		assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"), p)
	}

	w := get(r, http.MethodGet, "/config.js")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "window.location.origin", "the UI calls the API on its own origin")

	w = get(r, http.MethodGet, "/assets/index-abc.js")
	assert.Equal(t, "console.log(1)", w.Body.String())
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	assert.Contains(t, w.Header().Get("Content-Type"), "javascript")

	w = get(r, http.MethodGet, "/assets/index-abc.js", "Accept-Encoding", "gzip, br")
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, gz("console.log(1)"), w.Body.Bytes())

	w = get(r, http.MethodGet, "/gohead_logo.svg")
	assert.Equal(t, "<svg/>", w.Body.String())

	assert.Equal(t, http.StatusNotFound, get(r, http.MethodGet, "/assets/missing.js").Code)
	assert.Equal(t, http.StatusOK, get(r, http.MethodHead, "/login").Code)
}

func TestLeavesTheAPIAlone(t *testing.T) {
	r := testRouter(t)
	assert.Equal(t, "api", get(r, http.MethodGet, "/api/collections/x").Body.String())
	for _, p := range []string{"/api/nope", "/admin/nope", "/auth/nope", "/uploads/x.png", "/_nope", "/agents/webhook/3"} {
		w := get(r, http.MethodGet, p)
		assert.Equal(t, http.StatusNotFound, w.Code, p)
		assert.NotContains(t, w.Body.String(), "<html>", p)
	}
	assert.Equal(t, http.StatusNotFound, get(r, http.MethodPost, "/login").Code)
	// Path tricks stay inside the UI files.
	w := get(r, http.MethodGet, "/../../etc/passwd")
	assert.NotContains(t, w.Body.String(), "root:")
}

func TestNoUIInThisBuild(t *testing.T) {
	prev := files
	t.Cleanup(func() { files = prev })
	files = nil
	assert.False(t, Register(gin.New()))
	assert.False(t, Available())
}
