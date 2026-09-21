package referral

import (
	"context"
	"database/sql"
	"errors"
)

type KOLStatus struct {
	Registered bool          `json:"registered"`
	Eligible   bool          `json:"eligible"`
	Status     PartnerStatus `json:"status,omitempty"`
}

func (s *Store) KOLStatus(ctx context.Context, userID string) (KOLStatus, error) {
	primary, err := s.ReferrerByUser(ctx, userID)
	if err == nil && primary.Role == RoleScout {
		// A Scout is never shown KOL onboarding merely because no historical
		// KOL capability exists. The two programs stay isolated.
		return KOLStatus{}, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return KOLStatus{}, err
	}
	partner, err := s.KOLByID(ctx, userID)
	if err == nil {
		return KOLStatus{
			Registered: true,
			Eligible:   partner.Status == PartnerActive,
			Status:     partner.Status,
		}, nil
	}
	if err != sql.ErrNoRows {
		return KOLStatus{}, err
	}
	eligible, err := s.paidCheck(ctx, userID)
	if err != nil {
		return KOLStatus{}, err
	}
	return KOLStatus{Registered: false, Eligible: eligible}, nil
}
