package aipay

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"
)

type goldenVectors struct {
	Algorithm struct {
		Signature         string `json:"signature"`
		Hash              string `json:"hash"`
		SignatureEncoding string `json:"signature_encoding"`
		PayloadEncoding   string `json:"payload_encoding"`
	} `json:"algorithm"`
	PublicKey struct {
		NHex string `json:"n_hex"`
		EInt int    `json:"e_int"`
	} `json:"public_key"`
	SellerBill struct {
		Params    map[string]string `json:"params"`
		Canonical string            `json:"canonical"`
		Signature string            `json:"signature"`
	} `json:"seller_bill"`
	PaymentNeeded struct {
		Decoded map[string]any `json:"decoded"`
		Encoded string         `json:"encoded"`
	} `json:"payment_needed"`
	PaymentProof struct {
		Decoded struct {
			Protocol struct {
				PaymentProof string `json:"payment_proof"`
				TradeNo      string `json:"trade_no"`
			} `json:"protocol"`
			Method struct {
				ClientSession string `json:"client_session"`
			} `json:"method"`
		} `json:"decoded"`
		Encoded string `json:"encoded"`
	} `json:"payment_proof"`
	SyncResponses struct {
		Verify  goldenSyncResponse `json:"verify"`
		Confirm goldenSyncResponse `json:"confirm"`
	} `json:"sync_responses"`
	NegativeCases struct {
		TamperedResponse struct {
			Content   string `json:"content"`
			Signature string `json:"signature"`
			Valid     bool   `json:"valid"`
		} `json:"tampered_response"`
	} `json:"negative_cases"`
}

type goldenSyncResponse struct {
	Method    string          `json:"method"`
	Raw       string          `json:"raw"`
	Node      json.RawMessage `json:"node"`
	Signature string          `json:"signature"`
	Content   string          `json:"content"`
}

func loadAipayGolden(t *testing.T) goldenVectors {
	t.Helper()
	raw, err := os.ReadFile("testdata/aipay/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors goldenVectors
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}

func goldenPublicKey(t *testing.T, vectors goldenVectors) *rsa.PublicKey {
	t.Helper()
	modulus, ok := new(big.Int).SetString(vectors.PublicKey.NHex, 16)
	if !ok {
		t.Fatal("invalid golden RSA modulus")
	}
	return &rsa.PublicKey{N: modulus, E: vectors.PublicKey.EInt}
}

func TestAipayGoldenAlgorithm(t *testing.T) {
	vectors := loadAipayGolden(t)
	if vectors.Algorithm.Signature != "RSASSA-PKCS1-v1_5" ||
		vectors.Algorithm.Hash != "SHA-256" ||
		vectors.Algorithm.SignatureEncoding != "base64-standard" ||
		vectors.Algorithm.PayloadEncoding != "base64url-no-padding" {
		t.Fatalf("unaipayed golden algorithm: %+v", vectors.Algorithm)
	}
}

func TestAipayGoldenSellerBillMatchesGoCanonicalForm(t *testing.T) {
	vectors := loadAipayGolden(t)
	client := &Client{publicKey: goldenPublicKey(t, vectors)}

	if got := signContent(vectors.SellerBill.Params); got != vectors.SellerBill.Canonical {
		t.Fatalf("canonical content mismatch\nGo:    %q\nGolden: %q", got, vectors.SellerBill.Canonical)
	}
	if err := client.verify(vectors.SellerBill.Canonical, vectors.SellerBill.Signature); err != nil {
		t.Fatalf("verify seller bill: %v", err)
	}
}

func TestAipayGoldenPayloadEncodings(t *testing.T) {
	vectors := loadAipayGolden(t)

	raw, err := decodeBase64URL(vectors.PaymentNeeded.Encoded)
	if err != nil {
		t.Fatalf("decode Payment-Needed: %v", err)
	}
	var bill map[string]any
	if err := json.Unmarshal(raw, &bill); err != nil {
		t.Fatalf("decode Payment-Needed JSON: %v", err)
	}
	assertEqualJSON(t, "Payment-Needed", bill, vectors.PaymentNeeded.Decoded)

	payload, err := parseProof(vectors.PaymentProof.Encoded)
	if err != nil {
		t.Fatalf("parse Payment-Proof: %v", err)
	}
	if payload.Protocol.PaymentProof != vectors.PaymentProof.Decoded.Protocol.PaymentProof ||
		payload.Protocol.TradeNo != vectors.PaymentProof.Decoded.Protocol.TradeNo ||
		payload.Method.ClientSession != vectors.PaymentProof.Decoded.Method.ClientSession {
		t.Fatalf("Payment-Proof mismatch: %#v", payload)
	}
}

func TestAipayGoldenSyncResponseSignature(t *testing.T) {
	vectors := loadAipayGolden(t)
	client := &Client{publicKey: goldenPublicKey(t, vectors)}

	for name, response := range map[string]goldenSyncResponse{
		"verify":  vectors.SyncResponses.Verify,
		"confirm": vectors.SyncResponses.Confirm,
	} {
		t.Run(name, func(t *testing.T) {
			node, err := client.decodeSignedResponse(response.Method, []byte(response.Raw))
			if err != nil {
				t.Fatalf("decode signed response: %v", err)
			}
			assertEqualJSON(t, "response node", responseFields(t, node), responseFields(t, response.Node))
			compact := compactJSON(t, response.Node)
			if compact != response.Content {
				t.Fatalf("response content mismatch\nGo:    %q\nGolden: %q", compact, response.Content)
			}
		})
	}
}

func TestAipayGoldenNegativeCases(t *testing.T) {
	vectors := loadAipayGolden(t)
	client := &Client{publicKey: goldenPublicKey(t, vectors)}

	if err := client.verify(vectors.NegativeCases.TamperedResponse.Content, vectors.NegativeCases.TamperedResponse.Signature); err == nil {
		t.Fatal("tampered response signature unexpectedly verified")
	}
}

func responseFields(t *testing.T, node json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(node, &fields); err != nil {
		t.Fatalf("decode golden response node: %v", err)
	}
	return fields
}

func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		t.Fatalf("compact JSON: %v", err)
	}
	return buf.String()
}

func assertEqualJSON(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s mismatch\nGot:  %#v\nWant: %#v", name, got, want)
	}
}
