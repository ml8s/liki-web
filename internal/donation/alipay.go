package donation

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	verifyMethod    = "alipay.aipay.agent.payment.verify"
	confirmMethod   = "alipay.aipay.agent.fulfillment.confirm"
	maxGatewayBytes = 1024 * 1024
)

type Client struct {
	config     Config
	privateKey *rsa.PrivateKey
	publicKey  crypto.PublicKey
	httpClient *http.Client
	nowFun     func() time.Time
}

func NewClient(config Config) (*Client, error) {
	if !config.Enabled() {
		return nil, ErrDisabled
	}
	amount, err := config.normalizedAmount()
	if err != nil {
		return nil, err
	}
	config.Amount = amount
	if config.GoodsName == "" {
		config.GoodsName = DefaultGoodsName
	}
	privateKey, err := parsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("donation: parse alipay private key: %w", err)
	}
	publicKey, err := parsePublicKey(config.AlipayPublicKey)
	if err != nil {
		return nil, fmt.Errorf("donation: parse alipay public key: %w", err)
	}
	return &Client{
		config:     config,
		privateKey: privateKey,
		publicKey:  publicKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		nowFun:     func() time.Time { return time.Now() },
	}, nil
}

func (c *Client) PaymentNeededHeader(_ context.Context, bill Bill) (string, error) {
	signature, err := c.signSellerBill(bill)
	if err != nil {
		return "", err
	}
	body := map[string]any{
		"protocol": map[string]any{
			"out_trade_no":     bill.OutTradeNo,
			"amount":           bill.Amount,
			"currency":         bill.Currency,
			"resource_id":      bill.ResourceID,
			"pay_before":       bill.PayBefore,
			"seller_signature": signature,
			"seller_sign_type": "RSA2",
			"seller_unique_id": c.config.SellerID,
		},
		"method": map[string]any{
			"seller_name":          c.config.SellerName,
			"seller_id":            c.config.SellerID,
			"seller_app_id":        c.config.AppID,
			"goods_name":           bill.GoodsName,
			"seller_unique_id_key": "seller_id",
			"service_id":           c.config.ServiceID,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("donation: encode payment needed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (c *Client) Verify(ctx context.Context, proof string) (VerifiedPayment, error) {
	payload, err := parseProof(proof)
	if err != nil || payload.Protocol.PaymentProof == "" || payload.Protocol.TradeNo == "" {
		return VerifiedPayment{}, ErrProofInvalid
	}
	biz := map[string]string{
		"payment_proof": payload.Protocol.PaymentProof,
		"trade_no":      payload.Protocol.TradeNo,
	}
	if payload.Method.ClientSession != "" {
		biz["client_session"] = payload.Method.ClientSession
	}
	node, err := c.execute(ctx, verifyMethod, biz)
	if err != nil {
		return VerifiedPayment{}, err
	}
	var response struct {
		Code       string `json:"code"`
		TradeNo    string `json:"trade_no"`
		OutTradeNo string `json:"out_trade_no"`
		Amount     string `json:"amount"`
		ResourceID string `json:"resource_id"`
		Active     *bool  `json:"active"`
	}
	if err := json.Unmarshal(node, &response); err != nil {
		return VerifiedPayment{}, fmt.Errorf("donation: decode alipay verification: %w", err)
	}
	if response.Code != "10000" {
		return VerifiedPayment{}, ErrGatewayRejected
	}
	active := response.Active != nil && *response.Active
	return VerifiedPayment{
		TradeNo:    response.TradeNo,
		OutTradeNo: response.OutTradeNo,
		Amount:     response.Amount,
		ResourceID: response.ResourceID,
		Active:     active,
	}, nil
}

func (c *Client) Confirm(ctx context.Context, tradeNo string) error {
	if tradeNo == "" {
		return ErrProofInvalid
	}
	_, err := c.execute(ctx, confirmMethod, map[string]string{"trade_no": tradeNo})
	return err
}

func (c *Client) signSellerBill(bill Bill) (string, error) {
	params := map[string]string{
		"amount":       bill.Amount,
		"currency":     bill.Currency,
		"goods_name":   bill.GoodsName,
		"out_trade_no": bill.OutTradeNo,
		"pay_before":   bill.PayBefore,
		"resource_id":  bill.ResourceID,
		"seller_id":    c.config.SellerID,
		"service_id":   c.config.ServiceID,
	}
	return c.sign(signContent(params))
}

func (c *Client) execute(ctx context.Context, method string, biz map[string]string) (json.RawMessage, error) {
	bizRaw, err := json.Marshal(biz)
	if err != nil {
		return nil, fmt.Errorf("donation: encode alipay biz content: %w", err)
	}
	params := map[string]string{
		"app_id":      c.config.AppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "UTF-8",
		"sign_type":   "RSA2",
		"timestamp":   c.nowFun().Local().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizRaw),
	}
	signature, err := c.sign(signContent(params))
	if err != nil {
		return nil, err
	}
	params["sign"] = signature

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Gateway, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("donation: create alipay request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("donation: call alipay: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxGatewayBytes+1))
	if err != nil {
		return nil, fmt.Errorf("donation: read alipay response: %w", err)
	}
	if len(raw) > maxGatewayBytes {
		return nil, errors.New("donation: alipay response too large")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("donation: alipay HTTP %d", response.StatusCode)
	}
	return c.decodeSignedResponse(method, raw)
}

func (c *Client) decodeSignedResponse(method string, raw []byte) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("donation: decode alipay envelope: %w", err)
	}
	node, ok := envelope[method]
	if !ok {
		node, ok = envelope["error_response"]
	}
	if !ok {
		return nil, errors.New("donation: alipay response node missing")
	}
	signature, ok := envelope["sign"]
	if !ok {
		return nil, errors.New("donation: alipay response signature missing")
	}
	var signatureValue string
	if err := json.Unmarshal(signature, &signatureValue); err != nil {
		return nil, fmt.Errorf("donation: decode alipay signature: %w", err)
	}
	var responseFields map[string]json.RawMessage
	if err := json.Unmarshal(node, &responseFields); err != nil {
		return nil, fmt.Errorf("donation: decode alipay response node: %w", err)
	}
	delete(responseFields, "sign")
	delete(responseFields, "sign_type")
	if err := c.verify(signContentRaw(responseFields), signatureValue); err != nil {
		return nil, fmt.Errorf("donation: verify alipay response: %w", err)
	}
	return node, nil
}

func (c *Client) sign(content string) (string, error) {
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("donation: sign alipay request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (c *Client) verify(content, signature string) error {
	rawSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	publicKey, ok := c.publicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("donation: alipay public key is not RSA")
	}
	digest := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], rawSignature)
}

func signContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	first := true
	for _, key := range keys {
		if params[key] == "" {
			continue
		}
		if !first {
			out.WriteByte('&')
		}
		first = false
		out.WriteString(key)
		out.WriteByte('=')
		out.WriteString(params[key])
	}
	return out.String()
}

// signContentRaw preserves the exact response JSON values for signature input.
func signContentRaw(fields map[string]json.RawMessage) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	first := true
	for _, key := range keys {
		value := strings.TrimSpace(string(fields[key]))
		if value == "" || value == "null" || value == `""` {
			continue
		}
		if !first {
			out.WriteByte('&')
		}
		first = false
		out.WriteString(key)
		out.WriteByte('=')
		if unquoted, err := strconv.Unquote(value); err == nil && (value[0] == '"' || value[0] == '\'') {
			out.WriteString(unquoted)
		} else {
			out.WriteString(value)
		}
	}
	return out.String()
}

func parsePrivateKey(value string) (*rsa.PrivateKey, error) {
	raw, err := decodeKeyMaterial(value)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(raw); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(raw)
	if err != nil {
		return nil, errors.New("unsupported private key encoding")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func parsePublicKey(value string) (crypto.PublicKey, error) {
	raw, err := decodeKeyMaterial(value)
	if err != nil {
		return nil, err
	}
	parsed, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return nil, errors.New("unsupported public key encoding")
	}
	if _, ok := parsed.(*rsa.PublicKey); !ok {
		return nil, errors.New("public key is not RSA")
	}
	return parsed, nil
}

func decodeKeyMaterial(value string) ([]byte, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\n", "\n")
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return block.Bytes, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
	if err != nil {
		return nil, errors.New("key is not PEM or base64")
	}
	return raw, nil
}
