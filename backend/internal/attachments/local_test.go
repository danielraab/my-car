package attachments

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStore_RoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "missing", "files")
	s, err := NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Check(ctx); err != nil {
		t.Fatalf("Check() = %v", err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("storage dir = %v, %v; want created with 0700", info, err)
	}

	key := storageKey("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")
	if err := s.Put(ctx, key, strings.NewReader("hello"), 5, "application/pdf"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(key)))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stored file = %v, %v; want mode 0600", info, err)
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
		t.Fatalf("second Delete() = %v, want nil for a missing key", err)
	}
	if _, err := s.Get(ctx, key); !errors.Is(err, ErrBlobNotFound) {
		t.Fatalf("Get() after delete = %v, want ErrBlobNotFound", err)
	}
}

func TestLocalStore_RejectsKeysOutsideBase(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	s, err := NewLocalStore(filepath.Join(root, "files"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape", "a/../../escape", "../files-sibling/x", ""} {
		if err := s.Put(ctx, key, strings.NewReader("x"), 1, ""); err == nil {
			t.Errorf("Put(%q) succeeded, want rejection", key)
		}
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatalf("root contains %d entries, want only the storage dir", len(entries))
	}
}

func TestLocalStore_CheckFailsWhenNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}
	dir := t.TempDir()
	s, err := NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if err := s.Check(context.Background()); err == nil {
		t.Fatal("Check() = nil, want error for a read-only directory")
	}
}
