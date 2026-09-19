package donation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxProofLength = 64 * 1024

type Gateway interface {
	PaymentNeededHeader(ctx context.Context, bill Bill) (string, error)
	Verify(ctx context.Context, proof string) (VerifiedPayment, error)
	Confirm(ctx context.Context, tradeNo string) error
}

type Handler struct {
	store   *Store
	gateway Gateway
	config  Config
	amount  string
	nowFun  func() time.Time
	idFun   func() string
}

func NewHandler(store *Store, gateway Gateway, config Config) (*Handler, error) {
	if store == nil {
		return nil, errors.New("donation: store is required")
	}
	amount := ""
	if config.Enabled() && gateway == nil {
		return nil, errors.New("donation: gateway is required when enabled")
	}
	if config.Enabled() {
		var err error
		amount, err = config.normalizedAmount()
		if err != nil {
			return nil, err
		}
	}
	if config.GoodsName == "" {
		config.GoodsName = DefaultGoodsName
	}
	return &Handler{
		store:   store,
		gateway: gateway,
		config:  config,
		amount:  amount,
		nowFun:  func() time.Time { return time.Now().UTC() },
		idFun:   func() string { return uuid.NewString() },
	}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/donations/redeem", limit(h.HandleRedeem))
}

func (h *Handler) HandleRedeem(w http.ResponseWriter, r *http.Request) {
	if !h.config.Enabled() || h.gateway == nil {
		writeError(w, http.StatusServiceUnavailable, "donation_disabled",
			"捐赠通道未开放；所有 Liki 功能仍可正常使用")
		return
	}

	proof := strings.TrimSpace(r.Header.Get("Payment-Proof"))
	if proof == "" {
		h.paymentRequired(w, r)
		return
	}
	h.redeem(w, r, proof)
}

func (h *Handler) paymentRequired(w http.ResponseWriter, r *http.Request) {
	bill, err := h.store.CreateBill(r.Context(), h.amount, h.config.GoodsName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	encoded, err := h.gateway.PaymentNeededHeader(r.Context(), bill)
	if err != nil {
		// The bill remains pending and will be reconciled by expiry. Do not log
		// a signature value or expose gateway internals to the caller.
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	w.Header().Set("Payment-Needed", encoded)
	writeJSON(w, http.StatusPaymentRequired, map[string]any{
		"code":         "Payment-Needed",
		"message":      "Liki 是捐赠制。支付后获得捐赠凭证，所有功能仍然免费。",
		"out_trade_no": bill.OutTradeNo,
		"amount":       bill.Amount,
		"currency":     bill.Currency,
		"goods_name":   bill.GoodsName,
	})
}

func (h *Handler) redeem(w http.ResponseWriter, r *http.Request, proof string) {
	if len(proof) > maxProofLength || !validProofFormat(proof) {
		h.proofRejected(w)
		return
	}
	payload, err := parseProof(proof)
	if err != nil || payload.Protocol.PaymentProof == "" || payload.Protocol.TradeNo == "" {
		h.proofRejected(w)
		return
	}

	verified, err := h.gateway.Verify(r.Context(), proof)
	if err != nil || !verified.Active {
		h.proofRejected(w)
		return
	}
	if verified.TradeNo != payload.Protocol.TradeNo {
		h.proofRejected(w)
		return
	}

	receiptID := h.idFun()
	order, err := h.store.PrepareFulfillment(r.Context(), verified, receiptID)
	if errors.Is(err, ErrOrderNotFound) || errors.Is(err, ErrProofNotAllowed) {
		h.proofRejected(w)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
		return
	}
	if order.ResourceID != ResourceID || order.Currency != Currency ||
		verified.OutTradeNo != order.OutTradeNo {
		h.proofRejected(w)
		return
	}

	confirmed := true
	if order.FulfillmentStatus != FulfillmentConfirmed {
		if err := h.gateway.Confirm(r.Context(), order.TradeNo); err != nil {
			writeError(w, http.StatusBadGateway, "fulfillment_confirm_failed",
				"捐赠已收到，回执确认未完成；请使用同一 Payment-Proof 重试")
			return
		}
		confirmed = true
		if order.FulfillmentStatus != FulfillmentConfirmed {
			if err := h.store.MarkFulfilled(r.Context(), order.OutTradeNo, order.TradeNo); err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误")
				return
			}
			order.FulfillmentStatus = FulfillmentConfirmed
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resource_id": ResourceID,
		"content": map[string]any{
			"schema_version": SchemaVersion,
			"kind":           "donation-receipt",
			"status":         "received",
			"receipt_id":     order.ReceiptID,
			"out_trade_no":   order.OutTradeNo,
			"trade_no":       order.TradeNo,
			"amount":         order.Amount,
			"currency":       order.Currency,
			"donated_at":     order.PaidAt,
			"message":        "感谢支持。Liki 会继续免费开放。",
		},
		"credential_path":       CredentialPath,
		"fulfillment_confirmed": confirmed,
	})
}

func (h *Handler) proofRejected(w http.ResponseWriter) {
	writeError(w, http.StatusPaymentRequired, "Payment-Needed",
		"支付凭证无效或未通过校验；请重新完成捐赠后重试。")
}

func validProofFormat(proof string) bool {
	decoded, err := decodeBase64URL(proof)
	return err == nil && len(decoded) > 0
}

func parseProof(proof string) (proofPayload, error) {
	var payload proofPayload
	raw, err := decodeBase64URL(proof)
	if err != nil {
		return payload, ErrProofInvalid
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, ErrProofInvalid
	}
	return payload, nil
}

func decodeBase64URL(value string) ([]byte, error) {
	if pad := len(value) % 4; pad != 0 {
		value += strings.Repeat("=", 4-pad)
	}
	return base64.URLEncoding.DecodeString(value)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
