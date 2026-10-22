package sales

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/product"
)

type checkoutRequest struct {
	ProductID   string              `json:"product_id"`
	BillingMode product.BillingMode `json:"billing_mode"`
	Locale      string              `json:"locale"`
}

func (h *Handler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.BillingMode == "" {
		req.BillingMode = product.BillingModeOneTime
	}
	catalog := h.catalog
	if catalog == nil {
		catalog = product.NewCatalog(product.CatalogConfig{})
	}
	productID := product.Product(req.ProductID)
	if _, err := catalog.OfferingFor(productID); err != nil {
		httputil.RespondError(w, http.StatusNotFound, "product_not_found", "产品不存在")
		return
	}
	currency := h.currency.CurrencyFor(r)
	if req.BillingMode == product.BillingModeSubscription {
		h.handleSubscriptionCheckout(w, r, user, productID, currency, req.Locale)
		return
	}
	if req.BillingMode != product.BillingModeOneTime {
		httputil.RespondError(w, http.StatusBadRequest, "invalid_billing_mode", "unsupported billing mode")
		return
	}
	order, err := h.orders.CreateOrder(r.Context(), user.ID, user.Email, productID, string(currency))
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "order_failed", "创建订单失败")
		return
	}
	intent, err := h.runtime.CreateIntent(r.Context(), payment.CreateIntentInput{
		Product: string(productID), BusinessOrderID: order.ID, UserID: user.ID,
		Locale: LocaleOrDefault(req.Locale),
	})
	if err == nil {
		err = h.orders.AttachIntent(r.Context(), user.ID, order.ID, intent.ID)
	}
	if err == nil {
		var result *payment.CheckoutResult
		result, _, err = h.runtime.StartCheckout(r.Context(), intent.ID)
		if err == nil {
			httputil.RespondJSON(w, http.StatusCreated, checkoutView{
				OrderID:         order.ID,
				PaymentIntentID: intent.ID,
				Provider:        string(intent.Provider),
				Status:          string(intent.Status),
				CheckoutURL:     result.CheckoutURL,
				QRCodeURL:       result.QRCodeURL,
				StatusToken:     result.StatusToken,
			})
			return
		}
	}
	if cancelErr := h.orders.CancelOrder(r.Context(), user.ID, order.ID); cancelErr != nil {
		slog.ErrorContext(r.Context(), "sales: cancel failed checkout order", "order_id", order.ID, "err", cancelErr)
	}
	slog.ErrorContext(r.Context(), "sales: start checkout", "order_id", order.ID, "err", err)
	httputil.RespondError(w, http.StatusBadGateway, "checkout_failed", "发起支付失败")
}

// checkoutView is the browser-safe checkout resource. Provider session and
// order internals stay behind the BFF.
