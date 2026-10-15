package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"at.draab/my-car/internal/auth"
)

const (
	callerID = "11111111-1111-1111-1111-111111111111"
	otherID  = "99999999-9999-9999-9999-999999999999"
	refuelID = "22222222-2222-2222-2222-222222222222"
	foreign  = "33333333-3333-3333-3333-333333333333"
)

var (
	jpegBytes = append([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), bytes.Repeat([]byte{1}, 32)...)
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	gifBytes  = []byte("GIF89a\x01\x00\x01\x00")
	webpBytes = []byte("RIFF\x24\x00\x00\x00WEBPVP8 \x18\x00\x00\x00")
	pdfBytes  = []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
)

// memBlobs is an in-memory BlobStore.
type memBlobs struct {
	mu        sync.Mutex
	files     map[string][]byte
	deleteErr error
}

func newMemBlobs() *memBlobs { return &memBlobs{files: map[string][]byte{}} }

func (m *memBlobs) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key] = data
	return nil
}

func (m *memBlobs) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[key]
	if !ok {
		return nil, ErrBlobNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memBlobs) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.files, key)
	return nil
}

func (m *memBlobs) Check(context.Context) error { return nil }

func (m *memBlobs) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.files)
}

// fakeRepository models ownership like the real store: entries belong to
// accounts and attachments to entries.
type fakeRepository struct {
	entries     map[Owner]string
	attachments map[string]fakeAttachment
	insertErr   error
}

type fakeAttachment struct {
	Attachment
	owner   Owner
	account string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		entries:     map[Owner]string{{OwnerRefuel, refuelID}: callerID, {OwnerRefuel, foreign}: otherID},
		attachments: map[string]fakeAttachment{},
	}
}

func (f *fakeRepository) OwnerExists(_ context.Context, accountID string, owner Owner) error {
	if f.entries[owner] != accountID {
		return ErrNotFound
	}
	return nil
}

func (f *fakeRepository) List(ctx context.Context, accountID string, owner Owner) ([]Attachment, error) {
	if err := f.OwnerExists(ctx, accountID, owner); err != nil {
		return nil, err
	}
	var out []Attachment
	for _, a := range f.attachments {
		if a.owner == owner {
			out = append(out, a.Attachment)
		}
	}
	return out, nil
}

func (f *fakeRepository) Create(ctx context.Context, accountID string, owner Owner, in NewAttachment, put func(string) error) (Attachment, string, error) {
	if err := f.OwnerExists(ctx, accountID, owner); err != nil {
		return Attachment{}, "", err
	}
	list, _ := f.List(ctx, accountID, owner)
	if len(list) >= MaxFilesPerEntry {
		return Attachment{}, "", ErrLimitReached
	}
	id := uuid.NewString()
	key := storageKey(accountID, id)
	if err := put(key); err != nil {
		return Attachment{}, key, err
	}
	if f.insertErr != nil {
		return Attachment{}, key, f.insertErr
	}
	a := Attachment{ID: id, Filename: in.Filename, ContentType: in.ContentType, Size: in.Size, StorageKey: key, CreatedAt: time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)}
	f.attachments[id] = fakeAttachment{Attachment: a, owner: owner, account: accountID}
	return a, "", nil
}

func (f *fakeRepository) Get(_ context.Context, accountID, id string) (Attachment, error) {
	a, ok := f.attachments[id]
	if !ok || a.account != accountID {
		return Attachment{}, ErrNotFound
	}
	return a.Attachment, nil
}

func (f *fakeRepository) Delete(_ context.Context, accountID, id string) error {
	a, ok := f.attachments[id]
	if !ok || a.account != accountID {
		return ErrNotFound
	}
	delete(f.attachments, id)
	return nil
}

func sessionFor(accountID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithAccount(r.Context(), auth.Account{ID: accountID, Email: "me@example.com"})))
		})
	}
}

func noSession(next http.Handler) http.Handler { return next }

type env struct {
	repo    *fakeRepository
	blobs   *memBlobs
	mux     *http.ServeMux
	deleted int
}

func newEnv(t *testing.T, maxBytes int64, requireSession func(http.Handler) http.Handler) *env {
	t.Helper()
	e := &env{repo: newFakeRepository(), blobs: newMemBlobs(), mux: http.NewServeMux()}
	NewHandler(e.repo, e.blobs, maxBytes, func() { e.deleted++ }).RegisterRoutes(e.mux, requireSession)
	return e
}

func (e *env) do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

// uploadRequest builds a multipart request with a file part named field.
func uploadRequest(t *testing.T, target, field, filename, declaredType string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	header.Set("Content-Type", declaredType)
	part, err := mw.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return out
}

const refuelAttachments = "/api/v1/refuels/" + refuelID + "/attachments"

func TestUploadAcceptsImagesAndPDF(t *testing.T) {
	for want, content := range map[string][]byte{"image/jpeg": jpegBytes, "image/png": pngBytes, "image/gif": gifBytes, "image/webp": webpBytes, "application/pdf": pdfBytes} {
		e := newEnv(t, 1<<20, sessionFor(callerID))
		rec := e.do(uploadRequest(t, refuelAttachments, "file", "receipt", "application/octet-stream", content))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s: status = %d, body = %s", want, rec.Code, rec.Body)
		}
		body := decodeMap(t, rec)
		if body["contentType"] != want || body["size"] != float64(len(content)) || body["filename"] != "receipt" {
			t.Fatalf("%s: body = %v", want, body)
		}
		for _, key := range []string{"id", "filename", "contentType", "size", "createdAt"} {
			if _, ok := body[key]; !ok {
				t.Fatalf("%s: member %q missing", want, key)
			}
		}
		if e.blobs.count() != 1 {
			t.Fatalf("%s: stored %d files, want 1", want, e.blobs.count())
		}
	}
}

func TestUploadRejectsOtherContent(t *testing.T) {
	cases := map[string]struct {
		filename, declared string
		content            []byte
	}{
		"svg":          {"logo.svg", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		"html as jpeg": {"receipt.jpg", "image/jpeg", []byte("<!DOCTYPE html><html><script>alert(1)</script></html>")},
		"plain text":   {"notes.pdf", "application/pdf", []byte("just some text")},
		"zip":          {"archive.zip", "application/zip", []byte("PK\x03\x04\x14\x00\x00\x00")},
	}
	for name, tc := range cases {
		e := newEnv(t, 1<<20, sessionFor(callerID))
		rec := e.do(uploadRequest(t, refuelAttachments, "file", tc.filename, tc.declared, tc.content))
		if rec.Code != http.StatusUnsupportedMediaType || decodeMap(t, rec)["code"] != CodeUnsupportedMediaType {
			t.Fatalf("%s: status = %d, body = %s; want 415 unsupported_media_type", name, rec.Code, rec.Body)
		}
		if e.blobs.count() != 0 || len(e.repo.attachments) != 0 {
			t.Fatalf("%s: something was stored", name)
		}
	}
}

func TestUploadEnforcesSizeLimit(t *testing.T) {
	e := newEnv(t, int64(len(pdfBytes)), sessionFor(callerID))
	if rec := e.do(uploadRequest(t, refuelAttachments, "file", "exact.pdf", "application/pdf", pdfBytes)); rec.Code != http.StatusCreated {
		t.Fatalf("file at the limit: status = %d, body = %s", rec.Code, rec.Body)
	}
	tooLarge := append(append([]byte{}, pdfBytes...), 'x')
	rec := e.do(uploadRequest(t, refuelAttachments, "file", "big.pdf", "application/pdf", tooLarge))
	if rec.Code != http.StatusRequestEntityTooLarge || decodeMap(t, rec)["code"] != CodeFileTooLarge {
		t.Fatalf("status = %d, body = %s; want 413 file_too_large", rec.Code, rec.Body)
	}
	if len(e.repo.attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(e.repo.attachments))
	}
}

func TestUploadRejectsEleventhFile(t *testing.T) {
	e := newEnv(t, 1<<20, sessionFor(callerID))
	for i := 0; i < MaxFilesPerEntry; i++ {
		if rec := e.do(uploadRequest(t, refuelAttachments, "file", "r.pdf", "application/pdf", pdfBytes)); rec.Code != http.StatusCreated {
			t.Fatalf("upload %d: status = %d", i+1, rec.Code)
		}
	}
	rec := e.do(uploadRequest(t, refuelAttachments, "file", "r.pdf", "application/pdf", pdfBytes))
	if rec.Code != http.StatusConflict || decodeMap(t, rec)["code"] != CodeAttachmentLimitReached {
		t.Fatalf("status = %d, body = %s; want 409 attachment_limit_reached", rec.Code, rec.Body)
	}
	if e.blobs.count() != MaxFilesPerEntry {
		t.Fatalf("stored files = %d, want %d", e.blobs.count(), MaxFilesPerEntry)
	}
}

func TestUploadValidatesParts(t *testing.T) {
	e := newEnv(t, 1<<20, sessionFor(callerID))
	rec := e.do(uploadRequest(t, refuelAttachments, "document", "r.pdf", "application/pdf", pdfBytes))
	fields, _ := decodeMap(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["file"] != "required" || fields["document"] != "unknown" {
		t.Fatalf("status = %d, body = %s; want file required and document unknown", rec.Code, rec.Body)
	}

	rec = e.do(uploadRequest(t, refuelAttachments, "file", "empty.pdf", "application/pdf", nil))
	fields, _ = decodeMap(t, rec)["fields"].(map[string]any)
	if rec.Code != http.StatusBadRequest || fields["file"] != "empty" {
		t.Fatalf("empty file: status = %d, body = %s", rec.Code, rec.Body)
	}

	req := httptest.NewRequest(http.MethodPost, refuelAttachments, strings.NewReader(`{"file":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	if rec := e.do(req); rec.Code != http.StatusBadRequest {
		t.Fatalf("non-multipart: status = %d, want 400", rec.Code)
	}
	if e.blobs.count() != 0 {
		t.Fatal("something was stored")
	}
}

func TestUploadRemovesFileWhenRecordFails(t *testing.T) {
	e := newEnv(t, 1<<20, sessionFor(callerID))
	e.repo.insertErr = errors.New("insert failed")
	rec := e.do(uploadRequest(t, refuelAttachments, "file", "r.pdf", "application/pdf", pdfBytes))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if e.blobs.count() != 0 {
		t.Fatal("stored file without a record was not removed")
	}
}

func TestOtherAccountsAttachmentsAreNotFound(t *testing.T) {
	owner := newEnv(t, 1<<20, sessionFor(otherID))
	rec := owner.do(uploadRequest(t, "/api/v1/refuels/"+foreign+"/attachments", "file", "r.pdf", "application/pdf", pdfBytes))
	if rec.Code != http.StatusCreated {
		t.Fatalf("owner upload: status = %d", rec.Code)
	}
	id := decodeMap(t, rec)["id"].(string)

	// The caller shares the owner's data but not their session.
	caller := &env{repo: owner.repo, blobs: owner.blobs, mux: http.NewServeMux()}
	NewHandler(caller.repo, caller.blobs, 1<<20, func() { caller.deleted++ }).RegisterRoutes(caller.mux, sessionFor(callerID))
	for name, req := range map[string]*http.Request{
		"list":     httptest.NewRequest(http.MethodGet, "/api/v1/refuels/"+foreign+"/attachments", nil),
		"upload":   uploadRequest(t, "/api/v1/refuels/"+foreign+"/attachments", "file", "r.pdf", "application/pdf", pdfBytes),
		"download": httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+id+"/content", nil),
		"delete":   httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/"+id, nil),
	} {
		rec := caller.do(req)
		if rec.Code != http.StatusNotFound || bytes.Contains(rec.Body.Bytes(), pdfBytes) {
			t.Errorf("%s: status = %d, body = %q; want 404 without content", name, rec.Code, rec.Body)
		}
		if cc := rec.Header().Get("Cache-Control"); strings.Contains(cc, "max-age") {
			t.Errorf("%s: Cache-Control = %q; a 404 must not be cacheable", name, cc)
		}
	}
	if len(owner.repo.attachments) != 1 || owner.blobs.count() != 1 || caller.deleted != 0 {
		t.Fatal("another account's request changed the attachments")
	}
	if rec := caller.do(httptest.NewRequest(http.MethodGet, "/api/v1/attachments/not-a-uuid/content", nil)); rec.Code != http.StatusNotFound {
		t.Fatalf("malformed ID: status = %d, want 404", rec.Code)
	}
}

func TestAttachmentEndpointsRequireSession(t *testing.T) {
	e := newEnv(t, 1<<20, noSession)
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, refuelAttachments, nil),
		uploadRequest(t, refuelAttachments, "file", "r.pdf", "application/pdf", pdfBytes),
		httptest.NewRequest(http.MethodGet, "/api/v1/attachments/limits", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+refuelID+"/content", nil),
		httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/"+refuelID, nil),
	} {
		if rec := e.do(req); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestDownloadServesProtectiveHeaders(t *testing.T) {
	e := newEnv(t, 1<<20, sessionFor(callerID))
	rec := e.do(uploadRequest(t, refuelAttachments, "file", "Tankbeleg März.jpg", "image/jpeg", jpegBytes))
	id := decodeMap(t, rec)["id"].(string)

	rec = e.do(httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+id+"/content", nil))
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), jpegBytes) {
		t.Fatalf("status = %d, body length %d", rec.Code, rec.Body.Len())
	}
	h := rec.Header()
	for key, want := range map[string]string{
		"Content-Type":            "image/jpeg",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "sandbox",
		"Cache-Control":           "private, max-age=3600, immutable",
		"Content-Disposition":     "inline; filename*=utf-8''Tankbeleg%20M%C3%A4rz.jpg",
	} {
		if got := h.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestListAndDelete(t *testing.T) {
	e := newEnv(t, 1<<20, sessionFor(callerID))
	rec := e.do(uploadRequest(t, refuelAttachments, "file", "r.pdf", "application/pdf", pdfBytes))
	id := decodeMap(t, rec)["id"].(string)

	rec = e.do(httptest.NewRequest(http.MethodGet, refuelAttachments, nil))
	body := decodeMap(t, rec)
	if items, _ := body["items"].([]any); rec.Code != http.StatusOK || len(items) != 1 || body["nextCursor"] != nil {
		t.Fatalf("list: status = %d, body = %v", rec.Code, body)
	}
	if _, ok := body["nextCursor"]; !ok {
		t.Fatal("list: nextCursor member missing")
	}

	if rec := e.do(httptest.NewRequest(http.MethodDelete, "/api/v1/attachments/"+id, nil)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", rec.Code)
	}
	if e.deleted != 1 {
		t.Fatalf("deleted callback ran %d times, want 1", e.deleted)
	}
	if rec := e.do(httptest.NewRequest(http.MethodGet, "/api/v1/attachments/"+id+"/content", nil)); rec.Code != http.StatusNotFound {
		t.Fatalf("download after delete: status = %d, want 404", rec.Code)
	}
}

func TestLimits(t *testing.T) {
	e := newEnv(t, 10485760, sessionFor(callerID))
	rec := e.do(httptest.NewRequest(http.MethodGet, "/api/v1/attachments/limits", nil))
	body := decodeMap(t, rec)
	types, _ := body["allowedContentTypes"].([]any)
	if rec.Code != http.StatusOK || body["maxFileBytes"] != float64(10485760) || body["maxFilesPerEntry"] != float64(10) || len(types) != 5 {
		t.Fatalf("status = %d, body = %v", rec.Code, body)
	}
}

func TestSanitizeFilename(t *testing.T) {
	for raw, want := range map[string]string{
		`C:\scans\invoice.pdf`:          "invoice.pdf",
		"../../etc/passwd.pdf":          "passwd.pdf",
		"  receipt\x00\n.jpg ":          "receipt.jpg",
		"":                              "file",
		"..":                            "file",
		strings.Repeat("ä", 300) + ".x": strings.Repeat("ä", 255),
	} {
		if got := sanitizeFilename(raw); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", raw, got, want)
		}
	}
}
