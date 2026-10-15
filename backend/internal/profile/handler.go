// Package profile serves the authenticated caller's profile
// (GET/PATCH /api/v1/me), per specs/api-contract/profile.
package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

// MaxNameLength is the maximum length of a trimmed name, in characters.
const MaxNameLength = 100

const maxBodyBytes = 16 << 10

type Repository interface {
	UpdateAccountNames(ctx context.Context, accountID string, firstName, lastName *string) (auth.Account, error)
}

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the profile operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/me", requireSession(http.HandlerFunc(h.getMe)))
	mux.Handle("PATCH /api/v1/me", requireSession(http.HandlerFunc(h.updateMe)))
}

type profileBody struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func writeProfile(w http.ResponseWriter, a auth.Account) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(profileBody{ID: a.ID, Email: a.Email, FirstName: a.FirstName, LastName: a.LastName})
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	account, ok := auth.AccountFromContext(r.Context())
	if !ok {
		apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required", nil)
		return
	}
	writeProfile(w, account)
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	account, ok := auth.AccountFromContext(r.Context())
	if !ok {
		apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required", nil)
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&raw); err != nil || raw == nil {
		apierror.Validation(w, nil)
		return
	}
	firstName, lastName, fields := parseUpdate(raw)
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	updated, err := h.store.UpdateAccountNames(r.Context(), account.ID, firstName, lastName)
	if err != nil {
		httpserver.Logger(r.Context()).Error("update profile", "error", err)
		apierror.Internal(w)
		return
	}
	writeProfile(w, updated)
}

// parseUpdate validates a ProfileUpdate body, collecting every offending
// member, and returns the trimmed names that were supplied.
func parseUpdate(raw map[string]json.RawMessage) (firstName, lastName *string, fields map[string]string) {
	fields = map[string]string{}
	for key, value := range raw {
		switch key {
		case "firstName":
			firstName = parseName(key, value, fields)
		case "lastName":
			lastName = parseName(key, value, fields)
		case "email":
			fields[key] = apierror.ReasonReadOnly
		default:
			fields[key] = apierror.ReasonUnknown
		}
	}
	if len(fields) == 0 && firstName == nil && lastName == nil {
		fields[apierror.BodyField] = apierror.ReasonRequired
	}
	return firstName, lastName, fields
}

func parseName(key string, value json.RawMessage, fields map[string]string) *string {
	var name *string
	if err := json.Unmarshal(value, &name); err != nil || name == nil {
		fields[key] = apierror.ReasonInvalidType
		return nil
	}
	trimmed := strings.TrimSpace(*name)
	if utf8.RuneCountInString(trimmed) > MaxNameLength {
		fields[key] = apierror.ReasonTooLong
		return nil
	}
	return &trimmed
}
