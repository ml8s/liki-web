package account

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"log/slog"
	"net/http"

	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/liki"
	"github.com/ml8s/liki-web/internal/payment"
	"github.com/ml8s/liki-web/internal/profile"
	"github.com/ml8s/liki-web/internal/referral"
)

type AuthEraser interface {
	EraseUser(ctx context.Context, userID, tombstone string) (auth.ErasureResult, error)
}

type ReferralEraser interface {
	EraseUser(ctx context.Context, userID string) (referral.ErasureResult, error)
}

type RequireUserFunc func(http.HandlerFunc) http.HandlerFunc

type Handler struct {
	profiles      *profile.Store
	chat          *liki.ChatStore
	auth          AuthEraser
	orders        *liki.OrderStore
	entitlement   *liki.EntitlementStore
	payments      *payment.IntentStore
	notifications *payment.NotificationStore
	referral      ReferralEraser
	requireUser   RequireUserFunc
}

func NewHandler(
	profiles *profile.Store,
	chatStore *liki.ChatStore,
	authStore AuthEraser,
	orders *liki.OrderStore,
	entitlement *liki.EntitlementStore,
	payments *payment.IntentStore,
	notifications *payment.NotificationStore,
	referral ReferralEraser,
	requireUser RequireUserFunc,
) *Handler {
	return &Handler{
		profiles: profiles, chat: chatStore, auth: authStore,
		orders: orders, entitlement: entitlement, payments: payments,
		notifications: notifications,
		referral:      referral,
		requireUser:   requireUser,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/account/erasure", limit(h.requireUser(h.HandleErasure)))
}

// HandleErasure 合规强删（个保法删除权）：匿名化该用户全部档案（清除出生信息等
// PII 并软删），并清除会话中内嵌 PII 的 system_profile 消息。会话结构保留。
func (h *Handler) HandleErasure(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "请先登录"}})
		return
	}
	referralResult, err := h.referral.EraseUser(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, referral.ErrReferralFinancialHold) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": map[string]string{
				"code": "referral_financial_hold", "message": "存在待结算提现，请先完成资金结算",
			}})
			return
		}
		slog.Error("account: erase referral", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	profilesCleared, err := h.profiles.AnonymizeUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: erase profiles", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	tombstone := "erased-" + uuid.NewString() + "@liki.invalid"
	erased := map[string]int64{}
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
	for _, domain := range domains {
		cleared, err := domain.erase(r.Context())
		if err != nil {
			slog.Error("account: erase domain", "user_id", user.ID, "domain", domain.name, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
			return
		}
		if cleared > 0 {
			erased[domain.name] = cleared
		}
	}
	messagesCleared, err := h.chat.AnonymizeUserMessages(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: erase chat PII", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	authResult, err := h.auth.EraseUser(r.Context(), user.ID, tombstone)
	if err != nil {
		slog.Error("account: erase auth identity", "user_id", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "internal_error", "message": "服务器内部错误"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"status": "erased", "profiles_cleared": profilesCleared, "messages_cleared": messagesCleared,
			"auth_identities_cleared": authResult.Identities,
			"auth_sessions_cleared":   authResult.Sessions,
			"auth_codes_cleared":      authResult.Codes,
			"domains":                 erased,
			"referral_finance_tasks":  referralResult.FinanceTasks,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
