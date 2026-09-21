package referral

import (
	"os"
	"strings"
	"testing"
)

// These are independent partner surfaces. A regression that links or exposes
// one program inside the other would reveal the existence of the other role.
func TestPartnerFrontendsAreIsolated(t *testing.T) {
	kol, err := os.ReadFile("../../web/kol.html")
	if err != nil {
		t.Fatal(err)
	}
	scout, err := os.ReadFile("../../web/scout.html")
	if err != nil {
		t.Fatal(err)
	}

	for _, banned := range []string{"scout", "/scout", "channel", "/channel"} {
		if strings.Contains(strings.ToLower(string(kol)), strings.ToLower(banned)) {
			t.Fatalf("KOL frontend leaked scout concept: %q", banned)
		}
	}
	for _, banned := range []string{"kol", "/kol"} {
		if strings.Contains(strings.ToLower(string(scout)), strings.ToLower(banned)) {
			t.Fatalf("Scout frontend leaked KOL concept: %q", banned)
		}
	}
}
