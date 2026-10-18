package media

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeS3Server is a minimal in-memory S3 endpoint (path-style) covering
// what the S3 driver uses: bucket location/existence, put, get and delete.
// Signatures are not checked.
type fakeS3Server struct {
	mu      sync.Mutex
	buckets map[string]map[string]fakeObject
}

type fakeObject struct {
	data        []byte
	contentType string
	disposition string
}

func (f *fakeS3Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	objects, ok := f.buckets[bucket]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchBucket</Code><Message>no such bucket</Message></Error>`)
		return
	}
	if key == "" {
		if _, loc := r.URL.Query()["location"]; loc {
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`)
			return
		}
		w.WriteHeader(http.StatusOK) // HEAD bucket
		return
	}
	switch r.Method {
	case http.MethodPut:
		body, err := readS3Body(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		objects[key] = fakeObject{data: body, contentType: r.Header.Get("Content-Type"), disposition: r.Header.Get("Content-Disposition")}
		w.Header().Set("ETag", `"fake"`)
	case http.MethodGet, http.MethodHead:
		obj, ok := objects[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>no such key</Message></Error>`)
			return
		}
		w.Header().Set("Content-Type", obj.contentType)
		if obj.disposition != "" {
			w.Header().Set("Content-Disposition", obj.disposition)
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(obj.data)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(obj.data)
		}
	case http.MethodDelete:
		delete(objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// readS3Body decodes aws-chunked uploads (used by minio-go over plain HTTP).
func readS3Body(r *http.Request) ([]byte, error) {
	if !strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		return io.ReadAll(r.Body)
	}
	var out bytes.Buffer
	br := bufio.NewReader(r.Body)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		sizeHex, _, _ := strings.Cut(strings.TrimSpace(line), ";")
		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			return nil, err
		}
		if size == 0 {
			return out.Bytes(), nil
		}
		if _, err := io.CopyN(&out, br, size); err != nil {
			return nil, err
		}
		if _, err := br.ReadString('\n'); err != nil { // chunk trailer CRLF
			return nil, err
		}
	}
}

// fakeS3 starts the fake with one bucket and returns its URL.
func fakeS3(t *testing.T, bucket string) string {
	t.Helper()
	srv := httptest.NewServer(&fakeS3Server{buckets: map[string]map[string]fakeObject{bucket: {}}})
	t.Cleanup(srv.Close)
	return srv.URL
}
