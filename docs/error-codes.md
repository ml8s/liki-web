# Error Codes

<!-- 本文件由 web/scripts/check-error-codes.js 生成：请勿手改。改动错误码后重跑 `node web/scripts/check-error-codes.js --update`。 -->

每个 Go 错误码都必须在 App 三语 `errors.code.*` 中有文案（检查器 `make check-web-static` 强制）。
状态类端点只读且响应不含 PII；`payment-status` 能力 token 与 intent id 属 capability-URL 凭证（见 ARCHITECTURE.md）。

| Code | 定义处 |
| --- | --- |
| `admin_already_exists` | `internal/adminauth/http.go` |
| `admin_not_found` | `internal/adminauth/http.go` |
| `aipay_disabled` | `internal/aipay/handler.go` |
| `already_kol` | `internal/partner/admin_handler.go` |
| `already_scout` | `internal/partner/admin_handler.go` |
| `application_not_found` | `internal/partner/admin_handler.go` |
| `application_not_pending` | `internal/partner/admin_handler.go` |
| `application_pending` | `internal/partner/handler.go` |
| `checkout_failed` | `internal/sales/checkout_http.go` |
| `code_cooldown` | `internal/auth/http.go` |
| `code_expired` | `internal/auth/http.go` |
| `code_locked` | `internal/auth/http.go` |
| `entitlement_already_revoked` | `internal/admin/handler.go` |
| `entitlement_not_found` | `internal/admin/handler.go` |
| `entitlement_revoked` | `internal/admin/handler.go` |
| `finance_task_invalid_transition` | `internal/partner/admin_handler.go` |
| `finance_task_not_found` | `internal/partner/admin_handler.go` |
| `forbidden` | `internal/admin/route.go` |
| `fulfillment_confirm_failed` | `internal/aipay/handler.go` |
| `internal_error` | `internal/admin/handler.go` |
| `invalid_billing_mode` | `internal/sales/checkout_http.go` |
| `invalid_capability` | `internal/adminauth/http.go` |
| `invalid_code` | `internal/auth/http.go` |
| `invalid_consultation_context` | `internal/chat/http.go` |
| `invalid_email` | `internal/auth/http.go` |
| `invalid_request` | `internal/admin/handler.go` |
| `last_super_admin` | `internal/adminauth/http.go` |
| `mailer_unavailable` | `internal/auth/http.go` |
| `mixed_currency` | `internal/partner/role_handlers.go` |
| `no_withdrawable_commission` | `internal/partner/role_handlers.go` |
| `not_found` | `internal/httputil/errors.go` |
| `order_already_paid` | `internal/sales/order_http.go` |
| `order_expired` | `internal/sales/order_http.go` |
| `order_failed` | `internal/sales/checkout_http.go` |
| `order_not_pending` | `internal/sales/order_http.go` |
| `order_refunded` | `internal/sales/order_http.go` |
| `partner_financial_hold` | `internal/account/account.go` |
| `partner_not_found` | `internal/partner/admin_handler.go` |
| `payout_invalid_transition` | `internal/partner/admin_handler.go` |
| `payout_minimum_not_met` | `internal/partner/role_handlers.go` |
| `payout_not_found` | `internal/partner/admin_handler.go` |
| `product_not_found` | `internal/sales/checkout_http.go` |
| `profile_conflict` | `internal/birthprofile/http.go` |
| `quota_exceeded` | `internal/birthprofile/http.go` |
| `rate_limited` | `internal/http/middleware.go` |
| `role_conflict` | `internal/partner/admin_handler.go` |
| `role_required` | `internal/partner/role_handlers.go` |
| `sales_unavailable` | `internal/admin/orders.go` |
| `self_management_forbidden` | `internal/adminauth/http.go` |
| `skill_forbidden` | `internal/chat/http.go` |
| `subscription_conflict` | `internal/sales/subscription_http.go` |
| `subscription_failed` | `internal/sales/subscription_http.go` |
| `subscription_unavailable` | `internal/sales/subscription_http.go` |
| `subscription_update_failed` | `internal/sales/subscription_http.go` |
| `super_requires_super` | `internal/adminauth/http.go` |
| `too_large` | `internal/httputil/json.go` |
| `unauthorized` | `internal/httputil/errors.go` |
| `unsupported_media_type` | `internal/httputil/json.go` |
| `user_disabled` | `internal/auth/http.go` |
| `user_not_active` | `internal/admin/support.go` |
| `user_not_found` | `internal/admin/support.go` |
| `validation_error` | `internal/adminauth/http.go` |
| `webhook_failed` | `internal/payment/http.go` |
