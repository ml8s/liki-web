package adminauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAdminRequestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"empty", "", true},
		{"missing at sign", "adminexample.com", true},
		{"too long", strings.Repeat("a", 250) + "@x.io", true},
		{"valid", "admin@example.com", false},
		{"normalized case", " Admin@Example.COM ", false},
		{"plus tag", "admin+ops@example.com", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := createAdminRequest{Email: tc.email}.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate(%q) error = %v, wantErr = %v", tc.email, err, tc.wantErr)
			}
		})
	}
}

func TestCapabilityRequestValidate(t *testing.T) {
	testCases := []struct {
		name       string
		capability string
		reason     string
		wantErr    error
	}{
		{"unknown capability", "not-a-capability", "because", ErrInvalidCapability},
		{"empty capability", "", "because", ErrInvalidCapability},
		{"empty reason", string(CapabilityAdminSuper), "", ErrReasonRequired},
		{"blank reason", string(CapabilityAdminSuper), "   ", ErrReasonRequired},
		{"valid", string(CapabilityAdminSuper), "incident response", nil},
		{"valid trims capability padding", " " + string(CapabilityPartnerApprove) + " ", "because", nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := capabilityRequest{Capability: tc.capability, Reason: tc.reason}.Validate()
			if err != tc.wantErr {
				t.Fatalf("Validate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestAdminStatusRequestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		status  string
		reason  string
		wantErr error
	}{
		{"unknown status", "pending", "because", ErrInvalidAdminStatus},
		{"empty status", "", "because", ErrInvalidAdminStatus},
		{"missing reason", string(AdminActive), "", ErrReasonRequired},
		{"active", string(AdminActive), "because", nil},
		{"disabled", string(AdminDisabled), "because", nil},
		{"disabled trims padding", " " + string(AdminDisabled) + " ", "because", nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adminStatusRequest{Status: tc.status, Reason: tc.reason}.Validate()
			if err != tc.wantErr {
				t.Fatalf("Validate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestHandleCreateAdminRejectsInvalidEmail(t *testing.T) {
	handler := NewHandler(newStore(t))
	req := httptest.NewRequest(http.MethodPost, "/admin/admins", strings.NewReader(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleCreateAdmin().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateAdminRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(newStore(t))
	req := httptest.NewRequest(http.MethodPost, "/admin/admins", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleCreateAdmin().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGrantCapabilityRejectsUnknownCapability(t *testing.T) {
	handler := NewHandler(newStore(t))
	body := `{"capability":"bogus","reason":"because"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/admins/a@example.com/capabilities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleGrantCapability().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleEnableAdminRejectsBlankReason(t *testing.T) {
	handler := NewHandler(newStore(t))
	body := `{"reason":"  "}`
	req := httptest.NewRequest(http.MethodPost, "/admin/admins/a@example.com/enable", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.HandleEnableAdmin().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}
