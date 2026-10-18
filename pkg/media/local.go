package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LocalPathPrefix is where the API serves local files.
const LocalPathPrefix = "/uploads"

// Local stores files in a directory. It suits development and single-replica
// deployments; replicas do not share it unless the directory is a shared
// volume.
type Local struct {
	root      string
	publicURL string
}

// NewLocal returns a Local store writing under root (created if missing).
// publicURL is the base download URL; empty means LocalPathPrefix on the API.
func NewLocal(root, publicURL string) (*Local, error) {
	if root == "" {
		root = "uploads"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("cannot create media directory %s: %w", abs, err)
	}
	return &Local{root: abs, publicURL: publicURL}, nil
}

func (l *Local) Driver() string { return "local" }

// Root is the directory files are stored in.
func (l *Local) Root() string { return l.root }

// Path returns the file path of key, refusing keys that leave the root.
func (l *Local) Path(key string) (string, error) {
	if err := checkKey(key); err != nil {
		return "", err
	}
	return filepath.Join(l.root, filepath.FromSlash(key)), nil
}

// Within reports whether path is inside root once both are cleaned.
func Within(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ ObjectMeta) error {
	p, err := l.Path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	// Write to a temporary file and rename, so readers never see a partial file.
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) URL(_ context.Context, key string) (string, error) {
	if err := checkKey(key); err != nil {
		return "", err
	}
	if l.publicURL != "" {
		return joinURL(l.publicURL, key), nil
	}
	return LocalPathPrefix + "/" + key, nil
}
