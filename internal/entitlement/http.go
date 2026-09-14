package entitlement

import (
	"net/http"

	"liki-web/internal/auth"
)

type Handler struct {
	store       *Store
	requireUser RequireUserFunc
}

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

func NewHandler(store *Store, requireUser RequireUserFunc) *Handler {
	return &Handler{store: store, requireUser: requireUser}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me/entitlements", h.requireUser(h.HandleList))
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	items, err := h.store.ListActiveByUser(r.Context(), user.ID)
	if err != nil {
		internal(w)
		return
	}
	if items == nil {
		items = []Entitlement{}
	}
	responseOK(w, map[string]any{
		"entitlements":   items,
		"allowed_skills": allowedSkillStrings(items),
	})
}

func allowedSkillStrings(items []Entitlement) []string {
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
	return out
}

func responseOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
}

func internal(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
}
