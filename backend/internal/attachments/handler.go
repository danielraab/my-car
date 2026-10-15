package attachments

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/httpserver"
)

// Error codes specific to uploads.
const (
	CodeUnsupportedMediaType   = "unsupported_media_type"
	CodeFileTooLarge           = "file_too_large"
	CodeAttachmentLimitReached = "attachment_limit_reached"
)

const (
	// multipartOverhead bounds the request body beyond the file itself:
	// boundaries and part headers.
	multipartOverhead = 64 << 10
	maxFilenameLength = 255
	// sniffLength is how much http.DetectContentType looks at.
	sniffLength = 512
)

// AllowedContentTypes are the detected types an upload may have, in the order
// the limits operation lists them.
var AllowedContentTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif", "application/pdf"}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Repository interface {
	OwnerExists(ctx context.Context, accountID string, owner Owner) error
	List(ctx context.Context, accountID string, owner Owner) ([]Attachment, error)
	Create(ctx context.Context, accountID string, owner Owner, in NewAttachment, put func(key string) error) (Attachment, string, error)
	Get(ctx context.Context, accountID, attachmentID string) (Attachment, error)
	Delete(ctx context.Context, accountID, attachmentID string) error
}

type Handler struct {
	store          Repository
	blobs          BlobStore
	maxUploadBytes int64
	// deleted is called after an attachment was deleted, so its queued file
	// is removed without waiting for the next periodic run.
	deleted func()
}

func NewHandler(store Repository, blobs BlobStore, maxUploadBytes int64, deleted func()) *Handler {
	if deleted == nil {
		deleted = func() {}
	}
	return &Handler{store: store, blobs: blobs, maxUploadBytes: maxUploadBytes, deleted: deleted}
}

// RegisterRoutes mounts the attachment operations behind requireSession.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireSession func(http.Handler) http.Handler) {
	for _, route := range []struct {
		prefix, param string
		kind          OwnerKind
	}{
		{"/api/v1/cars/{carId}", "carId", OwnerCar},
		{"/api/v1/refuels/{refuelId}", "refuelId", OwnerRefuel},
		{"/api/v1/repairs/{repairId}", "repairId", OwnerRepair},
		{"/api/v1/tickets/{ticketId}", "ticketId", OwnerTicket},
	} {
		mux.Handle("GET "+route.prefix+"/attachments", requireSession(h.withOwner(route.param, route.kind, h.list)))
		mux.Handle("POST "+route.prefix+"/attachments", requireSession(h.withOwner(route.param, route.kind, h.upload)))
	}
	mux.Handle("GET /api/v1/attachments/limits", requireSession(h.withAccount(h.limits)))
	mux.Handle("GET /api/v1/attachments/{attachmentId}/content", requireSession(h.withAttachment(h.download)))
	mux.Handle("DELETE /api/v1/attachments/{attachmentId}", requireSession(h.withAttachment(h.delete)))
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

// withOwner resolves the path's entry ID; one that is not a UUID cannot name
// an entry, so it is reported like any other missing entry.
func (h *Handler) withOwner(param string, kind OwnerKind, next func(http.ResponseWriter, *http.Request, string, Owner)) http.Handler {
	return h.withAccount(func(w http.ResponseWriter, r *http.Request, accountID string) {
		id := r.PathValue(param)
		if !uuidPattern.MatchString(id) {
			apierror.NotFound(w)
			return
		}
		next(w, r, accountID, Owner{Kind: kind, ID: id})
	})
}

func (h *Handler) withAttachment(next func(http.ResponseWriter, *http.Request, string, string)) http.Handler {
	return h.withAccount(func(w http.ResponseWriter, r *http.Request, accountID string) {
		id := r.PathValue("attachmentId")
		if !uuidPattern.MatchString(id) {
			apierror.NotFound(w)
			return
		}
		next(w, r, accountID, id)
	})
}

type attachmentBody struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"createdAt"`
}

func toBody(a Attachment) attachmentBody {
	return attachmentBody{ID: a.ID, Filename: a.Filename, ContentType: a.ContentType, Size: a.Size, CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierror.NotFound(w)
	case errors.Is(err, ErrLimitReached):
		apierror.Write(w, http.StatusConflict, CodeAttachmentLimitReached, "the entry already has the maximum number of attachments", nil)
	default:
		httpserver.Logger(r.Context()).Error("attachment failed", "op", op, "error", err)
		apierror.Internal(w)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, accountID string, owner Owner) {
	items, err := h.store.List(r.Context(), accountID, owner)
	if err != nil {
		h.fail(w, r, "list", err)
		return
	}
	page := struct {
		Items      []attachmentBody `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}{Items: make([]attachmentBody, 0, len(items))}
	for _, a := range items {
		page.Items = append(page.Items, toBody(a))
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) limits(w http.ResponseWriter, _ *http.Request, _ string) {
	writeJSON(w, http.StatusOK, struct {
		MaxFileBytes        int64    `json:"maxFileBytes"`
		MaxFilesPerEntry    int      `json:"maxFilesPerEntry"`
		AllowedContentTypes []string `json:"allowedContentTypes"`
	}{h.maxUploadBytes, MaxFilesPerEntry, AllowedContentTypes})
}

func (h *Handler) fileTooLarge(w http.ResponseWriter) {
	apierror.Write(w, http.StatusRequestEntityTooLarge, CodeFileTooLarge, "the file exceeds the maximum upload size of "+strconv.FormatInt(h.maxUploadBytes, 10)+" bytes", nil)
}

// upload checks ownership before reading the body, spools the file part to a
// temporary file (enforcing the size limit while reading), detects its type
// from its content, and only then stores it.
func (h *Handler) upload(w http.ResponseWriter, r *http.Request, accountID string, owner Owner) {
	if err := h.store.OwnerExists(r.Context(), accountID, owner); err != nil {
		h.fail(w, r, "upload", err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes+multipartOverhead)
	reader, err := r.MultipartReader()
	if err != nil {
		apierror.Validation(w, map[string]string{apierror.BodyField: apierror.ReasonInvalidType})
		return
	}
	fields := map[string]string{}
	var spooled *os.File
	var size int64
	var filename string
	defer func() {
		if spooled != nil {
			spooled.Close()
			os.Remove(spooled.Name())
		}
	}()
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if isTooLarge(err) {
				h.fileTooLarge(w)
				return
			}
			apierror.Validation(w, map[string]string{apierror.BodyField: apierror.ReasonInvalidType})
			return
		}
		name := part.FormName()
		if name != "file" {
			fields[name] = apierror.ReasonUnknown
			part.Close()
			continue
		}
		if spooled != nil || part.FileName() == "" && !hasFilenameParam(part) {
			fields["file"] = apierror.ReasonInvalidType
			part.Close()
			continue
		}
		spooled, err = os.CreateTemp("", "my-car-upload-*")
		if err != nil {
			h.fail(w, r, "upload", err)
			return
		}
		size, err = io.Copy(spooled, io.LimitReader(part, h.maxUploadBytes+1))
		if err != nil {
			if isTooLarge(err) {
				h.fileTooLarge(w)
				return
			}
			apierror.Validation(w, map[string]string{apierror.BodyField: apierror.ReasonInvalidType})
			return
		}
		if size > h.maxUploadBytes {
			h.fileTooLarge(w)
			return
		}
		filename = sanitizeFilename(part.FileName())
		part.Close()
	}
	if spooled == nil {
		if _, reported := fields["file"]; !reported {
			fields["file"] = apierror.ReasonRequired
		}
	} else if size == 0 {
		fields["file"] = apierror.ReasonEmpty
	}
	if len(fields) > 0 {
		apierror.Validation(w, fields)
		return
	}

	contentType, err := detectContentType(spooled)
	if err != nil {
		h.fail(w, r, "upload", err)
		return
	}
	if contentType == "" {
		apierror.Write(w, http.StatusUnsupportedMediaType, CodeUnsupportedMediaType, "only JPEG, PNG, WebP and GIF images and PDF documents are accepted", nil)
		return
	}

	in := NewAttachment{Filename: filename, ContentType: contentType, Size: size}
	a, orphan, err := h.store.Create(r.Context(), accountID, owner, in, func(key string) error {
		if _, err := spooled.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return h.blobs.Put(r.Context(), key, spooled, size, contentType)
	})
	if orphan != "" {
		if err := h.blobs.Delete(context.WithoutCancel(r.Context()), orphan); err != nil {
			httpserver.Logger(r.Context()).Error("remove unrecorded attachment file", "error", err)
		}
	}
	if err != nil {
		h.fail(w, r, "upload", err)
		return
	}
	writeJSON(w, http.StatusCreated, toBody(a))
}

func isTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

// hasFilenameParam reports a file part whose filename parameter is present
// but sanitizes to nothing, such as filename="".
func hasFilenameParam(part *multipart.Part) bool {
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return false
	}
	_, ok := params["filename"]
	return ok
}

// detectContentType sniffs the file's leading bytes and returns its type if
// it is allowed, or "" otherwise. Client-declared types are never consulted.
func detectContentType(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	head := make([]byte, sniffLength)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	detected, _, _ := strings.Cut(http.DetectContentType(head[:n]), ";")
	for _, allowed := range AllowedContentTypes {
		if detected == allowed {
			return detected, nil
		}
	}
	return "", nil
}

// sanitizeFilename keeps an uploaded name for display only: its last path
// segment (for either separator), without control characters, at most 255
// characters, and "file" when nothing usable remains.
func sanitizeFilename(raw string) string {
	if i := strings.LastIndexAny(raw, `/\`); i >= 0 {
		raw = raw[i+1:]
	}
	raw = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, raw)
	raw = strings.TrimSpace(raw)
	if utf8.RuneCountInString(raw) > maxFilenameLength {
		raw = strings.TrimSpace(string([]rune(raw)[:maxFilenameLength]))
	}
	if raw == "" || raw == "." || raw == ".." {
		return "file"
	}
	return raw
}

// download streams the stored file with headers that keep it from running
// in the app's origin or being kept by a shared cache.
func (h *Handler) download(w http.ResponseWriter, r *http.Request, accountID, attachmentID string) {
	a, err := h.store.Get(r.Context(), accountID, attachmentID)
	if err != nil {
		h.fail(w, r, "download", err)
		return
	}
	content, err := h.blobs.Get(r.Context(), a.StorageKey)
	if err != nil {
		h.fail(w, r, "download", err)
		return
	}
	defer content.Close()
	header := w.Header()
	header.Set("Content-Type", a.ContentType)
	header.Set("Content-Length", strconv.FormatInt(a.Size, 10))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "sandbox")
	header.Set("Cache-Control", "private, max-age=3600, immutable")
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": a.Filename})
	if disposition == "" {
		disposition = "inline"
	}
	header.Set("Content-Disposition", disposition)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, content); err != nil {
		httpserver.Logger(r.Context()).Warn("attachment download interrupted", "error", err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, accountID, attachmentID string) {
	if err := h.store.Delete(r.Context(), accountID, attachmentID); err != nil {
		h.fail(w, r, "delete", err)
		return
	}
	h.deleted()
	w.WriteHeader(http.StatusNoContent)
}
