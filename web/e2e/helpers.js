// Shared test helpers — regex, assertions, utilities.
// Mocks go in mocks.js.

// i18n key pattern: if a raw key like "site.name" or "services.bazi.label" appears
// as visible text, the i18n lookup failed and leaked the key.
// i18n keys are always [a-z]+.[a-z]+ with optional third segment.
// Domain "liki.hk" and the standard filename "llms.txt" happen to match — exclude explicitly.
export const I18N_KEY_RE = /\b(?!liki\.hk\b|llms\.txt\b)[a-z]{2,}\.[a-z]{2,}(?:\.[a-z]+)?\b/;
