package repairs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

const (
	callerID = "11111111-1111-1111-1111-111111111111"
	carID    = "22222222-2222-2222-2222-222222222222"
	repairID = "33333333-3333-3333-3333-333333333333"
)

var stamp = time.Date(2026, 9, 27, 10, 30, 0, 0, time.UTC)

func sampleRepair() Repair {
	return Repair{ID: repairID, CarID: carID, Date: stamp, Station: "Garage", Type: "service", Amount: "120.50"}
}

type fakeRepository struct {
	calls    int
	err      error
	account  string
	filter   Filter
	after    *Cursor
	limit    int
	input    Input
	patch    Patch
	items    []Repair
	next     *Cursor
	stations []string
}

func (f *fakeRepository) List(_ context.Context, accountID string, filter Filter, after *Cursor, limit int) ([]Repair, *Cursor, error) {
	f.calls++
	f.account, f.filter, f.after, f.limit = accountID, filter, after, limit
	return f.items, f.next, f.err
}

func (f *fakeRepository) Create(_ context.Context, accountID string, in Input) (Repair, error) {
	f.calls++
	f.account, f.input = accountID, in
	if f.err != nil {
		return Repair{}, f.err
	}
	r := sampleRepair()
	r.CarID, r.OdometerReading, r.Description = in.CarID, in.OdometerReading, in.Description
	return r, nil
}

func (f *fakeRepository) Get(_ context.Context, accountID, _ string) (Repair, error) {
	f.calls++
	f.account = accountID
	if f.err != nil {
		return Repair{}, f.err
	}
	return sampleRepair(), nil
}

func (f *fakeRepository) Update(_ context.Context, accountID, _ string, p Patch) (Repair, error) {
	f.calls++
	f.account, f.patch = accountID, p
	if f.err != nil {
		return Repair{}, f.err
	}
	r := sampleRepair()
	if p.Amount != nil {
		r.Amount = *p.Amount
	}
	if p.OdometerReading.Set {
		r.OdometerReading = p.OdometerReading.Value
	}
	return r, nil
}

func (f *fakeRepository) Delete(_ context.Context, accountID, _ string) error {
	f.calls++
	f.account = accountID
	return f.err
}

func (f *fakeRepository) Stations(_ context.Context, accountID string) ([]string, error) {
	f.calls++
	f.account = accountID
	return f.stations, f.err
}

func passSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(auth.WithAccount(r.Context(), auth.Account{ID: callerID, Email: "me@example.com"})))
	})
}

func serve(t *testing.T, repo Repository, requireSession func(http.Handler) http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(repo).RegisterRoutes(mux, requireSession)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, strings.NewReader(body)))
	return rec
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

const validCreate = `{"carId":"` + carID + `","date":"2026-09-27T12:30:00+02:00","station":" Garage ","type":"service","amount":"120.50"}`

func TestCreateRepair(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/repairs", validCreate)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	in := repo.input
	if repo.account != callerID || in.CarID != carID || !in.Date.Equal(stamp) || in.Station != "Garage" ||
		in.Type != "service" || in.Amount != "120.50" || in.OdometerReading != nil || in.Description != "" {
		t.Fatalf("store input = %+v for %s", in, repo.account)
	}
	body := decodeMap(t, rec)
	for _, key := range []string{"id", "carId", "date", "station", "odometerReading", "type", "amount", "description"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("repair member %q missing: %v", key, body)
		}
	}
	if body["odometerReading"] != nil || body["description"] != "" || body["date"] != "2026-09-27T10:30:00Z" {
		t.Fatalf("body = %v", body)
	}
}

func TestCreateRepairWithOptionalFields(t *testing.T) {
	repo := &fakeRepository{}
	body := strings.Replace(validCreate, `{`, `{"odometerReading":0,"description":" Brakes ",`, 1)
	body = strings.Replace(body, `"120.50"`, `"0"`, 1)
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/repairs", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	in := repo.input
	if in.OdometerReading == nil || *in.OdometerReading != 0 || in.Description != "Brakes" || in.Amount != "0" {
		t.Fatalf("store input = %+v", in)
	}
}

func TestCreateOnUnownedCarIsNotFound(t *testing.T) {
	repo := &fakeRepository{err: ErrNotFound}
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/repairs", validCreate)
	if rec.Code != http.StatusNotFound || decodeMap(t, rec)["code"] != apierror.CodeNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestRepairValidation(t *testing.T) {
	cases := []struct {
		name, method, body string
		want               map[string]string
	}{
		{"unsupported type", http.MethodPost, strings.Replace(validCreate, `"service"`, `"tuning"`, 1), map[string]string{"type": apierror.ReasonInvalidEnum}},
		{"unsupported type on update", http.MethodPatch, `{"type":"tuning"}`, map[string]string{"type": apierror.ReasonInvalidEnum}},
		{"undocumented member on create", http.MethodPost, strings.Replace(validCreate, `{`, `{"liters":"3",`, 1), map[string]string{"liters": apierror.ReasonUnknown}},
		{"undocumented member on update", http.MethodPatch, `{"station":"x","id":"y"}`, map[string]string{"id": apierror.ReasonUnknown}},
		{"car is fixed on update", http.MethodPatch, `{"carId":"` + carID + `"}`, map[string]string{"carId": apierror.ReasonUnknown}},
		{"missing required members", http.MethodPost, `{"description":"x"}`, map[string]string{"carId": apierror.ReasonRequired, "date": apierror.ReasonRequired, "station": apierror.ReasonRequired, "type": apierror.ReasonRequired, "amount": apierror.ReasonRequired}},
		{"blank station", http.MethodPatch, `{"station":"  "}`, map[string]string{"station": apierror.ReasonEmpty}},
		{"null required member", http.MethodPatch, `{"amount":null}`, map[string]string{"amount": apierror.ReasonInvalidType}},
		{"null description", http.MethodPatch, `{"description":null}`, map[string]string{"description": apierror.ReasonInvalidType}},
		{"wrong types", http.MethodPatch, `{"odometerReading":"12","station":3,"amount":12}`, map[string]string{"odometerReading": apierror.ReasonInvalidType, "station": apierror.ReasonInvalidType, "amount": apierror.ReasonInvalidType}},
		{"fractional odometer", http.MethodPatch, `{"odometerReading":12.5}`, map[string]string{"odometerReading": apierror.ReasonInvalidType}},
		{"malformed car id", http.MethodPost, strings.Replace(validCreate, carID, "car-1", 1), map[string]string{"carId": apierror.ReasonInvalidType}},
		{"malformed dates", http.MethodPatch, `{"date":"2026-09-27"}`, map[string]string{"date": apierror.ReasonInvalidType}},
		{"invalid decimal", http.MethodPatch, `{"amount":"12,50"}`, map[string]string{"amount": apierror.ReasonInvalidDecimal}},
		{"negative amount", http.MethodPatch, `{"amount":"-0.01"}`, map[string]string{"amount": apierror.ReasonNegative}},
		{"negative odometer", http.MethodPatch, `{"odometerReading":-1}`, map[string]string{"odometerReading": apierror.ReasonNegative}},
		{"empty update", http.MethodPatch, `{}`, map[string]string{apierror.BodyField: apierror.ReasonRequired}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			target := "/api/v1/repairs"
			if tc.method == http.MethodPatch {
				target += "/" + repairID
			}
			rec := serve(t, repo, passSession, tc.method, target, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
			if repo.calls != 0 {
				t.Fatalf("store called %d times on a rejected request", repo.calls)
			}
			body := decodeMap(t, rec)
			fields, _ := body["fields"].(map[string]any)
			if body["code"] != apierror.CodeValidationFailed || len(fields) != len(tc.want) {
				t.Fatalf("body = %v, want fields %v", body, tc.want)
			}
			for k, v := range tc.want {
				if fields[k] != v {
					t.Fatalf("fields[%q] = %v, want %q (all: %v)", k, fields[k], v, fields)
				}
			}
		})
	}
}

func TestMalformedBodyIsRejected(t *testing.T) {
	for _, body := range []string{`not json`, `[]`, `null`} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/repairs", body)
		if rec.Code != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("%q: status = %d, calls = %d", body, rec.Code, repo.calls)
		}
	}
}

func TestUpdateAppliesZeroAndClearsOdometer(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/repairs/"+repairID, `{"amount":"0","odometerReading":null,"description":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	p := repo.patch
	if *p.Amount != "0" || !p.OdometerReading.Set || p.OdometerReading.Value != nil || *p.Description != "" || p.Station != nil || p.Date != nil {
		t.Fatalf("patch = %+v", p)
	}
	if body := decodeMap(t, rec); body["amount"] != "0" || body["odometerReading"] != nil {
		t.Fatalf("body = %v", body)
	}
}

func TestGetRepair(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/repairs/"+repairID, "")
	if rec.Code != http.StatusOK || repo.account != callerID {
		t.Fatalf("status = %d, account = %s", rec.Code, repo.account)
	}
	if body := decodeMap(t, rec); body["id"] != repairID || body["carId"] != carID || body["station"] != "Garage" {
		t.Fatalf("body = %v", body)
	}
}

func TestDeleteRepair(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodDelete, "/api/v1/repairs/"+repairID, "")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || repo.calls != 1 {
		t.Fatalf("status = %d, body = %q, calls = %d", rec.Code, rec.Body, repo.calls)
	}
}

func TestUnownedOrMissingRepairIsNotFound(t *testing.T) {
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPatch, `{"station":"x"}`},
		{http.MethodDelete, ""},
	} {
		repo := &fakeRepository{err: ErrNotFound}
		rec := serve(t, repo, passSession, tc.method, "/api/v1/repairs/"+repairID, tc.body)
		if rec.Code != http.StatusNotFound || decodeMap(t, rec)["code"] != apierror.CodeNotFound {
			t.Fatalf("%s: status = %d, body = %s", tc.method, rec.Code, rec.Body)
		}

		repo = &fakeRepository{}
		rec = serve(t, repo, passSession, tc.method, "/api/v1/repairs/not-a-uuid", tc.body)
		if rec.Code != http.StatusNotFound || repo.calls != 0 {
			t.Fatalf("%s invalid id: status = %d, calls = %d", tc.method, rec.Code, repo.calls)
		}
	}
}

func TestStoreFailureIsInternalError(t *testing.T) {
	repo := &fakeRepository{err: errors.New("boom")}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/repairs/"+repairID, "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestListRepairs(t *testing.T) {
	next := &Cursor{Date: stamp, ID: repairID}
	repo := &fakeRepository{items: []Repair{sampleRepair()}, next: next}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/repairs", "")
	if rec.Code != http.StatusOK || repo.limit != defaultLimit || repo.after != nil || repo.account != callerID || repo.filter != (Filter{}) {
		t.Fatalf("status = %d, limit = %d, after = %v, filter = %+v", rec.Code, repo.limit, repo.after, repo.filter)
	}
	body := decodeMap(t, rec)
	items, _ := body["items"].([]any)
	if len(items) != 1 || body["nextCursor"] != next.Encode() {
		t.Fatalf("body = %v", body)
	}

	repo = &fakeRepository{}
	target := "/api/v1/repairs?limit=100&cursor=" + next.Encode() + "&carId=" + carID + "&from=2026-01-01T00:00:00Z&to=2026-07-01T00:00:00%2B02:00"
	rec = serve(t, repo, passSession, http.MethodGet, target, "")
	f := repo.filter
	if rec.Code != http.StatusOK || repo.limit != 100 || repo.after == nil || repo.after.ID != repairID || !repo.after.Date.Equal(stamp) ||
		f.CarID == nil || *f.CarID != carID || !f.From.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) ||
		!f.To.Equal(time.Date(2026, 6, 30, 22, 0, 0, 0, time.UTC)) {
		t.Fatalf("status = %d, limit = %d, after = %v, filter = %+v", rec.Code, repo.limit, repo.after, f)
	}
	body = decodeMap(t, rec)
	if items, ok := body["items"].([]any); !ok || len(items) != 0 || body["nextCursor"] != nil {
		t.Fatalf("empty page body = %v", body)
	}
}

func TestListRejectsBadQuery(t *testing.T) {
	for _, query := range []string{
		"limit=0", "limit=101", "limit=ten", "cursor=%21%21", "cursor=" + (Cursor{Date: stamp, ID: "x"}).Encode(),
		"carId=car-1", "from=2026-01-01", "to=yesterday",
	} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/repairs?"+query, "")
		if rec.Code != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("%s: status = %d, calls = %d", query, rec.Code, repo.calls)
		}
	}
}

func TestListStations(t *testing.T) {
	repo := &fakeRepository{stations: []string{"Alpha", "Zeta"}}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/repairs/stations", "")
	if rec.Code != http.StatusOK || repo.account != callerID {
		t.Fatalf("status = %d, account = %s", rec.Code, repo.account)
	}
	var stations []string
	if err := json.Unmarshal(rec.Body.Bytes(), &stations); err != nil || len(stations) != 2 || stations[0] != "Alpha" {
		t.Fatalf("body = %s, err = %v", rec.Body, err)
	}
}

// noSessionRepository satisfies auth.Repository for a service whose session
// lookup always fails; no other method is reached by these requests.
type noSessionRepository struct{ auth.Repository }

func (noSessionRepository) Session(context.Context, []byte, time.Time) (auth.Session, error) {
	return auth.Session{}, auth.ErrInvalidCredential
}

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// Mounted on the top-level mux with the real session middleware, every
// repair route answers 401 rather than the /api/ 404 fallback.
func TestRepairsRequireSessionOnTopLevelMux(t *testing.T) {
	requireSession := auth.NewService(noSessionRepository{}, nil, nil, "http://localhost").RequireSession
	repo := &fakeRepository{}
	mux := httpserver.NewMux(okPinger{}, nil, fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, func(mux *http.ServeMux) {
		NewHandler(repo).RegisterRoutes(mux, requireSession)
	})
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/api/v1/repairs"},
		{http.MethodPost, "/api/v1/repairs"},
		{http.MethodGet, "/api/v1/repairs/stations"},
		{http.MethodGet, "/api/v1/repairs/" + repairID},
		{http.MethodPatch, "/api/v1/repairs/" + repairID},
		{http.MethodDelete, "/api/v1/repairs/" + repairID},
	} {
		for _, cookie := range []bool{false, true} {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(validCreate))
			if cookie {
				req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "stale"})
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized || repo.calls != 0 {
				t.Fatalf("%s %s cookie=%v: status = %d, calls = %d", tc.method, tc.target, cookie, rec.Code, repo.calls)
			}
		}
	}
}

func TestAttachmentCountIsReadOnly(t *testing.T) {
	repo := &fakeRepository{}
	body := strings.TrimSuffix(validCreate, "}") + `,"attachmentCount":1}`
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/repairs", body)
	fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["attachmentCount"] != "unknown" {
		t.Fatalf("status = %d, body = %s; want 400 with attachmentCount unknown", rec.Code, rec.Body)
	}
}
