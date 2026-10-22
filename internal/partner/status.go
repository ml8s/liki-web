package partner

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
)

// PartnerProgramState is the user-facing state for one independent programme.
// Status is present only when the user's assigned programme role is this role.
type PartnerProgramState struct {
	Role        PartnerRole         `json:"role"`
	Status      RoleStatus          `json:"status,omitempty"`
	Application *PartnerApplication `json:"application"`
	CanSubmit   bool                `json:"can_submit"`
}

// PartnerRoles returns the roles the user has any relationship with — a
// partner record (any status) or an application (any decision state). It is
// the light identity summary consumed by /api/me so the app shell can gate the
// partnership entry; programme details stay in PartnerStates.
func (s *Store) PartnerRoles(ctx context.Context, userID string) ([]string, error) {
	userID = strings.TrimSpace(userID)
	roles := map[string]struct{}{}
	if userID != "" {
		if err := s.collectRoles(ctx, `SELECT role FROM partners WHERE user_id=?`, userID, roles); err != nil {
			return nil, err
		}
		if err := s.collectRoles(ctx, `SELECT DISTINCT type FROM partner_applications WHERE user_id=?`, userID, roles); err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(roles))
	for role := range roles {
		out = append(out, role)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Store) collectRoles(ctx context.Context, query, userID string, into map[string]struct{}) error {
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return err
		}
		into[role] = struct{}{}
	}
	return rows.Err()
}

// PartnerStates is the stable aggregate for the partner onboarding surface.
type PartnerStates struct {
	KOL   PartnerProgramState `json:"kol"`
	Scout PartnerProgramState `json:"scout"`
}

func (s *Store) PartnerStates(ctx context.Context, userID string) (PartnerStates, error) {
	states := PartnerStates{
		KOL:   PartnerProgramState{Role: RoleKOL},
		Scout: PartnerProgramState{Role: RoleScout},
	}

	// Programme participation is intentionally exclusive. This prevents a
	// person from occupying both roles and recreating a hierarchy.
	_, err := s.PartnerByUser(ctx, userID)
	hasPrimary := err == nil
	switch {
	case err == nil:
	case errors.Is(err, sql.ErrNoRows):
	default:
		return PartnerStates{}, err
	}

	for _, role := range []PartnerRole{RoleKOL, RoleScout} {
		state := &states.KOL
		if role == RoleScout {
			state = &states.Scout
		}
		application, found, err := s.LatestApplicationByUser(ctx, userID, role)
		if err != nil {
			return PartnerStates{}, err
		}
		if found {
			item := application
			state.Application = &item
		}
		status, assigned, err := s.ProgrammeRoleStatus(ctx, userID, role)
		if err != nil {
			return PartnerStates{}, err
		}
		if assigned {
			state.Status = status
		}
		state.CanSubmit = !hasPrimary &&
			(!found || state.Application.Status != ApplicationPending)
	}
	if _, err := s.PendingApplicationByUser(ctx, userID); err == nil {
		states.KOL.CanSubmit = false
		states.Scout.CanSubmit = false
	} else if !errors.Is(err, sql.ErrNoRows) {
		return PartnerStates{}, err
	}
	return states, nil
}
