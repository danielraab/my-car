// Package attachments stores files attached to cars, refuels, repairs and
// tickets, per specs/api-contract/attachments and specs/platform/file-storage.
// Metadata lives in Postgres; file contents live in a BlobStore, either the
// local filesystem or an S3-compatible bucket.
package attachments

import (
	"context"
	"errors"
	"io"
)

// ErrBlobNotFound reports a storage key without stored content.
var ErrBlobNotFound = errors.New("stored file not found")

// BlobStore holds attachment contents under server-generated keys.
type BlobStore interface {
	// Put stores size bytes read from r under key.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get opens the content stored under key, or returns ErrBlobNotFound.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the content stored under key; a missing key is not an
	// error, so a retried deletion succeeds.
	Delete(ctx context.Context, key string) error
	// Check confirms at startup that the store can be written to.
	Check(ctx context.Context) error
}

// storageKey builds the key of an attachment. It is made only of
// server-generated UUIDs, never of request input, and does not depend on the
// backend, so files can be moved between backends by copying them.
func storageKey(accountID, attachmentID string) string {
	return accountID + "/" + attachmentID
}
