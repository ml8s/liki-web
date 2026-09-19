// Package donation implements a voluntary Alipay AI Pay donation flow.
//
// It is deliberately separate from commercial payments and entitlements: a
// donation produces a receipt only and never grants access to RPC methods.
package donation

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

const (
	ResourceID       = "/api/donations/redeem"
	Currency         = "CNY"
	CredentialPath   = "~/.liki/donation.json"
	SchemaVersion    = "liki-donation-v1"
	DefaultGoodsName = "Liki 捐赠"
)

const (
	StatusPending        = "pending"
	StatusPaid           = "paid"
	FulfillmentNone      = "unfulfilled"
	FulfillmentPending   = "pending_confirm"
	FulfillmentConfirmed = "fulfilled"
)

var (
	ErrOrderNotFound      = errors.New("donation: order not found")
	ErrProofNotAllowed    = errors.New("donation: proof is not valid for this order")
	ErrFulfillmentInvalid = errors.New("donation: invalid fulfillment state")
	ErrProofInvalid       = errors.New("donation: invalid payment proof")
	ErrGatewayRejected    = errors.New("donation: alipay rejected payment verification")
	ErrDisabled           = errors.New("donation: disabled")
)

var amountPattern = regexp.MustCompile(`^\d+(?:\.\d{1,2})?$`)

// Config is an explicit A2M configuration. Empty monetary or identity fields
// disable the endpoint; there is no invented default price.
type Config struct {
	AppID           string
	PrivateKey      string
	AlipayPublicKey string
	Gateway         string
	SellerID        string
	SellerName      string
	ServiceID       string
	Amount          string
	GoodsName       string
}

func (c Config) Enabled() bool {
	return c.AppID != "" && c.PrivateKey != "" && c.AlipayPublicKey != "" &&
		c.Gateway != "" && c.SellerID != "" && c.SellerName != "" &&
		c.ServiceID != "" && c.Amount != ""
}

func (c Config) normalizedAmount() (string, error) {
	if !amountPattern.MatchString(c.Amount) {
		return "", errors.New("donation: ALIPAY_DONATION_AMOUNT must be a decimal amount with at most 2 fraction digits")
	}
	parts := strings.SplitN(c.Amount, ".", 2)
	integer, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return "", errors.New("donation: ALIPAY_DONATION_AMOUNT is out of range")
	}
	if len(parts) == 1 {
		if integer == 0 {
			return "", errors.New("donation: ALIPAY_DONATION_AMOUNT must be positive")
		}
		return strconv.FormatUint(integer, 10) + ".00", nil
	}
	if integer == 0 && strings.Trim(parts[1], "0") == "" {
		return "", errors.New("donation: ALIPAY_DONATION_AMOUNT must be positive")
	}
	return strconv.FormatUint(integer, 10) + "." + parts[1] + strings.Repeat("0", 2-len(parts[1])), nil
}

type Bill struct {
	OutTradeNo string
	Amount     string
	Currency   string
	ResourceID string
	GoodsName  string
	PayBefore  string
	Status     string
}

type Order struct {
	OutTradeNo        string
	TradeNo           string
	Amount            string
	Currency          string
	ResourceID        string
	GoodsName         string
	PayBefore         string
	Status            string
	FulfillmentStatus string
	ReceiptID         string
	PaidAt            string
}

type VerifiedPayment struct {
	TradeNo    string
	OutTradeNo string
	Amount     string
	ResourceID string
	Active     bool
}

type proofPayload struct {
	Protocol struct {
		PaymentProof string `json:"payment_proof"`
		TradeNo      string `json:"trade_no"`
	} `json:"protocol"`
	Method struct {
		ClientSession string `json:"client_session"`
	} `json:"method"`
}
