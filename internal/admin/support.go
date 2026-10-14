package admin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/adminauth"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/product"
	"github.com/ml8s/liki-web/internal/sales"
)

// UserByEmailReader is the minimum identity lookup needed to ensure that a
// compensation is attached to a real platform user, never to a typo or ghost.
type UserByEmailReader interface {
	GetUserByEmail(context.Context, string) (auth.User, error)
}

// PartnerFinanceReader decouples the partner bounded context from admin.
type PartnerFinanceReader interface {
	AdminFinanceSummary(ctx context.Context, start, end time.Time) (partner.AdminFinanceSummary, error)
}

type compensationRequest struct {
	Email          string `json:"email"`
	Template       string `json:"template"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (r compensationRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" || strings.TrimSpace(r.Reason) == "" {
		return errors.New("email and reason are required")
	}
	if len(strings.TrimSpace(r.IdempotencyKey)) > 128 {
		return errors.New("idempotency key must be at most 128 characters")
	}
	switch strings.TrimSpace(r.Template) {
	case "support_7d", "support_30d":
		return nil
	default:
		return errors.New("template must be support_7d or support_30d")
	}
}

func (h *AdminService) HandleCompensateEntitlement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := httputil.DecodeAndValidate[compensationRequest](w, r)
		if !ok {
			return
		}
		user, err := h.users.GetUserByEmail(r.Context(), req.Email)
		if errors.Is(err, auth.ErrUserNotFound) {
			httputil.RespondError(w, http.StatusNotFound, "user_not_found", "平台用户不存在")
			return
		}
		if err != nil {
			slog.Error("admin: lookup compensation user", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "读取用户失败")
			return
		}
		if user.Status != auth.UserActive {
			httputil.RespondError(w, http.StatusConflict, "user_not_active", "仅有效账号可补偿")
			return
		}

		duration := 7 * 24 * time.Hour
		if strings.TrimSpace(req.Template) == "support_30d" {
			duration = 30 * 24 * time.Hour
		}
		sourceID := compensationSourceID(req.IdempotencyKey, user.ID)
		entitlement, err := h.entitlements.GrantForProduct(
			r.Context(), user.ID, user.Email, product.Chat,
			"admin_compensation", sourceID, duration,
		)
		if err != nil {
			slog.Error("admin: grant compensation", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "发放补偿失败")
			return
		}

		after, err := json.Marshal(map[string]any{
			"entitlement_id": entitlement.ID,
			"user_id":        user.ID,
			"template":       req.Template,
			"starts_at":      entitlement.StartsAt,
			"expires_at":     entitlement.ExpiresAt,
		})
		if err != nil {
			slog.Error("admin: encode compensation audit", "err", err)
		}
		h.auditEntitlementChange(r, "entitlement.compensated", entitlement.ID, "", string(after), req.Reason)
		emailQueued := true
		if h.compensationNotifier != nil {
			// The grant and audit are committed. Delivery receives a detached
			// context so an operator closing the browser cannot cancel the email.
			if err := h.compensationNotifier(context.WithoutCancel(r.Context()), user, entitlement, strings.TrimSpace(req.Template)); err != nil {
				emailQueued = false
				h.auditCompensationNotificationFailure(r, entitlement.ID, req.Reason, err)
				slog.Error("admin: queue compensation notification", "entitlement_id", entitlement.ID, "err", err)
			}
		}
		httputil.RespondJSON(w, http.StatusCreated, map[string]any{
			"template":     strings.TrimSpace(req.Template),
			"entitlement":  entitlement,
			"email_queued": emailQueued,
		})
	}
}

func compensationSourceID(idempotencyKey, userID string) string {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return uuid.NewString()
	}
	// Derive a UUID from the operator key plus beneficiary so retries preserve
	// idempotency without letting one key collide across different users.
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(
		"liki:admin_compensation\x00"+userID+"\x00"+idempotencyKey,
	)).String()
}

func (h *AdminService) auditCompensationNotificationFailure(r *http.Request, entitlementID, reason string, cause error) {
	actor, _ := auth.UserFromContext(r.Context())
	err := h.authorizer.Audit(r.Context(), adminauth.AuditEntry{
		ActorUserID: actor.ID, ActorEmail: actor.Email,
		Capability: string(adminauth.CapabilityEntitlementManage),
		Action:     "entitlement.compensation_notification_failed",
		ObjectType: "entitlement", ObjectID: entitlementID,
		After: "queued=false", Reason: reason + " | queue error: " + cause.Error(),
	})
	if err != nil {
		slog.Error("admin: audit compensation notification failure", "entitlement_id", entitlementID, "err", err)
	}
}

type AdminFinanceReport struct {
	PeriodStart          string             `json:"period_start"`
	PeriodEnd            string             `json:"period_end"`
	OneTimePaid          []sales.AdminMoney `json:"one_time_paid"`
	SubscriptionPaid     []sales.AdminMoney `json:"subscription_paid"`
	SubscriptionRefunded []sales.AdminMoney `json:"subscription_refunded"`
	NetProductRevenue    []sales.AdminMoney `json:"net_product_revenue"`
	PartnerCommissionNet []sales.AdminMoney `json:"partner_commission_net"`
	PartnerPayoutPaid    []sales.AdminMoney `json:"partner_payout_paid"`
	PartnerPendingPayout []sales.AdminMoney `json:"partner_pending_payout"`
}

func (h *AdminService) HandleFinanceSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		days := 30
		switch strings.TrimSpace(r.URL.Query().Get("days")) {
		case "7":
			days = 7
		case "90":
			days = 90
		case "", "30":
		default:
			httputil.RespondError(w, http.StatusBadRequest, "invalid_request", "days must be 7, 30, or 90")
			return
		}
		// Use an exclusive next-second boundary because stored timestamps have
		// second precision in SQLite.
		end := time.Now().UTC().Truncate(time.Second).Add(time.Second)
		start := end.AddDate(0, 0, -days)
		salesSummary, err := h.sales.AdminFinanceSummary(r.Context(), start, end)
		if err != nil {
			slog.Error("admin: sales finance summary", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "生成财务汇总失败")
			return
		}
		partner, err := h.partnerFinance.AdminFinanceSummary(r.Context(), start, end)
		if err != nil {
			slog.Error("admin: partner finance summary", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "生成财务汇总失败")
			return
		}

		grossProductPaid := addMoneyMaps(
			moneyMap(salesSummary.OneTimePaid), moneyMap(salesSummary.SubscriptionPaid),
		)
		productRefunds := moneyMap(salesSummary.SubscriptionRefunded)
		httputil.RespondJSON(w, http.StatusOK, AdminFinanceReport{
			PeriodStart:          start.Format(time.RFC3339),
			PeriodEnd:            end.Format(time.RFC3339),
			OneTimePaid:          salesSummary.OneTimePaid,
			SubscriptionPaid:     salesSummary.SubscriptionPaid,
			SubscriptionRefunded: salesSummary.SubscriptionRefunded,
			NetProductRevenue:    subtractAdminMoney(grossProductPaid, productRefunds),
			PartnerCommissionNet: partnerAdminMoney(partner.CommissionNet),
			PartnerPayoutPaid:    partnerAdminMoney(partner.PayoutPaid),
			PartnerPendingPayout: partnerAdminMoney(partner.PendingPayoutAmount),
		})
	}
}

func partnerAdminMoney(items []partner.AdminMoney) []sales.AdminMoney {
	converted := make([]sales.AdminMoney, 0, len(items))
	for _, item := range items {
		converted = append(converted, sales.AdminMoney{Currency: item.Currency, AmountMinor: item.AmountMinor})
	}
	return converted
}
