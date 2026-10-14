package partner

import (
	"net/http"
	"strconv"

	"github.com/ml8s/liki-web/internal/httputil"
)

const (
	defaultAdminPageSize = 100
	maxAdminPageSize     = 100
)

// AdminPageRequest is the canonical offset pagination input for admin lists.
// Invalid values are clamped instead of rejected so an operator typo cannot turn
// a read-only index into a 400 response.
type AdminPageRequest struct {
	Page     int
	PageSize int
}

func ParseAdminPageRequest(r *http.Request) AdminPageRequest {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), defaultAdminPageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultAdminPageSize
	}
	if pageSize > maxAdminPageSize {
		pageSize = maxAdminPageSize
	}
	return AdminPageRequest{Page: page, PageSize: pageSize}
}

func (p AdminPageRequest) Limit() int { return p.PageSize }
func (p AdminPageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// AdminPage pairs paged items with the pagination contract consumed by the
// admin client. Items is intentionally not tagged; handlers choose the domain
// key so existing response vocabulary remains stable.
type AdminPage[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int64
	HasMore  bool
}

func NewAdminPage[T any](request AdminPageRequest, items []T, total int64) AdminPage[T] {
	return AdminPage[T]{
		Items:    items,
		Page:     request.Page,
		PageSize: request.PageSize,
		Total:    total,
		HasMore:  int64(request.Offset()+len(items)) < total,
	}
}

func RespondAdminPage[T any](w http.ResponseWriter, key string, page AdminPage[T]) {
	httputil.RespondJSON(w, http.StatusOK, map[string]any{
		key:         page.Items,
		"page":      page.Page,
		"page_size": page.PageSize,
		"total":     page.Total,
		"has_more":  page.HasMore,
	})
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
