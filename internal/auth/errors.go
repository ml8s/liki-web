package auth

import "errors"

var (
	ErrInvalidEmail      = errors.New("auth: invalid email")
	ErrEmailUnavailable  = errors.New("auth: email unavailable")
	ErrCodeCooldown      = errors.New("auth: code cooldown")
	ErrCodeInvalid       = errors.New("auth: invalid code")
	ErrCodeExpired       = errors.New("auth: code expired")
	ErrCodeLocked        = errors.New("auth: code locked")
	ErrUserNotFound      = errors.New("auth: user not found")
	ErrUserDisabled      = errors.New("auth: user disabled")
	ErrSessionNotFound   = errors.New("auth: session not found")
	ErrSessionExpired    = errors.New("auth: session expired")
	ErrIdentityNotFound  = errors.New("auth: identity not found")
	ErrIdentityForbidden = errors.New("auth: identity forbidden")
	ErrMailerUnavailable = errors.New("auth: mailer unavailable")
)
