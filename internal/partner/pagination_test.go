package partner

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestParseAdminPageRequest(t *testing.T) {
	request := httptest.NewRequest("GET", "/?page=3&page_size=25", nil)
	page := ParseAdminPageRequest(request)
	if page.Page != 3 || page.PageSize != 25 {
		t.Fatalf("page = %+v, want page 3 size 25", page)
	}
	if page.Limit() != 25 || page.Offset() != 50 {
		t.Fatalf("limit/offset = %d/%d, want 25/50", page.Limit(), page.Offset())
	}

	request = httptest.NewRequest("GET", "/?page=-2&page_size=500", nil)
	page = ParseAdminPageRequest(request)
	if page.Page != 1 || page.PageSize != maxAdminPageSize {
		t.Fatalf("clamped page = %+v, want page 1 size %d", page, maxAdminPageSize)
	}
}

func TestAdminListsReturnPaginationMeta(t *testing.T) {
	ctx := context.Background()
	store := newAdminStore(t)
	const total = 205
	for i := 0; i < total; i++ {
		if _, err := store.SubmitApplication(ctx, RoleKOL, fmt.Sprintf("applicant-%03d", i), "pagination test"); err != nil {
			t.Fatal(err)
		}
	}

	first, err := store.ListApplications(ctx, ApplicationPending, "", AdminPageRequest{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 100 || first.Total != total || !first.HasMore {
		t.Fatalf("first page = page:%d size:%d total:%d has_more:%v len:%d",
			first.Page, first.PageSize, first.Total, first.HasMore, len(first.Items))
	}

	third, err := store.ListApplications(ctx, ApplicationPending, "", AdminPageRequest{Page: 3, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Items) != 5 || third.HasMore {
		t.Fatalf("last page = len:%d has_more:%v items:%+v", len(third.Items), third.HasMore, third.Items)
	}

	outOfRange, err := store.ListApplications(ctx, ApplicationPending, "", AdminPageRequest{Page: 4, PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(outOfRange.Items) != 0 || outOfRange.Total != total || outOfRange.HasMore {
		t.Fatalf("out-of-range page = %+v", outOfRange)
	}
}
