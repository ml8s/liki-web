package arch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoundedContextDependencies(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string][]string{
		"github.com/ml8s/liki-web/internal/product": {
			"sales", "payment", "entitlement", "chat", "partner", "birthprofile",
		},
		"github.com/ml8s/liki-web/internal/sales": {
			"chat", "entitlement", "partner",
		},
		"github.com/ml8s/liki-web/internal/payment": {
			"product", "sales", "chat", "partner", "entitlement", "birthprofile",
		},
		"github.com/ml8s/liki-web/internal/entitlement": {
			"sales", "payment", "chat", "partner",
		},
		"github.com/ml8s/liki-web/internal/chat": {
			"sales", "payment", "partner",
		},
		"github.com/ml8s/liki-web/internal/partner": {
			"product", "sales", "chat", "entitlement", "birthprofile",
		},
	}
	for pkg, banned := range forbidden {
		t.Run(pkg, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", pkg)
			cmd.Dir = repo
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("go list: %v", err)
			}
			deps := map[string]bool{}
			for _, dep := range strings.Fields(string(out)) {
				deps[dep] = true
			}
			for _, bannedPkg := range banned {
				full := "github.com/ml8s/liki-web/internal/" + bannedPkg
				if deps[full] {
					t.Errorf("%s imports forbidden bounded context %s", pkg, bannedPkg)
				}
			}
		})
	}
}

func TestLegacyDomainPackagesAreRemoved(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"liki", "referral", "profile", "commerce", "fulfillment"} {
		if _, err := os.Stat(filepath.Join(repo, "internal", name)); err == nil {
			t.Errorf("legacy package directory internal/%s still exists", name)
		}
	}
}
