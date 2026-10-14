package account

import (
	"context"
	"errors"

	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/chat"
	"github.com/ml8s/liki-web/internal/entitlement"
	"github.com/ml8s/liki-web/internal/httputil"
	"github.com/ml8s/liki-web/internal/partner"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/sales"
)

type AuthEraser interface {
	EraseUser(ctx context.Context, userID, tombstone string) (auth.ErasureResult, error)
}

type ReferralEraser interface {
	EraseUser(ctx context.Context, userID string) (partner.ErasureResult, error)
}

// AdminAuditEraser pseudonymizes the acting admin's email on historical
// admin audit rows. Feedback is intentionally absent from the erasure
// domains: submissions are anonymous (no user linkage is stored or
// accepted), so there is nothing to scope to a deleted user.
type AdminAuditEraser interface {
	AnonymizeAuditLogs(ctx context.Context, email, tombstone string) (int64, error)
}

type SubscriptionEraser interface {
	AnonymizeUser(ctx context.Context, userID, oldEmail, tombstone string) (int64, error)
}

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	profiles      *birthprofile.Store
	chat          *chat.ChatStore
	auth          AuthEraser
	orders        *sales.OrderStore
	entitlement   *entitlement.Store
	payments      *payment.IntentStore
	notifications *payment.NotificationStore
	subscriptions SubscriptionEraser
	referral      ReferralEraser
	adminAudit    AdminAuditEraser
	erasureQueue  *ErasureStore
	requireUser   RequireUserFunc
}

func NewHandler(
	profiles *birthprofile.Store,
	chatStore *chat.ChatStore,
	authStore AuthEraser,
	orders *sales.OrderStore,
	entitlement *entitlement.Store,
	payments *payment.IntentStore,
	notifications *payment.NotificationStore,
	referral ReferralEraser,
	adminAudit AdminAuditEraser,
	erasureQueue *ErasureStore,
	requireUser RequireUserFunc,
) *Handler {
	return &Handler{
		profiles: profiles, chat: chatStore, auth: authStore,
		orders: orders, entitlement: entitlement, payments: payments,
		notifications: notifications,
		referral:      referral,
		adminAudit:    adminAudit,
		erasureQueue:  erasureQueue,
		requireUser:   requireUser,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/account/erasure", limit(h.requireUser(h.HandleErasure)))
}

// SetSubscriptionEraser keeps the existing constructor stable while making the
// newer commerce domain explicit in production wiring.
func (h *Handler) SetSubscriptionEraser(subscriptions SubscriptionEraser) {
	h.subscriptions = subscriptions
}

// HandleErasure 合规强删（个保法删除权）：匿名化该用户全部档案（清除出生信息等
// PII 并软删），并清除会话中内嵌 PII 的 system_profile 消息。会话结构保留。
func (h *Handler) HandleErasure(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.Unauthorized(w)
		return
	}
	tombstone := "erased-" + uuid.NewString() + "@liki.invalid"
	var requestID string
	if h.erasureQueue != nil {
		id, err := h.erasureQueue.Enqueue(r.Context(), user.ID, user.Email, tombstone)
		if err != nil {
			slog.Error("account: enqueue erasure", "user_id", user.ID, "err", err)
			httputil.InternalError(w)
			return
		}
		requestID = id
	}
	outcome, err := h.runErasureSteps(r.Context(), user, tombstone, requestID)
	if err != nil {
		if errors.Is(err, partner.ErrReferralFinancialHold) {
			httputil.RespondError(w, http.StatusConflict, "referral_financial_hold", "存在待结算提现，请先完成资金结算")
			return
		}
		slog.Error("account: erasure", "user_id", user.ID, "request_id", requestID, "err", err)
		httputil.InternalError(w)
		return
	}
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "erased", "profiles_cleared": outcome.ProfilesCleared,
		"messages_cleared":        outcome.MessagesCleared,
		"auth_identities_cleared": outcome.AuthResult.Identities,
		"auth_sessions_cleared":   outcome.AuthResult.Sessions,
		"auth_codes_cleared":      outcome.AuthResult.Codes,
		"domains":                 outcome.Erased,
		"referral_finance_tasks":  outcome.ReferralResult.FinanceTasks,
	})
}

// runErasureSteps executes every erasure domain in order, skipping steps
// already recorded on the durable request (resume after failure). The auth
// identity tombstone is always the final step.
func (h *Handler) runErasureSteps(ctx context.Context, user auth.User, tombstone, requestID string) (erasureOutcome, error) {
	outcome := erasureOutcome{Erased: map[string]int64{}}
	mark := func(step string) {
		if requestID != "" && h.erasureQueue != nil {
			_ = h.erasureQueue.MarkStep(ctx, requestID, step)
		}
	}
	referralResult, err := h.referral.EraseUser(ctx, user.ID)
	if err != nil {
		return outcome, err
	}
	outcome.ReferralResult = referralResult
	mark("referral")
	profilesCleared, err := h.profiles.AnonymizeUser(ctx, user.ID)
	if err != nil {
		return outcome, err
	}
	outcome.ProfilesCleared = profilesCleared
	mark("profiles")
	domains := []struct {
		name  string
		erase func(context.Context) (int64, error)
	}{
		{name: "commerce_orders", erase: func(ctx context.Context) (int64, error) {
			return h.orders.AnonymizeUser(ctx, user.ID, user.Email, tombstone)
		}},
		{name: "entitlements", erase: func(ctx context.Context) (int64, error) {
			return h.entitlement.AnonymizeUser(ctx, user.ID, user.Email, tombstone)
		}},
		{name: "payment_intents", erase: func(ctx context.Context) (int64, error) {
			return h.payments.AnonymizeUser(ctx, user.ID, user.Email, tombstone)
		}},
		{name: "payment_notifications", erase: func(ctx context.Context) (int64, error) {
			return h.notifications.AnonymizeUser(ctx, user.ID, user.Email, tombstone)
		}},
	}
	if h.subscriptions != nil {
		domains = append(domains, struct {
			name  string
			erase func(context.Context) (int64, error)
		}{name: "commerce_subscriptions", erase: func(ctx context.Context) (int64, error) {
			return h.eraseSubscriptions(ctx, user.ID, user.Email, tombstone)
		}})
	}
	for _, domain := range domains {
		cleared, err := domain.erase(ctx)
		if err != nil {
			return outcome, err
		}
		if cleared > 0 {
			outcome.Erased[domain.name] = cleared
		}
		mark(domain.name)
	}
	messagesCleared, err := h.chat.AnonymizeUserMessages(ctx, user.ID)
	if err != nil {
		return outcome, err
	}
	outcome.MessagesCleared = messagesCleared
	mark("chat")
	if h.adminAudit != nil {
		if _, err := h.adminAudit.AnonymizeAuditLogs(ctx, user.Email, tombstone); err != nil {
			return outcome, err
		}
	}
	mark("admin_audit")
	authResult, err := h.auth.EraseUser(ctx, user.ID, tombstone)
	if err != nil {
		return outcome, err
	}
	outcome.AuthResult = authResult
	mark("auth_identity")
	return outcome, nil
}

func (h *Handler) eraseSubscriptions(ctx context.Context, userID, oldEmail, tombstone string) (int64, error) {
	if h.subscriptions == nil {
		return 0, nil
	}
	return h.subscriptions.AnonymizeUser(ctx, userID, oldEmail, tombstone)
}
