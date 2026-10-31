package chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	dateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe     = regexp.MustCompile(`^\d{2}:\d{2}$`)
	timezoneRe = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9_+-]*(?:/[A-Za-z0-9_+-]+)+|UTC)$`)
	instantRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$`)
)

type ConsultationBirth struct {
	Gender   string               `json:"gender"`
	Date     string               `json:"date"`
	Time     string               `json:"time"`
	Location ConsultationLocation `json:"location"`
}

type ConsultationLocation struct {
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

type ConsultationLayout struct {
	ZuoShan    int    `json:"zuo_shan"`
	PeriodDate string `json:"period_date"`
	DoorGua    string `json:"door_gua,omitempty"`
	MasterGua  string `json:"master_gua,omitempty"`
	StoveGua   string `json:"stove_gua,omitempty"`
}

type ConsultationContext struct {
	Version           int                   `json:"version"`
	Intent            string                `json:"intent"`
	Birth             *ConsultationBirth    `json:"birth,omitempty"`
	SecondBirth       *ConsultationBirth    `json:"second_birth,omitempty"`
	BirthYear         *int                  `json:"birth_year,omitempty"`
	Gender            *string               `json:"gender,omitempty"`
	Year              *int                  `json:"year,omitempty"`
	Question          *string               `json:"question,omitempty"`
	Casting           json.RawMessage       `json:"casting,omitempty"`
	CastingInstant    *string               `json:"casting_instant,omitempty"`
	Location          *ConsultationLocation `json:"location,omitempty"`
	Event             *string               `json:"event,omitempty"`
	StartDate         *string               `json:"start_date,omitempty"`
	EndDate           *string               `json:"end_date,omitempty"`
	Layout            *ConsultationLayout   `json:"layout,omitempty"`
	ThirdPartyConsent *bool                 `json:"third_party_consent,omitempty"`
}

type consultationCoinRound struct {
	Position int      `json:"position"`
	Coins    []string `json:"coins"`
	Value    int      `json:"value"`
	Changing bool     `json:"changing"`
}

func validConsultationIntent(intent string) bool {
	switch intent {
	case "natal", "liunian", "hepan", "divination", "huangli", "fengshui", "naming":
		return true
	default:
		return false
	}
}

func validateConsultationBirth(value *ConsultationBirth) error {
	if value == nil {
		return nil
	}
	if value.Gender != "male" && value.Gender != "female" {
		return errors.New("birth.gender must be male or female")
	}
	if err := validateConsultationDate("birth.date", value.Date); err != nil {
		return err
	}
	if !timeRe.MatchString(value.Time) {
		return errors.New("birth.time must be HH:mm")
	}
	if _, err := time.Parse("15:04", value.Time); err != nil {
		return errors.New("birth.time must be HH:mm")
	}
	return validateConsultationLocation(&value.Location)
}

func validateConsultationDate(name, value string) error {
	if !dateRe.MatchString(value) {
		return fmt.Errorf("%s must be YYYY-MM-DD", name)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s is not a valid calendar date", name)
	}
	return nil
}

func validateConsultationLocation(value *ConsultationLocation) error {
	if value == nil {
		return nil
	}
	if value.Longitude < -180 || value.Longitude > 180 {
		return errors.New("location.longitude must be in [-180,180]")
	}
	if strings.TrimSpace(value.Timezone) == "" {
		return errors.New("location.timezone is required")
	}
	if !timezoneRe.MatchString(value.Timezone) {
		return errors.New("location.timezone must be an explicit IANA timezone")
	}
	if _, err := time.LoadLocation(value.Timezone); err != nil {
		return errors.New("location.timezone must be a valid IANA timezone")
	}
	return nil
}

func validateConsultationCasting(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) || raw[0] != '{' {
		return errors.New("casting must be a JSON object")
	}
	var casting struct {
		Mode      string                  `json:"mode"`
		Order     string                  `json:"order"`
		CastingID string                  `json:"casting_id"`
		Rounds    []consultationCoinRound `json:"rounds"`
		Yaos      []int                   `json:"yaos"`
		DongYao   *[]int                  `json:"dong_yao"`
	}
	if err := json.Unmarshal(raw, &casting); err != nil {
		return fmt.Errorf("casting: %w", err)
	}
	if casting.Order != "bottom_up" {
		return errors.New("casting order must be bottom_up")
	}
	if strings.TrimSpace(casting.CastingID) == "" {
		return errors.New("casting casting_id is required")
	}
	switch casting.Mode {
	case "auto", "yaos":
		if len(casting.Rounds) != 0 {
			return errors.New("casting rounds are only valid for coins mode")
		}
	case "coins":
		if err := validateConsultationCoinRounds(casting.Rounds); err != nil {
			return err
		}
	default:
		return errors.New("casting mode must be auto, coins, or yaos")
	}
	if len(casting.Yaos) != 6 {
		return errors.New("casting yaos must contain six lines")
	}
	for _, yao := range casting.Yaos {
		if yao < 6 || yao > 9 {
			return errors.New("casting yaos values must be 6, 7, 8, or 9")
		}
	}
	if casting.DongYao == nil {
		return errors.New("casting dong_yao is required")
	}
	if err := validateConsultationDongYao(*casting.DongYao, casting.Yaos); err != nil {
		return err
	}
	return nil
}

func validateConsultationCoinRounds(rounds []consultationCoinRound) error {
	if len(rounds) != 6 {
		return errors.New("casting rounds must contain six lines")
	}
	for index, round := range rounds {
		if round.Position != index+1 {
			return errors.New("casting round positions must be bottom_up 1 through 6")
		}
		if len(round.Coins) != 3 {
			return errors.New("casting rounds must contain three coins")
		}
		sum := 0
		for _, coin := range round.Coins {
			switch coin {
			case "正":
				sum += 3
			case "反":
				sum += 2
			default:
				return errors.New("casting coins must be 正 or 反")
			}
		}
		if round.Value < 6 || round.Value > 9 || sum != round.Value {
			return errors.New("casting round values must match their three coins")
		}
		if round.Changing != (round.Value == 6 || round.Value == 9) {
			return errors.New("casting round changing flags must match the yao value")
		}
	}
	return nil
}

func validateConsultationDongYao(positions, yaos []int) error {
	seen := make(map[int]struct{}, len(positions))
	for _, position := range positions {
		if position < 1 || position > 6 {
			return errors.New("casting dong_yao positions must be in [1,6]")
		}
		if _, duplicate := seen[position]; duplicate {
			return errors.New("casting dong_yao positions must be unique")
		}
		seen[position] = struct{}{}
	}
	for position, yao := range yaos {
		if (yao == 6 || yao == 9) != containsInt(positions, position+1) {
			return errors.New("casting dong_yao positions must match changing yaos")
		}
	}
	return nil
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateConsultationLayout(value *ConsultationLayout) error {
	if value == nil {
		return nil
	}
	if value.ZuoShan < 0 || value.ZuoShan > 23 {
		return errors.New("layout.zuo_shan must be in [0,23]")
	}
	if value.PeriodDate != "" {
		if err := validateConsultationDate("layout.period_date", value.PeriodDate); err != nil {
			return err
		}
	}
	for name, gua := range map[string]string{
		"door_gua":   value.DoorGua,
		"master_gua": value.MasterGua,
		"stove_gua":  value.StoveGua,
	} {
		if gua == "" {
			continue
		}
		if !validConsultationGua(gua) {
			return fmt.Errorf("layout.%s must be one of the eight trigrams", name)
		}
	}
	return nil
}

func validConsultationGua(value string) bool {
	switch value {
	case "乾", "坎", "艮", "震", "巽", "离", "坤", "兑":
		return true
	default:
		return false
	}
}

func (context *ConsultationContext) Validate() error {
	if context == nil {
		return nil
	}
	if context.Version != 1 {
		return errors.New("consultation context version must be 1")
	}
	if !validConsultationIntent(context.Intent) {
		return errors.New("consultation context intent is invalid")
	}
	if err := context.validateAllowedFields(); err != nil {
		return err
	}
	if err := validateConsultationBirth(context.Birth); err != nil {
		return err
	}
	if err := validateConsultationBirth(context.SecondBirth); err != nil {
		return err
	}
	if err := validateConsultationLocation(context.Location); err != nil {
		return err
	}
	if err := validateConsultationCasting(context.Casting); err != nil {
		return err
	}
	if context.Year != nil && (*context.Year < 1900 || *context.Year > 2100) {
		return errors.New("year must be in [1900,2100]")
	}
	if context.BirthYear != nil && (*context.BirthYear < 1900 || *context.BirthYear > 2100) {
		return errors.New("birth_year must be in [1900,2100]")
	}
	if context.Gender != nil && *context.Gender != "male" && *context.Gender != "female" {
		return errors.New("gender must be male or female")
	}
	if context.ThirdPartyConsent != nil && context.Intent != "hepan" {
		return errors.New("third_party_consent is only valid for hepan consultation")
	}
	if context.CastingInstant != nil && !instantRe.MatchString(*context.CastingInstant) {
		return errors.New("casting_instant must be RFC 3339")
	}
	if context.CastingInstant != nil {
		if _, err := time.Parse(time.RFC3339, *context.CastingInstant); err != nil {
			return errors.New("casting_instant must be a valid RFC 3339 instant")
		}
	}
	for name, value := range map[string]*string{
		"question":   context.Question,
		"event":      context.Event,
		"start_date": context.StartDate,
		"end_date":   context.EndDate,
	} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	if context.StartDate != nil {
		if err := validateConsultationDate("start_date", *context.StartDate); err != nil {
			return err
		}
	}
	if context.EndDate != nil {
		if err := validateConsultationDate("end_date", *context.EndDate); err != nil {
			return err
		}
	}
	if context.StartDate != nil && context.EndDate != nil {
		if *context.EndDate < *context.StartDate {
			return errors.New("end_date must not be before start_date")
		}
	}
	if err := validateConsultationLayout(context.Layout); err != nil {
		return err
	}
	return context.validateIntentRequirements()
}

func (context *ConsultationContext) validateAllowedFields() error {
	allowed := map[string]bool{}
	switch context.Intent {
	case "natal", "naming":
		allowed["birth"] = true
	case "liunian":
		allowed["birth"] = true
		allowed["year"] = true
	case "hepan":
		allowed["second_birth"] = true
		allowed["third_party_consent"] = true
	case "divination":
		allowed["question"] = true
		allowed["casting"] = true
		allowed["casting_instant"] = true
		allowed["location"] = true
	case "huangli":
		allowed["event"] = true
		allowed["start_date"] = true
		allowed["end_date"] = true
	case "fengshui":
		allowed["layout"] = true
		allowed["birth_year"] = true
		allowed["gender"] = true
	}

	present := map[string]bool{
		"birth":               context.Birth != nil,
		"second_birth":        context.SecondBirth != nil,
		"birth_year":          context.BirthYear != nil,
		"gender":              context.Gender != nil,
		"year":                context.Year != nil,
		"question":            context.Question != nil,
		"casting":             len(context.Casting) != 0,
		"casting_instant":     context.CastingInstant != nil,
		"location":            context.Location != nil,
		"event":               context.Event != nil,
		"start_date":          context.StartDate != nil,
		"end_date":            context.EndDate != nil,
		"layout":              context.Layout != nil,
		"third_party_consent": context.ThirdPartyConsent != nil,
	}
	for name, isPresent := range present {
		if !allowed[name] && isPresent {
			return fmt.Errorf("%s is not valid for %s consultation", name, context.Intent)
		}
	}
	if context.Intent == "fengshui" && (context.BirthYear == nil) != (context.Gender == nil) {
		return errors.New("fengshui birth_year and gender must be provided together")
	}
	return nil
}

func (context *ConsultationContext) validateIntentRequirements() error {
	stringRequired := func(name string, value *string) error {
		if value == nil || strings.TrimSpace(*value) == "" {
			return fmt.Errorf("%s is required for %s consultation", name, context.Intent)
		}
		return nil
	}
	switch context.Intent {
	case "hepan":
		if context.SecondBirth == nil {
			return errors.New("second_birth is required for hepan consultation")
		}
		if context.ThirdPartyConsent == nil || !*context.ThirdPartyConsent {
			return errors.New("third_party_consent must be true for hepan consultation")
		}
	case "natal", "naming":
		if context.Birth == nil {
			return fmt.Errorf("birth is required for %s consultation", context.Intent)
		}
	case "liunian":
		if context.Birth == nil {
			return errors.New("birth is required for liunian consultation")
		}
		if context.Year == nil {
			return errors.New("year is required for liunian consultation")
		}
	case "divination":
		if err := stringRequired("question", context.Question); err != nil {
			return err
		}
		if len(context.Casting) == 0 {
			return errors.New("casting is required for divination consultation")
		}
		if context.CastingInstant == nil {
			return errors.New("casting_instant is required for divination consultation")
		}
		if context.Location == nil {
			return errors.New("location is required for divination consultation")
		}
	case "huangli":
		if err := stringRequired("event", context.Event); err != nil {
			return err
		}
		if err := stringRequired("start_date", context.StartDate); err != nil {
			return err
		}
		if err := stringRequired("end_date", context.EndDate); err != nil {
			return err
		}
	case "fengshui":
		if context.Layout == nil || context.Layout.PeriodDate == "" {
			return errors.New("layout with period_date is required for fengshui consultation")
		}
	}
	return nil
}

func normalizeConsultationContext(raw json.RawMessage) (json.RawMessage, *ConsultationContext, error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}
	var context ConsultationContext
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&context); err != nil {
		return nil, nil, fmt.Errorf("chat: decode consultation context: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("chat: consultation context must contain one JSON object")
	}
	if err := context.Validate(); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrConsultationContextInvalid, err)
	}
	normalized, err := json.Marshal(context)
	if err != nil {
		return nil, nil, fmt.Errorf("chat: encode consultation context: %w", err)
	}
	return normalized, &context, nil
}
