// Package httputil provides shared HTTP utilities used by product packages.
//
// # Decode policy
//
// First-party request DTOs use DecodeStrict: unknown fields are rejected so
// client typos fail loudly. Third-party payloads (AG-UI input, provider
// webhooks, verbatim storage such as feedback submissions) use Decode or a
// manual body read — their senders do not sign up to our field list.
package httputil
