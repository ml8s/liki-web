// Package liki is the conversational agent product. Users purchase access to
// Liki — a 命理 consultant — not a report. This package owns the product
// definition, entitlement grants, order lifecycle, and chat runtime.
package product

import "fmt"

// Product is the commercial Chat product. It is intentionally independent from
// legacy report products such as naming, mingshu, and hepan.
type Product string

const Chat Product = "liki-chat"

// Skill identifies a consultable domain inside the Liki agent.
type Skill string

// SkillConsultation is the single Liki Chat entitlement skill. The four
// experts (natal / divination / fengshui / naming) are deployment routing
// dimensions, not entitlements. 未发布，无历史 skill 兼容负担。
const SkillConsultation Skill = "liki"

// ProductDefinition describes a commercial product and the skills it grants.
type ProductDefinition struct {
	ID     Product
	Name   string
	Skills []Skill
}

// productCatalog holds the entitlement semantics: which skills each product
// grants. It is the source of truth for entitlement grants. The
// commerce-facing card (name/description/prices) lives in commerceCatalog —
// the two answer different questions on purpose. The legacy liki.db
// `products` table mirrored this catalog but is no longer created or
// seeded; old database files may still carry an inert copy.
var productCatalog = map[Product]ProductDefinition{
	Chat: {
		ID:   Chat,
		Name: "Liki Chat",
		// One entitlement skill; expert selection is a session routing dimension.
		Skills: []Skill{SkillConsultation},
	},
}

// ProductDefinitionFor fails closed for unknown commercial products.
func DefinitionFor(id Product) (ProductDefinition, error) {
	item, ok := productCatalog[id]
	if !ok {
		return ProductDefinition{}, fmt.Errorf("product: unknown chat product %q", id)
	}
	return item, nil
}
