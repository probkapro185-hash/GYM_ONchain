package domain

import "errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrAlreadyExists        = errors.New("already exists")
	ErrInvalidInput         = errors.New("invalid input")
	ErrInvalidAccountLink   = errors.New("invalid or expired link")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrInvalidPhone         = errors.New("invalid phone number")
	ErrInvalidEmail         = errors.New("invalid email")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrNoActiveSubscription = errors.New("no active subscription")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrConflict             = errors.New("conflict")
	ErrServiceUnavailable   = errors.New("service unavailable")
)
