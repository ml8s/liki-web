package partner

import (
	"context"
	"database/sql"
)

// adminPagedQuery runs a count/list pair with identical filters. The two
// statements are intentionally separate: SQLite window counts on every row are
// harder to index, while these admin indexes are ordered for the list read.
func adminPagedQuery[T any](
	ctx context.Context,
	db *sql.DB,
	countQuery string,
	listQuery string,
	whereArgs []any,
	limit int,
	offset int,
	scanRow func(scan func(dest ...any) error) (T, error),
) (AdminPage[T], error) {
	var total int64
	if err := db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return AdminPage[T]{}, err
	}

	args := append(append([]any{}, whereArgs...), limit, offset)
	rows, err := db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return AdminPage[T]{}, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]T, 0)
	for rows.Next() {
		item, err := scanRow(rows.Scan)
		if err != nil {
			return AdminPage[T]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminPage[T]{}, err
	}
	return AdminPage[T]{Items: items, Total: total}, nil
}
