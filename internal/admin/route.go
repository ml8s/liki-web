package admin

import (
	"context"
	"net/http"

	"liki-web/internal/httputil"
	"liki-web/internal/naming"
	"liki-web/internal/referral"
)

// IdentityFunc resolves the caller from the app's unified session.
type IdentityFunc func(r *http.Request) (email string, ok bool)

// RegisterRoutes mounts admin handler routes on the given mux.
func RegisterRoutes(
	mux *http.ServeMux,
	svc *naming.AdminService,
	onPaid func(ctx context.Context, orderID string),
	referralStore *referral.Store,
	identity IdentityFunc,
	adminEmail string,
) {
	h := NewHandler(svc, onPaid)
	guard := adminOnly(identity, adminEmail)

	mux.HandleFunc("GET /api/admin/session", guard(h.HandleSession(identity)))
	mux.HandleFunc("GET /api/admin/orders", guard(h.HandleListOrders()))
	mux.HandleFunc("POST /api/admin/orders", guard(h.HandleCreateOrder()))
	mux.HandleFunc("PATCH /api/admin/orders/{id}/expire", guard(h.HandleUpdateExpire()))
	mux.HandleFunc("POST /api/admin/verification-codes", guard(h.HandleGenerateCodes()))
	mux.HandleFunc("GET /api/admin/verification-codes", guard(h.HandleListCodes()))

	if referralStore != nil {
		referralAdmin := referral.NewAdminHandler(referralStore)
		mux.HandleFunc("GET /api/admin/referral/partners", guard(referralAdmin.HandleListPartners()))
		mux.HandleFunc("GET /api/admin/referral/applications", guard(referralAdmin.HandleListApplications()))
		mux.HandleFunc("POST /api/admin/referral/applications/{id}/decision", guard(referralAdmin.HandleApplicationDecision()))
		mux.HandleFunc("POST /api/admin/referral/partners/{userId}/status", guard(referralAdmin.HandlePartnerStatus()))
		mux.HandleFunc("GET /api/admin/referral/actions", guard(referralAdmin.HandleAdminActions()))
	}
}

func adminOnly(identity IdentityFunc, adminEmail string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			email, ok := identity(r)
			if !ok || email == "" || email != adminEmail {
				httputil.RespondError(w, http.StatusForbidden, "forbidden", "无权访问")
				return
			}
			next(w, r)
		}
	}
}
