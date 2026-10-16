package legacyimport

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Problem is a legacy value the import cannot represent without a
// transformation the mapping rules do not document.
type Problem struct {
	Table    string
	LegacyID string
	Column   string
	Value    *string
	Reason   string
}

func (p Problem) String() string {
	var b strings.Builder
	b.WriteString(p.Table)
	if p.LegacyID != "" {
		b.WriteString(" id=" + p.LegacyID)
	}
	if p.Column != "" {
		b.WriteString(" " + p.Column)
		if p.Value == nil {
			b.WriteString("=NULL")
		} else {
			fmt.Fprintf(&b, "=%q", *p.Value)
		}
	}
	b.WriteString(": " + p.Reason)
	return b.String()
}

// Stamps are a row's preserved creation and modification times.
type Stamps struct{ CreatedAt, UpdatedAt time.Time }

type Account struct {
	ID, Email, FirstName, LastName string
	Stamps
}

// Car mirrors the cars columns; dates are YYYY-MM-DD and the price is a
// decimal literal, as the stores pass them to Postgres.
type Car struct {
	ID, AccountID, Type, Make, Name, Fuel                             string
	FirstRegistration, LicensePlate, FIN, PurchaseDate, PurchasePrice *string
	IsActive                                                          bool
	Stamps
}

type Refuel struct {
	ID, CarID, Station, Fuel, Quantity, Amount string
	Date                                       time.Time
	OdometerReading                            *string
	Stamps
}

type Repair struct {
	ID, CarID, Station, Type, Amount string
	Date                             time.Time
	OdometerReading, Description     *string
	Stamps
}

type Ticket struct {
	ID, CarID, Type, Location, Amount string
	Date                              time.Time
	Description                       *string
	Stamps
}

// Plan is everything the import writes, with new UUIDs already assigned.
type Plan struct {
	Accounts []Account
	Cars     []Car
	Refuels  []Refuel
	Repairs  []Repair
	Tickets  []Ticket
}

var (
	carFuels    = map[string]string{"Others": "other", "Diesel": "diesel", "Gasoline": "gasoline", "Electric": "electric"}
	refuelFuels = map[string]string{"Normal": "normal", "Special": "special", "Others": "other"}
	// The legacy app had no charging subtypes; refuels of an electric car
	// map to the charging subtypes that must fit it.
	chargeFuels = map[string]string{"Normal": "ac", "Special": "dc", "Others": "other"}
	repairTypes = map[string]string{"Check": "check", "Service": "service", "Wearing part": "wearing_part", "Crash repair": "crash_repair"}
	ticketTypes = map[string]string{"Parking": "parking", "Velocity": "velocity", "Others": "other"}

	// Columns each table must declare. Users' credential columns are not
	// listed: the import never reads them.
	requiredColumns = map[string][]string{
		"Users":   {"id", "email", "firstname", "lastname", "createdAt", "updatedAt"},
		"Cars":    {"id", "UserId", "type", "carMake", "name", "fuel", "firstRegistration", "licensePlate", "fin", "isActive", "purchaseDate", "purchasePrice", "createdAt", "updatedAt"},
		"Refuels": {"id", "CarId", "date", "station", "odometerReading", "fuel", "liter", "amount", "createdAt", "updatedAt"},
		"Repairs": {"id", "CarId", "date", "station", "odometerReading", "type", "amount", "description", "createdAt", "updatedAt"},
		"Tickets": {"id", "CarId", "date", "type", "location", "amount", "description", "createdAt", "updatedAt"},
	}

	idPattern      = regexp.MustCompile(`^[0-9]+$`)
	decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
)

const legacyTimestamp = "2006-01-02 15:04:05"

// Map validates every legacy row and maps it to the v1 schema. Legacy ids
// resolve through lookup tables that live only for this call. It reports
// every problem rather than stopping at the first; the plan is only
// meaningful when there are none.
func Map(d Dump) (Plan, []Problem) {
	m := mapper{users: map[string]string{}, cars: map[string]string{}, electric: map[string]bool{}, emails: map[string]bool{}}
	if d.TimeZone != "+00:00" {
		m.problems = append(m.problems, Problem{Table: "dump", Reason: fmt.Sprintf("session time zone is %q, want \"+00:00\"", d.TimeZone)})
	}
	m.each(d, "Users", m.user)
	m.each(d, "Cars", m.car)
	m.each(d, "Refuels", m.refuel)
	m.each(d, "Repairs", m.repair)
	m.each(d, "Tickets", m.ticket)
	return m.plan, m.problems
}

type mapper struct {
	plan     Plan
	problems []Problem
	users    map[string]string // legacy user id → account UUID
	cars     map[string]string // legacy car id → car UUID
	electric map[string]bool   // car UUID → whether the car is electric
	emails   map[string]bool
}

func (m *mapper) each(d Dump, name string, mapRow func(*row)) {
	table := d.Tables[name]
	if table == nil {
		m.problems = append(m.problems, Problem{Table: name, Reason: "table missing from dump"})
		return
	}
	declared := map[string]bool{}
	for _, c := range table.Columns {
		declared[c] = true
	}
	missing := false
	for _, c := range requiredColumns[name] {
		if !declared[c] {
			m.problems = append(m.problems, Problem{Table: name, Reason: fmt.Sprintf("column %q missing from dump", c)})
			missing = true
		}
	}
	if missing {
		return
	}
	for _, values := range table.Rows {
		r := &row{m: m, table: name, values: values}
		r.id = r.legacyID("id")
		mapRow(r)
	}
}

// row reads one legacy row's columns, recording a problem for each value
// that a rule rejects.
type row struct {
	m      *mapper
	table  string
	id     string
	values Row
	failed bool
}

func (r *row) problem(column, reason string) {
	v := r.values[column]
	p := Problem{Table: r.table, LegacyID: r.id, Column: column, Reason: reason}
	if !v.Null {
		text := v.Text
		p.Value = &text
	}
	r.m.problems = append(r.m.problems, p)
	r.failed = true
}

func (r *row) legacyID(column string) string {
	v := r.values[column]
	if v.Null || !idPattern.MatchString(v.Text) {
		r.problem(column, "not a legacy id")
		return ""
	}
	return v.Text
}

// reference resolves a legacy foreign key through ids mapped earlier.
func (r *row) reference(column string, ids map[string]string, what string) string {
	v := r.values[column]
	if v.Null {
		r.problem(column, "missing "+what)
		return ""
	}
	id, ok := ids[v.Text]
	if !ok {
		r.problem(column, "unknown or unimported "+what)
	}
	return id
}

func (r *row) requiredText(column string) string {
	v := r.values[column]
	if v.Null || strings.TrimSpace(v.Text) == "" {
		r.problem(column, "required value is missing or blank")
		return ""
	}
	return v.Text
}

// optionalText maps a missing or blank value to null.
func (r *row) optionalText(column string) *string {
	v := r.values[column]
	if v.Null || strings.TrimSpace(v.Text) == "" {
		return nil
	}
	text := v.Text
	return &text
}

func (r *row) category(column string, codes map[string]string) string {
	v := r.values[column]
	code, ok := codes[v.Text]
	if v.Null || !ok {
		r.problem(column, "unknown category")
	}
	return code
}

func (r *row) timestamp(column string) time.Time {
	v := r.values[column]
	t, err := time.ParseInLocation(legacyTimestamp, v.Text, time.UTC)
	if v.Null || err != nil {
		r.problem(column, "not a timestamp")
	}
	return t
}

// optionalDate maps a midnight timestamp to its calendar date. A time of
// day is rejected rather than truncated, since it could hide a zone shift.
func (r *row) optionalDate(column string) *string {
	v := r.values[column]
	if v.Null {
		return nil
	}
	t, err := time.ParseInLocation(legacyTimestamp, v.Text, time.UTC)
	if err != nil {
		r.problem(column, "not a timestamp")
		return nil
	}
	if !t.Equal(t.Truncate(24 * time.Hour)) {
		r.problem(column, "calendar date has a time of day")
		return nil
	}
	date := t.Format(time.DateOnly)
	return &date
}

// decimal returns a plain decimal literal, or nil for NULL.
func (r *row) decimal(column string) *string {
	v := r.values[column]
	if v.Null {
		return nil
	}
	if !decimalPattern.MatchString(v.Text) {
		r.problem(column, "not a plain decimal number")
		return nil
	}
	text := v.Text
	return &text
}

func (r *row) amount(column string) string {
	d := r.decimal(column)
	switch {
	case d == nil && r.values[column].Null:
		r.problem(column, "required value is missing")
	case d != nil && isNegative(*d):
		r.problem(column, "negative")
	case d != nil:
		return *d
	}
	return ""
}

func (r *row) odometer(column string) *string {
	v := r.values[column]
	if v.Null {
		return nil
	}
	if !idPattern.MatchString(v.Text) {
		r.problem(column, "not a non-negative whole number")
		return nil
	}
	text := v.Text
	return &text
}

func (r *row) stamps() Stamps {
	return Stamps{CreatedAt: r.timestamp("createdAt"), UpdatedAt: r.timestamp("updatedAt")}
}

func isZero(decimal string) bool {
	return strings.Trim(strings.TrimPrefix(decimal, "-"), "0.") == ""
}

func isNegative(decimal string) bool {
	return strings.HasPrefix(decimal, "-") && !isZero(decimal)
}

func (m *mapper) user(r *row) {
	a := Account{ID: newUUID(), FirstName: r.values["firstname"].Text, LastName: r.values["lastname"].Text, Stamps: r.stamps()}
	if email := r.requiredText("email"); email != "" {
		a.Email = strings.ToLower(strings.TrimSpace(email))
		if m.emails[a.Email] {
			r.problem("email", "duplicate email address after normalization")
		}
		m.emails[a.Email] = true
	}
	if r.failed {
		return
	}
	m.users[r.id] = a.ID
	m.plan.Accounts = append(m.plan.Accounts, a)
}

func (m *mapper) car(r *row) {
	c := Car{
		ID:                newUUID(),
		AccountID:         r.reference("UserId", m.users, "user"),
		Type:              r.requiredText("type"),
		Make:              r.requiredText("carMake"),
		Name:              r.requiredText("name"),
		Fuel:              r.category("fuel", carFuels),
		FirstRegistration: r.optionalDate("firstRegistration"),
		LicensePlate:      r.optionalText("licensePlate"),
		FIN:               r.optionalText("fin"),
		PurchaseDate:      r.optionalDate("purchaseDate"),
		IsActive:          true,
		Stamps:            r.stamps(),
	}
	switch active := r.values["isActive"]; {
	case active.Null || active.Text == "1":
	case active.Text == "0":
		c.IsActive = false
	default:
		r.problem("isActive", "not a boolean")
	}
	if price := r.decimal("purchasePrice"); price != nil {
		switch {
		case isNegative(*price):
			r.problem("purchasePrice", "negative")
		case !isZero(*price):
			c.PurchasePrice = price
		}
	}
	if r.failed {
		return
	}
	m.cars[r.id] = c.ID
	m.electric[c.ID] = c.Fuel == "electric"
	m.plan.Cars = append(m.plan.Cars, c)
}

func (m *mapper) refuel(r *row) {
	f := Refuel{
		ID:              newUUID(),
		CarID:           r.reference("CarId", m.cars, "car"),
		Date:            r.timestamp("date"),
		Station:         r.requiredText("station"),
		OdometerReading: r.odometer("odometerReading"),
		Amount:          r.amount("amount"),
		Stamps:          r.stamps(),
	}
	fuels := refuelFuels
	if m.electric[f.CarID] {
		fuels = chargeFuels
	}
	f.Fuel = r.category("fuel", fuels)
	switch liters := r.decimal("liter"); {
	case liters == nil && r.values["liter"].Null:
		r.problem("liter", "required value is missing")
	case liters != nil && (isNegative(*liters) || isZero(*liters)):
		r.problem("liter", "not greater than zero")
	case liters != nil:
		f.Quantity = *liters
	}
	if !r.failed {
		m.plan.Refuels = append(m.plan.Refuels, f)
	}
}

func (m *mapper) repair(r *row) {
	p := Repair{
		ID:              newUUID(),
		CarID:           r.reference("CarId", m.cars, "car"),
		Date:            r.timestamp("date"),
		Station:         r.requiredText("station"),
		OdometerReading: r.odometer("odometerReading"),
		Type:            r.category("type", repairTypes),
		Amount:          r.amount("amount"),
		Description:     r.optionalText("description"),
		Stamps:          r.stamps(),
	}
	if !r.failed {
		m.plan.Repairs = append(m.plan.Repairs, p)
	}
}

func (m *mapper) ticket(r *row) {
	t := Ticket{
		ID:          newUUID(),
		CarID:       r.reference("CarId", m.cars, "car"),
		Date:        r.timestamp("date"),
		Type:        r.category("type", ticketTypes),
		Location:    r.requiredText("location"),
		Amount:      r.amount("amount"),
		Description: r.optionalText("description"),
		Stamps:      r.stamps(),
	}
	if !r.failed {
		m.plan.Tickets = append(m.plan.Tickets, t)
	}
}

// newUUID returns a random (version 4) UUID, the same shape as Postgres'
// gen_random_uuid().
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
