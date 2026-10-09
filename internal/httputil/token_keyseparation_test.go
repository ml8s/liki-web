package httputil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 密钥分离回归：capability token 的 MAC 必须来自 HKDF 派生密钥；
// 用原始平台密钥直接 HMAC 同一 payload 的伪造 token 不得通过校验。
func TestCapabilityTokenKeySeparation(t *testing.T) {
	t.Setenv("JWT_SECRET", "liki-ci-e2e-secret-0123456789abcdef")
	now := time.Now()

	token, err := NewCapabilityToken("payment-status", "intent-1", now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyCapabilityToken("payment-status", "intent-1", token, now.Add(time.Minute)) {
		t.Fatal("valid capability token rejected")
	}

	parts := strings.Split(token, ".")
	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	payload := fmt.Sprintf("capability:v1|payment-status|intent-1|%d", expiresAt)
	mac := hmac.New(sha256.New, []byte("liki-ci-e2e-secret-0123456789abcdef"))
	mac.Write([]byte(payload))
	forged := parts[0] + "." + parts[1] + "." + hex.EncodeToString(mac.Sum(nil))

	if VerifyCapabilityToken("payment-status", "intent-1", forged, now.Add(time.Minute)) {
		t.Fatal("raw-secret MAC accepted: key separation broken")
	}
}
