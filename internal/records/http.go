package records

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc
type LimitFunc func(http.HandlerFunc) http.HandlerFunc

// Handler exposes the tool-record routes. Records stay behind explicit user
// scoping; every mutation checks ownership in the store.
type Handler struct {
	store       *Store
	requireUser RequireUserFunc
	limit       LimitFunc
}

func NewHandler(store *Store, requireUser RequireUserFunc, limit LimitFunc) *Handler {
	if requireUser == nil {
		requireUser = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	return &Handler{store: store, requireUser: requireUser, limit: limit}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/records", h.requireUser(h.HandleList))
	mux.HandleFunc("POST /api/records", h.requireUser(h.limit(h.HandleCreate)))
	mux.HandleFunc("PATCH /api/records/{id}", h.requireUser(h.limit(h.HandleRename)))
	mux.HandleFunc("DELETE /api/records/{id}", h.requireUser(h.limit(h.HandleDelete)))
}

type createRequest struct {
	Kind   string          `json:"kind"`
	Title  string          `json:"title"`
	Input  json.RawMessage `json:"input"`
	Output json.RawMessage `json:"output"`
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	kind := Kind(r.URL.Query().Get("kind"))
	if kind != "" && !kind.Valid() {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "未知的记录类型")
		return
	}
	items, err := h.store.List(r.Context(), user.ID, kind, 50)
	if err != nil {
		slog.Error("records: list", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"records": items})
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	req, ok := httputil.Decode[createRequest](w, r)
	if !ok {
		return
	}
	record, err := h.store.Create(r.Context(), user.ID, CreateInput{
		Kind:   Kind(req.Kind),
		Title:  req.Title,
		Input:  req.Input,
		Output: req.Output,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "记录参数无效")
		return
	}
	httputil.RespondJSON(w, http.StatusOK, record)
}

type renameRequest struct {
	Title string `json:"title"`
}

func (h *Handler) HandleRename(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	req, ok := httputil.Decode[renameRequest](w, r)
	if !ok {
		return
	}
	record, err := h.store.Rename(r.Context(), user.ID, r.PathValue("id"), req.Title)
	if err != nil {
		writeRecordError(w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, record)
}

func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	if err := h.store.SoftDelete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		writeRecordError(w, err)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeRecordError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.NotFound(w)
	default:
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "记录参数无效")
	}
}
