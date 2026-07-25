package naming

import (
	"context"

	"liki-web/internal/product"
)

// AdminService provides naming-domain operations for the admin panel.
// This is the only entry point for admin to interact with naming data.
type AdminService struct {
	store *Store
}

func NewAdminService(store *Store) *AdminService {
	return &AdminService{store: store}
}

type OrderSummary struct {
	OrderID       string `json:"order_id"`
	Product       string `json:"product"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	Email         string `json:"email"`
	Status        string `json:"status"`
	ChatExpiresAt string `json:"expires_at"`
	CreatedAt     string `json:"created_at"`
}

func (s *AdminService) ListOrders(ctx context.Context, email string) ([]OrderSummary, error) {
	orders, err := s.store.ListOrdersByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	summaries := make([]OrderSummary, len(orders))
	for i, o := range orders {
		summaries[i] = OrderSummary{
			OrderID:       o.OrderID,
			Product:       o.Product,
			Amount:        o.Amount,
			Currency:      o.Currency,
			Email:         o.Email,
			Status:        o.Status,
			ChatExpiresAt: o.ChatExpiresAt,
			CreatedAt:     o.CreatedAt,
		}
	}
	return summaries, nil
}

func (s *AdminService) CreatePaidOrder(ctx context.Context, email, prod string, amount int, currency, expiresAt, locale string) (string, error) {
	orderID := product.NewOrderID()
	if err := s.store.CreatePaidOrder(ctx, orderID, prod, amount, currency, email, expiresAt, locale); err != nil {
		return "", err
	}
	return orderID, nil
}

func (s *AdminService) UpdateExpire(ctx context.Context, orderID, expiresAt string) error {
	return s.store.OverrideChatExpiresAt(ctx, orderID, expiresAt)
}

func (s *AdminService) GenerateVerificationCode(ctx context.Context, createdBy string) (string, error) {
	return s.store.GenerateVerificationCode(ctx, createdBy)
}

func (s *AdminService) BatchGenerateVerificationCodes(ctx context.Context, count int, createdBy string) ([]string, error) {
	return s.store.BatchGenerateVerificationCodes(ctx, count, createdBy)
}

func (s *AdminService) ListVerificationCodes(ctx context.Context) ([]VerificationCode, error) {
	return s.store.ListVerificationCodes(ctx)
}
