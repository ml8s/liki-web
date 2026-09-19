# Alipay AI Pay Golden vectors

This directory is a **test-only protocol oracle** for the donation endpoint.
It is not production code, is not imported by the server, and must never be
wired into request handling.

## Purpose

`golden.py` independently fixes the A2M protocol rules that Liki's Go client
must implement:

- sorted `key=value` signing content;
- omission of empty/null values without introducing a leading `&`;
- RSASSA-PKCS1-v1_5 with SHA-256 and standard Base64 signatures;
- Base64URL without padding for `Payment-Needed` and `Payment-Proof`;
- required protocol/method fields in the 402 bill;
- required verify fields and signed OpenAPI response-node verification.

`vectors.json` contains signed fixture data and an RSA **public key only**.
The associated key is generated solely for fixtures and is not a production
secret. Go tests consume the committed vectors directly, so CI does not require
Python.

The local official Alipay quick-sandbox flow supports Java, Python, Node.js,
C#, and PHP; it does not offer Go as a first-class runtime. Go is still valid
for production because it is an AOT native binary. Liki therefore implements
the official A2M contract directly and uses these vectors to check protocol
behavior.

## Commands

Verify committed vectors:

```bash
python3 golden.py verify --vectors vectors.json
```

Rotate the test-only fixture key and vectors locally:

```bash
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out /tmp/aipay-test.pem
python3 golden.py generate --private-key /tmp/aipay-test.pem --out vectors.json
python3 golden.py verify --vectors vectors.json
rm /tmp/aipay-test.pem
```

Do not commit a private key, real merchant key, sandbox credential, or
production gateway value here. Sandbox runtime credentials belong in an ignored,
mode-0600 `.alipay-sandbox.json` file supplied by the official local sandbox
tooling.
