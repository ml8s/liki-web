package mingshu

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"liki-web/internal/httputil"
	"liki-web/internal/product"

	"github.com/google/uuid"
)

// ── HandleCreateOrder ──

func HandleCreateOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Birth struct {
				Name   string `json:"name"`
				Year   int    `json:"year"`
				Month  int    `json:"month"`
				Day    int    `json:"day"`
				Hour   int    `json:"hour"`
				Minute int    `json:"minute"`
				City   string `json:"city"`
				Gender string `json:"gender"`
			} `json:"birth"`
			Email    string `json:"email"`
			Currency string `json:"currency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		b := &req.Birth
		if b.Name == "" || b.Year == 0 {
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "missing birth fields")
			return
		}

		locale := httputil.LocaleFromAccept(r)
		orderID := uuid.New().String()
		birthStr := fmt.Sprintf("%s | %d-%02d-%02d %02d:%02d | %s | %s",
			b.Name, b.Year, b.Month, b.Day, b.Hour, b.Minute, b.City, b.Gender)

		curr := strings.ToUpper(req.Currency)
		if curr == "" {
			curr = "CNY"
		}
		var amount int
		if curr == "CNY" {
			amount = 990
		} else {
			amount = 990
		}

		if err := store.CreateOrder(r.Context(), orderID, string(product.ProductMingShu), amount, curr, req.Email, birthStr, locale); err != nil {
			slog.Error("mingshu: create order", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "创建订单失败")
			return
		}

		httputil.RespondJSON(w, http.StatusOK, map[string]string{"order_id": orderID})
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
