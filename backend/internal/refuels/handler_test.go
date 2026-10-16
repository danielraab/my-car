package refuels

import (
	"at.draab/my-car/internal/auth"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const callerID = "11111111-1111-1111-1111-111111111111"
const carID = "22222222-2222-2222-2222-222222222222"
const refuelID = "33333333-3333-3333-3333-333333333333"

var stamp = time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)

type fakeRepository struct {
	items    []Refuel
	previous map[string]int64
	input    Input
	patch    Patch
	filter   Filter
	err      error
}

func (f *fakeRepository) List(_ context.Context, _ string, x Filter, _ *Cursor, _ int) ([]Refuel, *Cursor, error) {
	f.filter = x
	return f.items, nil, f.err
}
func (f *fakeRepository) Chart(_ context.Context, _ string, x Filter) ([]Refuel, map[string]int64, error) {
	f.filter = x
	return f.items, f.previous, f.err
}
func (f *fakeRepository) Create(_ context.Context, _ string, x Input) (Refuel, error) {
	f.input = x
	return Refuel{ID: refuelID, CarID: x.CarID, Date: x.Date, Station: x.Station, OdometerReading: x.OdometerReading, Fuel: x.Fuel, Quantity: x.Quantity, Amount: x.Amount}, f.err
}
func (f *fakeRepository) Get(context.Context, string, string) (Refuel, error) {
	if f.err != nil {
		return Refuel{}, f.err
	}
	return f.items[0], f.err
}
func (f *fakeRepository) Update(_ context.Context, _ string, _ string, x Patch) (Refuel, error) {
	f.patch = x
	if f.err != nil {
		return Refuel{}, f.err
	}
	return f.items[0], nil
}
func (f *fakeRepository) Delete(context.Context, string, string) error { return f.err }
func (f *fakeRepository) Stations(context.Context, string) ([]string, error) {
	return []string{"A", "B"}, f.err
}
func pass(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(auth.WithAccount(r.Context(), auth.Account{ID: callerID, Email: "x@y.z"})))
	})
}
func serve(t *testing.T, repo Repository, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	m := http.NewServeMux()
	NewHandler(repo).RegisterRoutes(m, pass)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(method, target, strings.NewReader(body)))
	return w
}
func sample(odo *int64) Refuel {
	return Refuel{ID: refuelID, CarID: carID, Date: stamp, Station: "Fuel", OdometerReading: odo, Fuel: "normal", Quantity: "40", Amount: "60"}
}
func TestCreateAndValidation(t *testing.T) {
	f := &fakeRepository{}
	w := serve(t, f, "POST", "/api/v1/refuels", `{"carId":"`+carID+`","date":"2026-09-28T10:00:00Z","station":" Fuel ","fuel":"normal","quantity":"40","amount":"60"}`)
	if w.Code != 201 || f.input.Quantity != "40" {
		t.Fatalf("code=%d input=%+v body=%s", w.Code, f.input, w.Body.String())
	}
	w = serve(t, f, "POST", "/api/v1/refuels", `{"carId":"`+carID+`","date":"2026-09-28T10:00:00Z","station":"Fuel","fuel":"normal","quantity":"0","amount":"60"}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "non_positive") {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	w = serve(t, f, "POST", "/api/v1/refuels", `{"unknown":true}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "unknown") {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}
func TestChartDerivedValuesAcrossCars(t *testing.T) {
	a, b := int64(1000), int64(1500)
	other := int64(2000)
	f := &fakeRepository{items: []Refuel{sample(&a), {ID: "44444444-4444-4444-4444-444444444444", CarID: "55555555-5555-5555-5555-555555555555", Date: stamp, Station: "X", OdometerReading: &other, Fuel: "special", Quantity: "20", Amount: "30"}, {ID: "66666666-6666-6666-6666-666666666666", CarID: carID, Date: stamp.Add(time.Hour), Station: "Y", OdometerReading: &b, Fuel: "normal", Quantity: "40", Amount: "60"}}}
	w := serve(t, f, "GET", "/api/v1/refuels/chart", "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var got struct {
		Items []refuelBody `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Items[0].UnitPrice != "1.5" || got.Items[0].Consumption != nil || got.Items[2].Distance == nil || *got.Items[2].Distance != 500 || *got.Items[2].Consumption != "8" {
		t.Fatalf("%+v", got.Items)
	}
}
func TestChartSeedsPredecessorBeforeFrom(t *testing.T) {
	first, second := int64(1500), int64(1900)
	otherCar := "55555555-5555-5555-5555-555555555555"
	f := &fakeRepository{
		items: []Refuel{
			sample(&first),
			{ID: "44444444-4444-4444-4444-444444444444", CarID: otherCar, Date: stamp, Station: "X", OdometerReading: &second, Fuel: "special", Quantity: "20", Amount: "30"},
		},
		// Only carID has a refuel before the range; otherCar's first refuel
		// in range is its first ever.
		previous: map[string]int64{carID: 1000},
	}
	w := serve(t, f, "GET", "/api/v1/refuels/chart?from=2026-01-01T00:00:00Z", "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var got struct {
		Items []refuelBody `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if f.filter.From == nil || !f.filter.From.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", f.filter.From)
	}
	if got.Items[0].Distance == nil || *got.Items[0].Distance != 500 || got.Items[0].Consumption == nil || *got.Items[0].Consumption != "8" {
		t.Fatalf("seeded refuel = %+v", got.Items[0])
	}
	if got.Items[1].Distance != nil || got.Items[1].Consumption != nil {
		t.Fatalf("first-ever refuel = %+v", got.Items[1])
	}
}
func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{Date: stamp, ID: refuelID}
	got, ok := DecodeCursor(c.Encode())
	if !ok || got.ID != c.ID || !got.Date.Equal(c.Date) {
		t.Fatalf("%+v %v", got, ok)
	}
}

func TestAuthenticationAndNotFound(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(&fakeRepository{}).RegisterRoutes(mux, func(next http.Handler) http.Handler { return next })
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/refuels", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated code = %d", w.Code)
	}
	w = serve(t, &fakeRepository{err: ErrNotFound}, "GET", "/api/v1/refuels/"+refuelID, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing code = %d", w.Code)
	}
}

func TestFormerMemberNamesAreUnknown(t *testing.T) {
	f := &fakeRepository{}
	w := serve(t, f, "POST", "/api/v1/refuels", `{"carId":"`+carID+`","date":"2026-09-28T10:00:00Z","station":"Fuel","fuel":"normal","liters":"40","amount":"60"}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"liters":"unknown"`) || f.input.CarID != "" {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestChargeBody(t *testing.T) {
	a, b := int64(1000), int64(1200)
	f := &fakeRepository{items: []Refuel{
		{ID: refuelID, CarID: carID, Date: stamp, Station: "Home", OdometerReading: &a, Fuel: "home", Quantity: "40", Amount: "14.00"},
		{ID: "66666666-6666-6666-6666-666666666666", CarID: carID, Date: stamp.Add(time.Hour), Station: "Ionity", OdometerReading: &b, Fuel: "dc", Quantity: "34.4", Amount: "20"},
	}}
	w := serve(t, f, "GET", "/api/v1/refuels/chart", "")
	if strings.Contains(w.Body.String(), "liters") || strings.Contains(w.Body.String(), "perLiter") {
		t.Fatalf("body still carries former members: %s", w.Body.String())
	}
	var got struct{ Items []refuelBody }
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Items[0].UnitPrice != "0.35" || *got.Items[1].Distance != 200 || *got.Items[1].Consumption != "17.2" {
		t.Fatalf("%+v", got.Items)
	}
}

func TestChargingSubtypesAreAcceptedAndMismatchIsReported(t *testing.T) {
	f := &fakeRepository{}
	w := serve(t, f, "POST", "/api/v1/refuels", `{"carId":"`+carID+`","date":"2026-09-28T10:00:00Z","station":"Ionity","fuel":"dc","quantity":"40","amount":"20"}`)
	if w.Code != 201 || f.input.Fuel != "dc" {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	f = &fakeRepository{err: ErrFuelMismatch}
	w = serve(t, f, "POST", "/api/v1/refuels", `{"carId":"`+carID+`","date":"2026-09-28T10:00:00Z","station":"Shell","fuel":"special","quantity":"40","amount":"60"}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"fuel":"fuel_mismatch"`) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	w = serve(t, f, "PATCH", "/api/v1/refuels/"+refuelID, `{"fuel":"home"}`)
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"fuel":"fuel_mismatch"`) {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestFitsCarFuel(t *testing.T) {
	for _, tc := range []struct {
		carFuel, subtype string
		want             bool
	}{
		{"electric", "home", true}, {"electric", "ac", true}, {"electric", "dc", true}, {"electric", "other", true},
		{"electric", "normal", false}, {"electric", "special", false},
		{"diesel", "normal", true}, {"gasoline", "special", true}, {"other", "other", true},
		{"diesel", "home", false}, {"gasoline", "ac", false}, {"other", "dc", false},
	} {
		if got := FitsCarFuel(tc.carFuel, tc.subtype); got != tc.want {
			t.Errorf("FitsCarFuel(%q, %q) = %v, want %v", tc.carFuel, tc.subtype, got, tc.want)
		}
	}
}
