package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestDetectType(t *testing.T) {
	pngData := pngBytes(t, 3, 2)
	cases := []struct {
		name string
		head []byte
		file string
		want string
	}{
		{"png by content, whatever the name", pngData, "photo.pdf", "image/png"},
		{"html is caught even named .png", []byte("<!DOCTYPE html><html><script>alert(1)</script>"), "cat.png", "text/html"},
		{"svg from text + extension", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`), "logo.svg", "image/svg+xml"},
		{"csv refined from text", []byte("a,b\n1,2\n"), "data.csv", "text/csv"},
		{"docx refined from zip", []byte("PK\x03\x04rest-of-zip"), "report.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"binary never becomes text", []byte{0x00, 0x01, 0x02, 0xff, 0xfe}, "notes.txt", "application/octet-stream"},
		{"text never becomes binary", []byte("just words"), "movie.mp4", "text/plain"},
		{"pdf", []byte("%PDF-1.7\n"), "x.bin", "application/pdf"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, DetectType(c.head, c.file), c.name)
	}
}

func TestAllowedAndInline(t *testing.T) {
	assert.True(t, Allowed("image/png", []string{"image/*"}))
	assert.True(t, Allowed("application/pdf", []string{"image/*", "application/pdf"}))
	assert.False(t, Allowed("text/html", []string{"image/*"}))
	assert.True(t, Allowed("text/html", []string{"*/*"}))
	assert.False(t, Allowed("image/svg+xml", DefaultAllowedTypes), "SVG is not allowed by default")
	assert.False(t, Allowed("text/html", DefaultAllowedTypes))

	assert.True(t, InlineSafe("image/jpeg"))
	assert.True(t, InlineSafe("video/mp4"))
	assert.False(t, InlineSafe("image/svg+xml"))
	assert.False(t, InlineSafe("text/html"))
	assert.False(t, InlineSafe("application/zip"))
}

func TestImageSize(t *testing.T) {
	w, h, ok := ImageSize(bytes.NewReader(pngBytes(t, 40, 30)))
	require.True(t, ok)
	assert.Equal(t, 40, w)
	assert.Equal(t, 30, h)
	_, _, ok = ImageSize(strings.NewReader("not an image"))
	assert.False(t, ok)
}

func TestWithin(t *testing.T) {
	assert.True(t, Within("/data/uploads", "/data/uploads/2026/x.png"))
	assert.False(t, Within("/data/uploads", "/data/uploads"))
	assert.False(t, Within("/data/uploads", "/data/uploads/../secret"))
	assert.False(t, Within("/data/uploads", "/data/uploads-other/x"))
	assert.False(t, Within("/data/uploads", "/etc/passwd"))
}

func TestCheckKey(t *testing.T) {
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../../x", "a//b", "a/./b", `a\b`} {
		assert.ErrorIs(t, checkKey(bad), ErrInvalidKey, bad)
	}
	assert.NoError(t, checkKey("2026/10/abc-photo.jpg"))
}

func TestLocalStore(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(filepath.Join(dir, "up"), "")
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, s.Put(ctx, "2026/10/a-x.txt", strings.NewReader("hello"), 5, ObjectMeta{}))
	got, err := os.ReadFile(filepath.Join(dir, "up", "2026", "10", "a-x.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))

	u, err := s.URL(ctx, "2026/10/a-x.txt")
	require.NoError(t, err)
	assert.Equal(t, "/uploads/2026/10/a-x.txt", u)

	withCDN, err := NewLocal(filepath.Join(dir, "up"), "https://cms.example.com/uploads/")
	require.NoError(t, err)
	u, _ = withCDN.URL(ctx, "2026/10/a-x.txt")
	assert.Equal(t, "https://cms.example.com/uploads/2026/10/a-x.txt", u)

	assert.ErrorIs(t, s.Put(ctx, "../escape.txt", strings.NewReader("x"), 1, ObjectMeta{}), ErrInvalidKey)
	_, err = os.Stat(filepath.Join(dir, "escape.txt"))
	assert.True(t, os.IsNotExist(err))

	require.NoError(t, s.Delete(ctx, "2026/10/a-x.txt"))
	require.NoError(t, s.Delete(ctx, "2026/10/a-x.txt"), "deleting twice is fine")
}

func TestS3Store(t *testing.T) {
	endpoint := fakeS3(t, "media")
	ctx := context.Background()
	cfg := S3Config{
		Endpoint: endpoint, Region: "us-east-1", Bucket: "media",
		AccessKey: "key", SecretKey: "secret", PathStyle: true, Prefix: "site-a",
		PresignExpiry: 10 * time.Minute,
	}
	s, err := NewS3(ctx, cfg, "")
	require.NoError(t, err)
	assert.Equal(t, "s3", s.Driver())

	data := pngBytes(t, 2, 2)
	require.NoError(t, s.Put(ctx, "2026/10/k-photo.png", bytes.NewReader(data), int64(len(data)), ObjectMeta{ContentType: "image/png"}))
	require.NoError(t, s.Put(ctx, "2026/10/k-doc.zip", strings.NewReader("PK.."), 4, ObjectMeta{ContentType: "application/zip", Attachment: true, FileName: "doc.zip"}))

	// Without a public URL, downloads use presigned URLs under the prefix.
	u, err := s.URL(ctx, "2026/10/k-photo.png")
	require.NoError(t, err)
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	assert.Equal(t, "/media/site-a/2026/10/k-photo.png", parsed.Path)
	assert.NotEmpty(t, parsed.Query().Get("X-Amz-Signature"))

	resp, err := http.Get(u)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, data, body)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	public, err := NewS3(ctx, cfg, "https://cdn.example.com")
	require.NoError(t, err)
	u, _ = public.URL(ctx, "2026/10/k-photo.png")
	assert.Equal(t, "https://cdn.example.com/site-a/2026/10/k-photo.png", u)

	require.NoError(t, s.Delete(ctx, "2026/10/k-photo.png"))
	resp, err = http.Get(must(s.URL(ctx, "2026/10/k-photo.png")))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.NoError(t, s.Delete(ctx, "2026/10/k-photo.png"), "deleting a missing key is fine")

	assert.ErrorIs(t, s.Put(ctx, "../x", strings.NewReader("x"), 1, ObjectMeta{}), ErrInvalidKey)
}

func TestS3BareEndpoint(t *testing.T) {
	endpoint := fakeS3(t, "media")
	host := strings.TrimPrefix(endpoint, "http://")
	s, err := NewS3(context.Background(), S3Config{
		Endpoint: host, Bucket: "media", AccessKey: "k", SecretKey: "s", PathStyle: true, UseSSL: false,
	}, "")
	require.NoError(t, err, "host:port uses UseSSL")
	u, _ := s.URL(context.Background(), "a/b.png")
	assert.True(t, strings.HasPrefix(u, "http://"+host+"/media/a/b.png"))
}

func TestS3StoreChecksTheBucket(t *testing.T) {
	endpoint := fakeS3(t, "media")
	_, err := NewS3(context.Background(), S3Config{
		Endpoint: endpoint, Bucket: "missing", AccessKey: "k", SecretKey: "s", PathStyle: true,
	}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `bucket "missing" does not exist`)

	_, err = New(context.Background(), Config{Driver: "ftp"})
	assert.Error(t, err)
	_, err = New(context.Background(), Config{Driver: "s3"})
	assert.Error(t, err, "endpoint and bucket are required")
}

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}
