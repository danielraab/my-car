package profile

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"at.draab/my-car/internal/auth"
)

// Store persists profile changes in the accounts table.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// UpdateProfile overwrites each supplied member (empty names included) and
// returns the updated account. Display settings only relabel values, so no
// car or expense row is touched.
func (s *Store) UpdateProfile(ctx context.Context, accountID string, u Update) (auth.Account, error) {
	var a auth.Account
	err := s.pool.QueryRow(ctx, `UPDATE accounts SET first_name=COALESCE($2,first_name), last_name=COALESCE($3,last_name), currency=COALESCE($4,currency), volume_unit=COALESCE($5,volume_unit), distance_unit=COALESCE($6,distance_unit), updated_at=now() WHERE id=$1 RETURNING `+auth.AccountColumns, accountID, u.FirstName, u.LastName, u.Currency, u.VolumeUnit, u.DistanceUnit).Scan(a.ScanTargets()...)
	return a, err
}
