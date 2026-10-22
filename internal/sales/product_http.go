package sales

import (
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/product"
)

func (h *Handler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	allowed, err := h.skills.AllowedSkills(r.Context(), user.ID)
	if err != nil {
		httputil.InternalError(w)
		return
	}
	allowedSet := map[string]bool{}
	for _, skill := range allowed {
		allowedSet[string(skill)] = true
	}
	catalog := h.catalog
	if catalog == nil {
		catalog = product.NewCatalog(product.CatalogConfig{})
	}
	items := make([]map[string]any, 0, 1)
	for _, commercial := range catalog.List() {
		definition, err := product.DefinitionFor(commercial.ID)
		if err != nil {
			httputil.InternalError(w)
			return
		}
		skills := make([]string, 0, len(definition.Skills))
		for _, skill := range definition.Skills {
			skills = append(skills, string(skill))
		}
		displayCurrency := h.currency.CurrencyFor(r)
		displayPrice, priceErr := catalog.Price(commercial.ID, product.BillingModeOneTime, displayCurrency)
		if priceErr != nil {
			httputil.InternalError(w)
			return
		}
		options := make([]map[string]any, 0, len(commercial.Offers))
		for _, offer := range commercial.Offers {
			if offer.BillingMode == product.BillingModeSubscription && h.subscriptionCheckout == nil {
				continue
			}
			// The catalog holds one offer per billing mode and currency, so
			// matching the offer against the server-selected display currency is
			// the same gate catalog.Price applies to order creation and
			// subscription checkout. Advertising another region's offer (for
			// example the USD-only subscription to a CNY buyer) would only fail
			// closed on click.
			if offer.Price.Currency != string(displayCurrency) {
				continue
			}
			options = append(options, map[string]any{
				"billing_mode": string(offer.BillingMode),
				"price": map[string]any{
					"currency_code": string(offer.Price.Currency),
					"minor_units":   offer.Price.MinorUnits,
				},
			})
		}
		purchased := len(skills) > 0
		for _, skill := range skills {
			if !allowedSet[skill] {
				purchased = false
				break
			}
		}
		items = append(items, map[string]any{
			"id":                   string(commercial.ID),
			"name":                 commercial.Name,
			"description":          commercial.Description,
			"skills":               skills,
			"display_currency":     string(displayCurrency),
			"display_amount_minor": displayPrice.MinorUnits,
			"purchase_options":     options,
			"purchased":            purchased,
		})
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{"products": items})
}
