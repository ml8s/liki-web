package admin

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ml8s/liki-web/internal/httputil"
)

func (h *AdminService) HandleAdminOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.sales == nil || h.sales.Orders == nil {
			httputil.RespondError(w, http.StatusServiceUnavailable, "sales_unavailable", "订单存储不可用")
			return
		}
		status := r.URL.Query().Get("status")
		search := r.URL.Query().Get("search")
		page := parsePositiveQueryInt(r, "page", 1)
		pageSize := parsePositiveQueryInt(r, "page_size", 50)

		result, err := h.sales.Orders.ListOrdersForAdmin(r.Context(), status, search, page, pageSize)
		if err != nil {
			slog.Error("admin: list sales orders", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "查询订单失败")
			return
		}
		httputil.RespondJSON(w, http.StatusOK, map[string]any{
			"orders": result.Items,
			"meta": map[string]any{
				"page":      result.Page,
				"page_size": result.PageSize,
				"total":     result.Total,
				"has_more":  result.HasMore,
			},
		})
	}
}

func parsePositiveQueryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
