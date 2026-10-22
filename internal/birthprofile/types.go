package birthprofile

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("birthprofile: not found")
	ErrConflict      = errors.New("birthprofile: version conflict")
	ErrQuotaExceeded = errors.New("birthprofile: quota exceeded")
)

type BirthProfile struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Version           int        `json:"version"`
	Nickname          string     `json:"nickname"`
	Name              string     `json:"name"`
	Gender            string     `json:"gender"`
	BirthPlace        string     `json:"birth_place"`
	BirthLongitude    *float64   `json:"birth_longitude"`
	BirthTimezone     *string    `json:"birth_timezone"`
	BirthInstant      *time.Time `json:"birth_instant"`
	BirthDate         string     `json:"birth_date"`
	BirthTime         string     `json:"birth_time"`
	Note              string     `json:"note"`
	ThirdPartyConsent bool       `json:"third_party_consent"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// BirthProfileInput is the mutable user-info subset of a profile. Pure platform data; no
// chart/pan fields by design (排盘不落库).
type BirthProfileInput struct {
	Nickname          string   `json:"nickname"`
	Name              string   `json:"name"`
	Gender            string   `json:"gender"`
	BirthDate         string   `json:"birth_date"`
	BirthTime         string   `json:"birth_time"`
	BirthPlace        string   `json:"birth_place"`
	BirthLongitude    *float64 `json:"birth_longitude"`
	BirthTimezone     *string  `json:"birth_timezone"`
	Note              string   `json:"note"`
	ThirdPartyConsent bool     `json:"third_party_consent"`
}

var (
	solarDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe      = regexp.MustCompile(`^\d{2}:\d{2}$`)
)

// Validate requires the complete chart input.
func (in BirthProfileInput) Validate(now time.Time) error {
	in.Nickname = strings.TrimSpace(in.Nickname)
	if in.Nickname == "" {
		return errors.New("nickname is required")
	}
	if len([]rune(in.Nickname)) > 40 {
		return errors.New("nickname is too long")
	}
	switch in.Gender {
	case "male", "female":
	default:
		return errors.New("gender must be male or female")
	}
	if !solarDateRe.MatchString(in.BirthDate) {
		return errors.New("invalid birth_date; expected Gregorian YYYY-MM-DD")
	}
	parsed, err := time.Parse("2006-01-02", in.BirthDate)
	if err != nil {
		return errors.New("invalid birth_date")
	}
	if parsed.Year() < 1900 || parsed.After(now) {
		return errors.New("birth_date out of range")
	}
	if !timeRe.MatchString(in.BirthTime) {
		return errors.New("birth_time required (HH:mm)")
	}
	if in.BirthLongitude != nil {
		lng := *in.BirthLongitude
		if math.IsNaN(lng) || math.IsInf(lng, 0) || lng < -180 || lng > 180 {
			return errors.New("birth_longitude must be a finite number between -180 and 180")
		}
	}
	if in.BirthTimezone != nil {
		if strings.TrimSpace(*in.BirthTimezone) == "" || strings.TrimSpace(*in.BirthTimezone) != *in.BirthTimezone {
			return errors.New("birth_timezone must be an IANA timezone")
		}
		if _, err := time.LoadLocation(*in.BirthTimezone); err != nil {
			return errors.New("birth_timezone must be an IANA timezone")
		}
	} else {
		return errors.New("birth_timezone is required")
	}
	if in.BirthLongitude == nil {
		return errors.New("birth_longitude is required")
	}
	return nil
}

// CanonicalInstant resolves civil time into one UTC instant. DST gaps return no
// candidate; DST folds return two and are rejected instead of silently picking.
func (in BirthProfileInput) CanonicalInstant() (time.Time, error) {
	day, err := time.Parse("2006-01-02", in.BirthDate)
	if err != nil {
		return time.Time{}, errors.New("invalid birth_date")
	}
	hour, err := time.Parse("15:04", in.BirthTime)
	if err != nil {
		return time.Time{}, errors.New("invalid birth_time")
	}
	location, err := time.LoadLocation(*in.BirthTimezone)
	if err != nil {
		return time.Time{}, errors.New("invalid birth_timezone")
	}

	hh, mm, _ := hour.Clock()
	base := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, location)
	seen := make(map[string]time.Time)
	for _, delta := range []time.Duration{-26 * time.Hour, 0, 26 * time.Hour} {
		probe := base.Add(delta)
		_, offset := probe.Zone()
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0,
			time.FixedZone("", offset))
		local := candidate.In(location)
		if local.Year() != day.Year() || local.Month() != day.Month() || local.Day() != day.Day() ||
			local.Hour() != hh || local.Minute() != mm {
			continue
		}
		utc := candidate.UTC()
		seen[utc.Format(time.RFC3339Nano)] = utc
	}
	if len(seen) == 0 {
		return time.Time{}, fmt.Errorf("birth time %s %s does not exist in %s", in.BirthDate, in.BirthTime, *in.BirthTimezone)
	}
	if len(seen) > 1 {
		return time.Time{}, fmt.Errorf("birth time %s %s is ambiguous in %s", in.BirthDate, in.BirthTime, *in.BirthTimezone)
	}
	for _, value := range seen {
		return value, nil
	}
	return time.Time{}, errors.New("unreachable")
}
