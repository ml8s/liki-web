package donation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSandboxConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".alipay-sandbox.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSandboxConfigMapsNonJavaFields(t *testing.T) {
	path := writeSandboxConfig(t, `{
		"appIds":[{
			"appId":"sandbox-app",
			"appPrivatePkcsKey":"private-key",
			"alipayPublicKey":"alipay-key",
			"appPublicKey":"app-key"
		}],
		"sandboxAccounts":{"partner":{"userId":"2088seller"}}
	}`)

	config, err := LoadSandboxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.AppID != "sandbox-app" || config.PrivateKey != "private-key" ||
		config.AlipayPublicKey != "alipay-key" || config.SellerID != "2088seller" {
		t.Fatalf("config = %+v", config)
	}
	if config.Gateway != SandboxGateway || config.ServiceID != SandboxServiceID {
		t.Fatalf("sandbox endpoints = %s %s", config.Gateway, config.ServiceID)
	}
}

func TestLoadSandboxConfigUsesPIDWhenPartnerMissing(t *testing.T) {
	path := writeSandboxConfig(t, `{
		"appIds":[{
			"appId":"sandbox-app",
			"appPrivatePkcsKey":"private-key",
			"alipayPublicKey":"alipay-key",
			"appPublicKey":"app-key",
			"pid":"2088pid"
		}],
		"sandboxAccounts":{"user":{"userId":"2088buyer"}}
	}`)

	config, err := LoadSandboxConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.SellerID != "2088pid" {
		t.Fatalf("seller id = %q", config.SellerID)
	}
}

func TestLoadSandboxConfigRejectsInsecureOrInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	insecure := filepath.Join(dir, "insecure.json")
	if err := os.WriteFile(insecure, []byte(`{"appIds":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSandboxConfig(insecure); err == nil || !strings.Contains(err.Error(), "mode 0600") {
		t.Fatalf("insecure file error = %v", err)
	}

	valid := writeSandboxConfig(t, `{"appIds":[{"appId":"app"}]}`)
	if _, err := LoadSandboxConfig(valid); err == nil || !strings.Contains(err.Error(), "missing app credentials") {
		t.Fatalf("invalid app error = %v", err)
	}

	missing := filepath.Join(dir, "missing.json")
	if _, err := LoadSandboxConfig(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file error = %v", err)
	}
}
