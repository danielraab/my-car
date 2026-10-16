// Package seed replaces application data with realistic sample records.
package seed

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	carsPerAccount    = 5
	refuelsPerAccount = 500
	repairsPerAccount = 75
	ticketsPerAccount = 90
)

// Run replaces all application rows atomically. The caller validates emails first.
func Run(ctx context.Context, pool *pgxpool.Pool, emails []string, now time.Time) error {
	if len(emails) == 0 {
		return fmt.Errorf("at least one email is required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer tx.Rollback(ctx)
	// TRUNCATE fires no delete triggers, so the stored files of the removed
	// attachments are queued for removal explicitly; the server removes them.
	_, err = tx.Exec(ctx, `INSERT INTO attachment_file_deletions (storage_key)
		SELECT storage_key FROM attachments ON CONFLICT (storage_key) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("queue attachment files: %w", err)
	}
	// Explicit list protects migration history and unrelated tables.
	_, err = tx.Exec(ctx, `TRUNCATE TABLE sessions, webauthn_challenges, webauthn_credentials,
		oidc_identities, oidc_login_attempts, magic_link_challenges, attachments, refuels,
		repairs, tickets, cars, accounts`)
	if err != nil {
		return fmt.Errorf("clear application data: %w", err)
	}

	rng := rand.New(rand.NewSource(now.UnixNano()))
	start := now.AddDate(-5, 0, 0)
	models := []struct{ make, name, fuel string }{
		{"Volkswagen", "Golf", "diesel"}, {"Toyota", "Corolla", "gasoline"},
		{"Skoda", "Octavia", "diesel"}, {"Honda", "Civic", "gasoline"},
		{"Volkswagen", "ID.3", "electric"},
	}
	stations := []string{"Shell", "OMV", "BP", "Avanti", "Jet"}
	repairTypes := []string{"check", "service", "wearing_part", "crash_repair"}
	ticketTypes := []string{"parking", "velocity", "other"}
	for userIndex, email := range emails {
		var accountID string
		if err := tx.QueryRow(ctx, `INSERT INTO accounts (email, first_name, last_name) VALUES ($1, 'Test', 'Driver') RETURNING id`, email).Scan(&accountID); err != nil {
			return fmt.Errorf("insert account %s: %w", email, err)
		}
		var carIDs [carsPerAccount]string
		for i, model := range models {
			plate := fmt.Sprintf("TEST-%02d-%02d", userIndex+1, i+1)
			if err := tx.QueryRow(ctx, `INSERT INTO cars
				(account_id, type, make, name, fuel, first_registration, license_plate, is_active, purchase_date, purchase_price)
				VALUES ($1, 'Car', $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
				accountID, model.make, model.name, model.fuel, start.AddDate(-2-i, 0, 0), plate,
				i != 0, start.AddDate(0, 0, -30), 12000+i*3500).Scan(&carIDs[i]); err != nil {
				return fmt.Errorf("insert car for %s: %w", email, err)
			}
		}
		for i := 0; i < refuelsPerAccount; i++ {
			car := i % carsPerAccount
			date := distributedDate(start, now, i, refuelsPerAccount, rng)
			quantity := float64(25+rng.Intn(40)) + float64(rng.Intn(100))/100
			price := 1.25 + float64(rng.Intn(85))/100
			station, fuel := stations[i%len(stations)], "normal"
			if i%7 == 0 {
				fuel = "special"
			}
			if models[car].fuel == "electric" {
				var charge chargeKind
				quantity, price, charge = chargeSample(rng)
				station, fuel = charge.station, charge.fuel
			}
			_, err := tx.Exec(ctx, `INSERT INTO refuels (car_id, date, station, odometer_reading, fuel, quantity, amount)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, carIDs[car], date, station,
				odometer(start, date, car), fuel, quantity, float64(int(quantity*price*100))/100)
			if err != nil {
				return fmt.Errorf("insert refuel for %s: %w", email, err)
			}
		}
		for i := 0; i < repairsPerAccount; i++ {
			car := i % carsPerAccount
			date := distributedDate(start, now, i, repairsPerAccount, rng)
			_, err := tx.Exec(ctx, `INSERT INTO repairs (car_id, date, station, odometer_reading, type, amount, description)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`, carIDs[car], date, "Local garage",
				odometer(start, date, car), repairTypes[i%len(repairTypes)], 75+rng.Intn(1250), "Maintenance and parts")
			if err != nil {
				return fmt.Errorf("insert repair for %s: %w", email, err)
			}
		}
		for i := 0; i < ticketsPerAccount; i++ {
			date := distributedDate(start, now, i, ticketsPerAccount, rng)
			_, err := tx.Exec(ctx, `INSERT INTO tickets (car_id, date, type, location, amount, description)
				VALUES ($1, $2, $3, $4, $5, $6)`, carIDs[i%carsPerAccount], date,
				ticketTypes[i%len(ticketTypes)], "Vienna", 25+rng.Intn(170), "Traffic fine")
			if err != nil {
				return fmt.Errorf("insert ticket for %s: %w", email, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

// chargeKind is a charging subtype with a typical station and price range
// per kWh.
type chargeKind struct {
	fuel, station      string
	minPrice, maxPrice float64
	weight             int
}

// chargeKinds favour home and AC charging, as most electric driving does.
var chargeKinds = []chargeKind{
	{"home", "Home", 0.25, 0.32, 10},
	{"ac", "Wien Energie", 0.39, 0.55, 5},
	{"dc", "Ionity", 0.59, 0.79, 4},
	{"other", "Workplace", 0.25, 0.80, 1},
}

// chargeSample returns a charge of 15–60 kWh, its price per kWh and kind.
func chargeSample(rng *rand.Rand) (float64, float64, chargeKind) {
	total := 0
	for _, k := range chargeKinds {
		total += k.weight
	}
	pick := rng.Intn(total)
	kind := chargeKinds[len(chargeKinds)-1]
	for _, k := range chargeKinds {
		if pick < k.weight {
			kind = k
			break
		}
		pick -= k.weight
	}
	quantity := float64(15+rng.Intn(45)) + float64(rng.Intn(100))/100
	price := kind.minPrice + rng.Float64()*(kind.maxPrice-kind.minPrice)
	return quantity, price, kind
}

func distributedDate(start, end time.Time, index, total int, rng *rand.Rand) time.Time {
	span := end.Sub(start)
	segment := span / time.Duration(total)
	return start.Add(time.Duration(index)*segment + time.Duration(rng.Int63n(int64(segment))))
}

func odometer(start, date time.Time, car int) int64 {
	return int64(12000+car*5000) + int64(date.Sub(start).Hours()/24)*int64(28+car*7)
}
