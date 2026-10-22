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

// BirthLocation is one atomic place fact. Place is display-only; computation
// requires longitude and timezone together.
type BirthLocation struct {
	Place     string   `json:"place"`
	Longitude *float64 `json:"longitude"`
	Timezone  *string  `json:"timezone"`
}

// BirthFacts is the canonical chartable subject. Date and time are projections
// of Instant in Location.Timezone; Instant is the canonical stored fact.
type BirthFacts struct {
	Gender   string        `json:"gender"`
	Date     string        `json:"date"`
	Time     string        `json:"time"`
	Location BirthLocation `json:"location"`
	Instant  *time.Time    `json:"instant"`
}

type BirthProfile struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Version           int        `json:"version"`
	Nickname          string     `json:"nickname"`
	Name              string     `json:"name"`
	Birth             BirthFacts `json:"birth"`
	Note              string     `json:"note"`
	ThirdPartyConsent bool       `json:"third_party_consent"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// BirthFactsInput is the mutable birth subset of a profile. Instant is never
// accepted from the client; the server derives it from exact civil time.
type BirthFactsInput struct {
	Gender   string             `json:"gender"`
	Date     string             `json:"date"`
	Time     string             `json:"time"`
	Location BirthLocationInput `json:"location"`
}

type BirthLocationInput struct {
	Place     string   `json:"place"`
	Longitude *float64 `json:"longitude"`
	Timezone  *string  `json:"timezone"`
}

// BirthProfileInput is the mutable user-info subset of a profile. Pure platform
// data; no chart/pan fields by design (排盘不落库).
type BirthProfileInput struct {
	Nickname          string          `json:"nickname"`
	Name              string          `json:"name"`
	Birth             BirthFactsInput `json:"birth"`
	Note              string          `json:"note"`
	ThirdPartyConsent bool            `json:"third_party_consent"`
}

var (
	solarDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe      = regexp.MustCompile(`^\d{2}:\d{2}$`)
)

// Validate requires the complete chart input.
func (in BirthProfileInput) Validate(now time.Time) error {
	if strings.TrimSpace(in.Nickname) == "" {
		return errors.New("nickname is required")
	}
	if len([]rune(in.Nickname)) > 40 {
		return errors.New("nickname is too long")
	}
	switch in.Birth.Gender {
	case "male", "female":
	default:
		return errors.New("birth.gender must be male or female")
	}
	if !solarDateRe.MatchString(in.Birth.Date) {
		return errors.New("invalid birth.date; expected Gregorian YYYY-MM-DD")
	}
	parsed, err := time.Parse("2006-01-02", in.Birth.Date)
	if err != nil {
		return errors.New("invalid birth.date")
	}
	if parsed.Year() < 1900 || parsed.After(now) {
		return errors.New("birth.date out of range")
	}
	if !timeRe.MatchString(in.Birth.Time) {
		return errors.New("birth.time is required (HH:mm)")
	}
	longitude := in.Birth.Location.Longitude
	if longitude == nil {
		return errors.New("birth.location.longitude is required")
	}
	if math.IsNaN(*longitude) || math.IsInf(*longitude, 0) || *longitude < -180 || *longitude > 180 {
		return errors.New("birth.location.longitude must be a finite number between -180 and 180")
	}
	timezone := in.Birth.Location.Timezone
	if timezone == nil {
		return errors.New("birth.location.timezone is required")
	}
	if strings.TrimSpace(*timezone) == "" || strings.TrimSpace(*timezone) != *timezone {
		return errors.New("birth.location.timezone must be an IANA timezone")
	}
	if _, err := time.LoadLocation(*timezone); err != nil {
		return errors.New("birth.location.timezone must be an IANA timezone")
	}
	return nil
}

// CanonicalInstant resolves civil time into one UTC instant. DST gaps return no
// candidate; DST folds return two and are rejected instead of silently picking.
func (in BirthProfileInput) CanonicalInstant() (time.Time, error) {
	day, err := time.Parse("2006-01-02", in.Birth.Date)
	if err != nil {
		return time.Time{}, errors.New("invalid birth.date")
	}
	hour, err := time.Parse("15:04", in.Birth.Time)
	if err != nil {
		return time.Time{}, errors.New("invalid birth.time")
	}
	location, err := time.LoadLocation(*in.Birth.Location.Timezone)
	if err != nil {
		return time.Time{}, errors.New("invalid birth.location.timezone")
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
		return time.Time{}, fmt.Errorf("birth time %s %s does not exist in %s",
			in.Birth.Date, in.Birth.Time, *in.Birth.Location.Timezone)
	}
	if len(seen) > 1 {
		return time.Time{}, fmt.Errorf("birth time %s %s is ambiguous in %s",
			in.Birth.Date, in.Birth.Time, *in.Birth.Location.Timezone)
	}
	for _, value := range seen {
		return value, nil
	}
	return time.Time{}, errors.New("birth instant not resolved")
}
