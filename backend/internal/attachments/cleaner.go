package attachments

import (
	"context"
	"log/slog"
	"time"
)

// FileDeletionQueue is the database queue of stored files whose attachment
// record is gone, filled by a trigger on every attachment deletion,
// including cascades from a deleted entry or car.
type FileDeletionQueue interface {
	PendingFileDeletions(ctx context.Context, limit int) ([]string, error)
	FileDeleted(ctx context.Context, key string) error
}

const cleanBatchSize = 100

// Cleaner removes queued files from the blob store, periodically and
// whenever kicked. A key leaves the queue only after its file is gone, so a
// failed removal is retried on the next run.
type Cleaner struct {
	queue FileDeletionQueue
	blobs BlobStore
	kick  chan struct{}
}

func NewCleaner(queue FileDeletionQueue, blobs BlobStore) *Cleaner {
	return &Cleaner{queue: queue, blobs: blobs, kick: make(chan struct{}, 1)}
}

// Kick asks for a run soon without blocking.
func (c *Cleaner) Kick() {
	select {
	case c.kick <- struct{}{}:
	default:
	}
}

// Run cleans once immediately, then on every interval or kick until ctx ends.
func (c *Cleaner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		c.Clean(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-c.kick:
		}
	}
}

// Clean drains the queue, stopping at the first failure so an unreachable
// store is not hammered; it returns how many files were removed.
func (c *Cleaner) Clean(ctx context.Context) int {
	removed := 0
	for {
		keys, err := c.queue.PendingFileDeletions(ctx, cleanBatchSize)
		if err != nil {
			slog.Error("read attachment file deletion queue", "error", err)
			return removed
		}
		for _, key := range keys {
			if err := c.blobs.Delete(ctx, key); err != nil {
				slog.Error("remove attachment file; will retry", "storage_key", key, "error", err)
				return removed
			}
			if err := c.queue.FileDeleted(ctx, key); err != nil {
				slog.Error("dequeue removed attachment file", "storage_key", key, "error", err)
				return removed
			}
			removed++
		}
		if len(keys) < cleanBatchSize {
			return removed
		}
	}
}
