package xunhu

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // Xunhu API requires MD5 signing
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ml8s/liki-web/internal/payment"
)

const (
	defaultPayURL   = "https://api.xunhupay.com/payment/do.html"
	defaultQueryURL = "https://api.xunhupay.com/payment/query.html"
	requestTimeout  = 30 * time.Second
)

// Client implements payment.Provider for XunhuPay (虎皮椒).
type Client struct {
	appID      string
	appSecret  string
	webhookURL string
	httpClient *http.Client
	payURL     string
	queryURL   string
}

// New creates a XunhuPay client. webhookURL is the provider-level notify URL.
func New(appID, appSecret, webhookURL string) *Client {
	return &Client{
		appID:      appID,
		appSecret:  appSecret,
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: requestTimeout},
		payURL:     defaultPayURL,
		queryURL:   defaultQueryURL,
	}
}

// FetchPayment queries Xunhu's provider-side order status for reconciliation.
//
// Xunhu signs the query request, but its published PHP SDK does not define a
// canonical signature for the query response's nested JSON object. Server-to-
// server queries therefore rely on HTTPS and the signed request; payment
// callbacks remain signature-verified by VerifyWebhook.
func (c *Client) FetchPayment(ctx context.Context, intent payment.PaymentIntent) (*payment.WebhookEvent, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("xunhu: generate nonce: %w", err)
	}
	params := map[string]string{
		"appid":           c.appID,
		"out_trade_order": intent.ID,
		"time":            strconv.FormatInt(time.Now().Unix(), 10),
		"nonce_str":       hex.EncodeToString(nonce),
	}
	params["hash"] = sign(params, c.appSecret)

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.queryURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("xunhu: build query: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("xunhu: query request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("xunhu: read query response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xunhu: query http %d: %s", response.StatusCode, string(responseBody))
	}

	var result struct {
		ErrCode int            `json:"errcode"`
		ErrMsg  string         `json:"errmsg"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("xunhu: parse query response: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("xunhu: %s (errcode=%d)", result.ErrMsg, result.ErrCode)
	}
	if result.Data["out_trade_order"] != intent.ID {
		return nil, fmt.Errorf("xunhu: query order mismatch: got %v want %s", result.Data["out_trade_order"], intent.ID)
	}

	status, _ := result.Data["status"].(string)
	if status != "OD" {
		return nil, nil
	}

	amountText, ok := result.Data["total_amount"].(string)
	if !ok || amountText == "" {
		return nil, fmt.Errorf("xunhu: paid query response has no total_amount")
	}
	amount, err := parseMinorUnits(amountText)
	if err != nil {
		return nil, fmt.Errorf("xunhu: query invalid total_amount: %w", err)
	}
	if amount != intent.AmountMinor {
		return nil, fmt.Errorf("xunhu: query amount mismatch: got %d want %d", amount, intent.AmountMinor)
	}
	paymentID, _ := result.Data["transaction_id"].(string)
	if paymentID == "" {
		return nil, fmt.Errorf("xunhu: paid query response has no transaction_id")
	}
	return &payment.WebhookEvent{
		Provider: payment.ProviderXunhu,
		Type:     payment.EventPaymentSucceeded,
		Data: payment.WebhookEventData{
			OrderID:   intent.ID,
			Amount:    amount,
			Currency:  intent.Currency,
			Email:     intent.Email,
			PaymentID: paymentID,
			Metadata: map[string]string{
				"status":          status,
				"amount_verified": "true",
			},
		},
	}, nil
}

// WebhookAck returns the exact body required by Xunhu after a verified callback.
func (c *Client) WebhookAck() []byte { return []byte("success") }

// CreateCheckout creates a XunhuPay checkout session from a normalized request.
func (c *Client) CreateCheckout(ctx context.Context, req payment.CheckoutRequest) (*payment.CheckoutResult, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("xunhu: generate nonce: %w", err)
	}

	params := map[string]string{
		"appid":          c.appID,
		"time":           strconv.FormatInt(time.Now().Unix(), 10),
		"version":        "1.1",
		"nonce_str":      hex.EncodeToString(nonce),
		"trade_order_id": req.IntentID,
		"total_fee":      formatMinorUnits(req.AmountMinor),
		"title":          req.Title,
		"notify_url":     c.webhookURL,
		"return_url":     req.ReturnURL,
	}
	if req.CancelURL != "" {
		params["callback_url"] = req.CancelURL
	}
	params["hash"] = sign(params, c.appSecret)

	// The official payment demo signs the flat parameter map, then sends the
	// JSON object to the payment gateway. Query and provider callbacks remain
	// form-encoded by their own official demos.
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("xunhu: encode request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.payURL, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("xunhu: build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("xunhu: request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("xunhu: read response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xunhu: http %d: %s", response.StatusCode, string(responseBody))
	}

	if err := verifyFlatResponseHash(responseBody, c.appSecret); err != nil {
		return nil, err
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		URL     string `json:"url"`
		QRCode  string `json:"url_qrcode"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("xunhu: parse response: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("xunhu: %s (errcode=%d)", result.ErrMsg, result.ErrCode)
	}
	return &payment.CheckoutResult{
		SessionID:   req.IntentID,
		CheckoutURL: result.URL,
		QRCodeURL:   result.QRCode,
	}, nil
}

// VerifyWebhook verifies and normalizes a XunhuPay webhook event.
func (c *Client) VerifyWebhook(rawBody []byte, headers http.Header) (*payment.WebhookEvent, error) {
	if len(rawBody) == 0 {
		return nil, fmt.Errorf("xunhu: empty body")
	}
	values, err := url.ParseQuery(string(rawBody))
	if err != nil {
		return nil, fmt.Errorf("xunhu: parse webhook: %w", err)
	}
	hash := values.Get("hash")
	if hash == "" {
		return nil, fmt.Errorf("xunhu: missing hash in webhook")
	}
	params := make(map[string]string, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		params[key] = values.Get(key)
	}
	expected := sign(params, c.appSecret)
	if !hmacEqual(hash, expected) {
		return nil, fmt.Errorf("xunhu: signature mismatch")
	}
	amount, err := parseMinorUnits(values.Get("total_fee"))
	if err != nil {
		return nil, fmt.Errorf("xunhu: invalid total_fee: %w", err)
	}
	eventType := values.Get("status")
	if eventType == "OD" {
		eventType = payment.EventPaymentSucceeded
	}
	orderID := values.Get("trade_order_id")
	if orderID == "" {
		orderID = values.Get("out_trade_no")
	}
	return &payment.WebhookEvent{
		Provider: payment.ProviderXunhu,
		Type:     eventType,
		Data: payment.WebhookEventData{
			OrderID:   orderID,
			Amount:    amount,
			PaymentID: values.Get("transaction_id"),
			Metadata:  map[string]string{"status": values.Get("status")},
		},
	}, nil
}

func flatResponseParams(response map[string]any) (map[string]string, error) {
	params := make(map[string]string, len(response))
	for key, value := range response {
		if key == "hash" {
			continue
		}
		switch typed := value.(type) {
		case string:
			params[key] = typed
		case json.Number:
			params[key] = typed.String()
		case int:
			params[key] = strconv.Itoa(typed)
		case int64:
			params[key] = strconv.FormatInt(typed, 10)
		case float64:
			params[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			params[key] = strconv.FormatBool(typed)
		case nil:
		default:
			return nil, fmt.Errorf("xunhu: unsupported response field %s", key)
		}
	}
	return params, nil
}

// verifyFlatResponseHash verifies Xunhu's flat JSON API responses. Creation
// responses are flat; query responses intentionally use a different nested
// contract and are handled by FetchPayment.
func verifyFlatResponseHash(raw []byte, secret string) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var response map[string]any
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("xunhu: parse response: %w", err)
	}
	provided, _ := response["hash"].(string)
	if provided == "" {
		return fmt.Errorf("xunhu: missing response hash")
	}
	params, err := flatResponseParams(response)
	if err != nil {
		return err
	}
	if !hmacEqual(provided, sign(params, secret)) {
		return fmt.Errorf("xunhu: response signature mismatch")
	}
	return nil
}

func formatMinorUnits(amount int64) string {
	return fmt.Sprintf("%d.%02d", amount/100, amount%100)
}

// parseMinorUnits converts XunhuPay's fixed 2-decimal amount string into
// integer minor units. Amounts with more than 2 decimal places are rejected
// (not truncated) — consistent with aipay.parseAmountMinor: a protocol
// violation must fail loudly on a money path, never silently lose precision.
func parseMinorUnits(value string) (int64, error) {
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return 0, fmt.Errorf("xunhu: amount %q has more than 2 decimal places", value)
		}
		if len(fraction) < 2 {
			fraction += "0"
		}
	}
	fractionMinor, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + fractionMinor, nil
}

// sign generates the MD5 signature required by XunhuPay. The official rule
// excludes hash and empty values, then signs non-empty values in ASCII order
// with APPSECRET appended directly.
func sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	raw := strings.Join(parts, "&") + secret
	sum := md5.Sum([]byte(raw)) //nolint:gosec
	return hex.EncodeToString(sum[:])
}

func hmacEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
