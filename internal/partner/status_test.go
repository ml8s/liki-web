package partner

import (
	"context"
	"reflect"
	"testing"
)

// PartnerRoles 是 /api/me 的身份摘要：只回答"该账号与哪些合作角色有关系"，
// 含审核中与被拒（入口保留以便回访/重申），不含任何计划细节。
func TestPartnerRolesSummarisesRelationships(t *testing.T) {
	ctx := context.Background()
	store := newHandlerStore(t)

	roles, err := store.PartnerRoles(ctx, "user-none")
	if err != nil {
		t.Fatal(err)
	}
	if roles == nil || len(roles) != 0 {
		t.Fatalf("no relationship: roles=%v, want empty non-nil", roles)
	}

	if _, err := store.SubmitApplication(ctx, RoleKOL, "user-applied", "content creator"); err != nil {
		t.Fatal(err)
	}
	roles, err = store.PartnerRoles(ctx, "user-applied")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roles, []string{"kol"}) {
		t.Fatalf("pending application: roles=%v, want [kol]", roles)
	}

	if _, err := store.createKOLPartner(ctx, "user-kol"); err != nil {
		t.Fatal(err)
	}
	roles, err = store.PartnerRoles(ctx, "user-kol")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roles, []string{"kol"}) {
		t.Fatalf("active KOL: roles=%v, want [kol]", roles)
	}

	if _, err := store.CreateScoutPartner(ctx, "user-scout", "admin", "invited scout"); err != nil {
		t.Fatal(err)
	}
	roles, err = store.PartnerRoles(ctx, "user-scout")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roles, []string{"scout"}) {
		t.Fatalf("active scout: roles=%v, want [scout]", roles)
	}
}
