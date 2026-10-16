// Package apierror writes the API's common error representation
// (components/schemas/Error in openapi/openapi.yaml): a stable,
// machine-readable code, a human-readable message and, for validation
// failures, machine-readable reasons keyed by request member name.
package apierror

import (
	"encoding/json"
	"net/http"
)

// Stable error codes.
const (
	CodeValidationFailed = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeNotFound         = "not_found"
	CodeInternal         = "internal_error"
)

// Field reasons reported in the error's fields map.
const (
	ReasonReadOnly       = "read_only"
	ReasonUnknown        = "unknown"
	ReasonInvalidType    = "invalid_type"
	ReasonTooLong        = "too_long"
	ReasonRequired       = "required"
	ReasonEmpty          = "empty"
	ReasonInvalidEnum    = "invalid_enum"
	ReasonInvalidDate    = "invalid_date"
	ReasonInvalidDecimal = "invalid_decimal"
	ReasonNegative       = "negative"
	ReasonNonPositive    = "non_positive"
	// ReasonBeforeFirstRegistration rejects a deregistration date earlier
	// than the car's first registration.
	ReasonBeforeFirstRegistration = "before_first_registration"
	// ReasonFuelMismatch rejects a refuel subtype that does not fit the
	// car's fuel (charging subtypes on an electric car, fuel grades otherwise).
	ReasonFuelMismatch = "fuel_mismatch"
	// ReasonFuelLocked rejects a car fuel change to or from electric while
	// the car has refuels.
	ReasonFuelLocked = "fuel_locked"
)

// BodyField is the fields key for a problem with the request body as a whole.
const BodyField = "_body"

type body struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Write sends an error response with the given status, code, message and
// optional field reasons.
func Write(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body{Code: code, Message: message, Fields: fields})
}

// Validation sends a 400 validation_failed response carrying field reasons.
func Validation(w http.ResponseWriter, fields map[string]string) {
	Write(w, http.StatusBadRequest, CodeValidationFailed, "request validation failed", fields)
}

// NotFound sends a 404 not_found response. It is also used for resources
// owned by another account, so their existence is not disclosed.
func NotFound(w http.ResponseWriter) {
	Write(w, http.StatusNotFound, CodeNotFound, "resource not found", nil)
}

// Internal sends a 500 internal_error response without exposing details.
func Internal(w http.ResponseWriter) {
	Write(w, http.StatusInternalServerError, CodeInternal, "internal server error", nil)
}
