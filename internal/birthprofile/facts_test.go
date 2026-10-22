package birthprofile

import (
	"testing"
	"time"
)

func birthFacts() BirthFactsInput {
	longitude := 121.4737
	timezone := "Asia/Shanghai"
	return BirthFactsInput{
		Gender: "female",
		Date:   "1958-05-12",
		Time:   "06:00",
		Location: BirthLocationInput{
			Place:     "上海",
			Longitude: &longitude,
			Timezone:  &timezone,
		},
	}
}

func TestNewBirthFactsResolvesCanonicalInstant(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	facts, err := NewBirthFacts(now, birthFacts())
	if err != nil {
		t.Fatal(err)
	}
	if facts.Instant == nil || facts.Instant.Format(time.RFC3339) != "1958-05-11T22:00:00Z" {
		t.Fatalf("facts = %+v", facts)
	}
	if facts.Date != "1958-05-12" || facts.Time != "06:00" {
		t.Fatalf("projections = %s %s", facts.Date, facts.Time)
	}
}

func TestResolveBirthInstantRejectsDSTFold(t *testing.T) {
	facts := birthFacts()
	facts.Date = "2024-11-03"
	facts.Time = "01:30"
	longitude := -74.006
	timezone := "America/New_York"
	facts.Location.Longitude = &longitude
	facts.Location.Timezone = &timezone

	if _, err := ResolveBirthInstant(facts); err == nil {
		t.Fatal("ambiguous fold must fail")
	}
}
