package hepan

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/product"
)

// HandleCreateOrder creates a hepan order.
func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PersonA PersonInfo `json:"person_a"`
			PersonB PersonInfo `json:"person_b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		locale := httputil.LocaleFromAccept(r)
		orderID := product.NewOrderID()
		pA, _ := json.Marshal(req.PersonA)  //nolint:errcheck
		pB, _ := json.Marshal(req.PersonB)  //nolint:errcheck
		amount, currency := hepanAmount(string(product.DetectCurrency(r.Header.Get("CF-IPCountry"), "")))

		if err := store.CreateOrder(r.Context(), orderID, string(ProductHepan), amount, currency, "", string(pA), string(pB), locale); err != nil {
			slog.Error("hepan: create order", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
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
