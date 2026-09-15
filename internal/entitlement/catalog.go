package entitlement

import "fmt"

// Product is the commercial Chat product. It is intentionally independent from
// legacy report products such as naming, mingshu, and hepan.
type Product string

const ProductChat Product = "liki-chat"

type ProductDefinition struct {
	ID     Product
	Name   string
	Skills []Skill
}

var productCatalog = map[Product]ProductDefinition{
	ProductChat: {
		ID:   ProductChat,
		Name: "Liki Chat",
		// One unified skill (the engine routes 八字/起名/问卦/风水 internally).
		Skills: []Skill{SkillLiki},
	},
}

// ProductCatalog returns deterministic ordering for schema seeding and payment
// integrations.
func ProductCatalog() []ProductDefinition {
	return []ProductDefinition{productCatalog[ProductChat]}
}

// ProductDefinitionFor fails closed for unknown commercial products.
func ProductDefinitionFor(id Product) (ProductDefinition, error) {
	item, ok := productCatalog[id]
	if !ok {
		return ProductDefinition{}, fmt.Errorf("entitlement: unknown chat product %q", id)
	}
	return item, nil
}

func skillsForProduct(id Product) ([]Skill, error) {
	item, err := ProductDefinitionFor(id)
	if err != nil {
		return nil, err
	}
	return item.Skills, nil
}
