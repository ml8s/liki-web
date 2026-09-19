package aipay

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	SandboxGateway   = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	SandboxServiceID = "api_mock_service_id"
)

// sandboxFile mirrors the Alipay quick sandbox JSON shape. Go is not a
// first-class language in the official quick-sandbox launcher, but non-Java
// projects use the same PKCS#1 private key field.
type sandboxFile struct {
	AppIDs []struct {
		AppID           string `json:"appId"`
		AppPrivatePKCS1 string `json:"appPrivatePkcsKey"`
		AlipayPublicKey string `json:"alipayPublicKey"`
		AppPublicKey    string `json:"appPublicKey"`
		PID             string `json:"pid"`
	} `json:"appIds"`
	SandboxAccounts struct {
		Partner struct {
			UserID string `json:"userId"`
		} `json:"partner"`
		User struct {
			UserID string `json:"userId"`
		} `json:"user"`
	} `json:"sandboxAccounts"`
}

// LoadSandboxConfig reads an explicit local quick-sandbox file. It never reads
// production credentials and always pins the sandbox gateway and service ID.
func LoadSandboxConfig(path string) (Config, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Config{}, fmt.Errorf("aipay: read alipay sandbox config: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Config{}, errors.New("aipay: alipay sandbox config must be a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		return Config{}, errors.New("aipay: alipay sandbox config must have mode 0600")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("aipay: read alipay sandbox config: %w", err)
	}
	var file sandboxFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return Config{}, fmt.Errorf("aipay: decode alipay sandbox config: %w", err)
	}
	if len(file.AppIDs) == 0 {
		return Config{}, errors.New("aipay: alipay sandbox config has no appId")
	}
	app := file.AppIDs[0]
	if app.AppID == "" || app.AppPrivatePKCS1 == "" || app.AlipayPublicKey == "" || app.AppPublicKey == "" {
		return Config{}, errors.New("aipay: alipay sandbox config is missing app credentials")
	}
	sellerID := file.SandboxAccounts.Partner.UserID
	if sellerID == "" {
		sellerID = app.PID
	}
	if sellerID == "" {
		return Config{}, errors.New("aipay: alipay sandbox config has no seller id")
	}
	return Config{
		AppID:           app.AppID,
		PrivateKey:      app.AppPrivatePKCS1,
		AlipayPublicKey: app.AlipayPublicKey,
		Gateway:         SandboxGateway,
		SellerID:        sellerID,
		ServiceID:       SandboxServiceID,
	}, nil
}
