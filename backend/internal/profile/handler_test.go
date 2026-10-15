package profile

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

	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

var caller = auth.Account{ID: "11111111-1111-1111-1111-111111111111", Email: "me@example.com", FirstName: "Ada", LastName: "Lovelace", Currency: "EUR", VolumeUnit: "liter", DistanceUnit: "kilometer"}

type fakeRepository struct {
	calls       int
	first, last *string
	update      Update
	err         error
}

func (f *fakeRepository) UpdateProfile(_ context.Context, id string, u Update) (auth.Account, error) {
	f.calls++
	f.update = u
	f.first, f.last = u.FirstName, u.LastName
	if f.err != nil {
		return auth.Account{}, f.err
	}
	a := caller
	a.ID = id
	for dst, src := range map[*string]*string{&a.FirstName: u.FirstName, &a.LastName: u.LastName, &a.Currency: u.Currency, &a.VolumeUnit: u.VolumeUnit, &a.DistanceUnit: u.DistanceUnit} {
		if src != nil {
			*dst = *src
		}
	}
	return a, nil
}

// passSession stands in for auth.Service.RequireSession with an
// already-authenticated caller.
func passSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(auth.WithAccount(r.Context(), caller)))
	})
}

func serve(t *testing.T, repo Repository, requireSession func(http.Handler) http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(repo).RegisterRoutes(mux, requireSession)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, "/api/v1/me", strings.NewReader(body)))
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

func TestGetMeReturnsCallerProfile(t *testing.T) {
	rec := serve(t, &fakeRepository{}, passSession, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decodeMap(t, rec)
	want := map[string]any{"id": caller.ID, "email": caller.Email, "firstName": caller.FirstName, "lastName": caller.LastName, "currency": "EUR", "volumeUnit": "liter", "distanceUnit": "kilometer"}
	if len(got) != len(want) {
		t.Fatalf("body = %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestPatchMeUpdatesNames(t *testing.T) {
	tests := []struct {
		name, body          string
		wantFirst, wantLast string
		firstSet, lastSet   bool
	}{
		{"first name only", `{"firstName":"  Grace "}`, "Grace", "Lovelace", true, false},
		{"clears last name", `{"lastName":""}`, "Ada", "", false, true},
		{"both names", `{"firstName":"Grace","lastName":"Hopper"}`, "Grace", "Hopper", true, true},
		{"exactly the limit", `{"firstName":"` + strings.Repeat("ä", MaxNameLength) + `"}`, strings.Repeat("ä", MaxNameLength), "Lovelace", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := serve(t, repo, passSession, http.MethodPatch, tt.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
			}
			if (repo.first != nil) != tt.firstSet || (repo.last != nil) != tt.lastSet {
				t.Fatalf("supplied first=%v last=%v", repo.first, repo.last)
			}
			got := decodeMap(t, rec)
			if got["firstName"] != tt.wantFirst || got["lastName"] != tt.wantLast || got["email"] != caller.Email {
				t.Fatalf("body = %v", got)
			}
		})
	}
}

func TestPatchMeUpdatesDisplaySettings(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, `{"currency":"USD","volumeUnit":"gallon","distanceUnit":"mile"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if repo.update.FirstName != nil || repo.update.LastName != nil {
		t.Fatalf("names supplied: %+v", repo.update)
	}
	got := decodeMap(t, rec)
	if got["currency"] != "USD" || got["volumeUnit"] != "gallon" || got["distanceUnit"] != "mile" || got["firstName"] != "Ada" {
		t.Fatalf("body = %v", got)
	}
}

func TestPatchMeRejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name, body string
		fields     map[string]string
	}{
		{"email alone", `{"email":"new@example.com"}`, map[string]string{"email": "read_only"}},
		{"occupied email with name", `{"email":"other@example.com","firstName":"Grace"}`, map[string]string{"email": "read_only"}},
		{"unknown member", `{"firstName":"Grace","nickname":"G"}`, map[string]string{"nickname": "unknown"}},
		{"null name", `{"firstName":null}`, map[string]string{"firstName": "invalid_type"}},
		{"numeric name", `{"lastName":7}`, map[string]string{"lastName": "invalid_type"}},
		{"overlong after trim", `{"lastName":" ` + strings.Repeat("x", MaxNameLength+1) + ` "}`, map[string]string{"lastName": "too_long"}},
		{"all errors collected", `{"email":"x@example.com","firstName":false,"id":"1"}`, map[string]string{"email": "read_only", "firstName": "invalid_type", "id": "unknown"}},
		{"unknown currency", `{"currency":"XYZ"}`, map[string]string{"currency": "invalid_enum"}},
		{"lowercase currency", `{"currency":"usd"}`, map[string]string{"currency": "invalid_enum"}},
		{"long currency", `{"currency":"EURO"}`, map[string]string{"currency": "invalid_enum"}},
		{"numeric currency", `{"currency":978}`, map[string]string{"currency": "invalid_type"}},
		{"unknown volume unit", `{"volumeUnit":"barrel"}`, map[string]string{"volumeUnit": "invalid_enum"}},
		{"unknown distance unit", `{"distanceUnit":"furlong"}`, map[string]string{"distanceUnit": "invalid_enum"}},
		{"null unit", `{"distanceUnit":null}`, map[string]string{"distanceUnit": "invalid_type"}},
		{"valid name with invalid setting", `{"firstName":"Grace","volumeUnit":"Liter"}`, map[string]string{"volumeUnit": "invalid_enum"}},
		{"empty object", `{}`, map[string]string{"_body": "required"}},
		{"malformed json", `{"firstName":`, nil},
		{"not an object", `["firstName"]`, nil},
		{"null body", `null`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := serve(t, repo, passSession, http.MethodPatch, tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
			}
			if repo.calls != 0 {
				t.Fatal("repository called for an invalid request")
			}
			got := decodeMap(t, rec)
			if got["code"] != "validation_failed" {
				t.Fatalf("code = %v", got["code"])
			}
			fields, _ := got["fields"].(map[string]any)
			if len(fields) != len(tt.fields) {
				t.Fatalf("fields = %v, want %v", fields, tt.fields)
			}
			for k, v := range tt.fields {
				if fields[k] != v {
					t.Fatalf("fields[%s] = %v, want %v", k, fields[k], v)
				}
			}
		})
	}
}

func TestPatchMeRejectsOversizedBody(t *testing.T) {
	repo := &fakeRepository{}
	rec := serve(t, repo, passSession, http.MethodPatch, `{"firstName":"`+strings.Repeat("x", maxBodyBytes)+`"}`)
	if rec.Code != http.StatusBadRequest || repo.calls != 0 {
		t.Fatalf("status = %d calls = %d", rec.Code, repo.calls)
	}
}

func TestPatchMeStoreFailure(t *testing.T) {
	rec := serve(t, &fakeRepository{err: errors.New("boom")}, passSession, http.MethodPatch, `{"firstName":"Grace"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decodeMap(t, rec); got["code"] != "internal_error" || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("body = %s", rec.Body)
	}
}

// noSessionRepository satisfies auth.Repository for a service whose session
// lookup always fails; no other method is reached by these requests.
type noSessionRepository struct{ auth.Repository }

func (noSessionRepository) Session(context.Context, []byte, time.Time) (auth.Session, error) {
	return auth.Session{}, auth.ErrInvalidCredential
}

func TestMeRequiresSession(t *testing.T) {
	requireSession := auth.NewService(noSessionRepository{}, nil, nil, "http://localhost").RequireSession
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		for _, cookie := range []bool{false, true} {
			repo := &fakeRepository{}
			mux := http.NewServeMux()
			NewHandler(repo).RegisterRoutes(mux, requireSession)
			req := httptest.NewRequest(method, "/api/v1/me", strings.NewReader(`{"firstName":"Grace"}`))
			if cookie {
				req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "stale"})
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized || repo.calls != 0 {
				t.Fatalf("%s cookie=%v: status = %d calls = %d", method, cookie, rec.Code, repo.calls)
			}
		}
	}
}

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

func TestMeIsServedByTopLevelMux(t *testing.T) {
	requireSession := auth.NewService(noSessionRepository{}, nil, nil, "http://localhost").RequireSession
	mux := httpserver.NewMux(okPinger{}, nil, fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}, func(mux *http.ServeMux) {
		NewHandler(&fakeRepository{}).RegisterRoutes(mux, requireSession)
	})
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, "/api/v1/me", strings.NewReader(`{}`)))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s /api/v1/me = %d, want 401 (not the /api/ 404 fallback)", method, rec.Code)
		}
	}
}
