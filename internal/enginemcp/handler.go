package enginemcp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ml8s/liki-web/internal/httputil"
)

// Free product pages use this explicit allowlist. The BFF is not an open MCP
// proxy; adding a public tool remains a deliberate web API change.
var freeTools = map[string]struct{}{
	"tianwen_time":    {},
	"city_coords":     {},
	"bazi_chart":      {},
	"bazi_fullchart":  {},
	"ziwei_chart":     {},
	"ziwei_fullchart": {},
	"ziwei_daxian":    {},
	"ziwei_liunian":   {},
	"bazhai_chart":    {},
	"bazi_bond":       {},
	"liuyao_qigua":    {},
	"liuyao_chart":    {},
	"huangli_days":    {},
}

type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/engine/tools/{tool}", limit(h.handleCall))
}

func (h *Handler) handleCall(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	if _, ok := freeTools[tool]; !ok {
		httputil.RespondError(w, http.StatusNotFound, "engine_tool_not_found", "Engine tool not found")
		return
	}

	var arguments json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&arguments); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid engine tool request")
		return
	}
	if len(arguments) == 0 || arguments[0] != '{' {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Engine tool arguments must be an object")
		return
	}

	data, err := h.client.CallTool(r.Context(), tool, arguments)
	if err != nil {
		var toolErr *ToolError
		switch {
		case errors.As(err, &toolErr):
			httputil.RespondError(w, http.StatusUnprocessableEntity, "engine_tool_failed", toolErr.Message)
		case errors.Is(err, ErrUnavailable):
			httputil.RespondError(w, http.StatusBadGateway, "engine_unavailable", "Engine unavailable")
		default:
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "Engine call failed")
		}
		return
	}
	httputil.RespondRawJSON(w, http.StatusOK, map[string]any{"data": data})
}
