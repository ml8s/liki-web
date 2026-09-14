package profile

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("profile: not found")
	ErrConflict      = errors.New("profile: version conflict")
	ErrQuotaExceeded = errors.New("profile: quota exceeded")
	ErrForbidden     = errors.New("profile: forbidden")
)

type Profile struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Version           int        `json:"version"`
	Nickname          string     `json:"nickname"`
	Name              string     `json:"name"`
	Gender            string     `json:"gender"`
	Relation          string     `json:"relation"`
	Calendar          string     `json:"calendar"`
	BirthDate         string     `json:"birth_date"`
	BirthTime         string     `json:"birth_time"`
	BirthTimeKnown    string     `json:"birth_time_known"`
	BirthPlace        string     `json:"birth_place"`
	Note              string     `json:"note"`
	ThirdPartyConsent bool       `json:"third_party_consent"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Input is the mutable user-info subset of a profile. Pure platform data; no
// chart/pan fields by design (排盘不落库).
type Input struct {
	Nickname          string `json:"nickname"`
	Name              string `json:"name"`
	Gender            string `json:"gender"`
	Relation          string `json:"relation"`
	Calendar          string `json:"calendar"`
	BirthDate         string `json:"birth_date"`
	BirthTime         string `json:"birth_time"`
	BirthTimeKnown    string `json:"birth_time_known"`
	BirthPlace        string `json:"birth_place"`
	Note              string `json:"note"`
	ThirdPartyConsent bool   `json:"third_party_consent"`
}

var (
	solarDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	lunarDateRe = regexp.MustCompile(`^\d{4}-闰?\d{1,2}-\d{1,2}$`)
	timeRe      = regexp.MustCompile(`^\d{2}:\d{2}$`)
)

// Validate checks user-info fields only. Birth-time rules follow the two-route
// semantics documented in the FS (known/approximate/unknown).
func (in Input) Validate(now time.Time) error {
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
	switch in.Calendar {
	case "solar", "lunar":
	default:
		return errors.New("calendar must be solar or lunar")
	}
	switch in.BirthTimeKnown {
	case "known", "approximate", "unknown":
	default:
		return errors.New("birth_time_known must be known, approximate, or unknown")
	}
	dateRe := solarDateRe
	if in.Calendar == "lunar" {
		dateRe = lunarDateRe
	}
	if !dateRe.MatchString(in.BirthDate) {
		return fmt.Errorf("invalid birth_date for calendar %s", in.Calendar)
	}
	if in.Calendar == "solar" {
		parsed, err := time.Parse("2006-01-02", in.BirthDate)
		if err != nil {
			return errors.New("invalid birth_date")
		}
		if parsed.Year() < 1900 || parsed.After(now) {
			return errors.New("birth_date out of range")
		}
	}
	if in.BirthTimeKnown != "unknown" && !timeRe.MatchString(in.BirthTime) {
		return errors.New("birth_time required (HH:mm) unless unknown")
	}
	return nil
}
