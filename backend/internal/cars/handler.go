// Package cars serves the caller's cars (/api/v1/cars), per
// specs/api-contract/cars.
package cars

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

const (
	defaultLimit = 25
	maxLimit     = 100
	maxBodyBytes = 16 << 10
	// maxInformationLength is information's maxLength in characters.
	maxInformationLength = 2000
)

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
	fuels          = map[string]bool{"other": true, "diesel": true, "gasoline": true, "electric": true}
)

func isUUID(s string) bool { return uuidPattern.MatchString(s) }

type Repository interface {
	List(ctx context.Context, accountID string, after *Cursor, limit int) ([]Car, *Cursor, error)
	Create(ctx context.Context, accountID string, in Input) (Car, error)
	Get(ctx context.Context, accountID, carID string) (Car, error)
	Update(ctx context.Context, accountID, carID string, p Patch) (Car, error)
	Delete(ctx context.Context, accountID, carID string) error
}

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the car operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/cars", requireSession(h.withAccount(h.list)))
	mux.Handle("POST /api/v1/cars", requireSession(h.withAccount(h.create)))
	mux.Handle("GET /api/v1/cars/{carId}", requireSession(h.withCar(h.get)))
	mux.Handle("PATCH /api/v1/cars/{carId}", requireSession(h.withCar(h.update)))
	mux.Handle("DELETE /api/v1/cars/{carId}", requireSession(h.withCar(h.delete)))
}

func (h *Handler) withAccount(next func(http.ResponseWriter, *http.Request, string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		account, ok := auth.AccountFromContext(r.Context())
		if !ok {
			apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required", nil)
			return
		}
		next(w, r, account.ID)
	})
}

// withCar also resolves the path's car ID; one that is not a UUID cannot
// name a car, so it is reported like any other missing car.
func (h *Handler) withCar(next func(http.ResponseWriter, *http.Request, string, string)) http.Handler {
	return h.withAccount(func(w http.ResponseWriter, r *http.Request, accountID string) {
		carID := r.PathValue("carId")
		if !isUUID(carID) {
			apierror.NotFound(w)
			return
		}
		next(w, r, accountID, carID)
	})
}

type carBody struct {
	ID                 string  `json:"id"`
	Type               string  `json:"type"`
	Make               string  `json:"make"`
	Name               string  `json:"name"`
	Fuel               string  `json:"fuel"`
	FirstRegistration  *string `json:"firstRegistration"`
	LicensePlate       *string `json:"licensePlate"`
	FIN                *string `json:"fin"`
	IsActive           bool    `json:"isActive"`
	PurchaseDate       *string `json:"purchaseDate"`
	PurchasePrice      *string `json:"purchasePrice"`
	DeregistrationDate *string `json:"deregistrationDate"`
	Information        *string `json:"information"`
	AttachmentCount    int     `json:"attachmentCount"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

func toBody(c Car) carBody {
	return carBody{
		ID: c.ID, Type: c.Type, Make: c.Make, Name: c.Name, Fuel: c.Fuel,
		FirstRegistration: c.FirstRegistration, LicensePlate: c.LicensePlate, FIN: c.FIN,
		IsActive: c.IsActive, PurchaseDate: c.PurchaseDate, PurchasePrice: c.PurchasePrice,
		DeregistrationDate: c.DeregistrationDate, Information: c.Information, AttachmentCount: c.AttachmentCount,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	if errors.Is(err, ErrNotFound) {
		apierror.NotFound(w)
		return
	}
	if errors.Is(err, ErrDeregistrationBeforeFirstRegistration) {
		apierror.Validation(w, map[string]string{"deregistrationDate": apierror.ReasonBeforeFirstRegistration})
		return
	}
	httpserver.Logger(r.Context()).Error("car failed", "op", op, "error", err)
	apierror.Internal(w)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, accountID string) {
	query := r.URL.Query()
	fields := map[string]string{}
	limit := defaultLimit
	if raw := query.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			fields["limit"] = apierror.ReasonInvalidType
		}
		limit = n
	}
	var after *Cursor
	if raw := query.Get("cursor"); raw != "" {
		c, ok := DecodeCursor(raw)
		if !ok {
			fields["cursor"] = apierror.ReasonInvalidType
		}
		after = &c
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	items, next, err := h.store.List(r.Context(), accountID, after, limit)
	if err != nil {
		h.fail(w, r, "list", err)
		return
	}
	page := struct {
		Items      []carBody `json:"items"`
		NextCursor *string   `json:"nextCursor"`
	}{Items: make([]carBody, 0, len(items))}
	for _, c := range items {
		page.Items = append(page.Items, toBody(c))
	}
	if next != nil {
		encoded := next.Encode()
		page.NextCursor = &encoded
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, accountID string) {
	p, fields, ok := decodeBody(w, r)
	if !ok {
		return
	}
	for key, value := range map[string]*string{
		"type": p.Type, "make": p.Make, "name": p.Name, "fuel": p.Fuel,
	} {
		if value == nil {
			if _, reported := fields[key]; !reported {
				fields[key] = apierror.ReasonRequired
			}
		}
	}
	if len(fields) == 0 && deregisteredBeforeFirstRegistration(p.FirstRegistration.Value, p.DeregistrationDate.Value) {
		fields["deregistrationDate"] = apierror.ReasonBeforeFirstRegistration
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	in := Input{
		Type: *p.Type, Make: *p.Make, Name: *p.Name, Fuel: *p.Fuel,
		FirstRegistration: p.FirstRegistration.Value, LicensePlate: p.LicensePlate.Value,
		FIN: p.FIN.Value, PurchaseDate: p.PurchaseDate.Value, PurchasePrice: p.PurchasePrice.Value,
		DeregistrationDate: p.DeregistrationDate.Value, Information: p.Information.Value,
		IsActive: p.IsActive == nil || *p.IsActive,
	}
	c, err := h.store.Create(r.Context(), accountID, in)
	if err != nil {
		h.fail(w, r, "create", err)
		return
	}
	writeJSON(w, http.StatusCreated, toBody(c))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, accountID, carID string) {
	c, err := h.store.Get(r.Context(), accountID, carID)
	if err != nil {
		h.fail(w, r, "get", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(c))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, accountID, carID string) {
	p, fields, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if len(fields) == 0 && p == (Patch{}) {
		fields[apierror.BodyField] = apierror.ReasonRequired
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	if p.FirstRegistration.Set || p.DeregistrationDate.Set {
		ok, err := h.datesInOrder(r.Context(), accountID, carID, p)
		if err != nil {
			h.fail(w, r, "update", err)
			return
		}
		if !ok {
			apierror.Validation(w, map[string]string{"deregistrationDate": apierror.ReasonBeforeFirstRegistration})
			return
		}
	}
	c, err := h.store.Update(r.Context(), accountID, carID, p)
	if err != nil {
		h.fail(w, r, "update", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(c))
}

// datesInOrder checks the patched car's deregistration date against its
// first registration, taking a date the patch does not supply from the
// stored car. The database check still guards against a concurrent change.
func (h *Handler) datesInOrder(ctx context.Context, accountID, carID string, p Patch) (bool, error) {
	stored, err := h.store.Get(ctx, accountID, carID)
	if err != nil {
		return false, err
	}
	first, deregistration := stored.FirstRegistration, stored.DeregistrationDate
	if p.FirstRegistration.Set {
		first = p.FirstRegistration.Value
	}
	if p.DeregistrationDate.Set {
		deregistration = p.DeregistrationDate.Value
	}
	return !deregisteredBeforeFirstRegistration(first, deregistration), nil
}

// deregisteredBeforeFirstRegistration compares two validated full-dates,
// whose fixed-width form orders lexically.
func deregisteredBeforeFirstRegistration(first, deregistration *string) bool {
	return first != nil && deregistration != nil && *deregistration < *first
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, accountID, carID string) {
	if err := h.store.Delete(r.Context(), accountID, carID); err != nil {
		h.fail(w, r, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeBody reads a JSON object and validates every member, collecting a
// reason for each offending one. A malformed body is answered here and
// reported as not ok.
func decodeBody(w http.ResponseWriter, r *http.Request) (Patch, map[string]string, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&raw); err != nil || raw == nil {
		apierror.Validation(w, nil)
		return Patch{}, nil, false
	}
	var p Patch
	fields := map[string]string{}
	for key, value := range raw {
		switch key {
		case "type":
			p.Type = requiredString(key, value, fields)
		case "make":
			p.Make = requiredString(key, value, fields)
		case "name":
			p.Name = requiredString(key, value, fields)
		case "fuel":
			if s := requiredString(key, value, fields); s != nil {
				if !fuels[*s] {
					fields[key] = apierror.ReasonInvalidEnum
				} else {
					p.Fuel = s
				}
			}
		case "firstRegistration":
			p.FirstRegistration = nullableString(key, value, fields, checkDate)
		case "licensePlate":
			p.LicensePlate = nullableString(key, value, fields, checkNonBlank)
		case "isActive":
			var b *bool
			if err := json.Unmarshal(value, &b); err != nil || b == nil {
				fields[key] = apierror.ReasonInvalidType
			} else {
				p.IsActive = b
			}
		case "fin":
			p.FIN = nullableString(key, value, fields, checkNonBlank)
		case "purchaseDate":
			p.PurchaseDate = nullableString(key, value, fields, checkDate)
		case "deregistrationDate":
			p.DeregistrationDate = nullableString(key, value, fields, checkDate)
		case "information":
			p.Information = nullableString(key, value, fields, checkInformation)
		case "purchasePrice":
			p.PurchasePrice = nullableString(key, value, fields, func(s string) (string, string) {
				if !decimalPattern.MatchString(s) {
					return "", apierror.ReasonInvalidDecimal
				}
				if strings.HasPrefix(s, "-") && strings.Trim(s[1:], "0.") != "" {
					return "", apierror.ReasonNegative
				}
				return s, ""
			})
		default:
			fields[key] = apierror.ReasonUnknown
		}
	}
	return p, fields, true
}

// requiredString decodes a non-null string, trimmed, that must not be blank.
func requiredString(key string, value json.RawMessage, fields map[string]string) *string {
	var s *string
	if err := json.Unmarshal(value, &s); err != nil || s == nil {
		fields[key] = apierror.ReasonInvalidType
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		fields[key] = apierror.ReasonEmpty
		return nil
	}
	return &trimmed
}

// nullableString decodes a string-or-null member; check normalizes a string
// value or returns a reason rejecting it.
func nullableString(key string, value json.RawMessage, fields map[string]string, check func(string) (string, string)) Nullable {
	var s *string
	if err := json.Unmarshal(value, &s); err != nil {
		fields[key] = apierror.ReasonInvalidType
		return Nullable{}
	}
	if s == nil {
		return Nullable{Set: true}
	}
	normalized, reason := check(*s)
	if reason != "" {
		fields[key] = reason
		return Nullable{}
	}
	return Nullable{Set: true, Value: &normalized}
}

// checkNonBlank trims a nullable text member and rejects a blank one.
func checkNonBlank(s string) (string, string) {
	if s = strings.TrimSpace(s); s == "" {
		return "", apierror.ReasonEmpty
	}
	return s, ""
}

// checkInformation trims the free-form text, rejecting a blank or overlong one.
func checkInformation(s string) (string, string) {
	s, reason := checkNonBlank(s)
	if reason == "" && utf8.RuneCountInString(s) > maxInformationLength {
		return "", apierror.ReasonTooLong
	}
	return s, reason
}

func checkDate(s string) (string, string) {
	if !isDate(s) {
		return "", apierror.ReasonInvalidDate
	}
	return s, ""
}

func isDate(s string) bool {
	t, err := time.Parse(time.DateOnly, s)
	return err == nil && t.Year() >= 1
}
