package attachments

import (
	"context"
	"testing"

	"at.draab/my-car/internal/cars"
	"at.draab/my-car/internal/refuels"
	"at.draab/my-car/internal/repairs"
	"at.draab/my-car/internal/tickets"
)

// TestAttachmentCountsOnEntries checks the attachmentCount each resource
// reports: a car counts only its own attachments, not its expenses'.
func TestAttachmentCountsOnEntries(t *testing.T) {
	s, pool := testStore(t)
	blobs := newMemBlobs()
	accountID, owners := testEntries(t, pool)
	ctx := context.Background()
	for _, owner := range []Owner{owners[OwnerCar], owners[OwnerCar], owners[OwnerRefuel], owners[OwnerTicket], owners[OwnerTicket], owners[OwnerTicket]} {
		if _, err := create(t, s, blobs, accountID, owner, "f.pdf"); err != nil {
			t.Fatal(err)
		}
	}

	car, err := cars.NewStore(pool).Get(ctx, accountID, owners[OwnerCar].ID)
	if err != nil || car.AttachmentCount != 2 {
		t.Errorf("car AttachmentCount = %d, %v; want 2", car.AttachmentCount, err)
	}
	carList, _, err := cars.NewStore(pool).List(ctx, accountID, nil, 10)
	if err != nil || len(carList) != 1 || carList[0].AttachmentCount != 2 {
		t.Errorf("car list = %+v, %v; want one car counting 2", carList, err)
	}
	refuel, err := refuels.NewStore(pool).Get(ctx, accountID, owners[OwnerRefuel].ID)
	if err != nil || refuel.AttachmentCount != 1 {
		t.Errorf("refuel AttachmentCount = %d, %v; want 1", refuel.AttachmentCount, err)
	}
	repair, err := repairs.NewStore(pool).Get(ctx, accountID, owners[OwnerRepair].ID)
	if err != nil || repair.AttachmentCount != 0 {
		t.Errorf("repair AttachmentCount = %d, %v; want 0", repair.AttachmentCount, err)
	}
	ticket, err := tickets.NewStore(pool).Get(ctx, accountID, owners[OwnerTicket].ID)
	if err != nil || ticket.AttachmentCount != 3 {
		t.Errorf("ticket AttachmentCount = %d, %v; want 3", ticket.AttachmentCount, err)
	}
}
