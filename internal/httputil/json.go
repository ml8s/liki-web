package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

const (
	// DefaultJSONBodyLimit bounds all first-party JSON request bodies. A
	// route may pass a smaller limit through DecodeOptions.
	DefaultJSONBodyLimit = 1 << 20
)

type DecodeOptions struct {
	// MaxBytes bounds the request body. Zero uses DefaultJSONBodyLimit.
	MaxBytes int64
}

// DecodeStrictJSON decodes one first-party JSON request document. It owns the
// transport contract: JSON content type, body size, malformed JSON, unknown
// fields, and trailing data. Domain validation stays in the handler/domain.
func DecodeStrictJSON[T any](w http.ResponseWriter, r *http.Request, opts DecodeOptions) (T, bool) {
	var req T

	if !hasJSONContentType(r) {
		RespondError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return req, false
	}

	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultJSONBodyLimit
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		decodeJSONFailure(w, r, err)
		return req, false
	}

	// encoding/json only reads the first value. Reject trailing data (for
	// example `{...} {...}`) so the request body has exactly one document.
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		decodeJSONFailure(w, r, err)
		return req, false
	}

	return req, true
}

func hasJSONContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}

func decodeJSONFailure(w http.ResponseWriter, r *http.Request, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		slog.WarnContext(r.Context(), "json request body too large", "path", r.URL.Path, "limit", maxBytesErr.Limit)
		RespondError(w, http.StatusRequestEntityTooLarge, "too_large", "Request body too large")
		return
	}

	slog.WarnContext(r.Context(), "strict json decode failed", "path", r.URL.Path, "err", err)
	RespondError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid JSON body")
}
