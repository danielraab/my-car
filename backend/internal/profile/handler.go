// Package profile serves the authenticated caller's profile
// (GET/PATCH /api/v1/me), per specs/api-contract/profile.
package profile

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/currency"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

// MaxNameLength is the maximum length of a trimmed name, in characters.
const MaxNameLength = 100

const maxBodyBytes = 16 << 10

// Update holds the supplied members of a ProfileUpdate; nil means the member
// was absent and the stored value is kept.
type Update struct {
	FirstName, LastName                *string
	Currency, VolumeUnit, DistanceUnit *string
}

type Repository interface {
	UpdateProfile(ctx context.Context, accountID string, update Update) (auth.Account, error)
}

var (
	volumeUnits   = map[string]bool{"liter": true, "gallon": true}
	distanceUnits = map[string]bool{"kilometer": true, "mile": true}
)

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the profile operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/me", requireSession(http.HandlerFunc(h.getMe)))
	mux.Handle("PATCH /api/v1/me", requireSession(http.HandlerFunc(h.updateMe)))
}

func writeProfile(w http.ResponseWriter, a auth.Account) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a.Profile())
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
	update, fields := parseUpdate(raw)
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	updated, err := h.store.UpdateProfile(r.Context(), account.ID, update)
	if err != nil {
		httpserver.Logger(r.Context()).Error("update profile", "error", err)
		apierror.Internal(w)
		return
	}
	writeProfile(w, updated)
}

// parseUpdate validates a ProfileUpdate body, collecting every offending
// member, and returns the supplied members with names trimmed.
func parseUpdate(raw map[string]json.RawMessage) (update Update, fields map[string]string) {
	fields = map[string]string{}
	for key, value := range raw {
		switch key {
		case "firstName":
			update.FirstName = parseName(key, value, fields)
		case "lastName":
			update.LastName = parseName(key, value, fields)
		case "currency":
			update.Currency = parseSetting(key, value, fields, isCurrency)
		case "volumeUnit":
			update.VolumeUnit = parseSetting(key, value, fields, func(v string) bool { return volumeUnits[v] })
		case "distanceUnit":
			update.DistanceUnit = parseSetting(key, value, fields, func(v string) bool { return distanceUnits[v] })
		case "email":
			fields[key] = apierror.ReasonReadOnly
		default:
			fields[key] = apierror.ReasonUnknown
		}
	}
	if len(fields) == 0 && update == (Update{}) {
		fields[apierror.BodyField] = apierror.ReasonRequired
	}
	return update, fields
}

// isCurrency accepts exactly an uppercase ISO 4217 alphabetic code;
// currency.ParseISO alone also accepts lowercase input.
func isCurrency(code string) bool {
	if len(code) != 3 || strings.ToUpper(code) != code {
		return false
	}
	_, err := currency.ParseISO(code)
	return err == nil
}

func parseSetting(key string, value json.RawMessage, fields map[string]string, valid func(string) bool) *string {
	var setting *string
	if err := json.Unmarshal(value, &setting); err != nil || setting == nil {
		fields[key] = apierror.ReasonInvalidType
		return nil
	}
	if !valid(*setting) {
		fields[key] = apierror.ReasonInvalidEnum
		return nil
	}
	return setting
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
