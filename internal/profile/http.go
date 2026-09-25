package profile

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type LimitFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store       *Store
	requireUser RequireUserFunc
}

func NewHandler(store *Store, requireUser RequireUserFunc) *Handler {
	return &Handler{store: store, requireUser: requireUser}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit LimitFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/profiles", h.requireUser(limit(h.HandleCreate)))
	mux.HandleFunc("GET /api/profiles", h.requireUser(h.HandleList))
	mux.HandleFunc("GET /api/profiles/{id}", h.requireUser(h.HandleGet))
	mux.HandleFunc("PATCH /api/profiles/{id}", h.requireUser(limit(h.HandleUpdate)))
	mux.HandleFunc("DELETE /api/profiles/{id}", h.requireUser(limit(h.HandleDelete)))
}

type createRequest struct {
	Input
}

type updateRequest struct {
	Input
	ExpectedVersion int `json:"expected_version"`
}

func (r updateRequest) Validate() error {
	if r.ExpectedVersion <= 0 {
		return errors.New("expected_version is required")
	}
	return nil
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var req createRequest
	if !decode(w, r, &req) {
		return
	}
	if err := req.Validate(time.Now()); err != nil {
		validation(w, err.Error())
		return
	}
	p, err := h.store.Create(r.Context(), user.ID, req.Input)
	if err != nil {
		switch {
		case errors.Is(err, ErrQuotaExceeded):
			writeError(w, http.StatusTooManyRequests, "quota_exceeded", "档案数量已达上限")
		default:
			slog.Error("profile: create", "user_id", user.ID, "err", err)
			internal(w)
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": p})
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	items, nextCursor, err := h.store.List(r.Context(), user.ID, cursor, limit)
	if err != nil {
		slog.Error("profile: list", "user_id", user.ID, "err", err)
		internal(w)
		return
	}
	if items == nil {
		items = []Profile{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"profiles": items, "next_cursor": nextCursor},
	})
}

func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	p, err := h.store.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			notFound(w)
			return
		}
		slog.Error("profile: get", "user_id", user.ID, "err", err)
		internal(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var req updateRequest
	if !decode(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		validation(w, err.Error())
		return
	}
	if err := req.Input.Validate(time.Now()); err != nil {
		validation(w, err.Error())
		return
	}
	p, err := h.store.Update(r.Context(), user.ID, r.PathValue("id"), req.ExpectedVersion, req.Input)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			notFound(w)
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, "profile_conflict", "档案已被其他设备修改，请刷新后重试")
		default:
			slog.Error("profile: update", "user_id", user.ID, "err", err)
			internal(w)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	if err := h.store.SoftDelete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			notFound(w)
			return
		}
		slog.Error("profile: delete", "user_id", user.ID, "err", err)
		internal(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return false
	}
	return true
}

func validation(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]string{"code": "validation_error", "message": message}})
}

func unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "not_found", "message": "档案不存在"}})
}

func internal(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
