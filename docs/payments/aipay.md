# AI Pay

AI Pay is a separate donation-style or per-call payment integration. It is not part of the Liki Chat commercial checkout and does not create `liki-chat` entitlement by itself.

## Boundary

```text
Skill / caller
→ liki-web AI Pay endpoint
→ payment proof
→ explicit business action
```

AI Pay handlers do not mutate chat entitlements unless a separate commercial flow explicitly requires it.

## Configuration

See `.env.example`.

Core variables:

```text
ALIPAY_APP_ID
ALIPAY_PRIVATE_KEY
ALIPAY_PUBLIC_KEY
ALIPAY_GATEWAY
ALIPAY_SELLER_ID
ALIPAY_SELLER_NAME
ALIPAY_SERVICE_ID
ALIPAY_AIPAY_AMOUNT
```

Missing private configuration disables the public AI Pay flow.

## Security

- Private keys remain in deployment configuration.
- Payment proofs are verified before business effects.
- Receipt storage does not include secrets.
