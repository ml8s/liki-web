package sales

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/product"
)

const (
	AdminOrderDefaultPageSize = 50
	AdminOrderMaxPageSize     = 200
)

// AdminOrderPage is a read-only operations index. It does not authorize or
// mutate anything; Admin owns capability checks and audit for actions.
type AdminOrderPage struct {
	Items    []Order
	Total    int64
	Page     int
	PageSize int
	HasMore  bool
}

func normalizeAdminOrderPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = AdminOrderDefaultPageSize
	}
	if pageSize > AdminOrderMaxPageSize {
		pageSize = AdminOrderMaxPageSize
	}
	return page, pageSize
}

func normalizeAdminOrderStatus(status string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return "", true
	case string(OrderPending):
		return string(OrderPending), true
	case string(OrderPaid):
		return string(OrderPaid), true
	case string(OrderCanceled):
		return string(OrderCanceled), true
	default:
		return "", false
	}
}

// ListOrdersForAdmin is the operations index for one-time product orders.
// Search covers identifiers operators receive from support tickets and
// provider consoles; email remains available because Admin explicitly holds
// sales read capability.
func (s *OrderStore) ListOrdersForAdmin(ctx context.Context, status, search string, page, pageSize int) (AdminOrderPage, error) {
	result := AdminOrderPage{}
	if s == nil || s.db == nil {
		return result, nil
	}
	status, ok := normalizeAdminOrderStatus(status)
	if !ok {
		return result, sql.ErrNoRows
	}
	page, pageSize = normalizeAdminOrderPage(page, pageSize)
	search = strings.TrimSpace(search)

	where := []string{}
	args := []any{}
	if status != "" {
		where = append(where, `status=?`)
		args = append(args, status)
	}
	if search != "" {
		like := "%" + search + "%"
		where = append(where, `(id LIKE ? OR email LIKE ? OR payment_intent_id LIKE ?)`)
		args = append(args, like, like, like)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = ` WHERE ` + strings.Join(where, ` AND `)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sales_orders`+whereSQL, args...).Scan(&total); err != nil {
		return result, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT id,user_id,email,product_id,amount_minor,currency,status,payment_intent_id,created_at,updated_at,paid_at
FROM sales_orders`+whereSQL+`
ORDER BY created_at DESC
LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]Order, 0)
	for rows.Next() {
		var order Order
		var productID, orderStatus, createdAt, updatedAt string
		var intentID, paidAt sql.NullString
		if err := rows.Scan(&order.ID, &order.UserID, &order.Email, &productID, &order.AmountMinor,
			&order.Currency, &orderStatus, &intentID, &createdAt, &updatedAt, &paidAt); err != nil {
			return result, err
		}
		order.ProductID = product.Product(productID)
		order.Status = OrderStatus(orderStatus)
		order.PaymentIntentID = intentID.String
		if order.CreatedAt, err = parseOrderTime(createdAt); err != nil {
			return result, err
		}
		if order.UpdatedAt, err = parseOrderTime(updatedAt); err != nil {
			return result, err
		}
		if paidAt.Valid {
			parsed, err := parseOrderTime(paidAt.String)
			if err != nil {
				return result, err
			}
			order.PaidAt = &parsed
		}
		items = append(items, order)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Items = items
	result.Total = total
	result.Page = page
	result.PageSize = pageSize
	result.HasMore = int64(page*pageSize) < total
	return result, nil
}

func parseOrderTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
