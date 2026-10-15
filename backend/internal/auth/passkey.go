package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/httpserver"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

const (
	passkeyRegistrationCookie = "my_car_passkey_registration"
	passkeyLoginCookie        = "my_car_passkey_login"

	// freshLoginWindow is how recently a session must have been established
	// to register a passkey.
	freshLoginWindow = 5 * time.Minute

	maxPasskeyNameLength = 100
	maxPasskeyBodyBytes  = 64 << 10

	codeReauthenticationRequired = "reauthentication_required"
	codePasskeyRejected          = "passkey_rejected"
	codePasskeyAlreadyRegistered = "passkey_already_registered"
)

var passkeyIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *Service) registerPasskeyRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/auth/passkey/options", s.limit(groupPasskeyLogin, limitJSON, s.startPasskeyLogin))
	mux.Handle("POST /api/v1/auth/passkey", s.limit(groupPasskeyLogin, limitJSON, s.completePasskeyLogin))
	mux.Handle("GET /api/v1/passkeys", s.RequireSession(http.HandlerFunc(s.listPasskeys)))
	mux.Handle("POST /api/v1/passkeys", s.RequireSession(http.HandlerFunc(s.registerPasskey)))
	mux.Handle("POST /api/v1/passkeys/registration-options", s.RequireSession(http.HandlerFunc(s.startPasskeyRegistration)))
	mux.Handle("PATCH /api/v1/passkeys/{passkeyId}", s.RequireSession(http.HandlerFunc(s.renamePasskey)))
	mux.Handle("DELETE /api/v1/passkeys/{passkeyId}", s.RequireSession(http.HandlerFunc(s.deletePasskey)))
}

// passkeyUser adapts an account and its passkeys to the WebAuthn library. The
// user handle is the account UUID's 16 bytes.
type passkeyUser struct {
	account  Account
	passkeys []Passkey
}

func (u passkeyUser) WebAuthnID() []byte {
	id, err := uuid.Parse(u.account.ID)
	if err != nil {
		return nil
	}
	return id[:]
}
func (u passkeyUser) WebAuthnName() string { return u.account.Email }
func (u passkeyUser) WebAuthnDisplayName() string {
	if name := strings.TrimSpace(u.account.FirstName + " " + u.account.LastName); name != "" {
		return name
	}
	return u.account.Email
}
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	out := make([]webauthn.Credential, 0, len(u.passkeys))
	for _, p := range u.passkeys {
		transports := make([]protocol.AuthenticatorTransport, 0, len(p.Transports))
		for _, t := range p.Transports {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		out = append(out, webauthn.Credential{
			ID: p.CredentialID, PublicKey: p.PublicKey, AttestationFormat: p.AttestationFormat, Transport: transports,
			Flags:         webauthn.CredentialFlags{UserPresent: true, UserVerified: p.UserVerified, BackupEligible: p.BackupEligible, BackupState: p.BackupState},
			Authenticator: webauthn.Authenticator{AAGUID: p.AAGUID, SignCount: p.SignCount},
		})
	}
	return out
}

type passkeySummary struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	CreatedAt         string  `json:"createdAt"`
	LastUsedAt        *string `json:"lastUsedAt"`
	AuthenticatorName *string `json:"authenticatorName"`
	BackedUp          bool    `json:"backedUp"`
}

func toPasskeySummary(p Passkey) passkeySummary {
	out := passkeySummary{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano), AuthenticatorName: AuthenticatorName(p.AAGUID), BackedUp: p.BackupState}
	if p.LastUsedAt != nil {
		used := p.LastUsedAt.UTC().Format(time.RFC3339Nano)
		out.LastUsedAt = &used
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func currentSession(r *http.Request) authenticatedSession {
	return r.Context().Value(sessionContextKey{}).(authenticatedSession)
}

// requireFreshLogin answers 403 reauthentication_required unless the session
// was established within the fresh-login window.
func (s *Service) requireFreshLogin(w http.ResponseWriter, session Session) bool {
	if s.now().Sub(session.CreatedAt) <= freshLoginWindow {
		return true
	}
	apierror.Write(w, http.StatusForbidden, codeReauthenticationRequired, "a login within the last 5 minutes is required", nil)
	return false
}

func setCeremonyCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/api/v1", MaxAge: int(passkeyCeremonyTTL / time.Second), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
}
func clearCeremonyCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/api/v1", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
}

// beginCeremony persists the library's ceremony state under a fresh
// correlation token, sets that token as an HttpOnly cookie, and returns the
// browser options.
func (s *Service) beginCeremony(w http.ResponseWriter, ctx context.Context, cookie, kind, accountID string, data *webauthn.SessionData, options any) {
	token, digest, err := NewCredential()
	if err != nil {
		apierror.Internal(w)
		return
	}
	state, err := json.Marshal(data)
	if err != nil {
		apierror.Internal(w)
		return
	}
	if err = s.store.CreatePasskeyChallenge(ctx, digest, kind, accountID, state, s.now().Add(passkeyCeremonyTTL)); err != nil {
		httpserver.Logger(ctx).Error("create passkey challenge", "kind", kind, "error", err)
		apierror.Internal(w)
		return
	}
	setCeremonyCookie(w, cookie, token)
	writeJSON(w, http.StatusOK, options)
}

// consumeCeremony resolves and consumes the ceremony named by the cookie and
// clears the cookie.
func (s *Service) consumeCeremony(w http.ResponseWriter, r *http.Request, cookie, kind, accountID string) (webauthn.SessionData, error) {
	var data webauthn.SessionData
	c, err := r.Cookie(cookie)
	if err != nil || c.Value == "" {
		return data, ErrInvalidCredential
	}
	clearCeremonyCookie(w, cookie)
	state, err := s.store.ConsumePasskeyChallenge(r.Context(), Digest(c.Value), kind, accountID, s.now())
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(state, &data); err != nil {
		return data, err
	}
	return data, nil
}

func (s *Service) startPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	session := currentSession(r).Session
	if !s.requireFreshLogin(w, session) {
		return
	}
	passkeys, err := s.store.ListPasskeys(r.Context(), session.ID)
	if err != nil {
		httpserver.Logger(r.Context()).Error("list passkeys", "error", err)
		apierror.Internal(w)
		return
	}
	user := passkeyUser{account: session.Account, passkeys: passkeys}
	creation, data, err := s.rp.BeginRegistration(user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, RequireResidentKey: protocol.ResidentKeyRequired(), UserVerification: protocol.VerificationRequired}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
	)
	if err != nil {
		httpserver.Logger(r.Context()).Error("begin passkey registration", "error", err)
		apierror.Internal(w)
		return
	}
	s.beginCeremony(w, r.Context(), passkeyRegistrationCookie, ceremonyRegistration, session.ID, data, creation)
}

// parsePasskeyName validates a raw JSON name member, recording a reason in
// fields when it is invalid.
func parsePasskeyName(raw json.RawMessage, fields map[string]string) string {
	var name string
	if raw == nil {
		fields["name"] = apierror.ReasonRequired
		return ""
	}
	if json.Unmarshal(raw, &name) != nil {
		fields["name"] = apierror.ReasonInvalidType
		return ""
	}
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		fields["name"] = apierror.ReasonEmpty
	case utf8.RuneCountInString(name) > maxPasskeyNameLength:
		fields["name"] = apierror.ReasonTooLong
	}
	return name
}

// decodeObject reads a JSON object body, reporting unknown members.
func decodeObject(w http.ResponseWriter, r *http.Request, known ...string) (map[string]json.RawMessage, map[string]string, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPasskeyBodyBytes)).Decode(&raw); err != nil || raw == nil {
		apierror.Validation(w, nil)
		return nil, nil, false
	}
	fields := map[string]string{}
	for key := range raw {
		isKnown := false
		for _, k := range known {
			isKnown = isKnown || key == k
		}
		if !isKnown {
			fields[key] = apierror.ReasonUnknown
		}
	}
	return raw, fields, true
}

func (s *Service) registerPasskey(w http.ResponseWriter, r *http.Request) {
	session := currentSession(r).Session
	if !s.requireFreshLogin(w, session) {
		return
	}
	raw, fields, ok := decodeObject(w, r, "name", "credential")
	if !ok {
		return
	}
	name := parsePasskeyName(raw["name"], fields)
	credential := raw["credential"]
	if len(credential) == 0 {
		fields["credential"] = apierror.ReasonRequired
	} else if credential[0] != '{' {
		fields["credential"] = apierror.ReasonInvalidType
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	data, err := s.consumeCeremony(w, r, passkeyRegistrationCookie, ceremonyRegistration, session.ID)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, codePasskeyRejected, "passkey registration could not be verified", nil)
		return
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(credential)
	if err != nil {
		apierror.Write(w, http.StatusBadRequest, codePasskeyRejected, "passkey registration could not be verified", nil)
		return
	}
	cred, err := s.rp.CreateCredential(passkeyUser{account: session.Account}, data, parsed)
	if err != nil {
		httpserver.Logger(r.Context()).Warn("verify passkey registration", "error", err)
		apierror.Write(w, http.StatusBadRequest, codePasskeyRejected, "passkey registration could not be verified", nil)
		return
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	var aaguid []byte
	if len(cred.Authenticator.AAGUID) == 16 {
		aaguid = cred.Authenticator.AAGUID
	}
	created, err := s.store.CreatePasskey(r.Context(), Passkey{
		AccountID: session.ID, CredentialID: cred.ID, PublicKey: cred.PublicKey, AAGUID: aaguid,
		SignCount: cred.Authenticator.SignCount, Transports: transports, AttestationFormat: cred.AttestationFormat,
		UserVerified: cred.Flags.UserVerified, BackupEligible: cred.Flags.BackupEligible, BackupState: cred.Flags.BackupState,
		Name: name,
	})
	if errors.Is(err, ErrDuplicatePasskey) {
		apierror.Write(w, http.StatusConflict, codePasskeyAlreadyRegistered, "passkey is already registered", nil)
		return
	}
	if err != nil {
		httpserver.Logger(r.Context()).Error("store passkey", "error", err)
		apierror.Internal(w)
		return
	}
	writeJSON(w, http.StatusCreated, toPasskeySummary(created))
}

func (s *Service) listPasskeys(w http.ResponseWriter, r *http.Request) {
	passkeys, err := s.store.ListPasskeys(r.Context(), currentSession(r).ID)
	if err != nil {
		httpserver.Logger(r.Context()).Error("list passkeys", "error", err)
		apierror.Internal(w)
		return
	}
	items := make([]passkeySummary, 0, len(passkeys))
	for _, p := range passkeys {
		items = append(items, toPasskeySummary(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) renamePasskey(w http.ResponseWriter, r *http.Request) {
	passkeyID := r.PathValue("passkeyId")
	if !passkeyIDPattern.MatchString(passkeyID) {
		apierror.NotFound(w)
		return
	}
	raw, fields, ok := decodeObject(w, r, "name")
	if !ok {
		return
	}
	name := parsePasskeyName(raw["name"], fields)
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}
	renamed, err := s.store.RenamePasskey(r.Context(), currentSession(r).ID, passkeyID, name)
	if errors.Is(err, ErrPasskeyNotFound) {
		apierror.NotFound(w)
		return
	}
	if err != nil {
		httpserver.Logger(r.Context()).Error("rename passkey", "error", err)
		apierror.Internal(w)
		return
	}
	writeJSON(w, http.StatusOK, toPasskeySummary(renamed))
}

func (s *Service) deletePasskey(w http.ResponseWriter, r *http.Request) {
	passkeyID := r.PathValue("passkeyId")
	if !passkeyIDPattern.MatchString(passkeyID) {
		apierror.NotFound(w)
		return
	}
	session := currentSession(r).Session
	err := s.store.DeletePasskey(r.Context(), session.ID, passkeyID, s.now())
	if errors.Is(err, ErrPasskeyNotFound) {
		apierror.NotFound(w)
		return
	}
	if err != nil {
		httpserver.Logger(r.Context()).Error("delete passkey", "error", err)
		apierror.Internal(w)
		return
	}
	if strings.EqualFold(session.PasskeyID, passkeyID) {
		ClearSessionCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) startPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	assertion, data, err := s.rp.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		httpserver.Logger(r.Context()).Error("begin passkey login", "error", err)
		apierror.Internal(w)
		return
	}
	s.beginCeremony(w, r.Context(), passkeyLoginCookie, ceremonyLogin, "", data, assertion)
}

func rejectPasskeyLogin(w http.ResponseWriter) {
	apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized, "passkey could not be verified", nil)
}

func (s *Service) completePasskeyLogin(w http.ResponseWriter, r *http.Request) {
	data, err := s.consumeCeremony(w, r, passkeyLoginCookie, ceremonyLogin, "")
	if err != nil {
		rejectPasskeyLogin(w)
		return
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(http.MaxBytesReader(w, r.Body, maxPasskeyBodyBytes))
	if err != nil {
		rejectPasskeyLogin(w)
		return
	}
	// The library checks that the user handle names this account and that
	// the asserted credential is one of its passkeys.
	owner := func(_, userHandle []byte) (webauthn.User, error) {
		id, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, err
		}
		account, passkeys, err := s.store.AccountPasskeys(r.Context(), id.String())
		if err != nil {
			return nil, err
		}
		return passkeyUser{account: account, passkeys: passkeys}, nil
	}
	_, cred, err := s.rp.ValidatePasskeyLogin(owner, data, parsed)
	if err != nil {
		httpserver.Logger(r.Context()).Warn("verify passkey login", "error", err)
		rejectPasskeyLogin(w)
		return
	}
	if cred.Authenticator.CloneWarning {
		httpserver.Logger(r.Context()).Warn("verify passkey login: signature counter did not increase; possible cloned authenticator")
		rejectPasskeyLogin(w)
		return
	}
	token, digest, err := NewCredential()
	if err != nil {
		apierror.Internal(w)
		return
	}
	expires := s.now().Add(sessionLifetime)
	err = s.store.CompletePasskeyLogin(r.Context(), cred.ID, cred.Authenticator.SignCount, cred.Flags.BackupState, s.now(), digest, expires)
	if errors.Is(err, ErrInvalidCredential) {
		rejectPasskeyLogin(w)
		return
	}
	if err != nil {
		httpserver.Logger(r.Context()).Error("record passkey login", "error", err)
		apierror.Internal(w)
		return
	}
	SetSessionCookie(w, token, expires)
	w.WriteHeader(http.StatusNoContent)
}
