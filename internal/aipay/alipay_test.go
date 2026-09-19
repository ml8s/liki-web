package aipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type alipayHarness struct {
	client     *Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	server     *httptest.Server
}

func newAlipayHarness(t *testing.T, handler http.HandlerFunc) *alipayHarness {
	t.Helper()
	publicKey := &newTestPrivateKey(t).PublicKey
	return newAlipayHarnessWithHandler(t, publicKey, handler)
}

func newAlipayHarnessWithHandler(t *testing.T, publicKey *rsa.PublicKey, handler http.HandlerFunc) *alipayHarness {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(Config{
		AppID:           "app-id",
		PrivateKey:      privateKeyPEM(t, privateKey),
		AlipayPublicKey: base64.StdEncoding.EncodeToString(publicDER),
		Gateway:         server.URL + "/gateway.do",
		SellerID:        "2088",
		SellerName:      "Liki",
		ServiceID:       "service-id",
		Amount:          "1.00",
		GoodsName:       "Liki 服务支持",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &alipayHarness{client: client, privateKey: privateKey, publicKey: publicKey, server: server}
}

func privateKeyPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	raw := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: raw}))
}

func signRSA(t *testing.T, key *rsa.PrivateKey, content string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func verifyRSA(t *testing.T, key *rsa.PublicKey, content, signature string) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := sha256.Sum256([]byte(content))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], raw); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

func signedResponse(t *testing.T, key *rsa.PrivateKey, method string, response map[string]any) []byte {
	t.Helper()
	responseRaw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	envelope := map[string]json.RawMessage{}
	if err := json.Unmarshal(responseRaw, &envelope); err != nil {
		t.Fatal(err)
	}
	responseKey := strings.ReplaceAll(method, ".", "_") + "_response"
	envelope[responseKey] = responseRaw
	envelope["sign"] = json.RawMessage(jsonQuote(signRSA(t, key, string(responseRaw))))
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func jsonQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func newTestPrivateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestAlipayClientSignsAndVerifiesA2MCalls(t *testing.T) {
	var sawVerifyBiz map[string]string
	var sawConfirmBiz map[string]string
	var testKey *rsa.PrivateKey
	var harness *alipayHarness
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/gateway.do") {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		params := map[string]string{}
		for key := range r.PostForm {
			params[key] = r.PostForm.Get(key)
		}
		signature := params["sign"]
		delete(params, "sign")
		content := signContent(params)
		verifyRSA(t, &harness.privateKey.PublicKey, content, signature)
		switch params["method"] {
		case verifyMethod:
			sawVerifyBiz = map[string]string{}
			if err := json.Unmarshal([]byte(params["biz_content"]), &sawVerifyBiz); err != nil {
				t.Errorf("decode verify biz: %v", err)
			}
			writeRaw(t, w, signedResponse(t, testKey, verifyMethod, map[string]any{
				"code":         "10000",
				"msg":          "Success",
				"trade_no":     "trade-1",
				"out_trade_no": "order-1",
				"amount":       "1.00",
				"resource_id":  ResourceID,
				"active":       true,
			}))
		case confirmMethod:
			sawConfirmBiz = map[string]string{}
			if err := json.Unmarshal([]byte(params["biz_content"]), &sawConfirmBiz); err != nil {
				t.Errorf("decode confirm biz: %v", err)
			}
			writeRaw(t, w, signedResponse(t, testKey, confirmMethod, map[string]any{
				"code": "10000",
				"msg":  "Success",
			}))
		default:
			t.Errorf("unexpected method %q", params["method"])
			http.Error(w, "bad method", http.StatusBadRequest)
		}
	}
	testKey = newTestPrivateKey(t)
	harness = newAlipayHarnessWithHandler(t, &testKey.PublicKey, http.HandlerFunc(handler))

	bill, err := harness.client.PaymentNeededHeader(context.Background(), Bill{
		OutTradeNo: "order-1",
		Amount:     "1.00",
		Currency:   Currency,
		ResourceID: ResourceID,
		GoodsName:  "Liki 服务支持",
		PayBefore:  time.Now().Format(time.RFC3339),
		Status:     StatusPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(bill); err != nil ||
		!strings.Contains(string(decoded), ResourceID) {
		t.Fatalf("bill = %q decode=%q err=%v", bill, decoded, err)
	}

	proof := base64.RawURLEncoding.EncodeToString([]byte(`{"protocol":{"payment_proof":"inner-proof","trade_no":"trade-1"}}`))
	verified, err := harness.client.Verify(context.Background(), proof)
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Active || verified.TradeNo != "trade-1" || verified.OutTradeNo != "order-1" || verified.Amount != "1.00" {
		t.Fatalf("verified = %+v", verified)
	}
	if sawVerifyBiz["payment_proof"] != "inner-proof" || sawVerifyBiz["trade_no"] != "trade-1" {
		t.Fatalf("verify biz = %#v", sawVerifyBiz)
	}
	if err := harness.client.Confirm(context.Background(), "trade-1"); err != nil {
		t.Fatal(err)
	}
	if sawConfirmBiz["trade_no"] != "trade-1" {
		t.Fatalf("confirm biz = %#v", sawConfirmBiz)
	}
}

func TestAlipayClientRejectsTamperedResponse(t *testing.T) {
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	harness := newAlipayHarness(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		writeRaw(t, w, signedResponse(t, wrongKey, verifyMethod, map[string]any{
			"code": "10000", "active": true,
		}))
	})
	proof := base64.RawURLEncoding.EncodeToString([]byte(`{"protocol":{"payment_proof":"p","trade_no":"t"}}`))
	if _, err := harness.client.Verify(context.Background(), proof); err == nil {
		t.Fatal("expected tampered response to fail")
	}
}

func TestAlipayClientRejectsGatewayHTTPErrors(t *testing.T) {
	for _, code := range []int{http.StatusInternalServerError, http.StatusNotFound, http.StatusBadGateway} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			harness := newAlipayHarness(t, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "server error", code)
			})
			proof := base64.RawURLEncoding.EncodeToString(
				[]byte(`{"protocol":{"payment_proof":"p","trade_no":"t"}}`))
			_, err := harness.client.Verify(context.Background(), proof)
			if err == nil || !strings.Contains(err.Error(), "alipay HTTP") {
				t.Fatalf("err = %v, want alipay HTTP %d", err, code)
			}
		})
	}
}

func TestAlipayClientRejectsMalformedGatewayJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `not-json`},
		{name: "missing response node", body: `{"sign":"abc"}`},
		{name: "missing signature", body: `{"alipay_aipay_agent_payment_verify_response":{"code":"10000"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newAlipayHarness(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.body))
			})
			proof := base64.RawURLEncoding.EncodeToString(
				[]byte(`{"protocol":{"payment_proof":"p","trade_no":"t"}}`))
			_, err := harness.client.Verify(context.Background(), proof)
			if err == nil {
				t.Fatal("expected malformed gateway response to fail")
			}
		})
	}
}

func writeRaw(t *testing.T, w http.ResponseWriter, raw []byte) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if _, err := w.Write(raw); err != nil {
		t.Errorf("write response: %v", err)
	}
}
