// Package stats serves the dashboard's expense statistics
// (/api/v1/stats/expenses), per specs/api-contract/expenses.
package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Repository interface {
	Expenses(ctx context.Context, accountID string, f Filter) ([]Expense, error)
}

type Handler struct{ store Repository }

func NewHandler(store Repository) *Handler { return &Handler{store: store} }

// RegisterRoutes mounts the statistics operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/stats/expenses", requireSession(h.withAccount(h.expenses)))
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

type expenseBody struct {
	Date   string `json:"date"`
	Kind   string `json:"kind"`
	Amount string `json:"amount"`
}

// expenses returns rows rather than per-day or per-month totals: the
// buckets depend on the viewer's time zone, which only the client knows.
func (h *Handler) expenses(w http.ResponseWriter, r *http.Request, accountID string) {
	query := r.URL.Query()
	fields := map[string]string{}
	var f Filter
	if raw := query.Get("carId"); raw != "" {
		if !uuidPattern.MatchString(raw) {
			fields["carId"] = apierror.ReasonInvalidType
		}
		f.CarID = &raw
	}
	for key, target := range map[string]*time.Time{"from": &f.From, "to": &f.To} {
		raw := query.Get(key)
		if raw == "" {
			fields[key] = apierror.ReasonRequired
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			fields[key] = apierror.ReasonInvalidType
		}
		*target = t
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	items, err := h.store.Expenses(r.Context(), accountID, f)
	if err != nil {
		httpserver.Logger(r.Context()).Error("expense statistics", "error", err)
		apierror.Internal(w)
		return
	}
	body := struct {
		Items []expenseBody `json:"items"`
	}{Items: make([]expenseBody, 0, len(items))}
	for _, e := range items {
		body.Items = append(body.Items, expenseBody{Date: e.Date.UTC().Format(time.RFC3339Nano), Kind: e.Kind, Amount: e.Amount})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}
