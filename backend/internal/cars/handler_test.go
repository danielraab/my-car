package cars

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/config"
	"at.draab/my-car/internal/httpserver"
)

const (
	callerID = "11111111-1111-1111-1111-111111111111"
	carID    = "22222222-2222-2222-2222-222222222222"
)

var stamp = time.Date(2026, 9, 27, 10, 0, 0, 123456000, time.UTC)

func sampleCar() Car {
	return Car{ID: carID, Type: "Hatchback", Make: "VW", Name: "Golf", Fuel: "diesel", FirstRegistration: ptr("2019-03-01"),
		LicensePlate: ptr("W-123AB"), IsActive: true, CreatedAt: stamp, UpdatedAt: stamp}
}

type fakeRepository struct {
	calls   int
	err     error
	account string
	after   *Cursor
	limit   int
	input   Input
	patch   Patch
	items   []Car
	next    *Cursor
}

func (f *fakeRepository) List(_ context.Context, accountID string, after *Cursor, limit int) ([]Car, *Cursor, error) {
	f.calls++
	f.account, f.after, f.limit = accountID, after, limit
	return f.items, f.next, f.err
}

func (f *fakeRepository) Create(_ context.Context, accountID string, in Input) (Car, error) {
	f.calls++
	f.account, f.input = accountID, in
	if f.err != nil {
		return Car{}, f.err
	}
	c := sampleCar()
	c.Name, c.Fuel, c.FIN, c.IsActive = in.Name, in.Fuel, in.FIN, in.IsActive
	c.FirstRegistration, c.LicensePlate = in.FirstRegistration, in.LicensePlate
	c.DeregistrationDate, c.Information = in.DeregistrationDate, in.Information
	return c, nil
}

func (f *fakeRepository) Get(_ context.Context, accountID, _ string) (Car, error) {
	f.calls++
	f.account = accountID
	if f.err != nil {
		return Car{}, f.err
	}
	return sampleCar(), nil
}

func (f *fakeRepository) Update(_ context.Context, accountID, _ string, p Patch) (Car, error) {
	f.calls++
	f.account, f.patch = accountID, p
	if f.err != nil {
		return Car{}, f.err
	}
	c := sampleCar()
	if p.IsActive != nil {
		c.IsActive = *p.IsActive
	}
	for _, field := range []struct {
		patch  Nullable
		column **string
	}{{p.FirstRegistration, &c.FirstRegistration}, {p.LicensePlate, &c.LicensePlate}, {p.FIN, &c.FIN},
		{p.DeregistrationDate, &c.DeregistrationDate}, {p.Information, &c.Information}} {
		if field.patch.Set {
			*field.column = field.patch.Value
		}
	}
	return c, nil
}

func (f *fakeRepository) Delete(_ context.Context, accountID, _ string) error {
	f.calls++
	f.account = accountID
	return f.err
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

const validCreate = `{"type":" Hatchback ","make":"VW","name":"Golf","fuel":"diesel","firstRegistration":"2019-03-01","licensePlate":"W-123AB"}`

func TestCreateCar(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", validCreate)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if repo.account != callerID || repo.input.Type != "Hatchback" || !repo.input.IsActive || repo.input.FIN != nil {
		t.Fatalf("store input = %+v for %s", repo.input, repo.account)
	}
	body := decodeMap(t, rec)
	for _, key := range []string{"id", "type", "make", "name", "fuel", "firstRegistration", "licensePlate", "fin", "isActive", "purchaseDate", "purchasePrice", "deregistrationDate", "information", "createdAt", "updatedAt"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("car member %q missing: %v", key, body)
		}
	}
	if body["fin"] != nil || body["createdAt"] != "2026-09-27T10:00:00.123456Z" {
		t.Fatalf("body = %v", body)
	}
}

func TestCreateCarWithOptionalFields(t *testing.T) {
	repo := &fakeRepository{}
	body := `{"type":"Hatchback","make":"VW","name":"Golf","fuel":"electric","firstRegistration":"2019-03-01","licensePlate":"W-1","fin":" ABC ","isActive":false,"purchaseDate":"2019-04-15","purchasePrice":"18500.50"}`
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	in := repo.input
	if in.IsActive || *in.FIN != "ABC" || *in.PurchaseDate != "2019-04-15" || *in.PurchasePrice != "18500.50" || in.Fuel != "electric" {
		t.Fatalf("store input = %+v", in)
	}
}

func TestCreateCarWithoutRegistrationDetails(t *testing.T) {
	for _, body := range []string{
		`{"type":"Car","make":"VW","name":"Golf","fuel":"diesel"}`,
		`{"type":"Car","make":"VW","name":"Golf","fuel":"diesel","firstRegistration":null,"licensePlate":null}`,
	} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s: status = %d, body = %s", body, rec.Code, rec.Body)
		}
		if repo.input.FirstRegistration != nil || repo.input.LicensePlate != nil {
			t.Fatalf("%s: store input = %+v", body, repo.input)
		}
		got := decodeMap(t, rec)
		for _, key := range []string{"firstRegistration", "licensePlate"} {
			if value, ok := got[key]; !ok || value != nil {
				t.Fatalf("%s: %s = %v (present %v), want null", body, key, value, ok)
			}
		}
	}
}

func TestCarValidation(t *testing.T) {
	cases := []struct {
		name, method, body string
		want               map[string]string
	}{
		{"unsupported fuel", http.MethodPost, strings.Replace(validCreate, `"diesel"`, `"petrol"`, 1), map[string]string{"fuel": apierror.ReasonInvalidEnum}},
		{"unsupported fuel on update", http.MethodPatch, `{"fuel":"petrol"}`, map[string]string{"fuel": apierror.ReasonInvalidEnum}},
		{"undocumented member on create", http.MethodPost, strings.Replace(validCreate, `{`, `{"owner":"x",`, 1), map[string]string{"owner": apierror.ReasonUnknown}},
		{"undocumented member on update", http.MethodPatch, `{"name":"Golf","id":"x"}`, map[string]string{"id": apierror.ReasonUnknown}},
		{"missing required members", http.MethodPost, `{"type":"Hatchback"}`, map[string]string{"make": apierror.ReasonRequired, "name": apierror.ReasonRequired, "fuel": apierror.ReasonRequired}},
		{"blank strings", http.MethodPatch, `{"name":"  ","fin":""}`, map[string]string{"name": apierror.ReasonEmpty, "fin": apierror.ReasonEmpty}},
		{"blank license plate on create", http.MethodPost, strings.Replace(validCreate, `"W-123AB"`, `" "`, 1), map[string]string{"licensePlate": apierror.ReasonEmpty}},
		{"blank license plate on update", http.MethodPatch, `{"licensePlate":""}`, map[string]string{"licensePlate": apierror.ReasonEmpty}},
		{"null required member", http.MethodPatch, `{"make":null}`, map[string]string{"make": apierror.ReasonInvalidType}},
		{"wrong types", http.MethodPatch, `{"isActive":"yes","type":3,"purchasePrice":12}`, map[string]string{"isActive": apierror.ReasonInvalidType, "type": apierror.ReasonInvalidType, "purchasePrice": apierror.ReasonInvalidType}},
		{"invalid dates", http.MethodPatch, `{"firstRegistration":"2019-02-30","purchaseDate":"15.04.2019"}`, map[string]string{"firstRegistration": apierror.ReasonInvalidDate, "purchaseDate": apierror.ReasonInvalidDate}},
		{"invalid decimal", http.MethodPatch, `{"purchasePrice":"12,50"}`, map[string]string{"purchasePrice": apierror.ReasonInvalidDecimal}},
		{"negative price", http.MethodPatch, `{"purchasePrice":"-0.01"}`, map[string]string{"purchasePrice": apierror.ReasonNegative}},
		{"blank information on create", http.MethodPost, strings.Replace(validCreate, `{`, `{"information":" \n\t ",`, 1), map[string]string{"information": apierror.ReasonEmpty}},
		{"blank information on update", http.MethodPatch, `{"information":""}`, map[string]string{"information": apierror.ReasonEmpty}},
		{"oversized information", http.MethodPatch, `{"information":"` + strings.Repeat("ä", maxInformationLength+1) + `"}`, map[string]string{"information": apierror.ReasonTooLong}},
		{"invalid deregistration date", http.MethodPatch, `{"deregistrationDate":"2024-13-01"}`, map[string]string{"deregistrationDate": apierror.ReasonInvalidDate}},
		{"deregistration before first registration on create", http.MethodPost, strings.Replace(validCreate, `{`, `{"deregistrationDate":"2019-02-28",`, 1), map[string]string{"deregistrationDate": apierror.ReasonBeforeFirstRegistration}},
		{"empty update", http.MethodPatch, `{}`, map[string]string{apierror.BodyField: apierror.ReasonRequired}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			target := "/api/v1/cars"
			if tc.method == http.MethodPatch {
				target += "/" + carID
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
		rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", body)
		if rec.Code != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("%q: status = %d, calls = %d", body, rec.Code, repo.calls)
		}
	}
}

func TestUpdateClearsOptionalField(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"fin":null,"purchasePrice":"100","firstRegistration":null,"licensePlate":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	p := repo.patch
	if !p.FIN.Set || p.FIN.Value != nil || !p.FirstRegistration.Set || p.FirstRegistration.Value != nil ||
		!p.LicensePlate.Set || p.LicensePlate.Value != nil || !p.PurchasePrice.Set || *p.PurchasePrice.Value != "100" || p.PurchaseDate.Set || p.Name != nil {
		t.Fatalf("patch = %+v", p)
	}
	if body := decodeMap(t, rec); body["fin"] != nil || body["firstRegistration"] != nil || body["licensePlate"] != nil {
		t.Fatalf("body = %v", body)
	}
}

func TestCreateCarWithDeregistrationAndInformation(t *testing.T) {
	repo := &fakeRepository{}
	body := strings.Replace(validCreate, `{`, `{"isActive":false,"deregistrationDate":"2019-03-01","information":"  Sold\nto Anna  ",`, 1)
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	got := decodeMap(t, rec)
	if got["isActive"] != false || got["deregistrationDate"] != "2019-03-01" || got["information"] != "Sold\nto Anna" {
		t.Fatalf("body = %v", got)
	}
}

func TestUpdateCarInformationAtMaximumLength(t *testing.T) {
	repo := &fakeRepository{}
	text := strings.Repeat("ä", maxInformationLength)
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"information":"`+text+`"}`)
	if rec.Code != http.StatusOK || repo.patch.Information.Value == nil || *repo.patch.Information.Value != text {
		t.Fatalf("status = %d, patch = %+v", rec.Code, repo.patch)
	}
}

func TestUpdateActiveStateLeavesDeregistration(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"isActive":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if repo.calls != 1 || repo.patch.DeregistrationDate.Set || repo.patch.IsActive == nil || *repo.patch.IsActive {
		t.Fatalf("calls = %d, patch = %+v", repo.calls, repo.patch)
	}
	if got := decodeMap(t, rec); got["isActive"] != false || got["deregistrationDate"] != nil {
		t.Fatalf("body = %v", got)
	}
}

func TestUpdateClearsDeregistrationAndInformation(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"deregistrationDate":null,"information":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	p := repo.patch
	if !p.DeregistrationDate.Set || p.DeregistrationDate.Value != nil || !p.Information.Set || p.Information.Value != nil {
		t.Fatalf("patch = %+v", p)
	}
}

func TestUpdateChecksDateOrderAgainstStoredCar(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		// sampleCar's stored first registration is 2019-03-01.
		{"deregistration before stored first registration", `{"deregistrationDate":"2019-02-28"}`, http.StatusBadRequest},
		{"deregistration on stored first registration", `{"deregistrationDate":"2019-03-01"}`, http.StatusOK},
		{"both dates out of order", `{"firstRegistration":"2020-01-01","deregistrationDate":"2019-12-31"}`, http.StatusBadRequest},
		{"first registration cleared", `{"firstRegistration":null,"deregistrationDate":"2000-01-01"}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
			if tc.want != http.StatusBadRequest {
				return
			}
			if repo.patch != (Patch{}) {
				t.Fatalf("store updated on a rejected request: %+v", repo.patch)
			}
			fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
			if fields["deregistrationDate"] != apierror.ReasonBeforeFirstRegistration {
				t.Fatalf("fields = %v", fields)
			}
		})
	}
}

func TestUpdateDateOrderForMissingCarIsNotFound(t *testing.T) {
	repo := &fakeRepository{err: ErrNotFound}
	rec := serve(t, repo, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"deregistrationDate":"2020-01-01"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestDatabaseDateOrderRejectionIsValidationError(t *testing.T) {
	repo := &fakeRepository{err: ErrDeregistrationBeforeFirstRegistration}
	rec := serve(t, repo, passSession, http.MethodPost, "/api/v1/cars", validCreate)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fields, _ := decodeMap(t, rec)["fields"].(map[string]any); fields["deregistrationDate"] != apierror.ReasonBeforeFirstRegistration {
		t.Fatalf("fields = %v", fields)
	}
}

func TestGetCar(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/cars/"+carID, "")
	if rec.Code != http.StatusOK || repo.account != callerID {
		t.Fatalf("status = %d, account = %s", rec.Code, repo.account)
	}
	if body := decodeMap(t, rec); body["id"] != carID || body["licensePlate"] != "W-123AB" {
		t.Fatalf("body = %v", body)
	}
}

func TestDeleteCar(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodDelete, "/api/v1/cars/"+carID, "")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || repo.calls != 1 {
		t.Fatalf("status = %d, body = %q, calls = %d", rec.Code, rec.Body, repo.calls)
	}
}

func TestUnownedOrMissingCarIsNotFound(t *testing.T) {
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPatch, `{"name":"x"}`},
		{http.MethodDelete, ""},
	} {
		repo := &fakeRepository{err: ErrNotFound}
		rec := serve(t, repo, passSession, tc.method, "/api/v1/cars/"+carID, tc.body)
		if rec.Code != http.StatusNotFound || decodeMap(t, rec)["code"] != apierror.CodeNotFound {
			t.Fatalf("%s: status = %d, body = %s", tc.method, rec.Code, rec.Body)
		}

		repo = &fakeRepository{}
		rec = serve(t, repo, passSession, tc.method, "/api/v1/cars/not-a-uuid", tc.body)
		if rec.Code != http.StatusNotFound || repo.calls != 0 {
			t.Fatalf("%s invalid id: status = %d, calls = %d", tc.method, rec.Code, repo.calls)
		}
	}
}

func TestStoreFailureIsInternalError(t *testing.T) {
	repo := &fakeRepository{err: errors.New("boom")}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/cars/"+carID, "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestListCars(t *testing.T) {
	next := &Cursor{CreatedAt: stamp, ID: carID}
	repo := &fakeRepository{items: []Car{sampleCar()}, next: next}
	rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/cars", "")
	if rec.Code != http.StatusOK || repo.limit != defaultLimit || repo.after != nil || repo.account != callerID {
		t.Fatalf("status = %d, limit = %d, after = %v", rec.Code, repo.limit, repo.after)
	}
	body := decodeMap(t, rec)
	items, _ := body["items"].([]any)
	if len(items) != 1 || body["nextCursor"] != next.Encode() {
		t.Fatalf("body = %v", body)
	}

	repo = &fakeRepository{}
	rec = serve(t, repo, passSession, http.MethodGet, "/api/v1/cars?limit=100&cursor="+next.Encode(), "")
	if rec.Code != http.StatusOK || repo.limit != 100 || repo.after == nil || repo.after.ID != carID || !repo.after.CreatedAt.Equal(stamp) {
		t.Fatalf("status = %d, limit = %d, after = %v", rec.Code, repo.limit, repo.after)
	}
	body = decodeMap(t, rec)
	if items, ok := body["items"].([]any); !ok || len(items) != 0 || body["nextCursor"] != nil {
		t.Fatalf("empty page body = %v", body)
	}
}

func TestListRejectsBadPaging(t *testing.T) {
	for _, query := range []string{"limit=0", "limit=101", "limit=ten", "cursor=%21%21", "cursor=" + (Cursor{CreatedAt: stamp, ID: "x"}).Encode()} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, http.MethodGet, "/api/v1/cars?"+query, "")
		if rec.Code != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("%s: status = %d, calls = %d", query, rec.Code, repo.calls)
		}
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

// Mounted on the top-level mux with the real session middleware, every car
// route answers 401 rather than the /api/ 404 fallback.
func TestCarsRequireSessionOnTopLevelMux(t *testing.T) {
	requireSession := auth.NewService(noSessionRepository{}, nil, nil, "http://localhost").RequireSession
	repo := &fakeRepository{}
	mux := httpserver.NewMux(okPinger{}, nil, fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, func(mux *http.ServeMux) {
		NewHandler(repo).RegisterRoutes(mux, requireSession)
	})
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/api/v1/cars"},
		{http.MethodPost, "/api/v1/cars"},
		{http.MethodGet, "/api/v1/cars/" + carID},
		{http.MethodPatch, "/api/v1/cars/" + carID},
		{http.MethodDelete, "/api/v1/cars/" + carID},
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

// Through the real mux and middleware chain, a store failure's error entry and
// the access entry carry the request ID returned in X-Request-Id, and neither
// contains the concrete car path.
func TestStoreFailureLogCorrelatesWithRequestID(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	repo := &fakeRepository{err: errors.New("boom")}
	mux := httpserver.NewMux(okPinger{}, nil, fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, func(mux *http.ServeMux) {
		NewHandler(repo).RegisterRoutes(mux, passSession)
	})
	rec := httptest.NewRecorder()
	httpserver.Wrap(mux, config.RequestLogAll).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/cars/"+carID, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	id := rec.Header().Get("X-Request-Id")
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if id == "" || len(lines) != 2 || !strings.Contains(lines[0], "level=ERROR") || !strings.Contains(lines[0], "error=boom") ||
		!strings.Contains(lines[1], `route="GET /api/v1/cars/{carId}"`) || !strings.Contains(lines[1], "status=500") {
		t.Fatalf("X-Request-Id = %q, log = %q", id, logs.String())
	}
	for _, line := range lines {
		if !strings.Contains(line, "request_id="+id) || strings.Contains(line, carID) {
			t.Errorf("line %q: want request_id=%s and no car ID", line, id)
		}
	}
}

func TestAttachmentCountIsReadOnly(t *testing.T) {
	for _, req := range []struct{ method, target string }{{http.MethodPost, "/api/v1/cars"}, {http.MethodPatch, "/api/v1/cars/" + carID}} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, req.method, req.target, `{"type":"Car","make":"VW","name":"Golf","fuel":"diesel","attachmentCount":3}`)
		fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
		if rec.Code != http.StatusBadRequest || fields["attachmentCount"] != apierror.ReasonUnknown || repo.calls != 0 {
			t.Fatalf("%s: status = %d, body = %s; want 400 with attachmentCount unknown", req.method, rec.Code, rec.Body)
		}
	}
	rec := serve(t, &fakeRepository{}, passSession, http.MethodGet, "/api/v1/cars/"+carID, "")
	if body := decodeMap(t, rec); body["attachmentCount"] != float64(0) {
		t.Fatalf("car body = %v, want attachmentCount 0", body)
	}
}

func TestRefuelCountIsReadOnlyAndFuelLockIsReported(t *testing.T) {
	for _, req := range []struct{ method, target string }{{http.MethodPost, "/api/v1/cars"}, {http.MethodPatch, "/api/v1/cars/" + carID}} {
		repo := &fakeRepository{}
		rec := serve(t, repo, passSession, req.method, req.target, `{"type":"Car","make":"VW","name":"Golf","fuel":"diesel","refuelCount":3}`)
		fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
		if rec.Code != http.StatusBadRequest || fields["refuelCount"] != apierror.ReasonUnknown || repo.calls != 0 {
			t.Fatalf("%s: status = %d, body = %s; want 400 with refuelCount unknown", req.method, rec.Code, rec.Body)
		}
	}
	rec := serve(t, &fakeRepository{}, passSession, http.MethodGet, "/api/v1/cars/"+carID, "")
	if body := decodeMap(t, rec); body["refuelCount"] != float64(0) {
		t.Fatalf("car body = %v, want refuelCount 0", body)
	}

	rec = serve(t, &fakeRepository{err: ErrFuelLocked}, passSession, http.MethodPatch, "/api/v1/cars/"+carID, `{"fuel":"electric"}`)
	fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["fuel"] != apierror.ReasonFuelLocked {
		t.Fatalf("status = %d, body = %s; want 400 with fuel_locked", rec.Code, rec.Body)
	}
}
