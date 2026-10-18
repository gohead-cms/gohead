// Package media stores uploaded files on a pluggable backend: the local
// filesystem (development, single replica) or any S3-compatible object
// store (AWS S3, MinIO, Cloudflare R2, Garage, Ceph...).
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Store saves and serves file contents. Keys are relative, slash-separated
// paths such as "2026/10/3f2a9c-photo.jpg".
type Store interface {
	// Driver names the backend ("local" or "s3").
	Driver() string
	// Put stores size bytes from r under key.
	Put(ctx context.Context, key string, r io.Reader, size int64, meta ObjectMeta) error
	// Delete removes key. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
	// URL returns the address clients use to download key.
	URL(ctx context.Context, key string) (string, error)
}

// ObjectMeta is stored with an object and sent back when it is served.
type ObjectMeta struct {
	ContentType string
	// Attachment makes browsers download the file instead of rendering it
	// (used for every type that is not safe to display inline).
	Attachment bool
	FileName   string
}

// Config selects and configures a Store.
type Config struct {
	Driver string // "local" (default) or "s3"
	// PublicURL is the base URL files are served from, e.g. a CDN
	// ("https://cdn.example.com") or the bucket's public endpoint. Keys are
	// appended to it. Optional: local files default to /uploads/<key> on
	// the API, and S3 files to short-lived presigned URLs.
	PublicURL string
	Local     LocalConfig
	S3        S3Config
}

// LocalConfig configures the filesystem driver.
type LocalConfig struct {
	Path string // directory files are written to
}

// S3Config configures the S3-compatible driver.
type S3Config struct {
	Endpoint  string // URL ("https://s3.amazonaws.com") or host[:port] with UseSSL
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// PathStyle uses endpoint/bucket/key URLs instead of bucket.endpoint/key
	// (needed by MinIO and most self-hosted stores).
	PathStyle bool
	// Prefix is prepended to every key, to share a bucket.
	Prefix string
	// PresignExpiry is the lifetime of download URLs when PublicURL is
	// empty.
	PresignExpiry time.Duration
}

// New builds the Store described by cfg.
func New(ctx context.Context, cfg Config) (Store, error) {
	switch strings.ToLower(cfg.Driver) {
	case "", "local":
		return NewLocal(cfg.Local.Path, cfg.PublicURL)
	case "s3":
		return NewS3(ctx, cfg.S3, cfg.PublicURL)
	default:
		return nil, fmt.Errorf("unknown media driver %q (expected local or s3)", cfg.Driver)
	}
}

// ErrInvalidKey is returned for keys that could escape the storage root.
var ErrInvalidKey = errors.New("invalid media key")

// checkKey rejects absolute paths, traversal and empty segments.
func checkKey(key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") {
		return ErrInvalidKey
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return ErrInvalidKey
		}
	}
	return nil
}

func joinURL(base, key string) string {
	return strings.TrimRight(base, "/") + "/" + key
}
