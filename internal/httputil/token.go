package httputil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func ReturnToken(product, orderID string) string {
	mac := hmac.New(sha256.New, JWTSecret())
	mac.Write([]byte(product + ":" + orderID))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyReturnToken(product, orderID, token string) bool {
	return hmac.Equal([]byte(ReturnToken(product, orderID)), []byte(token))
}
