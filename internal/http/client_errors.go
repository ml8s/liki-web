package http

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/metrics"
)

// clientErrorRequest is a deliberately small, PII-free telemetry payload.
// Callers must not put birth data, chat content, email addresses or raw
// provider payloads in Detail.
type clientErrorRequest struct {
	Code      string `json:"code"`
	View      string `json:"view,omitempty"`
	Locale    string `json:"locale,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Transport string `json:"transport,omitempty"`
	Agent     string `json:"agent,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

var (
	clientErrorCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	viewUUIDPattern        = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// Known codes keep metric cardinality bounded. New client codes are still
	// logged, but the metric folds them into "other" until this list grows.
	knownClientErrorCodes = map[string]bool{
		"engine_protocol_mismatch":  true,
		"engine_unavailable":        true,
		"engine_tool_failed":        true,
		"chat_send_failed":          true,
		"chat_transport_failed":     true,
		"chat_serialization_failed": true,
		"uncaught_error":            true,
		"unhandled_rejection":       true,
		"api_error":                 true,
		"unknown":                   true,
	}
)

func (r clientErrorRequest) Validate() error {
	if !clientErrorCodePattern.MatchString(r.Code) {
		return errors.New("code must be a lowercase identifier")
	}
	if len(r.View) > 256 || len(r.Locale) > 16 || len(r.RequestID) > 64 ||
		len(r.Transport) > 16 || len(r.Agent) > 32 || len(r.Detail) > 300 {
		return errors.New("client error field too long")
	}
	return nil
}

func normalizeClientErrorView(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") {
		return ""
	}
	value = viewUUIDPattern.ReplaceAllString(value, ":id")
	if len(value) > 160 {
		value = value[:160]
	}
	return value
}

func sanitizeClientErrorDetail(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	if len(value) > 300 {
		value = value[:300]
	}
	return value
}

// handleClientError records a sanitized client-side failure for operators.
// The endpoint is public but rate-limited and stores no PII.
func handleClientError() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeStrict[clientErrorRequest](w, r)
		if !ok {
			return
		}
		metricCode := req.Code
		if !knownClientErrorCodes[metricCode] {
			metricCode = "other"
		}
		metrics.Default.Inc(metrics.MetricClientError, map[string]string{"code": metricCode})
		slog.WarnContext(r.Context(), "client_error",
			"code", req.Code,
			"view", normalizeClientErrorView(req.View),
			"locale", strings.TrimSpace(req.Locale),
			"request_id", strings.TrimSpace(req.RequestID),
			"transport", strings.TrimSpace(req.Transport),
			"agent", strings.TrimSpace(req.Agent),
			"detail", sanitizeClientErrorDetail(req.Detail),
		)
		w.WriteHeader(http.StatusNoContent)
	}
}
