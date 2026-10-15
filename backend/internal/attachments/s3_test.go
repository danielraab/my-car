package attachments

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"at.draab/my-car/internal/config"
)

// TestS3Store_RoundTrip runs against a real S3-compatible service, such as
// the compose minio service:
//
//	S3_TEST_ENDPOINT=http://localhost:9000 S3_TEST_BUCKET=my-cars \
//	S3_TEST_ACCESS_KEY_ID=mycars S3_TEST_SECRET_ACCESS_KEY=mycars-secret go test ./internal/attachments/
func TestS3Store_RoundTrip(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set; skipping S3 integration test")
	}
	ctx := context.Background()
	cfg := config.S3Config{
		Endpoint: endpoint, Bucket: os.Getenv("S3_TEST_BUCKET"),
		AccessKeyID: os.Getenv("S3_TEST_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("S3_TEST_SECRET_ACCESS_KEY"),
		UsePathStyle: true, Prefix: "test/",
	}
	s, err := NewS3Store(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Check(ctx); err != nil {
		t.Fatalf("Check() = %v", err)
	}
	key := storageKey("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")
	if err := s.Put(ctx, key, strings.NewReader("hello"), 5, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "hello" {
		t.Fatalf("Get() = %q, want hello", got)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("second Delete() = %v, want nil", err)
	}
	if _, err := s.Get(ctx, key); !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("Get() after delete = %v, want ErrBlobNotFound", err)
	}

	missing := cfg
	missing.Bucket = "does-not-exist"
	other, _ := NewS3Store(missing)
	if err := other.Check(ctx); err == nil {
		t.Fatal("Check() for a missing bucket = nil, want error")
	}
}
