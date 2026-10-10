package httputil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 密钥分离回归：capability token 的签名密钥必须是 HKDF 派生密钥。
// 用原始平台密钥（JWT 签名密钥）验证的 capability token 必须被拒——
// 两个上下文不得共享原始密钥材料。
func TestCapabilityTokenKeySeparation(t *testing.T) {
	t.Setenv("JWT_SECRET", "liki-ci-e2e-secret-0123456789abcdef")
	now := time.Now()

	token, err := NewCapabilityToken("payment-status", "intent-1", now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// 派生密钥下验证通过
	if !VerifyCapabilityToken("payment-status", "intent-1", token, now.Add(time.Minute)) {
		t.Fatal("valid capability token rejected under derived key")
	}

	// 原始平台密钥（JWT 签名密钥）下验证必须失败：密钥分离
	rawParser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	var claims jwt.RegisteredClaims
	if _, err := rawParser.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) {
		return []byte("liki-ci-e2e-secret-0123456789abcdef"), nil
	}); err == nil {
		t.Fatal("capability token verified under raw JWT secret: key separation broken")
	}
}
