package mingshu

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

// ── HandleCreateOrder ──

func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.RespondError(w, http.StatusGone, "product_retired", "命书产品已下线，请使用 Liki Chat")
	}
}

// ── HandleGetReport ──

func HandleGetReport(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		if orderID == "" {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "订单不存在")
			return
		}
		order, err := store.GetOrder(r.Context(), orderID)
		if err != nil || order.Status != "paid" || order.Product != string(product.ProductMingShu) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}
		if !httputil.AuthorizeOrderAccess(r, string(product.ProductMingShu), orderID, order.Email, time.Now()) {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}

		content, err := store.GetReport(r.Context(), orderID)
		if err != nil {
			httputil.RespondError(w, http.StatusNotFound, "not_found", "报告不存在")
			return
		}

		birth, err := store.GetBirth(r.Context(), orderID)
		if err != nil {
			birth = ""
		}

		var report map[string]any
		if err := json.Unmarshal([]byte(content), &report); err != nil {
			report = map[string]any{"raw": content, "birth": birth}
		} else {
			report["birth"] = birth
		}

		httputil.RespondJSON(w, http.StatusOK, report)
	}
}
