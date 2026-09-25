package admin

import (
	"context"
	"net/http"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/naming"
	"github.com/ml8s/liki-web/internal/referral"
)

type IdentityFunc func(r *http.Request) (auth.User, bool)

func capabilityGuard(authorizer *adminauth.Store, identity IdentityFunc) func(adminauth.Capability) func(http.HandlerFunc) http.HandlerFunc {
	return func(capability adminauth.Capability) func(http.HandlerFunc) http.HandlerFunc {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				user, ok := identity(r)
				if !ok {
					_ = authorizer.Audit(r.Context(), adminauth.AuditEntry{
						ActorEmail: user.Email, Action: "admin.request", ObjectType: "route",
						ObjectID: r.URL.Path, Reason: "missing unified session",
					})
					httputil.RespondError(w, http.StatusForbidden, "forbidden", "无权访问")
					return
				}
				allowed, err := authorizer.Authorize(r.Context(), user, capability)
				if err != nil {
					httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "权限检查失败")
					return
				}
				if !allowed {
					_ = authorizer.Audit(r.Context(), adminauth.AuditEntry{
						ActorUserID: user.ID, ActorEmail: user.Email, Capability: string(capability),
						Action: "admin.request", ObjectType: "route", ObjectID: r.URL.Path,
						Reason: "capability denied",
					})
					httputil.RespondError(w, http.StatusForbidden, "forbidden", "无权访问")
					return
				}
				r = r.WithContext(auth.WithAuth(r.Context(), user, auth.AuthSession{}, ""))
				next(w, r)
			}
		}
	}
}

func RegisterRoutes(
	mux *http.ServeMux,
	svc *naming.AdminService,
	onPaid func(ctx context.Context, orderID string),
	referralStore *referral.Store,
	authorizer *adminauth.Store,
	identity IdentityFunc,
	partnerDecisionNotifier referral.DecisionNotifier,
) {
	h := NewHandler(svc, onPaid)
	guard := capabilityGuard(authorizer, identity)
	identityFunc := func(r *http.Request) auth.User { user, _ := identity(r); return user }
	adminAuthHandler := adminauth.NewHandler(authorizer)

	mux.HandleFunc("GET /api/admin/session", guard(adminauth.CapabilityPartnerRead)(func(w http.ResponseWriter, r *http.Request) {
		h.HandleSession(identityFunc(r))(w, r)
	}))

	mux.HandleFunc("GET /api/admin/admins", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleListAdmins()))
	mux.HandleFunc("POST /api/admin/admins", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleCreateAdmin()))
	mux.HandleFunc("POST /api/admin/admins/{email}/capabilities", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleGrantCapability()))
	mux.HandleFunc("POST /api/admin/admins/{email}/capabilities/revoke", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleRevokeCapability()))
	mux.HandleFunc("POST /api/admin/admins/{email}/enable", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleEnableAdmin()))
	mux.HandleFunc("POST /api/admin/admins/{email}/disable", guard(adminauth.CapabilityAdminUserManage)(adminAuthHandler.HandleDisableAdmin()))

	mux.HandleFunc("GET /api/admin/orders", guard(adminauth.CapabilityAdminSuper)(h.HandleListOrders()))
	mux.HandleFunc("POST /api/admin/orders", guard(adminauth.CapabilityAdminSuper)(h.HandleCreateOrder()))
	mux.HandleFunc("PATCH /api/admin/orders/{id}/expire", guard(adminauth.CapabilityAdminSuper)(h.HandleUpdateExpire()))
	mux.HandleFunc("POST /api/admin/verification-codes", guard(adminauth.CapabilityAdminSuper)(h.HandleGenerateCodes()))
	mux.HandleFunc("GET /api/admin/verification-codes", guard(adminauth.CapabilityAdminSuper)(h.HandleListCodes()))

	if referralStore != nil {
		referralAdmin := referral.NewAdminHandler(referralStore, partnerDecisionNotifier)
		mux.HandleFunc("GET /api/admin/referral/partners", guard(adminauth.CapabilityPartnerRead)(referralAdmin.HandleListPartners()))
		mux.HandleFunc("GET /api/admin/referral/applications", guard(adminauth.CapabilityPartnerRead)(referralAdmin.HandleListApplications()))
		mux.HandleFunc("POST /api/admin/referral/applications/{id}/decision", guard(adminauth.CapabilityPartnerApprove)(referralAdmin.HandleApplicationDecision()))
		mux.HandleFunc("POST /api/admin/referral/partners/{userId}/freeze", guard(adminauth.CapabilityPartnerFreeze)(referralAdmin.HandlePartnerFreeze()))
		mux.HandleFunc("POST /api/admin/referral/partners/{userId}/unfreeze", guard(adminauth.CapabilityPartnerUnfreeze)(referralAdmin.HandlePartnerUnfreeze()))
		mux.HandleFunc("GET /api/admin/referral/payouts", guard(adminauth.CapabilityPayoutRead)(referralAdmin.HandleListPayouts()))
		mux.HandleFunc("POST /api/admin/referral/payouts/{id}/approve", guard(adminauth.CapabilityPayoutApprove)(referralAdmin.HandlePayoutAction()))
		mux.HandleFunc("POST /api/admin/referral/payouts/{id}/reject", guard(adminauth.CapabilityPayoutReject)(referralAdmin.HandlePayoutAction()))
		mux.HandleFunc("POST /api/admin/referral/payouts/{id}/mark-paid", guard(adminauth.CapabilityPayoutMarkPaid)(referralAdmin.HandlePayoutAction()))
		mux.HandleFunc("POST /api/admin/referral/payouts/{id}/reveal-account", guard(adminauth.CapabilityPayoutMarkPaid)(referralAdmin.HandleRevealPayoutAccount()))
		mux.HandleFunc("GET /api/admin/referral/actions", guard(adminauth.CapabilityAuditRead)(referralAdmin.HandleAdminActions()))
		mux.HandleFunc("GET /api/admin/referral/finance-tasks", guard(adminauth.CapabilityCommissionRead)(referralAdmin.HandleListFinanceTasks()))
		mux.HandleFunc("POST /api/admin/referral/finance-tasks/{id}/resolve", guard(adminauth.CapabilityCommissionVoid)(referralAdmin.HandleResolveFinanceTask()))
	}
}
