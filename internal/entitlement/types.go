package entitlement

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
	StatusExpired Status = "expired"
)

type Skill string

// SkillLiki is the single unified skill (the engine routes 八字/起名/问卦/风水
// internally). 未发布，无历史 skill 兼容负担。
const SkillLiki Skill = "liki"

type Entitlement struct {
	ID        string
	UserID    string
	Email     string
	ProductID Product
	Skills    []Skill
	Status    Status
	Source    string
	SourceID  string
	StartsAt  time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e Entitlement) MarshalJSON() ([]byte, error) {
	skills := make([]string, 0, len(e.Skills))
	for _, skill := range e.Skills {
		skills = append(skills, string(skill))
	}
	return json.Marshal(struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		Email     string    `json:"email"`
		ProductID string    `json:"product_id"`
		Skills    []string  `json:"skills"`
		Status    Status    `json:"status"`
		StartsAt  time.Time `json:"starts_at"`
		ExpiresAt time.Time `json:"expires_at"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID: e.ID, UserID: e.UserID, Email: e.Email, ProductID: string(e.ProductID),
		Skills: skills, Status: e.Status, StartsAt: e.StartsAt, ExpiresAt: e.ExpiresAt,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}
