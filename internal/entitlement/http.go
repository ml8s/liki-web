package entitlement

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
)

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	store       *Store
	requireUser RequireUserFunc
}

func NewHandler(store *Store, requireUser RequireUserFunc) *Handler {
	return &Handler{store: store, requireUser: requireUser}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me/entitlements", h.requireUser(h.HandleEntitlements))
}

func (h *Handler) HandleEntitlements(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	items, err := h.store.ListActiveByUser(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	if items == nil {
		items = []Entitlement{}
	}
	allowed := make(map[string]bool)
	out := make([]string, 0)
	for _, item := range items {
		for _, skill := range item.Skills {
			value := string(skill)
			if !allowed[value] {
				allowed[value] = true
				out = append(out, value)
			}
		}
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"entitlements":   items,
		"allowed_skills": out,
	})
}
