// Package refuels serves the caller's refuels (/api/v1/refuels), per
// specs/api-contract/expenses.
package refuels

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

const (
	defaultLimit = 25
	maxLimit     = 100
	maxBodyBytes = 16 << 10
)

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
	fuelTypes      = map[string]bool{"normal": true, "special": true, "home": true, "ac": true, "dc": true, "other": true}
)

func isUUID(s string) bool { return uuidPattern.MatchString(s) }

type Repository interface {
	List(ctx context.Context, accountID string, f Filter, after *Cursor, limit int) ([]Refuel, *Cursor, error)
	Create(ctx context.Context, accountID string, in Input) (Refuel, error)
	Get(ctx context.Context, accountID, refuelID string) (Refuel, error)
	Update(ctx context.Context, accountID, refuelID string, p Patch) (Refuel, error)
	Delete(ctx context.Context, accountID, refuelID string) error
	Stations(ctx context.Context, accountID string) ([]string, error)
	Chart(ctx context.Context, accountID string, f Filter) ([]Refuel, map[string]int64, error)
}

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the refuel operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/refuels", requireSession(h.withAccount(h.list)))
	mux.Handle("POST /api/v1/refuels", requireSession(h.withAccount(h.create)))
	mux.Handle("GET /api/v1/refuels/stations", requireSession(h.withAccount(h.stations)))
	mux.Handle("GET /api/v1/refuels/chart", requireSession(h.withAccount(h.chart)))
	mux.Handle("GET /api/v1/refuels/{refuelId}", requireSession(h.withRefuel(h.get)))
	mux.Handle("PATCH /api/v1/refuels/{refuelId}", requireSession(h.withRefuel(h.update)))
	mux.Handle("DELETE /api/v1/refuels/{refuelId}", requireSession(h.withRefuel(h.delete)))
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

// withRefuel also resolves the path's refuel ID; one that is not a UUID
// cannot name a refuel, so it is reported like any other missing refuel.
func (h *Handler) withRefuel(next func(http.ResponseWriter, *http.Request, string, string)) http.Handler {
	return h.withAccount(func(w http.ResponseWriter, r *http.Request, accountID string) {
		refuelID := r.PathValue("refuelId")
		if !isUUID(refuelID) {
			apierror.NotFound(w)
			return
		}
		next(w, r, accountID, refuelID)
	})
}

type refuelBody struct {
	ID              string  `json:"id"`
	CarID           string  `json:"carId"`
	Date            string  `json:"date"`
	Station         string  `json:"station"`
	OdometerReading *int64  `json:"odometerReading"`
	Fuel            string  `json:"fuel"`
	Quantity        string  `json:"quantity"`
	Amount          string  `json:"amount"`
	UnitPrice       string  `json:"unitPrice"`
	Distance        *int64  `json:"distance"`
	Consumption     *string `json:"consumption"`
	AttachmentCount int     `json:"attachmentCount"`
}

func toBody(r Refuel) refuelBody {
	return refuelBody{
		ID: r.ID, CarID: r.CarID, Date: r.Date.UTC().Format(time.RFC3339Nano), Station: r.Station,
		OdometerReading: r.OdometerReading, Fuel: r.Fuel, Quantity: r.Quantity, Amount: r.Amount,
		UnitPrice: ratio(r.Amount, r.Quantity, 3), AttachmentCount: r.AttachmentCount,
	}
}

func ratio(numerator, denominator string, places int) string {
	n, _ := strconv.ParseFloat(numerator, 64)
	d, _ := strconv.ParseFloat(denominator, 64)
	v := math.Round(n/d*math.Pow10(places)) / math.Pow10(places)
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// consumption is the quantity (litres, gallons or kWh) per 100 distance
// units.
func consumption(quantity string, distance int64) string {
	q, _ := strconv.ParseFloat(quantity, 64)
	v := math.Round(q/float64(distance)*10000) / 100
	return strconv.FormatFloat(v, 'f', -1, 64)
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
	if errors.Is(err, ErrFuelMismatch) {
		apierror.Validation(w, map[string]string{"fuel": apierror.ReasonFuelMismatch})
		return
	}
	httpserver.Logger(r.Context()).Error("refuel failed", "op", op, "error", err)
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
	var f Filter
	if raw := query.Get("carId"); raw != "" {
		if !isUUID(raw) {
			fields["carId"] = apierror.ReasonInvalidType
		}
		f.CarID = &raw
	}
	for key, target := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if raw := query.Get(key); raw != "" {
			t, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				fields[key] = apierror.ReasonInvalidType
			}
			*target = &t
		}
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	items, next, err := h.store.List(r.Context(), accountID, f, after, limit)
	if err != nil {
		h.fail(w, r, "list", err)
		return
	}
	page := struct {
		Items      []refuelBody `json:"items"`
		NextCursor *string      `json:"nextCursor"`
	}{Items: make([]refuelBody, 0, len(items))}
	for _, item := range items {
		page.Items = append(page.Items, toBody(item))
	}
	if next != nil {
		encoded := next.Encode()
		page.NextCursor = &encoded
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) stations(w http.ResponseWriter, r *http.Request, accountID string) {
	stations, err := h.store.Stations(r.Context(), accountID)
	if err != nil {
		h.fail(w, r, "list stations of", err)
		return
	}
	writeJSON(w, http.StatusOK, stations)
}

func parseFilter(r *http.Request) (Filter, map[string]string) {
	q, f, fields := r.URL.Query(), Filter{}, map[string]string{}
	if raw := q.Get("carId"); raw != "" {
		if !isUUID(raw) {
			fields["carId"] = apierror.ReasonInvalidType
		}
		f.CarID = &raw
	}
	for key, target := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if raw := q.Get(key); raw != "" {
			t, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				fields[key] = apierror.ReasonInvalidType
			} else {
				*target = &t
			}
		}
	}
	return f, fields
}

func (h *Handler) chart(w http.ResponseWriter, r *http.Request, accountID string) {
	f, fields := parseFilter(r)
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	items, previous, err := h.store.Chart(r.Context(), accountID, f)
	if err != nil {
		h.fail(w, r, "chart", err)
		return
	}
	if previous == nil {
		previous = map[string]int64{}
	}
	bodies := make([]refuelBody, 0, len(items))
	for _, item := range items {
		body := toBody(item)
		if item.OdometerReading != nil {
			if old, ok := previous[item.CarID]; ok {
				distance := *item.OdometerReading - old
				if distance > 0 {
					body.Distance = &distance
					c := consumption(item.Quantity, distance)
					body.Consumption = &c
				}
			}
			previous[item.CarID] = *item.OdometerReading
		}
		bodies = append(bodies, body)
	}
	writeJSON(w, http.StatusOK, struct {
		Items []refuelBody `json:"items"`
	}{bodies})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, accountID string) {
	p, carID, fields, ok := decodeBody(w, r, true)
	if !ok {
		return
	}
	for key, missing := range map[string]bool{
		"carId": carID == nil, "date": p.Date == nil, "station": p.Station == nil, "fuel": p.Fuel == nil, "quantity": p.Quantity == nil, "amount": p.Amount == nil,
	} {
		if _, reported := fields[key]; missing && !reported {
			fields[key] = apierror.ReasonRequired
		}
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	in := Input{
		CarID: *carID, Date: *p.Date, Station: *p.Station, OdometerReading: p.OdometerReading.Value,
		Fuel: *p.Fuel, Quantity: *p.Quantity, Amount: *p.Amount,
	}
	created, err := h.store.Create(r.Context(), accountID, in)
	if err != nil {
		h.fail(w, r, "create", err)
		return
	}
	writeJSON(w, http.StatusCreated, toBody(created))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, accountID, refuelID string) {
	found, err := h.store.Get(r.Context(), accountID, refuelID)
	if err != nil {
		h.fail(w, r, "get", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(found))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, accountID, refuelID string) {
	p, _, fields, ok := decodeBody(w, r, false)
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
	updated, err := h.store.Update(r.Context(), accountID, refuelID, p)
	if err != nil {
		h.fail(w, r, "update", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(updated))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, accountID, refuelID string) {
	if err := h.store.Delete(r.Context(), accountID, refuelID); err != nil {
		h.fail(w, r, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeBody reads a JSON object and validates every member, collecting a
// reason for each offending one. carId is only a member of the creation
// request (RefuelInput); an update (RefuelUpdate) reports it as unknown. A
// malformed body is answered here and reported as not ok.
func decodeBody(w http.ResponseWriter, r *http.Request, create bool) (Patch, *string, map[string]string, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&raw); err != nil || raw == nil {
		apierror.Validation(w, nil)
		return Patch{}, nil, nil, false
	}
	var p Patch
	var carID *string
	fields := map[string]string{}
	for key, value := range raw {
		switch key {
		case "carId":
			if !create {
				fields[key] = apierror.ReasonUnknown
			} else if s := requiredString(key, value, fields); s != nil {
				if !isUUID(*s) {
					fields[key] = apierror.ReasonInvalidType
				} else {
					carID = s
				}
			}
		case "date":
			if s := requiredString(key, value, fields); s != nil {
				t, err := time.Parse(time.RFC3339Nano, *s)
				if err != nil {
					fields[key] = apierror.ReasonInvalidType
				} else {
					p.Date = &t
				}
			}
		case "station":
			p.Station = requiredString(key, value, fields)
		case "fuel":
			if s := requiredString(key, value, fields); s != nil {
				if !fuelTypes[*s] {
					fields[key] = apierror.ReasonInvalidEnum
				} else {
					p.Fuel = s
				}
			}
		case "quantity", "amount":
			if s := requiredString(key, value, fields); s != nil {
				switch {
				case !decimalPattern.MatchString(*s):
					fields[key] = apierror.ReasonInvalidDecimal
				case strings.HasPrefix(*s, "-") && strings.Trim((*s)[1:], "0.") != "":
					fields[key] = apierror.ReasonNegative
				case key == "quantity" && strings.Trim(*s, "0.") == "":
					fields[key] = apierror.ReasonNonPositive
				default:
					if key == "quantity" {
						p.Quantity = s
					} else {
						p.Amount = s
					}
				}
			}
		case "odometerReading":
			var n *int64
			if err := json.Unmarshal(value, &n); err != nil {
				fields[key] = apierror.ReasonInvalidType
			} else if n != nil && *n < 0 {
				fields[key] = apierror.ReasonNegative
			} else {
				p.OdometerReading = NullableInt{Set: true, Value: n}
			}
		default:
			fields[key] = apierror.ReasonUnknown
		}
	}
	return p, carID, fields, true
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
