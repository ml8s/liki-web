package hepan

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
)

// HandleCreateOrder creates a hepan order.
func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "合盘产品已下线，请使用 Liki Chat")
	}
}

// HandleGetReport returns the report JSON for a paid order.
func HandleGetReport(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "订单不存在")
			return
		}
		order, err := store.GetOrder(r.Context(), orderID)
		if err != nil || order.Status != "paid" || order.Product != string(ProductHepan) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}
		if !httputil.AuthorizeOrderAccess(r, string(ProductHepan), orderID, order.Email, time.Now()) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}

		content, err := store.GetReport(r.Context(), orderID)
		if err != nil {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}

		var report map[string]any
		if err := json.Unmarshal([]byte(content), &report); err != nil {
			slog.Error("hepan: unmarshal report", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "读取报告失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, report)
	}
}
