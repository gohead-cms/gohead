package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 stores files in an S3-compatible bucket.
type S3 struct {
	client    *minio.Client
	bucket    string
	prefix    string
	publicURL string
	expiry    time.Duration
}

// NewS3 connects to the bucket described by cfg and checks that it exists.
func NewS3(ctx context.Context, cfg S3Config, publicURL string) (*S3, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("media.s3.endpoint and media.s3.bucket are required")
	}
	endpoint := cfg.Endpoint
	useSSL := cfg.UseSSL
	// Accept a full URL too ("https://s3.eu-west-1.amazonaws.com").
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		endpoint = u.Host
		useSSL = u.Scheme == "https"
	}
	lookup := minio.BucketLookupAuto
	if cfg.PathStyle {
		lookup = minio.BucketLookupPath
	}
	var creds *credentials.Credentials
	if cfg.AccessKey != "" {
		creds = credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	} else {
		// Standard AWS environment variables or an instance/IRSA role.
		creds = credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{}, &credentials.IAM{},
		})
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        creds,
		Secure:       useSSL,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid S3 configuration: %w", err)
	}
	expiry := cfg.PresignExpiry
	if expiry <= 0 {
		expiry = time.Hour
	}
	s := &S3{
		client:    client,
		bucket:    cfg.Bucket,
		prefix:    strings.Trim(cfg.Prefix, "/"),
		publicURL: publicURL,
		expiry:    expiry,
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ok, err := client.BucketExists(checkCtx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("cannot reach bucket %q at %s: %w", cfg.Bucket, endpoint, err)
	}
	if !ok {
		return nil, fmt.Errorf("bucket %q does not exist at %s", cfg.Bucket, endpoint)
	}
	return s, nil
}

func (s *S3) Driver() string { return "s3" }

func (s *S3) objectKey(key string) (string, error) {
	if err := checkKey(key); err != nil {
		return "", err
	}
	if s.prefix == "" {
		return key, nil
	}
	return s.prefix + "/" + key, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, meta ObjectMeta) error {
	obj, err := s.objectKey(key)
	if err != nil {
		return err
	}
	opts := minio.PutObjectOptions{ContentType: meta.ContentType}
	if meta.Attachment {
		opts.ContentDisposition = mime.FormatMediaType("attachment", map[string]string{"filename": meta.FileName})
	}
	_, err = s.client.PutObject(ctx, s.bucket, obj, r, size, opts)
	return err
}

func (s *S3) Delete(ctx context.Context, key string) error {
	obj, err := s.objectKey(key)
	if err != nil {
		return err
	}
	// RemoveObject succeeds for missing keys.
	return s.client.RemoveObject(ctx, s.bucket, obj, minio.RemoveObjectOptions{})
}

func (s *S3) URL(ctx context.Context, key string) (string, error) {
	obj, err := s.objectKey(key)
	if err != nil {
		return "", err
	}
	if s.publicURL != "" {
		return joinURL(s.publicURL, obj), nil
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, obj, s.expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
