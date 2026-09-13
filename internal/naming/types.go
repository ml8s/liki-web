package naming

import validation "github.com/go-ozzo/ozzo-validation/v4"

// ── request types ──

type chatRequest struct {
	Message string `json:"message"`
	Lang    string `json:"lang,omitempty"`
	// Context is the initial advisor greeting (frontend-provided). Injected
	// as first assistant message only when chat history is empty (first turn).
	Context string `json:"context,omitempty"`
}

func (r chatRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Message, validation.Required),
	)
}

type loginRequest struct {
	Email string `json:"email"`
}

func (r loginRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required),
	)
}

type selectOrderRequest struct {
	OrderID string `json:"order_id"`
	Email   string `json:"email"`
}

func (r selectOrderRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.OrderID, validation.Required),
		validation.Field(&r.Email, validation.Required),
	)
}

// ── response types ──

type LoginResponse struct {
	OrderID string              `json:"order_id,omitempty"`
	Orders  []LoginOrderSummary `json:"orders,omitempty"`
}

type LoginOrderSummary struct {
	OrderID   string `json:"order_id"`
	Summary   string `json:"summary"`
	ExpiresAt string `json:"expires_at"`
}

// ── data types ──

type OrderData struct {
	OrderID       string
	Product       string
	Amount        int64
	Currency      string
	Email         string
	Provider      string
	Status        string
	Locale        string
	ChatExpiresAt string
	CreatedAt     string
}

type ChatMessage struct {
	ID        int64  `json:"id"`
	OrderID   string `json:"order_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type VerificationCode struct {
	Code      string  `json:"code"`
	Status    string  `json:"status"`
	OrderID   *string `json:"order_id,omitempty"`
	Email     *string `json:"email,omitempty"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	UsedAt    *string `json:"used_at,omitempty"`
}

type RedeemCodeParams struct {
	Code          string
	Email         string
	OrderID       string
	Product       string
	Currency      string
	Amount        int64
	ChatExpiresAt string
	Locale        string
}
