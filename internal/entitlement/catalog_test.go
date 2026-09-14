package entitlement

import "testing"

func TestChatProductUnlocksAllSkillsAndRejectsLegacyProducts(t *testing.T) {
	item, err := ProductDefinitionFor(ProductChat)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Skills) != 4 {
		t.Fatalf("chat skills = %+v", item.Skills)
	}
	want := []Skill{SkillNaming, SkillBazi, SkillDivination, SkillFengshui}
	for i, skill := range want {
		if item.Skills[i] != skill {
			t.Fatalf("skills = %+v, want %+v", item.Skills, want)
		}
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
