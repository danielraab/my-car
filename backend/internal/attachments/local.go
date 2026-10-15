package attachments

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

// LocalStore keeps attachment contents below a directory on the backend's
// filesystem, accessible only to the backend's own user.
type LocalStore struct{ base string }

// NewLocalStore creates dir if needed and returns a store rooted there.
func NewLocalStore(dir string) (*LocalStore, error) {
	base, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	return &LocalStore{base: base}, nil
}

// path resolves a key below the base directory, refusing any key that would
// leave it. Keys are server-generated, so this is a safety net.
func (s *LocalStore) path(key string) (string, error) {
	p := filepath.Join(s.base, filepath.FromSlash(key))
	if !strings.HasPrefix(p, s.base+string(filepath.Separator)) {
		return "", fmt.Errorf("storage key %q resolves outside the storage directory", key)
	}
	return p, nil
}

// Put writes to a temporary file in the target directory and renames it into
// place, so a reader never sees a partly written file.
func (s *LocalStore) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// CreateTemp already uses 0600; stated explicitly as the at-rest rule.
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (s *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrBlobNotFound
	}
	return f, err
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Check writes and removes a probe file.
func (s *LocalStore) Check(_ context.Context) error {
	f, err := os.CreateTemp(s.base, ".probe-*")
	if err != nil {
		return err
	}
	f.Close()
	return os.Remove(f.Name())
}
