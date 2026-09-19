# Liki Alipay AI Pay Donation System Plan

Status: approved for implementation  
Date: 2026-09-19

## 1. Goal

Add an Alipay AI Pay based voluntary donation flow for agent users.

The donation system is intentionally independent from the existing commercial
payment system. It does not create entitlements, unlock RPC methods, change LLM
quality, or introduce a user tier.

```text
/api/payments/*     commercial products, orders, entitlements
/api/donations/*    voluntary donation receipts
/jsonrpc            free business methods
engine              payment-unaware
```

## 2. Product Rules

- All existing RPC features remain free.
- A donation gives gratitude and a receipt, not a service level or privilege.
- A session asks at most once.
- Ask only after a completed result has been positively acknowledged by the
  user, or when the user explicitly asks how to support Liki.
- Never ask before a task, during work, immediately after an unacknowledged
  result, or while the user is dissatisfied and requesting corrections.
- If there is no explicit satisfaction signal, do not ask.
- If `~/.liki/donation.json` already contains a valid receipt, do not ask.
- If the user declines, do not ask again for that session.
- Failure or decline never blocks normal use.
- Do not use religious wording such as “上香”. The product name is “捐赠”.

## 3. Alipay A2M Mapping

Alipay AI Pay standard flow:

```text
request without Payment-Proof
  -> 402 + Payment-Needed
buyer pays
  -> Payment-Proof
retry with Payment-Proof
  -> verify proof
  -> fulfill resource
  -> fulfillment.confirm
  -> 200 response
```

Liki donation mapping:

| Alipay concept | Liki donation meaning |
| --- | --- |
| Resource | `/api/donations/redeem` |
| Payment | Voluntary support |
| Resource result | Donation receipt |
| Fulfillment | Persist and return the receipt |
| Access control | None |

The endpoint still follows the A2M protocol and verifies the proof strictly.
It only changes the product meaning: the resource is a receipt, not access.

## 4. API

### 4.1 Create bill

```http
POST /api/donations/redeem
```

No `Payment-Proof` header. The response is HTTP 402 and contains a signed
`Payment-Needed` header. The response body is explanatory only.

```json
{
  "code": "Payment-Needed",
  "message": "Liki 是捐赠制。支付后获得捐赠凭证，所有功能仍然免费。",
  "out_trade_no": "...",
  "amount": "1.00",
  "currency": "CNY",
  "goods_name": "Liki 捐赠"
}
```

### 4.2 Redeem proof

```http
POST /api/donations/redeem
Payment-Proof: <base64url proof>
```

Server-side checks:

1. Decode the proof.
2. Extract `protocol.payment_proof`, `protocol.trade_no`, and optional
   `method.client_session`.
3. Call `alipay.aipay.agent.payment.verify`.
4. Require response `code=10000`, `active=true`.
5. Require returned `amount`, `out_trade_no`, `resource_id`, and currency to
   match the local order.
6. Reject expired, canceled, unknown, or otherwise unusable orders with 402.
7. Persist the verified order idempotently.
8. Call `alipay.aipay.agent.fulfillment.confirm`.
9. Return the receipt only after confirmation succeeds.

If fulfillment confirmation fails, return HTTP 502 and allow retry with the
same proof. Repeated successful redemption returns the same receipt.

### 4.3 Receipt response

The response resource is a whole-file credential object:

```json
{
  "resource_id": "/api/donations/redeem",
  "content": {
    "schema_version": "liki-donation-v1",
    "kind": "donation-receipt",
    "status": "received",
    "receipt_id": "...",
    "out_trade_no": "...",
    "trade_no": "...",
    "amount": "1.00",
    "currency": "CNY",
    "donated_at": "2026-09-19T00:00:00Z",
    "message": "感谢支持。Liki 会继续免费开放。"
  },
  "credential_path": "~/.liki/donation.json",
  "fulfillment_confirmed": true
}
```

The skill writes `content` as the whole credential file. It must not extract a
token or invent additional fields.

## 5. Backend Design

Add a dedicated package:

```text
internal/donation/
  types.go
  store.go
  handler.go
  alipay.go
```

Responsibilities:

| Unit | Responsibility |
| --- | --- |
| `types.go` | Data types and domain errors |
| `store.go` | SQLite schema, bills, idempotent fulfillment state |
| `handler.go` | HTTP protocol and strict verification flow |
| `alipay.go` | Gateway client for verify and fulfillment confirm |

SQLite table:

```text
donation_orders
  out_trade_no
  trade_no
  amount
  currency
  resource_id
  goods_name
  pay_before
  status
  fulfillment_status
  receipt_id
  created_at
  updated_at
  paid_at
```

States:

| Stage | Value |
| --- | --- |
| Created | `status=pending`, `fulfillment_status=unfulfilled` |
| Proof verified | `status=paid`, `fulfillment_status=pending_confirm` |
| Confirm accepted | `status=paid`, `fulfillment_status=fulfilled` |

No refund, revoke, rollback, or expire-backfill feature is included.

## 6. Configuration

```env
ALIPAY_APP_ID=
ALIPAY_PRIVATE_KEY=
ALIPAY_PUBLIC_KEY=
ALIPAY_GATEWAY=https://openapi.alipay.com/gateway.do
ALIPAY_SELLER_ID=
ALIPAY_SERVICE_ID=
ALIPAY_DONATION_AMOUNT=1.00
ALIPAY_CONFIG_FILE=
```

Rules:

- The endpoint is disabled unless every required Alipay configuration is set.
- Amount is explicit; there is no implicit default price.
- Production uses `https://openapi.alipay.com/gateway.do`.
- Production must not set `ALIPAY_CONFIG_FILE`; its gateway, service ID, and
  credentials must come from the production environment.
- Local sandbox may set `ALIPAY_CONFIG_FILE` to an explicit, ignored, mode-0600
  `.alipay-sandbox.json` file produced by the official local sandbox flow. The
  loader maps `appIds[0].appId`, `appPrivatePkcsKey`, and `alipayPublicKey`,
  pins the sandbox gateway, and uses the fixed sandbox service ID. It is not a
  mock and all protocol checks remain enabled.
- The official quick-sandbox bootstrap does not offer Go as a first-class
  runtime. Go is implemented directly against the A2M contract; protocol
  fixtures live in `internal/donation/testdata/aipay`.
- Keys are loaded only by the donation package and are never logged.
- Logs must not contain `Payment-Proof`, private keys, or full buyer data.

## 7. Skill Flow

Add a small donation helper and a concise `Donation` section to `SKILL.md`.

The helper is `web/skills/liki/donation.py`.

```bash
python3 donation.py status
python3 donation.py request
python3 donation.py confirm --proof-file /tmp/payment-proof
```

`confirm` also accepts the proof on stdin. Command-line arguments and logs must
not contain the proof.

The complete behavior is:

1. Read `~/.liki/donation.json`.
2. If `schema_version=liki-donation-v1`, `kind=donation-receipt`, and
   `status=received`, stop silently.
3. Offer only after the user confirms satisfaction with a completed result,
   thanks the agent, or explicitly asks how to support Liki.
4. Do not offer before the result, during analysis, after an unacknowledged
   result, or while the user is dissatisfied and asking for corrections.
5. If the user agrees, call the donation helper without proof.
6. Use the returned `Payment-Needed` bill with Alipay AI Pay.
7. After payment, retry with `Payment-Proof`.
8. Write the returned `content` whole to `~/.liki/donation.json`.
9. If the user declines or the flow fails, continue normally.

The skill must not say donation unlocks features.

## 8. Test Checklist

Backend:

- Missing proof returns 402 and a decodable `Payment-Needed` bill.
- Disabled configuration returns 503.
- Invalid proof returns 402.
- Gateway verification failure returns 402.
- Amount mismatch returns 402.
- Unknown local order returns 402.
- Resource ID mismatch returns 402.
- Successful proof creates one receipt and confirms fulfillment.
- Retrying the same proof returns the same receipt and does not confirm again.
- Confirmation failure returns 502 and permits retry.
- Database state remains consistent across retries.
- Alipay request signatures and signed response verification are exercised
  against a local protocol-conformant gateway.

Skill helper:

- Valid receipt suppresses prompting.
- Invalid receipt does not count as donated.
- Request mode returns a protocol bill without storing payment data.
- Confirm mode writes the returned credential as a whole object.
- Network and protocol failures do not create a local receipt.

Regression:

- `make test-go`
- `make test-python`
- Skill/static checks affected by `SKILL.md`
- Existing payment and entitlement tests remain unchanged and green.
