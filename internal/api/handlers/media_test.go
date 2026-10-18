package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/media"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type upload struct {
	name    string
	content []byte
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))))
	return buf.Bytes()
}

// newMediaAPI extends the content API with media routes and a local store.
func newMediaAPI(t *testing.T) (*contentAPI, *media.Local) {
	t.Helper()
	a := newContentAPI(t)
	store, err := media.NewLocal(filepath.Join(t.TempDir(), "uploads"), "")
	require.NoError(t, err)
	storage.InitMediaStore(store)
	prev := MediaPolicy
	MediaPolicy = storage.UploadPolicy{MaxSize: 64 << 10} // 64 KB
	t.Cleanup(func() { storage.InitMediaStore(nil); MediaPolicy = prev })

	api := a.router.Group("/api", middleware.AuthMiddleware())
	api.POST("/media", UploadMedia)
	api.GET("/media", ListMedia)
	api.GET("/media/:id", GetMedia)
	api.PUT("/media/:id", UpdateMedia)
	api.DELETE("/media/:id", DeleteMedia)
	a.router.GET("/uploads/*key", ServeLocalMedia(store))
	return a, store
}

func (a *contentAPI) upload(role string, files []upload, fields map[string]string) (int, map[string]any) {
	a.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range files {
		part, err := w.CreateFormFile("files", f.name)
		require.NoError(a.t, err)
		_, _ = part.Write(f.content)
	}
	for k, v := range fields {
		require.NoError(a.t, w.WriteField(k, v))
	}
	require.NoError(a.t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	token, err := auth.GenerateJWT(role+"-user", role)
	require.NoError(a.t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, req)
	var out map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return rr.Code, out
}

func (a *contentAPI) uploadOK(files ...upload) []map[string]any {
	a.t.Helper()
	code, out := a.upload("editor", files, nil)
	require.Equal(a.t, http.StatusCreated, code, "%v", out)
	var list []map[string]any
	for _, it := range out["data"].([]any) {
		list = append(list, it.(map[string]any))
	}
	return list
}

func TestMediaUploads(t *testing.T) {
	a, store := newMediaAPI(t)
	img := testPNG(t, 40, 30)

	code, out := a.upload("editor", []upload{{"Holiday Photo.PNG", img}}, map[string]string{"alt": "Beach", "caption": "Summer"})
	require.Equal(t, http.StatusCreated, code, "%v", out)
	m := out["data"].([]any)[0].(map[string]any)
	assert.Equal(t, "Holiday Photo.PNG", m["name"])
	assert.Equal(t, "image/png", m["mime"])
	assert.EqualValues(t, len(img), m["size"])
	assert.EqualValues(t, 40, m["width"])
	assert.EqualValues(t, 30, m["height"])
	assert.Equal(t, "Beach", m["alt"])
	assert.Equal(t, "editor-user", m["uploadedBy"])
	assert.Len(t, m["hash"], 64)
	url := m["url"].(string)
	assert.Regexp(t, `^/uploads/\d{4}/\d{2}/[0-9a-f]{16}-holiday-photo\.png$`, url)
	id := int(m["id"].(float64))

	t.Run("the file is served with safe headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, img, rr.Body.Bytes())
		assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "sandbox")
		assert.Empty(t, rr.Header().Get("Content-Disposition"), "images display inline")

		req = httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Range", "bytes=0-3")
		rr = httptest.NewRecorder()
		a.router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusPartialContent, rr.Code, "range requests work (video seeking)")
		assert.Equal(t, img[:4], rr.Body.Bytes())

		for _, bad := range []string{"/uploads/../../etc/passwd", "/uploads/2026/10/not-in-library.png"} {
			rr = httptest.NewRecorder()
			a.router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, bad, nil))
			assert.Equal(t, http.StatusNotFound, rr.Code, bad)
		}
	})

	t.Run("types come from the content", func(t *testing.T) {
		code, out := a.upload("editor", []upload{{"innocent.png", []byte("<html><script>alert(1)</script></html>")}}, nil)
		assert.Equal(t, http.StatusUnsupportedMediaType, code)
		assert.Contains(t, a.errMsg(out), "text/html")

		code, _ = a.upload("editor", []upload{{"logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)}}, nil)
		assert.Equal(t, http.StatusUnsupportedMediaType, code, "SVG is refused by default")

		zip := a.uploadOK(upload{"report.zip", append([]byte("PK\x03\x04"), make([]byte, 64)...)})[0]
		assert.Equal(t, "application/zip", zip["mime"])
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, zip["url"].(string), nil))
		assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment", "non-media types download")
	})

	t.Run("limits and permissions", func(t *testing.T) {
		code, out := a.upload("editor", []upload{{"big.png", append(testPNG(t, 2, 2), make([]byte, 70<<10)...)}}, nil)
		assert.Equal(t, http.StatusRequestEntityTooLarge, code, "%v", out)

		code, _ = a.upload("viewer", []upload{{"a.png", img}}, nil)
		assert.Equal(t, http.StatusForbidden, code)

		code, out = a.upload("editor", nil, map[string]string{"alt": "x"})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, a.errMsg(out), "files")

		code, _ = a.do("editor", "POST", "/api/media", map[string]any{"file": "x"})
		assert.Equal(t, http.StatusBadRequest, code, "JSON bodies are rejected")
	})

	t.Run("a batch is stored entirely or not at all", func(t *testing.T) {
		before := len(a.ok("viewer", "GET", "/api/media?pageSize=100", nil).([]any))
		code, _ := a.upload("editor", []upload{{"ok.png", img}, {"bad.exe", []byte("MZ\x90\x00binary")}}, nil)
		assert.Equal(t, http.StatusUnsupportedMediaType, code)
		assert.Len(t, a.ok("viewer", "GET", "/api/media?pageSize=100", nil), before)

		two := a.uploadOK(upload{"one.png", img}, upload{"two.png", img})
		assert.Len(t, two, 2)
	})

	t.Run("library: list, filter, update, delete", func(t *testing.T) {
		images := a.ok("viewer", "GET", "/api/media?mime=image/", nil).([]any)
		for _, it := range images {
			assert.True(t, strings.HasPrefix(it.(map[string]any)["mime"].(string), "image/"))
		}
		found := a.ok("viewer", "GET", "/api/media?search=holiday", nil).([]any)
		require.Len(t, found, 1)

		path := fmt.Sprintf("/api/media/%d", id)
		got := a.ok("viewer", "GET", path, nil).(map[string]any)
		assert.Equal(t, "Beach", got["alt"])
		upd := a.ok("editor", "PUT", path, map[string]any{"data": map[string]any{"alt": "Sunny beach"}}).(map[string]any)
		assert.Equal(t, "Sunny beach", upd["alt"])
		assert.Equal(t, "Summer", upd["caption"], "fields left out are kept")

		code, _ := a.do("viewer", "DELETE", path, nil)
		assert.Equal(t, http.StatusForbidden, code)

		key := strings.TrimPrefix(url, "/uploads/")
		file, _ := store.Path(key)
		_, err := os.Stat(file)
		require.NoError(t, err)
		a.ok("editor", "DELETE", path, nil)
		_, err = os.Stat(file)
		assert.True(t, os.IsNotExist(err), "the stored file is removed")
		code, _ = a.do("viewer", "GET", path, nil)
		assert.Equal(t, http.StatusNotFound, code)
	})
}

func TestMediaAttributes(t *testing.T) {
	a, _ := newMediaAPI(t)
	img := a.uploadOK(upload{"cover.png", testPNG(t, 8, 8)})[0]
	pdf := a.uploadOK(upload{"brochure.pdf", []byte("%PDF-1.7\n%…")})[0]
	imgID, pdfID := img["id"], pdf["id"]

	a.ok("admin", "POST", "/admin/components", map[string]any{
		"name": "card",
		"attributes": map[string]any{
			"title": map[string]any{"type": "string"},
			"icon":  map[string]any{"type": "media", "allowed_types": []any{"image/*"}},
		},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "people", "kind": "collection",
		"attributes": map[string]any{
			"name":   map[string]any{"type": "string"},
			"avatar": map[string]any{"type": "media", "allowed_types": []any{"image/*"}},
		},
	})
	a.ok("admin", "POST", "/admin/collections", map[string]any{
		"name": "articles", "kind": "collection",
		"attributes": map[string]any{
			"title":  map[string]any{"type": "string"},
			"cover":  map[string]any{"type": "media", "allowed_types": []any{"image/*"}},
			"files":  map[string]any{"type": "media", "multiple": true, "max": 3},
			"legacy": map[string]any{"type": "media"},
			"author": map[string]any{"type": "relation", "relation": "oneToOne", "target": "people"},
			"cards":  map[string]any{"type": "component", "component": "card", "repeatable": true},
		},
	})

	created := a.ok("editor", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{
		"title":  "With files",
		"cover":  imgID,
		"files":  []any{map[string]any{"id": pdfID}, fmt.Sprint(imgID)},
		"legacy": "https://example.com/old.jpg",
		"cards":  []any{map[string]any{"title": "A", "icon": imgID}},
	}}).(map[string]any)

	cover := created["cover"].(map[string]any)
	assert.Equal(t, img["url"], cover["url"])
	assert.Equal(t, "cover.png", cover["name"])
	assert.EqualValues(t, 8, cover["width"])
	files := created["files"].([]any)
	require.Len(t, files, 2)
	assert.Equal(t, "application/pdf", files[0].(map[string]any)["mime"])
	assert.Equal(t, "https://example.com/old.jpg", created["legacy"], "URL values are kept as they are")
	icon := created["cards"].([]any)[0].(map[string]any)["icon"].(map[string]any)
	assert.Equal(t, img["url"], icon["url"], "media inside components are populated too")

	path := fmt.Sprintf("/api/collections/articles/%d", int(created["id"].(float64)))
	raw := a.ok("viewer", "GET", path+"?level=0", nil).(map[string]any)
	assert.Equal(t, imgID, raw["cover"], "level=0 returns IDs")

	// A related item is an object, so its files are populated too.
	ada := a.ok("editor", "POST", "/api/collections/people", map[string]any{"data": map[string]any{
		"name": "Ada", "avatar": imgID,
	}}).(map[string]any)
	other := a.ok("editor", "POST", "/api/collections/articles", map[string]any{"data": map[string]any{
		"title": "Other", "author": ada["id"],
	}}).(map[string]any)
	rel := other["author"].(map[string]any)
	assert.Equal(t, img["url"], rel["avatar"].(map[string]any)["url"])

	bad := map[string]map[string]any{
		"unknown file":              {"cover": 999999},
		"wrong type":                {"cover": pdfID},
		"URL where uploads only":    {"cover": "https://example.com/x.png"},
		"too many files":            {"files": []any{imgID, imgID, imgID, imgID}},
		"multiple needs a list":     {"files": imgID},
		"wrong type in a component": {"cards": []any{map[string]any{"icon": pdfID}}},
		"not a reference":           {"cover": true},
	}
	for name, data := range bad {
		code, out := a.do("editor", "POST", "/api/collections/articles", map[string]any{"data": data})
		assert.Equal(t, http.StatusBadRequest, code, "%s: %v", name, out)
	}

	t.Run("deleted files read as null", func(t *testing.T) {
		a.ok("editor", "DELETE", fmt.Sprintf("/api/media/%d", int(pdfID.(float64))), nil)
		got := a.ok("viewer", "GET", path, nil).(map[string]any)
		files := got["files"].([]any)
		require.Len(t, files, 1, "the deleted file is left out of the list")
		assert.Equal(t, "image/png", files[0].(map[string]any)["mime"])
	})

	t.Run("singletons", func(t *testing.T) {
		a.ok("admin", "POST", "/admin/singleton", map[string]any{
			"name": "site", "attributes": map[string]any{"logo": map[string]any{"type": "media"}},
		})
		a.ok("editor", "PUT", "/api/singleton/site", map[string]any{"data": map[string]any{"logo": imgID}})
		got := a.ok("viewer", "GET", "/api/singleton/site", nil).(map[string]any)
		assert.Equal(t, img["url"], got["logo"].(map[string]any)["url"])
		code, _ := a.do("editor", "PUT", "/api/singleton/site", map[string]any{"data": map[string]any{"logo": 424242}})
		assert.Equal(t, http.StatusBadRequest, code)
	})
}
