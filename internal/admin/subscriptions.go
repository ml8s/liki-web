package admin

import (
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/httputil"
)

func (h *AdminService) HandleAdminSubscriptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.sales == nil || h.sales.Subscriptions == nil {
			httputil.RespondError(w, http.StatusServiceUnavailable, "sales_unavailable", "订阅存储不可用")
			return
		}
		status := r.URL.Query().Get("status")
		search := r.URL.Query().Get("search")
		page := parsePositiveQueryInt(r, "page", 1)
		pageSize := parsePositiveQueryInt(r, "page_size", 50)

		result, err := h.sales.Subscriptions.ListSubscriptionsForAdmin(r.Context(), status, search, page, pageSize)
		if err != nil {
			slog.ErrorContext(r.Context(), "admin: list sales subscriptions", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询订阅失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{
			"subscriptions": result.Items,
			"page":          result.Page,
			"page_size":     result.PageSize,
			"total":         result.Total,
			"has_more":      result.HasMore,
		})
	}
}
