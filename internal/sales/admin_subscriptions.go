package sales

import (
	"context"
	"database/sql"
	"strings"
)

const (
	AdminSubscriptionDefaultPageSize = 50
	AdminSubscriptionMaxPageSize     = 200
)

// AdminSubscriptionPage is a read-only operations index over subscription
// agreements. It does not authorize or mutate anything; Admin owns capability
// checks and audit for actions.
type AdminSubscriptionPage struct {
	Items    []Subscription
	Total    int64
	Page     int
	PageSize int
	HasMore  bool
}

func normalizeAdminSubscriptionPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = AdminSubscriptionDefaultPageSize
	}
	if pageSize > AdminSubscriptionMaxPageSize {
		pageSize = AdminSubscriptionMaxPageSize
	}
	return page, pageSize
}

// Every state in the sales_subscriptions CHECK constraint is filterable, so an
// operator can follow a subscription from pending through on_hold or failed
// without a second vocabulary.
func normalizeAdminSubscriptionStatus(status string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return "", true
	case string(SubscriptionPending):
		return string(SubscriptionPending), true
	case string(SubscriptionActive):
		return string(SubscriptionActive), true
	case string(SubscriptionOnHold):
		return string(SubscriptionOnHold), true
	case string(SubscriptionCancelled):
		return string(SubscriptionCancelled), true
	case string(SubscriptionExpired):
		return string(SubscriptionExpired), true
	case string(SubscriptionFailed):
		return string(SubscriptionFailed), true
	default:
		return "", false
	}
}

// ListSubscriptionsForAdmin is the operations index for subscription
// agreements. Search covers the identifiers operators receive from support
// tickets and provider consoles: local id, customer email, and the provider
// subscription id that must be reconciled after webhook delays.
func (s *SubscriptionStore) ListSubscriptionsForAdmin(ctx context.Context, status, search string, page, pageSize int) (AdminSubscriptionPage, error) {
	result := AdminSubscriptionPage{}
	if s == nil || s.db == nil {
		return result, nil
	}
	status, ok := normalizeAdminSubscriptionStatus(status)
	if !ok {
		return result, sql.ErrNoRows
	}
	page, pageSize = normalizeAdminSubscriptionPage(page, pageSize)
	search = strings.TrimSpace(search)

	where := []string{}
	args := []any{}
	if status != "" {
		where = append(where, `status=?`)
		args = append(args, status)
	}
	if search != "" {
		like := "%" + search + "%"
		where = append(where, `(id LIKE ? OR email LIKE ? OR provider_subscription_id LIKE ?)`)
		args = append(args, like, like, like)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = ` WHERE ` + strings.Join(where, ` AND `)
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sales_subscriptions`+whereSQL, args...).Scan(&total); err != nil {
		return result, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT `+subscriptionColumns()+`
FROM sales_subscriptions`+whereSQL+`
ORDER BY created_at DESC
LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]Subscription, 0)
	for rows.Next() {
		item, scanErr := scanSubscription(rows.Scan)
		if scanErr != nil {
			return result, scanErr
		}
		items = append(items, item)
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
