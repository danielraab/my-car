package attachments

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"at.draab/my-car/internal/db"
)

func testStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}
	if err := db.RunMigrations(url); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Serialize with other packages' database tests, one of which truncates
	// the domain tables while holding this lock.
	lock, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, `SELECT pg_advisory_lock(7242026)`); err != nil {
		lock.Release()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		lock.Exec(ctx, `SELECT pg_advisory_unlock(7242026)`)
		lock.Release()
	})
	return NewStore(pool), pool
}

// testEntries creates an account with a car, a refuel, a repair and a ticket.
func testEntries(t *testing.T, pool *pgxpool.Pool) (accountID string, owners map[OwnerKind]Owner) {
	t.Helper()
	ctx := context.Background()
	email := "attachments-test-" + time.Now().Format("150405.000000000") + "@example.com"
	if err := pool.QueryRow(ctx, `INSERT INTO accounts (email) VALUES ($1) RETURNING id`, email).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	var carID, refuelID, repairID, ticketID string
	if err := pool.QueryRow(ctx, `INSERT INTO cars (account_id, type, make, name, fuel) VALUES ($1, 'Car', 'VW', 'Golf', 'diesel') RETURNING id`, accountID).Scan(&carID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'OMV', 'normal', 40, 70) RETURNING id`, carID).Scan(&refuelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO repairs (car_id, date, station, type, amount) VALUES ($1, now(), 'Garage', 'service', 300) RETURNING id`, carID).Scan(&repairID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO tickets (car_id, date, type, location, amount) VALUES ($1, now(), 'parking', 'Linz', 30) RETURNING id`, carID).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM attachment_file_deletions WHERE storage_key LIKE $1`, accountID+"/%")
		pool.Exec(ctx, `DELETE FROM cars WHERE account_id = $1`, accountID)
		pool.Exec(ctx, `DELETE FROM attachment_file_deletions WHERE storage_key LIKE $1`, accountID+"/%")
		pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, accountID)
	})
	return accountID, map[OwnerKind]Owner{
		OwnerCar: {OwnerCar, carID}, OwnerRefuel: {OwnerRefuel, refuelID},
		OwnerRepair: {OwnerRepair, repairID}, OwnerTicket: {OwnerTicket, ticketID},
	}
}

func create(t *testing.T, s *Store, blobs BlobStore, accountID string, owner Owner, filename string) (Attachment, error) {
	t.Helper()
	ctx := context.Background()
	a, orphan, err := s.Create(ctx, accountID, owner, NewAttachment{Filename: filename, ContentType: "application/pdf", Size: 3}, func(key string) error {
		return blobs.Put(ctx, key, strings.NewReader("pdf"), 3, "application/pdf")
	})
	if orphan != "" {
		t.Fatalf("Create() reported orphan %q", orphan)
	}
	return a, err
}

func TestStore_CreateListGetForEveryOwnerKind(t *testing.T) {
	s, pool := testStore(t)
	blobs := newMemBlobs()
	accountID, owners := testEntries(t, pool)
	ctx := context.Background()
	for kind, owner := range owners {
		first, err := create(t, s, blobs, accountID, owner, "first.pdf")
		if err != nil {
			t.Fatalf("kind %d: Create() = %v", kind, err)
		}
		if first.StorageKey != accountID+"/"+first.ID {
			t.Fatalf("storage key = %q, want account/attachment IDs", first.StorageKey)
		}
		second, err := create(t, s, blobs, accountID, owner, "second.pdf")
		if err != nil {
			t.Fatal(err)
		}
		items, err := s.List(ctx, accountID, owner)
		if err != nil || len(items) != 2 || items[0].ID != first.ID || items[1].ID != second.ID {
			t.Fatalf("kind %d: List() = %+v, %v; want both, oldest first", kind, items, err)
		}
		got, err := s.Get(ctx, accountID, second.ID)
		if err != nil || got.Filename != "second.pdf" || got.Size != 3 {
			t.Fatalf("Get() = %+v, %v", got, err)
		}
	}
	if blobs.count() != 8 {
		t.Fatalf("stored files = %d, want 8", blobs.count())
	}
}

func TestStore_OtherAccountSeesNothing(t *testing.T) {
	s, pool := testStore(t)
	blobs := newMemBlobs()
	ownerID, owners := testEntries(t, pool)
	otherID, _ := testEntries(t, pool)
	ctx := context.Background()
	a, err := create(t, s, blobs, ownerID, owners[OwnerRepair], "invoice.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(ctx, otherID, owners[OwnerRepair]); !errors.Is(err, ErrNotFound) {
		t.Errorf("List() by other account = %v, want ErrNotFound", err)
	}
	if _, err := create(t, s, blobs, otherID, owners[OwnerRepair], "planted.pdf"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Create() by other account = %v, want ErrNotFound", err)
	}
	if _, err := s.Get(ctx, otherID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() by other account = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, otherID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() by other account = %v, want ErrNotFound", err)
	}
	if blobs.count() != 1 {
		t.Fatalf("stored files = %d, want only the owner's", blobs.count())
	}
}

func TestStore_LimitHoldsUnderConcurrentUploads(t *testing.T) {
	s, pool := testStore(t)
	blobs := newMemBlobs()
	accountID, owners := testEntries(t, pool)
	owner := owners[OwnerTicket]
	for i := 0; i < MaxFilesPerEntry-1; i++ {
		if _, err := create(t, s, blobs, accountID, owner, "fine.pdf"); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = create(t, s, blobs, accountID, owner, "race.pdf")
		}()
	}
	wg.Wait()
	succeeded, limited := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrLimitReached):
			limited++
		default:
			t.Fatalf("Create() = %v", err)
		}
	}
	if succeeded != 1 || limited != 3 {
		t.Fatalf("%d succeeded and %d were limited, want 1 and 3", succeeded, limited)
	}
	items, _ := s.List(context.Background(), accountID, owner)
	if len(items) != MaxFilesPerEntry || blobs.count() != MaxFilesPerEntry {
		t.Fatalf("attachments = %d, files = %d; want %d", len(items), blobs.count(), MaxFilesPerEntry)
	}
}

func TestStore_DeletionsAreQueuedAndCleaned(t *testing.T) {
	s, pool := testStore(t)
	blobs := newMemBlobs()
	accountID, owners := testEntries(t, pool)
	ctx := context.Background()
	direct, err := create(t, s, blobs, accountID, owners[OwnerCar], "direct.pdf")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []OwnerKind{OwnerCar, OwnerRefuel, OwnerRepair, OwnerTicket} {
		if _, err := create(t, s, blobs, accountID, owners[kind], "cascade.pdf"); err != nil {
			t.Fatal(err)
		}
	}
	cleaner := NewCleaner(s, blobs)

	if err := s.Delete(ctx, accountID, direct.ID); err != nil {
		t.Fatal(err)
	}
	if n := cleaner.Clean(ctx); n != 1 || blobs.count() != 4 {
		t.Fatalf("after one delete: cleaned %d, %d files left; want 1 and 4", n, blobs.count())
	}

	// Deleting the car cascades to its expenses and every attachment.
	if _, err := pool.Exec(ctx, `DELETE FROM cars WHERE id = $1`, owners[OwnerCar].ID); err != nil {
		t.Fatal(err)
	}
	blobs.deleteErr = errors.New("storage unavailable")
	if n := cleaner.Clean(ctx); n != 0 || blobs.count() != 4 {
		t.Fatalf("with failing storage: cleaned %d, %d files left; want 0 and 4", n, blobs.count())
	}
	blobs.deleteErr = nil
	if n := cleaner.Clean(ctx); n != 4 || blobs.count() != 0 {
		t.Fatalf("after recovery: cleaned %d, %d files left; want 4 and 0", n, blobs.count())
	}
	if pending, err := s.PendingFileDeletions(ctx, 100); err != nil || len(pending) != 0 {
		t.Fatalf("queue = %v, %v; want empty", pending, err)
	}
}
