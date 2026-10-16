package legacyimport

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMapFixture(t *testing.T) {
	plan, problems := Map(parseFixture(t))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if len(plan.Accounts) != 2 || len(plan.Cars) != 3 || len(plan.Refuels) != 3 || len(plan.Repairs) != 2 || len(plan.Tickets) != 2 {
		t.Fatalf("plan sizes = %d/%d/%d/%d/%d", len(plan.Accounts), len(plan.Cars), len(plan.Refuels), len(plan.Repairs), len(plan.Tickets))
	}

	ada, grace := plan.Accounts[0], plan.Accounts[1]
	if ada.Email != "ada@example.org" || ada.FirstName != "Ada" || grace.FirstName != "" || grace.LastName != "" {
		t.Errorf("accounts = %+v", plan.Accounts)
	}
	if want := time.Date(2024, 2, 2, 17, 21, 0, 0, time.UTC); !grace.UpdatedAt.Equal(want) {
		t.Errorf("updatedAt = %v, want %v", grace.UpdatedAt, want)
	}

	golf, octavia, ka := plan.Cars[0], plan.Cars[1], plan.Cars[2]
	if golf.AccountID != ada.ID || octavia.AccountID != ada.ID || ka.AccountID != grace.ID {
		t.Errorf("car owners = %s %s %s", golf.AccountID, octavia.AccountID, ka.AccountID)
	}
	if golf.Make != "VW" || golf.Fuel != "diesel" || *golf.FirstRegistration != "2003-08-01" || golf.LicensePlate != nil || *golf.PurchasePrice != "1000" || !golf.IsActive {
		t.Errorf("golf = %+v", golf)
	}
	if octavia.FIN != nil || octavia.IsActive || *octavia.PurchaseDate != "2021-10-01" || *octavia.PurchasePrice != "11100.5" || *octavia.LicensePlate != "W-12345X" {
		t.Errorf("octavia = %+v", octavia)
	}
	if ka.FirstRegistration != nil || ka.LicensePlate != nil || ka.PurchasePrice != nil || !ka.IsActive {
		t.Errorf("ka = %+v", ka)
	}

	for i, carID := range []string{golf.ID, octavia.ID, ka.ID} {
		if plan.Refuels[i].CarID != carID {
			t.Errorf("refuel %d car = %s, want %s", i, plan.Refuels[i].CarID, carID)
		}
	}
	if r := plan.Refuels[1]; r.Fuel != "special" || r.OdometerReading != nil || r.Quantity != "30" || r.Amount != "55.9" {
		t.Errorf("refuel = %+v", r)
	}
	if r := plan.Repairs[0]; r.Type != "wearing_part" || *r.OdometerReading != "121000" || r.CarID != golf.ID {
		t.Errorf("repair = %+v", r)
	}
	if r := plan.Repairs[1]; r.Description != nil || r.Amount != "0" {
		t.Errorf("repair with blank description = %+v", r)
	}
	if tk := plan.Tickets[1]; tk.Type != "other" || tk.CarID != ka.ID || *tk.Description != "Toll sticker" {
		t.Errorf("ticket = %+v", tk)
	}

	uuid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}
	for _, id := range []string{ada.ID, grace.ID, golf.ID, octavia.ID, ka.ID, plan.Refuels[0].ID, plan.Tickets[0].ID} {
		if !uuid.MatchString(id) || seen[id] {
			t.Errorf("id %q is not a fresh v4 UUID", id)
		}
		seen[id] = true
	}
}

// mutate returns the fixture with one value of one legacy row replaced.
func mutate(t *testing.T, table string, index int, column string, v Value) Dump {
	t.Helper()
	d := parseFixture(t)
	d.Tables[table].Rows[index][column] = v
	return d
}

func text(s string) Value { return Value{Text: s} }

var null = Value{Null: true}

func TestMapRejectsUnrepresentableValues(t *testing.T) {
	cases := []struct {
		name, table string
		index       int
		column      string
		value       Value
		want        string
	}{
		{"unknown refuel fuel", "Refuels", 0, "fuel", text("Premium"), `Refuels id=1 fuel="Premium": unknown category`},
		{"missing refuel fuel", "Refuels", 0, "fuel", null, `Refuels id=1 fuel=NULL: unknown category`},
		{"unknown car fuel", "Cars", 0, "fuel", text("diesel"), `Cars id=1 fuel="diesel": unknown category`},
		{"unknown repair type", "Repairs", 1, "type", text("Wearing Part"), `Repairs id=2 type="Wearing Part": unknown category`},
		{"unknown ticket type", "Tickets", 0, "type", text("Other"), `Tickets id=1 type="Other": unknown category`},
		{"blank car name", "Cars", 0, "name", text("  "), `Cars id=1 name="  ": required value is missing or blank`},
		{"missing make", "Cars", 1, "carMake", null, `Cars id=2 carMake=NULL: required value is missing or blank`},
		{"blank station", "Refuels", 2, "station", text(""), `Refuels id=3 station="": required value is missing or blank`},
		{"blank location", "Tickets", 1, "location", text(" "), `Tickets id=2 location=" ": required value is missing or blank`},
		{"missing email", "Users", 1, "email", null, `Users id=2 email=NULL: required value is missing or blank`},
		{"missing user", "Cars", 2, "UserId", null, `Cars id=3 UserId=NULL: missing user`},
		{"unknown user", "Cars", 2, "UserId", text("9"), `Cars id=3 UserId="9": unknown or unimported user`},
		{"unknown car", "Repairs", 0, "CarId", text("42"), `Repairs id=1 CarId="42": unknown or unimported car`},
		{"missing car", "Tickets", 0, "CarId", null, `Tickets id=1 CarId=NULL: missing car`},
		{"registration with time of day", "Cars", 0, "firstRegistration", text("2003-07-31 22:00:00"), `Cars id=1 firstRegistration="2003-07-31 22:00:00": calendar date has a time of day`},
		{"bad timestamp", "Refuels", 0, "date", text("0000-00-00 00:00:00"), `Refuels id=1 date="0000-00-00 00:00:00": not a timestamp`},
		{"exponent literal", "Refuels", 0, "amount", text("1.5e-7"), `Refuels id=1 amount="1.5e-7": not a plain decimal number`},
		{"zero liters", "Refuels", 1, "liter", text("0"), `Refuels id=2 liter="0": not greater than zero`},
		{"missing liters", "Refuels", 1, "liter", null, `Refuels id=2 liter=NULL: required value is missing`},
		{"negative amount", "Tickets", 0, "amount", text("-5"), `Tickets id=1 amount="-5": negative`},
		{"missing repair amount", "Repairs", 0, "amount", null, `Repairs id=1 amount=NULL: required value is missing`},
		{"negative price", "Cars", 0, "purchasePrice", text("-1"), `Cars id=1 purchasePrice="-1": negative`},
		{"bad active flag", "Cars", 0, "isActive", text("2"), `Cars id=1 isActive="2": not a boolean`},
		{"negative odometer", "Repairs", 0, "odometerReading", text("-3"), `Repairs id=1 odometerReading="-3": not a non-negative whole number`},
		{"bad legacy id", "Tickets", 1, "id", text("x"), `Tickets id="x": not a legacy id`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, problems := Map(mutate(t, tc.table, tc.index, tc.column, tc.value))
			if len(problems) == 0 {
				t.Fatal("no problem reported")
			}
			if got := problems[0].String(); got != tc.want {
				t.Fatalf("problem = %q, want %q (all: %v)", got, tc.want, problems)
			}
		})
	}
}

func TestMapDocumentedTransformations(t *testing.T) {
	for _, price := range []string{"0", "0.0", "-0"} {
		plan, problems := Map(mutate(t, "Cars", 0, "purchasePrice", text(price)))
		if len(problems) != 0 || plan.Cars[0].PurchasePrice != nil {
			t.Errorf("price %q: car = %+v, problems = %v", price, plan.Cars[0], problems)
		}
	}
	plan, problems := Map(mutate(t, "Cars", 1, "licensePlate", text("  ")))
	if len(problems) != 0 || plan.Cars[1].LicensePlate != nil {
		t.Errorf("blank plate: car = %+v, problems = %v", plan.Cars[1], problems)
	}
	plan, problems = Map(mutate(t, "Tickets", 1, "description", text(" ")))
	if len(problems) != 0 || plan.Tickets[1].Description != nil {
		t.Errorf("blank description: ticket = %+v, problems = %v", plan.Tickets[1], problems)
	}
	plan, problems = Map(mutate(t, "Users", 1, "email", text("ADA@example.org")))
	// The rejected user's car and its expenses are reported as unowned too.
	if len(problems) == 0 || problems[0].String() != `Users id=2 email="ADA@example.org": duplicate email address after normalization` {
		t.Errorf("duplicate email problems = %v", problems)
	}
}

func TestMapCollectsEveryProblem(t *testing.T) {
	d := parseFixture(t)
	d.Tables["Refuels"].Rows[0]["fuel"] = text("Premium")
	d.Tables["Refuels"].Rows[2]["liter"] = text("0")
	d.Tables["Tickets"].Rows[1]["type"] = text("Toll")
	_, problems := Map(d)
	if len(problems) != 3 {
		t.Fatalf("problems = %v, want 3", problems)
	}
}

func TestMapReportsDumpLevelProblems(t *testing.T) {
	d := parseFixture(t)
	d.TimeZone = "SYSTEM"
	delete(d.Tables, "Tickets")
	d.Tables["Repairs"].Columns = d.Tables["Repairs"].Columns[:6]
	_, problems := Map(d)
	var lines []string
	for _, p := range problems {
		lines = append(lines, p.String())
	}
	got := strings.Join(lines, "\n")
	for _, want := range []string{
		`dump: session time zone is "SYSTEM", want "+00:00"`,
		`Tickets: table missing from dump`,
		`Repairs: column "description" missing from dump`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("problems lack %q:\n%s", want, got)
		}
	}
}

func TestProblemsNeverContainCredentials(t *testing.T) {
	d := parseFixture(t)
	for _, r := range d.Tables["Users"].Rows {
		r["email"] = null
		r["createdAt"] = text("bad")
	}
	_, problems := Map(d)
	if len(problems) == 0 {
		t.Fatal("expected problems")
	}
	for _, p := range problems {
		if s := p.String(); strings.Contains(s, "SECRET") {
			t.Fatalf("problem leaks a credential: %s", s)
		}
	}
}

func TestMapRefuelsOfElectricCarToChargingSubtypes(t *testing.T) {
	plan, problems := Map(mutate(t, "Cars", 1, "fuel", text("Electric")))
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if plan.Cars[1].Fuel != "electric" {
		t.Fatalf("car fuel = %s, want electric", plan.Cars[1].Fuel)
	}
	if r := plan.Refuels[1]; r.CarID != plan.Cars[1].ID || r.Fuel != "dc" || r.Quantity != "30" {
		t.Errorf("charge = %+v, want subtype dc and quantity 30", r)
	}
	if r := plan.Refuels[0]; r.Fuel != "normal" {
		t.Errorf("refuel of a diesel car = %+v, want subtype normal", r)
	}
}
