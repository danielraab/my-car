package profile

import (
	"context"
	"os"
	"testing"
	"time"

	"at.draab/my-car/internal/db"
)

func TestStoreUpdateProfile(t *testing.T) {
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
	defer pool.Close()
	// Serialize with other packages' database tests, some of which truncate
	// the domain tables or require them empty while holding this lock.
	lock, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if _, err := lock.Exec(ctx, `SELECT pg_advisory_lock(7242026)`); err != nil {
		t.Fatal(err)
	}
	defer lock.Exec(ctx, `SELECT pg_advisory_unlock(7242026)`)
	email := "profile-test-" + time.Now().Format("150405.000000000") + "@example.com"
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO accounts (email,first_name,last_name) VALUES ($1,'Ada','Lovelace') RETURNING id`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	s := NewStore(pool)

	first := "Grace"
	a, err := s.UpdateProfile(ctx, id, Update{FirstName: &first})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != id || a.Email != email || a.FirstName != "Grace" || a.LastName != "Lovelace" || a.Currency != "EUR" || a.VolumeUnit != "liter" || a.DistanceUnit != "kilometer" {
		t.Fatalf("partial update = %+v", a)
	}

	empty := ""
	a, err = s.UpdateProfile(ctx, id, Update{LastName: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if a.FirstName != "Grace" || a.LastName != "" {
		t.Fatalf("empty update = %+v", a)
	}

	usd, gallon, mile := "USD", "gallon", "mile"
	a, err = s.UpdateProfile(ctx, id, Update{Currency: &usd, VolumeUnit: &gallon, DistanceUnit: &mile})
	if err != nil {
		t.Fatal(err)
	}
	if a.FirstName != "Grace" || a.Currency != "USD" || a.VolumeUnit != "gallon" || a.DistanceUnit != "mile" {
		t.Fatalf("settings update = %+v", a)
	}
}
