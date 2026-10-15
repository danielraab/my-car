package attachments

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"at.draab/my-car/internal/config"
)

// S3Store keeps attachment contents in an S3-compatible bucket. Objects are
// written without a public ACL and only ever served through the backend.
type S3Store struct {
	client *minio.Client
	bucket string
	prefix string
}

// NewS3Store builds a client for the configured endpoint; it does not
// contact the service (see Check).
func NewS3Store(cfg config.S3Config) (*S3Store, error) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	lookup := minio.BucketLookupAuto
	if cfg.UsePathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       endpoint.Scheme == "https",
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, err
	}
	return &S3Store{client: client, bucket: cfg.Bucket, prefix: cfg.Prefix}, nil
}

func (s *S3Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, s.prefix+key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Get stats the object first, since GetObject only reports a missing object
// on the first read.
func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, s.prefix+key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
			return nil, ErrBlobNotFound
		}
		return nil, err
	}
	return obj, nil
}

// Delete succeeds for a missing object, as S3 DELETE does.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, s.prefix+key, minio.RemoveObjectOptions{})
}

// Check confirms the bucket exists and the credentials can reach it.
func (s *S3Store) Check(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", s.bucket)
	}
	return nil
}
