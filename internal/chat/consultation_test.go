package chat

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ml8s/liki-web/internal/sqliteutil"
)

func stringPointer(value string) *string { return &value }

func boolPointer(value bool) *bool { return &value }

func TestConsultationContextValidatesScenarioContract(t *testing.T) {
	question := "這個項目能成嗎"
	instant := "2026-10-23T12:00:00Z"
	casting := `{"mode":"auto","order":"bottom_up","yaos":[7,8,8,7,8,9],"dong_yao":[6],"casting_id":"cast_1"}`
	context := ConsultationContext{
		Version:        1,
		Scenario:       "divination",
		Question:       &question,
		Casting:        json.RawMessage(casting),
		CastingInstant: &instant,
		Location:       &ConsultationLocation{Longitude: 121.4726, Timezone: "Asia/Shanghai"},
	}
	if err := context.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	context.Casting = json.RawMessage(`{"mode":"auto"}`)
	if err := context.Validate(); err == nil {
		t.Fatal("Validate() should reject an incomplete non-reproducible casting receipt")
	}
}

func TestConsultationContextValidatesAtomicFactFormats(t *testing.T) {
	context := ConsultationContext{
		Version:   1,
		Scenario:  "huangli",
		Event:     stringPointer("wedding"),
		StartDate: stringPointer("2026-02-30"),
		EndDate:   stringPointer("2026-03-01"),
	}
	if err := context.Validate(); err == nil || err.Error() != "start_date is not a valid calendar date" {
		t.Fatalf("Validate() calendar error = %v", err)
	}

	context.StartDate = stringPointer("2026-02-01")
	context.EndDate = stringPointer("2026-01-31")
	if err := context.Validate(); err == nil || err.Error() != "end_date must not be before start_date" {
		t.Fatalf("Validate() date order error = %v", err)

	}

	context = ConsultationContext{
		Version:  1,
		Scenario: "fengshui",
		Layout: &ConsultationLayout{
			ZuoShan:    24,
			PeriodDate: "2026-02-30",
			DoorGua:    "north",
		},
	}
	if err := context.Validate(); err == nil || err.Error() != "layout.zuo_shan must be in [0,23]" {
		t.Fatalf("Validate() mountain error = %v", err)
	}
	context.Layout.ZuoShan = 0
	if err := context.Validate(); err == nil || err.Error() != "layout.period_date is not a valid calendar date" {
		t.Fatalf("Validate() layout date error = %v", err)
	}
	context.Layout.PeriodDate = "2026-02-01"
	if err := context.Validate(); err == nil || err.Error() != "layout.door_gua must be one of the eight trigrams" {
		t.Fatalf("Validate() gua error = %v", err)
	}
	context.Layout.DoorGua = "坎"
	if err := context.Validate(); err != nil {
		t.Fatalf("Validate() valid fengshui context error = %v", err)
	}
}

func TestConsultationContextValidatesCastingReceiptShape(t *testing.T) {
	tests := map[string]string{
		"invalid mode":       `{"mode":"random","order":"bottom_up","yaos":[7,7,7,7,7,7],"dong_yao":[],"casting_id":"cast"}`,
		"invalid order":      `{"mode":"auto","order":"top_down","yaos":[7,7,7,7,7,7],"dong_yao":[],"casting_id":"cast"}`,
		"missing six yaos":   `{"mode":"auto","order":"bottom_up","yaos":[7,7,7],"dong_yao":[],"casting_id":"cast"}`,
		"invalid yao":        `{"mode":"auto","order":"bottom_up","yaos":[5,7,7,7,7,7],"dong_yao":[],"casting_id":"cast"}`,
		"missing dong_yao":   `{"mode":"auto","order":"bottom_up","yaos":[7,7,7,7,7,7],"casting_id":"cast"}`,
		"receipt mismatch":   `{"mode":"auto","order":"bottom_up","yaos":[7,7,7,7,7,9],"dong_yao":[],"casting_id":"cast"}`,
		"duplicate dongyao":  `{"mode":"auto","order":"bottom_up","yaos":[9,9,7,7,7,7],"dong_yao":[1,1],"casting_id":"cast"}`,
		"coins bad position": `{"mode":"coins","order":"bottom_up","rounds":[{"position":2,"coins":["正","正","正"],"value":9,"changing":true}],"yaos":[7,9,7,7,7,7],"dong_yao":[2],"casting_id":"cast"}`,
		"auto has rounds":    `{"mode":"auto","order":"bottom_up","rounds":[{"position":1,"coins":["正","正","正"],"value":9,"changing":true}],"yaos":[7,7,7,7,7,7],"dong_yao":[],"casting_id":"cast"}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateConsultationCasting(json.RawMessage(raw)); err == nil {
				t.Fatal("validateConsultationCasting() should fail")
			}
		})
	}

	coins := `{"mode":"coins","order":"bottom_up","rounds":[{"position":1,"coins":["正","正","正"],"value":9,"changing":true},{"position":2,"coins":["反","反","反"],"value":6,"changing":true},{"position":3,"coins":["正","正","反"],"value":8,"changing":false},{"position":4,"coins":["正","反","反"],"value":7,"changing":false},{"position":5,"coins":["正","正","反"],"value":8,"changing":false},{"position":6,"coins":["正","反","反"],"value":7,"changing":false}],"yaos":[9,6,8,7,8,7],"dong_yao":[1,2],"casting_id":"cast"}`
	if err := validateConsultationCasting(json.RawMessage(coins)); err != nil {
		t.Fatalf("validateConsultationCasting() coins error = %v", err)
	}
}

func TestConsultationContextUsesExplicitIANATimezones(t *testing.T) {
	if err := validateConsultationLocation(&ConsultationLocation{Longitude: 116.4074, Timezone: "Local"}); err == nil {
		t.Fatal("validateConsultationLocation() should reject Go's Local alias")
	}
	if err := validateConsultationLocation(&ConsultationLocation{Longitude: 116.4074, Timezone: "Asia/Shanghai"}); err != nil {
		t.Fatalf("validateConsultationLocation() error = %v", err)
	}
}

func TestConsultationContextRequiresHePanSecondBirth(t *testing.T) {
	context := ConsultationContext{Version: 1, Scenario: "hepan"}
	if err := context.Validate(); err == nil {
		t.Fatal("Validate() should reject He Pan context without second birth facts")
	}

	context.SecondBirth = &ConsultationBirth{
		Gender:   "female",
		Date:     "1990-05-12",
		Time:     "08:30",
		Location: ConsultationLocation{Longitude: 112.9823, Timezone: "Asia/Shanghai"},
	}
	context.ThirdPartyConsent = boolPointer(false)
	if err := context.Validate(); err == nil {
		t.Fatal("Validate() should reject He Pan context without affirmative third-party consent")
	}
	context.ThirdPartyConsent = boolPointer(true)
	if err := context.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestNormalizeConsultationContextRejectsUnknownMembers(t *testing.T) {
	raw := json.RawMessage(`{"version":1,"scenario":"huangli","event":"wedding","start_date":"2026-10-01","end_date":"2026-10-03","place":"長沙"}`)
	if _, _, err := normalizeConsultationContext(raw); err == nil {
		t.Fatal("normalizeConsultationContext() should reject a non-contract member")
	}

	raw = json.RawMessage(`{"version":1,"scenario":"huangli","event":"wedding","start_date":"2026-10-01","end_date":"2026-10-03"}`)
	normalized, _, err := normalizeConsultationContext(raw)
	if err != nil {
		t.Fatalf("normalizeConsultationContext() error = %v", err)
	}
	if string(normalized) != string(raw) {
		t.Fatalf("normalized context = %s", normalized)
	}
}

func TestConsultationContextRejectsIrrelevantConsent(t *testing.T) {
	context := ConsultationContext{
		Version:           1,
		Scenario:          "huangli",
		Event:             stringPointer("wedding"),
		StartDate:         stringPointer("2026-10-01"),
		EndDate:           stringPointer("2026-10-03"),
		ThirdPartyConsent: boolPointer(true),
	}
	if err := context.Validate(); err == nil {
		t.Fatal("Validate() should reject third-party consent outside He Pan")
	}
}

func TestSendMessageFailsClosedWhenPersistedContextIsInvalid(t *testing.T) {
	service, chats, remote := newAgentModeTestService(t)
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "user-1", "liki", ExpertNatal, "test", "")
	if err != nil {
		t.Fatal(err)
	}
	invalid := `{"version":1,"scenario":"divination","question":"項目能否成交"}`
	if _, err := chats.db.ExecContext(ctx, `UPDATE chat_sessions SET context_json = ? WHERE id = ?`, invalid, session.ID); err != nil {
		t.Fatal(err)
	}

	_, _, _, err = service.SendMessage(ctx, "user-1", session.ID, "继续解读", nil)
	if !errors.Is(err, ErrConsultationContextInvalid) {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if remote.calls != 0 {
		t.Fatalf("runner calls = %d", remote.calls)
	}
	messages, _, err := chats.ListMessages(ctx, "user-1", session.ID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestConsultationContextRequiresBirthScenarios(t *testing.T) {
	birth := &ConsultationBirth{
		Gender:   "female",
		Date:     "1990-05-12",
		Time:     "08:30",
		Location: ConsultationLocation{Longitude: 121.4726, Timezone: "Asia/Shanghai"},
	}
	year := 2026

	tests := map[string]ConsultationContext{
		"natal": {
			Version: 1, Scenario: "natal", Birth: birth,
		},
		"liunian": {
			Version: 1, Scenario: "liunian", Birth: birth, Year: &year,
		},
		"naming": {
			Version: 1, Scenario: "naming", Birth: birth,
		},
	}
	for name, context := range tests {
		t.Run(name, func(t *testing.T) {
			if err := context.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}

	emptyNatal := ConsultationContext{Version: 1, Scenario: "natal"}
	if err := emptyNatal.Validate(); err == nil {
		t.Fatal("Validate() should reject natal context without birth facts")
	}
	emptyNaming := ConsultationContext{Version: 1, Scenario: "naming"}
	if err := emptyNaming.Validate(); err == nil {
		t.Fatal("Validate() should reject naming context without birth facts")
	}
	liunianWithoutYear := ConsultationContext{Version: 1, Scenario: "liunian", Birth: birth}
	if err := liunianWithoutYear.Validate(); err == nil {
		t.Fatal("Validate() should reject liunian context without year")
	}
}

func TestChatStorePersistsContextButOmitsItFromListProjection(t *testing.T) {
	ctx := context.Background()
	db, err := sqliteutil.Open(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewChatStore(db)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	raw := json.RawMessage(`{"version":1,"scenario":"huangli","event":"wedding","start_date":"2026-10-01","end_date":"2026-10-03"}`)
	created, err := store.CreateSessionWithContext(ctx, "user-1", "liki", ExpertDivination, "擇日", "profile-1", raw)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := store.GetSession(ctx, "user-1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got ConsultationContext
	if err := json.Unmarshal(detail.Context, &got); err != nil {
		t.Fatal(err)
	}
	if got.Scenario != "huangli" || *got.Event != "wedding" {
		t.Fatalf("detail context = %+v", got)
	}

	sessions, err := store.ListSessions(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || len(sessions[0].Context) != 0 {
		t.Fatalf("list projection leaked context: %+v", sessions)
	}
}
