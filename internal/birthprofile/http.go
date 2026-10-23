package birthprofile

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
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
	mux.HandleFunc("POST /api/birth-profiles", h.requireUser(limit(h.HandleCreate)))
	mux.HandleFunc("GET /api/birth-profiles", h.requireUser(h.HandleList))
	mux.HandleFunc("GET /api/birth-profiles/{id}", h.requireUser(h.HandleGet))
	mux.HandleFunc("PATCH /api/birth-profiles/{id}", h.requireUser(limit(h.HandleUpdate)))
	mux.HandleFunc("DELETE /api/birth-profiles/{id}", h.requireUser(limit(h.HandleDelete)))
}

type updateRequest struct {
	BirthProfileDraft
	ExpectedVersion int `json:"expected_version"`
}

func (h *Handler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}

	req, ok := httputil.DecodeStrictJSON[BirthProfileDraft](
		w,
		r,
		httputil.DecodeOptions{},
	)
	if !ok {
		return
	}
	if err := ValidateBirthProfileDraft(time.Now(), req); err != nil {
		validation(w, err.Error())
		return
	}
	p, err := h.store.Create(r.Context(), user.ID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrQuotaExceeded):
			httputil.RespondError(w, http.StatusTooManyRequests, "quota_exceeded", "档案数量已达上限")
		default:
			slog.ErrorContext(r.Context(), "profile: create", "user_id", user.ID, "err", err)
			httputil.InternalError(w)
		}
		return
	}
	httputil.RespondRawJSON(w, http.StatusCreated, map[string]any{"data": p})
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	items, nextCursor, err := h.store.List(r.Context(), user.ID, cursor, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "profile: list", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	if items == nil {
		items = []BirthProfile{}
	}
	httputil.RespondRawJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"profiles": items, "next_cursor": nextCursor},
	})
}

func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	p, err := h.store.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "profile: get", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondRawJSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}

	req, ok := httputil.DecodeStrictJSON[updateRequest](
		w,
		r,
		httputil.DecodeOptions{},
	)
	if !ok {
		return
	}
	if req.ExpectedVersion <= 0 {
		validation(w, "expected_version is required")
		return
	}
	if err := ValidateBirthProfileDraft(time.Now(), req.BirthProfileDraft); err != nil {
		validation(w, err.Error())
		return
	}
	p, err := h.store.Update(r.Context(), user.ID, r.PathValue("id"), req.ExpectedVersion, req.BirthProfileDraft)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.NotFound(w)
		case errors.Is(err, ErrConflict):
			httputil.RespondError(w, http.StatusConflict, "profile_conflict", "档案已被其他设备修改，请刷新后重试")
		default:
			slog.ErrorContext(r.Context(), "profile: update", "user_id", user.ID, "err", err)
			httputil.InternalError(w)
		}
		return
	}
	httputil.RespondRawJSON(w, http.StatusOK, map[string]any{"data": p})
}

func (h *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	if err := h.store.SoftDelete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w)
			return
		}
		slog.ErrorContext(r.Context(), "profile: delete", "user_id", user.ID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondRawJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

func validation(w http.ResponseWriter, message string) {
	httputil.RespondError(w, http.StatusUnprocessableEntity, "validation_error", message)
}
