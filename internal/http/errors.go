package http

// Error codes used locally in the http package.
// Reuses httputil.ErrCodeInvalidRequest and httputil.ErrCodeForbidden from httputil.
const (
	ErrCodeValidationError = "validation_error"
	ErrCodeTooLarge        = "too_large"
)
