#!/usr/bin/env python3
"""Independent protocol oracle for Liki's Alipay AI Pay aipay flow.

The file is a test asset. It is not imported by production code and it uses
only the Python standard library. RSA aipay is intentionally limited to the
RSASSA-PKCS1-v1_5/SHA-256 algorithm used by the A2M contract.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import sys
from pathlib import Path
from typing import Any

SCHEMA_VERSION = "liki-aipay-golden-v1"
VERIFY_METHOD = "alipay.aipay.agent.payment.verify"
CONFIRM_METHOD = "alipay.aipay.agent.fulfillment.confirm"
SHA256_DIGEST_INFO = bytes.fromhex("3031300d060960864801650304020105000420")


def base64url_encode(value: bytes) -> str:
    return base64.urlsafe_b64encode(value).rstrip(b"=").decode("ascii")


def base64url_decode(value: str) -> bytes:
    if len(value) % 4:
        value += "=" * (4 - len(value) % 4)
    return base64.urlsafe_b64decode(value.encode("ascii"))


def compact_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"), sort_keys=True)


def canonical_value(value: Any) -> str:
    if isinstance(value, str):
        return value
    return compact_json(value)


def canonical_params(params: dict[str, Any]) -> str:
    """Alipay's sorted key=value format; empty values are omitted."""
    return "&".join(
        f"{key}={canonical_value(params[key])}"
        for key in sorted(params)
        if params[key] is not None and params[key] != ""
    )


def strip_signature_fields(fields: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in fields.items() if key not in {"sign", "sign_type"}}


def read_der_tlv(data: bytes, offset: int) -> tuple[int, bytes, int]:
    tag = data[offset]
    offset += 1
    length = data[offset]
    offset += 1
    if length & 0x80:
        size = length & 0x7F
        if size == 0 or size > 4:
            raise ValueError("unaipayed DER length")
        length = int.from_bytes(data[offset : offset + size], "big")
        offset += size
    value = data[offset : offset + length]
    if len(value) != length:
        raise ValueError("truncated DER value")
    return tag, value, offset + length


def parse_der_sequence(data: bytes) -> list[tuple[int, bytes]]:
    tag, body, consumed = read_der_tlv(data, 0)
    if tag != 0x30 or consumed != len(data):
        raise ValueError("expected one DER SEQUENCE")
    items: list[tuple[int, bytes]] = []
    offset = 0
    while offset < len(body):
        item_tag, value, next_offset = read_der_tlv(body, offset)
        items.append((item_tag, value))
        offset = next_offset
    return items


def parse_private_key_pem(pem: str) -> dict[str, int]:
    lines = [line for line in pem.strip().splitlines() if "-----" not in line]
    der = base64.b64decode("".join(lines), validate=True)
    items = parse_der_sequence(der)
    integers = items
    if items and items[0][0] == 0x02 and len(items) >= 3 and items[1][0] == 0x02:
        # PKCS#1: version, n, e, d, p, q, dp, dq, qinv.
        pass
    elif len(items) >= 3 and items[0][0] == 0x02 and items[1][0] == 0x30 and items[2][0] == 0x04:
        # PKCS#8: version, AlgorithmIdentifier, OCTET wrapping a PKCS#1 key.
        integers = parse_der_sequence(items[2][1])
    else:
        raise ValueError("unaipayed private key DER structure")

    names = ("version", "n", "e", "d", "p", "q", "dp", "dq", "qinv")
    if len(integers) < 9 or any(tag != 0x02 for tag, _ in integers[:9]):
        raise ValueError("private key is not RSA PKCS#1")
    key = dict(zip(names, (int.from_bytes(value, "big") for _, value in integers[:9])))
    if key["e"] < 3 or key["d"] <= 0 or key["n"] <= 0:
        raise ValueError("invalid RSA private key")
    return key


def i2osp(value: int, length: int) -> bytes:
    return value.to_bytes(length, "big")


def emsa_pkcs1_v15(content: str, key_length: int) -> bytes:
    digest = hashlib.sha256(content.encode("utf-8")).digest()
    encoded = b"\x00\x01" + b"\xff" * (key_length - len(SHA256_DIGEST_INFO) - len(digest) - 3) + b"\x00" + SHA256_DIGEST_INFO + digest
    if len(encoded) != key_length or (key_length - len(digest) - 3) < 8:
        raise ValueError("RSA key is too small for SHA-256")
    return encoded


def rsa_sha256_sign(private_key: dict[str, int], content: str) -> str:
    key_length = (private_key["n"].bit_length() + 7) // 8
    message = int.from_bytes(emsa_pkcs1_v15(content, key_length), "big")
    signature = pow(message, private_key["d"], private_key["n"])
    return base64.b64encode(i2osp(signature, key_length)).decode("ascii")


def rsa_sha256_verify(public_key: dict[str, int], content: str, signature: str) -> bool:
    try:
        raw_signature = base64.b64decode(signature, validate=True)
    except (ValueError, base64.binascii.Error):
        return False
    key_length = (public_key["n"].bit_length() + 7) // 8
    if len(raw_signature) != key_length:
        return False
    recovered = pow(int.from_bytes(raw_signature, "big"), public_key["e"], public_key["n"])
    expected = emsa_pkcs1_v15(content, key_length)
    return i2osp(recovered, key_length) == expected


def seller_signature(params: dict[str, Any], private_key: dict[str, int]) -> str:
    bill_fields = {
        "amount": params["amount"],
        "currency": params["currency"],
        "goods_name": params["goods_name"],
        "out_trade_no": params["out_trade_no"],
        "pay_before": params["pay_before"],
        "resource_id": params["resource_id"],
        "seller_id": params["seller_id"],
        "service_id": params["service_id"],
    }
    return rsa_sha256_sign(private_key, canonical_params(bill_fields))


def payment_needed(params: dict[str, Any], private_key: dict[str, int]) -> dict[str, Any]:
    payload = {
        "protocol": {
            "out_trade_no": params["out_trade_no"],
            "amount": params["amount"],
            "currency": "CNY",
            "resource_id": params["resource_id"],
            "pay_before": params["pay_before"],
            "seller_signature": seller_signature(params, private_key),
            "seller_sign_type": "RSA2",
            "seller_unique_id": params["seller_id"],
        },
        "method": {
            "seller_name": params["seller_name"],
            "seller_id": params["seller_id"],
            "seller_app_id": params["app_id"],
            "goods_name": params["goods_name"],
            "seller_unique_id_key": "seller_id",
            "service_id": params["service_id"],
        },
    }
    return {"decoded": payload, "encoded": base64url_encode(compact_json(payload).encode("utf-8"))}


def proof(payment_proof: str, trade_no: str, client_session: str = "") -> dict[str, Any]:
    payload: dict[str, Any] = {"protocol": {"payment_proof": payment_proof, "trade_no": trade_no}}
    if client_session:
        payload["method"] = {"client_session": client_session}
    return {"decoded": payload, "encoded": base64url_encode(compact_json(payload).encode("utf-8"))}


def signed_sync_response(method: str, node: dict[str, Any], private_key: dict[str, int]) -> dict[str, Any]:
    content = compact_json(node)
    signature = rsa_sha256_sign(private_key, content)
    response_key = method.replace(".", "_") + "_response"
    envelope = {response_key: node, "sign": signature}
    raw = compact_json(envelope)
    return {
        "method": method,
        "raw": raw,
        "node": node,
        "signature": signature,
        "content": content,
    }


def default_vectors(private_key: dict[str, int]) -> dict[str, Any]:
    bill = {
        "app_id": "sandbox-app",
        "out_trade_no": "order-1",
        "amount": "1.00",
        "currency": "CNY",
        "resource_id": "/api/aipay",
        "goods_name": "Liki 服务支持",
        "pay_before": "2026-01-01T00:00:00+08:00",
        "seller_id": "2088seller",
        "seller_name": "Liki",
        "service_id": "api_mock_service_id",
    }
    bill_fields = {key: bill[key] for key in (
        "amount", "currency", "goods_name", "out_trade_no", "pay_before",
        "resource_id", "seller_id", "service_id",
    )}
    payment = payment_needed(bill, private_key)
    response = signed_sync_response(VERIFY_METHOD, {
        "code": "10000",
        "msg": "Success",
        "active": True,
        "trade_no": "trade-1",
        "out_trade_no": "order-1",
        "amount": "1.00",
        "resource_id": "/api/aipay",
    }, private_key)
    confirm = signed_sync_response(CONFIRM_METHOD, {"code": "10000", "msg": "Success"}, private_key)
    return {
        "schema_version": SCHEMA_VERSION,
        "algorithm": {
            "signature": "RSASSA-PKCS1-v1_5",
            "hash": "SHA-256",
            "signature_encoding": "base64-standard",
            "payload_encoding": "base64url-no-padding",
        },
        "public_key": {
            "n_hex": format(private_key["n"], "x"),
            "e_int": private_key["e"],
        },
        "seller_bill": {
            "params": bill_fields,
            "canonical": canonical_params(bill_fields),
            "signature": payment["decoded"]["protocol"]["seller_signature"],
        },
        "payment_needed": payment,
        "payment_proof": proof("platform-inner-proof", "trade-1", "client-session"),
        "sync_responses": {"verify": response, "confirm": confirm},
        "negative_cases": {
            "tampered_response": {
                "content": compact_json({**response["node"], "amount": "2.00"}),
                "signature": response["signature"],
                "valid": False,
            },
        },
    }


def validate_vectors(vectors: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    if vectors.get("schema_version") != SCHEMA_VERSION:
        errors.append("schema_version mismatch")
    public_key = {
        "n": int(vectors["public_key"]["n_hex"], 16),
        "e": int(vectors["public_key"]["e_int"]),
    }
    checks = (
        ("seller_bill", vectors["seller_bill"]["canonical"], vectors["seller_bill"]["signature"]),
        ("verify", vectors["sync_responses"]["verify"]["content"], vectors["sync_responses"]["verify"]["signature"]),
        ("confirm", vectors["sync_responses"]["confirm"]["content"], vectors["sync_responses"]["confirm"]["signature"]),
    )
    for name, canonical, signature in checks:
        if not rsa_sha256_verify(public_key, canonical, signature):
            errors.append(f"{name} signature verification failed")

    expected_bill = canonical_params(vectors["seller_bill"]["params"])
    if expected_bill != vectors["seller_bill"]["canonical"]:
        errors.append("seller bill canonical content mismatch")

    decoded_needed = json.loads(base64url_decode(vectors["payment_needed"]["encoded"]))
    if decoded_needed != vectors["payment_needed"]["decoded"]:
        errors.append("Payment-Needed round trip failed")
    decoded_proof = json.loads(base64url_decode(vectors["payment_proof"]["encoded"]))
    if decoded_proof != vectors["payment_proof"]["decoded"]:
        errors.append("Payment-Proof round trip failed")

    negative = vectors["negative_cases"]
    if rsa_sha256_verify(public_key, negative["tampered_response"]["content"], negative["tampered_response"]["signature"]):
        errors.append("tampered response unexpectedly verified")
    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    generate = commands.add_parser("generate", help="write signed protocol vectors")
    generate.add_argument("--private-key", required=True, help="path to a test-only PKCS#1/PKCS#8 PEM key")
    generate.add_argument("--out", default="vectors.json")
    verify = commands.add_parser("verify", help="verify a vectors.json file")
    verify.add_argument("--vectors", default="vectors.json")
    commands.add_parser("selftest", help="run structural checks with no external dependencies")
    args = parser.parse_args(argv)

    if args.command == "generate":
        key = parse_private_key_pem(Path(args.private_key).read_text(encoding="utf-8"))
        vectors = default_vectors(key)
        Path(args.out).write_text(json.dumps(vectors, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        errors = validate_vectors(vectors)
    elif args.command == "verify":
        vectors = json.loads(Path(args.vectors).read_text(encoding="utf-8"))
        errors = validate_vectors(vectors)
    else:
        sample = canonical_params({"z": "last", "a": "first", "empty": "", "none": None})
        errors = [] if sample == "a=first&z=last" else ["canonical_params selftest failed"]
        payload = proof("p", "t")
        if json.loads(base64url_decode(payload["encoded"])) != payload["decoded"]:
            errors.append("base64url selftest failed")

    if errors:
        for error in errors:
            print(f"error: {error}", file=sys.stderr)
        return 1
    print("OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
