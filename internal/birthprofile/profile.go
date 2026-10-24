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
	PlaceID   string   `json:"place_id,omitempty"`
	Longitude *float64 `json:"longitude"`
	Timezone  *string  `json:"timezone"`
}

// BirthFacts is the canonical chartable subject. It intentionally does not
// contain date/time projections: those are derived from Instant and
// Location.Timezone for API responses.
type BirthFacts struct {
	Gender   string        `json:"gender"`
	Location BirthLocation `json:"location"`
	Instant  *time.Time    `json:"instant"`
}

// BirthFactsView is the API projection of canonical birth facts. Date and time
// always derive from Instant in Location.Timezone.
type BirthFactsView struct {
	BirthFacts
	Date string `json:"date"`
	Time string `json:"time"`
}

type BirthProfile struct {
	ID                string         `json:"id"`
	UserID            string         `json:"user_id"`
	Version           int            `json:"version"`
	Nickname          string         `json:"nickname"`
	Name              string         `json:"name"`
	Birth             BirthFactsView `json:"birth"`
	Note              string         `json:"note"`
	ThirdPartyConsent bool           `json:"third_party_consent"`
	LastUsedAt        *time.Time     `json:"last_used_at"`
	DeletedAt         *time.Time     `json:"deleted_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
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
	PlaceID   string   `json:"place_id,omitempty"`
	Longitude *float64 `json:"longitude"`
	Timezone  *string  `json:"timezone"`
}

// BirthProfileDraft is the mutable user-info subset of a profile. Pure platform
// data; no chart/pan fields by design (排盘不落库).
type BirthProfileDraft struct {
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

// ValidateBirthFacts requires the complete chart input.
func ValidateBirthFacts(now time.Time, facts BirthFactsInput) error {
	switch facts.Gender {
	case "male", "female":
	default:
		return errors.New("birth.gender must be male or female")
	}
	if !solarDateRe.MatchString(facts.Date) {
		return errors.New("invalid birth.date; expected Gregorian YYYY-MM-DD")
	}
	parsed, err := time.Parse("2006-01-02", facts.Date)
	if err != nil {
		return errors.New("invalid birth.date")
	}
	if parsed.Year() < 1900 || parsed.After(now) {
		return errors.New("birth.date out of range")
	}
	if !timeRe.MatchString(facts.Time) {
		return errors.New("birth.time is required (HH:mm)")
	}
	if facts.Location.Longitude == nil {
		return errors.New("birth.location.longitude is required")
	}
	longitude := *facts.Location.Longitude
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return errors.New("birth.location.longitude must be a finite number between -180 and 180")
	}
	if len(facts.Location.PlaceID) > 80 {
		return errors.New("birth.location.place_id is too long")
	}
	if facts.Location.Timezone == nil {
		return errors.New("birth.location.timezone is required")
	}
	timezone := *facts.Location.Timezone
	if strings.TrimSpace(timezone) == "" || strings.TrimSpace(timezone) != timezone {
		return errors.New("birth.location.timezone must be an IANA timezone")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return errors.New("birth.location.timezone must be an IANA timezone")
	}
	return nil
}

// ValidateBirthProfileMetadata validates non-chart profile metadata.
func ValidateBirthProfileMetadata(draft BirthProfileDraft) error {
	if strings.TrimSpace(draft.Nickname) == "" {
		return errors.New("nickname is required")
	}
	if len([]rune(draft.Nickname)) > 40 {
		return errors.New("nickname is too long")
	}
	return nil
}

// ValidateBirthProfileDraft validates profile metadata and complete birth facts.
func ValidateBirthProfileDraft(now time.Time, draft BirthProfileDraft) error {
	if err := ValidateBirthProfileMetadata(draft); err != nil {
		return err
	}
	if !draft.ThirdPartyConsent {
		return errors.New("third_party_consent is required")
	}
	return ValidateBirthFacts(now, draft.Birth)
}

// ResolveBirthInstant returns the single civil-time reading. DST gaps produce no
// candidate; DST folds produce two and are rejected instead of picking one.
func ResolveBirthInstant(facts BirthFactsInput) (time.Time, error) {
	day, err := time.Parse("2006-01-02", facts.Date)
	if err != nil {
		return time.Time{}, errors.New("invalid birth.date")
	}
	hour, err := time.Parse("15:04", facts.Time)
	if err != nil {
		return time.Time{}, errors.New("invalid birth.time")
	}
	if facts.Location.Timezone == nil {
		return time.Time{}, errors.New("invalid birth.location.timezone")
	}
	location, err := time.LoadLocation(*facts.Location.Timezone)
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
			facts.Date, facts.Time, *facts.Location.Timezone)
	}
	if len(seen) > 1 {
		return time.Time{}, fmt.Errorf("birth time %s %s is ambiguous in %s",
			facts.Date, facts.Time, *facts.Location.Timezone)
	}
	for _, value := range seen {
		return value, nil
	}
	return time.Time{}, errors.New("birth instant not resolved")
}

// NewBirthFacts is the single boundary from draft facts to canonical runtime
// facts. Callers must not construct chartable BirthFacts by hand.
func NewBirthFacts(now time.Time, facts BirthFactsInput) (BirthFacts, error) {
	if err := ValidateBirthFacts(now, facts); err != nil {
		return BirthFacts{}, err
	}
	instant, err := ResolveBirthInstant(facts)
	if err != nil {
		return BirthFacts{}, err
	}
	return BirthFacts{
		Gender: facts.Gender,
		Location: BirthLocation{
			Place:     strings.TrimSpace(facts.Location.Place),
			PlaceID:   strings.TrimSpace(facts.Location.PlaceID),
			Longitude: facts.Location.Longitude,
			Timezone:  facts.Location.Timezone,
		},
		Instant: &instant,
	}, nil
}

// NewBirthFactsView derives the API date/time projection from canonical facts.
func NewBirthFactsView(facts BirthFacts) BirthFactsView {
	view := BirthFactsView{BirthFacts: facts}
	if facts.Instant == nil || facts.Location.Timezone == nil {
		return view
	}
	location, err := time.LoadLocation(*facts.Location.Timezone)
	if err != nil {
		return view
	}
	local := facts.Instant.In(location)
	view.Date = local.Format("2006-01-02")
	view.Time = local.Format("15:04")
	return view
}
