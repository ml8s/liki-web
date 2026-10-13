package referral

import (
	"context"
	"database/sql"
	"time"
)

func sumMoney(ctx context.Context, db *sql.DB, query string, start, end time.Time) ([]AdminMoney, error) {
	rows, err := db.QueryContext(ctx, query, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]AdminMoney, 0)
	for rows.Next() {
		var item AdminMoney
		if err := rows.Scan(&item.Currency, &item.AmountMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func sumMoneyAll(ctx context.Context, db *sql.DB, query string) ([]AdminMoney, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]AdminMoney, 0)
	for rows.Next() {
		var item AdminMoney
		if err := rows.Scan(&item.Currency, &item.AmountMinor); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
