// Package tickets serves the caller's tickets (/api/v1/tickets), per
// specs/api-contract/expenses.
package tickets

import (
	"context"
	"encoding/json"
	"errors"
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
	types          = map[string]bool{"parking": true, "velocity": true, "other": true}
)

func isUUID(s string) bool { return uuidPattern.MatchString(s) }

type Repository interface {
	List(ctx context.Context, accountID string, f Filter, after *Cursor, limit int) ([]Ticket, *Cursor, error)
	Create(ctx context.Context, accountID string, in Input) (Ticket, error)
	Get(ctx context.Context, accountID, ticketID string) (Ticket, error)
	Update(ctx context.Context, accountID, ticketID string, p Patch) (Ticket, error)
	Delete(ctx context.Context, accountID, ticketID string) error
	Locations(ctx context.Context, accountID string) ([]string, error)
}

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the ticket operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/tickets", requireSession(h.withAccount(h.list)))
	mux.Handle("POST /api/v1/tickets", requireSession(h.withAccount(h.create)))
	mux.Handle("GET /api/v1/tickets/locations", requireSession(h.withAccount(h.locations)))
	mux.Handle("GET /api/v1/tickets/{ticketId}", requireSession(h.withTicket(h.get)))
	mux.Handle("PATCH /api/v1/tickets/{ticketId}", requireSession(h.withTicket(h.update)))
	mux.Handle("DELETE /api/v1/tickets/{ticketId}", requireSession(h.withTicket(h.delete)))
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

// withTicket also resolves the path's ticket ID; one that is not a UUID
// cannot name a ticket, so it is reported like any other missing ticket.
func (h *Handler) withTicket(next func(http.ResponseWriter, *http.Request, string, string)) http.Handler {
	return h.withAccount(func(w http.ResponseWriter, r *http.Request, accountID string) {
		ticketID := r.PathValue("ticketId")
		if !isUUID(ticketID) {
			apierror.NotFound(w)
			return
		}
		next(w, r, accountID, ticketID)
	})
}

type ticketBody struct {
	ID              string `json:"id"`
	CarID           string `json:"carId"`
	Date            string `json:"date"`
	Type            string `json:"type"`
	Location        string `json:"location"`
	Amount          string `json:"amount"`
	Description     string `json:"description"`
	AttachmentCount int    `json:"attachmentCount"`
}

func toBody(t Ticket) ticketBody {
	return ticketBody{
		ID: t.ID, CarID: t.CarID, Date: t.Date.UTC().Format(time.RFC3339Nano), Type: t.Type,
		Location: t.Location, Amount: t.Amount, Description: t.Description,
		AttachmentCount: t.AttachmentCount,
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
	httpserver.Logger(r.Context()).Error("ticket failed", "op", op, "error", err)
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
		Items      []ticketBody `json:"items"`
		NextCursor *string      `json:"nextCursor"`
	}{Items: make([]ticketBody, 0, len(items))}
	for _, item := range items {
		page.Items = append(page.Items, toBody(item))
	}
	if next != nil {
		encoded := next.Encode()
		page.NextCursor = &encoded
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) locations(w http.ResponseWriter, r *http.Request, accountID string) {
	locations, err := h.store.Locations(r.Context(), accountID)
	if err != nil {
		h.fail(w, r, "list locations of", err)
		return
	}
	writeJSON(w, http.StatusOK, locations)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, accountID string) {
	p, carID, fields, ok := decodeBody(w, r, true)
	if !ok {
		return
	}
	for key, missing := range map[string]bool{
		"carId": carID == nil, "date": p.Date == nil, "type": p.Type == nil, "location": p.Location == nil, "amount": p.Amount == nil,
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
		CarID: *carID, Date: *p.Date, Type: *p.Type, Location: *p.Location, Amount: *p.Amount,
	}
	if p.Description != nil {
		in.Description = *p.Description
	}
	created, err := h.store.Create(r.Context(), accountID, in)
	if err != nil {
		h.fail(w, r, "create", err)
		return
	}
	writeJSON(w, http.StatusCreated, toBody(created))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, accountID, ticketID string) {
	found, err := h.store.Get(r.Context(), accountID, ticketID)
	if err != nil {
		h.fail(w, r, "get", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(found))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, accountID, ticketID string) {
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
	updated, err := h.store.Update(r.Context(), accountID, ticketID, p)
	if err != nil {
		h.fail(w, r, "update", err)
		return
	}
	writeJSON(w, http.StatusOK, toBody(updated))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, accountID, ticketID string) {
	if err := h.store.Delete(r.Context(), accountID, ticketID); err != nil {
		h.fail(w, r, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeBody reads a JSON object and validates every member, collecting a
// reason for each offending one. carId is only a member of the creation
// request (TicketInput); an update (TicketUpdate) reports it as unknown. A
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
		case "location":
			p.Location = requiredString(key, value, fields)
		case "type":
			if s := requiredString(key, value, fields); s != nil {
				if !types[*s] {
					fields[key] = apierror.ReasonInvalidEnum
				} else {
					p.Type = s
				}
			}
		case "amount":
			if s := requiredString(key, value, fields); s != nil {
				switch {
				case !decimalPattern.MatchString(*s):
					fields[key] = apierror.ReasonInvalidDecimal
				case strings.HasPrefix(*s, "-") && strings.Trim((*s)[1:], "0.") != "":
					fields[key] = apierror.ReasonNegative
				default:
					p.Amount = s
				}
			}
		case "description":
			var s *string
			if err := json.Unmarshal(value, &s); err != nil || s == nil {
				fields[key] = apierror.ReasonInvalidType
			} else {
				trimmed := strings.TrimSpace(*s)
				p.Description = &trimmed
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
