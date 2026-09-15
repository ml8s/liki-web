package entitlement

import "testing"

func TestChatProductUnlocksUnifiedSkillAndRejectsLegacyProducts(t *testing.T) {
	item, err := ProductDefinitionFor(ProductChat)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Skills) != 1 || item.Skills[0] != SkillLiki {
		t.Fatalf("chat skills = %+v, want [liki]", item.Skills)
	}

	if got := len(ProductCatalog()); got != 1 {
		t.Fatalf("catalog size = %d, want 1", got)
	}

	for _, legacy := range []Product{"naming", "mingshu", "hepan", "liki-naming", "liki-bazi", "liki-divination", "liki-fengshui", "liki-chat-all"} {
		if _, err := skillsForProduct(legacy); err == nil {
			t.Fatalf("product %q was accepted", legacy)
		}
	}
}
