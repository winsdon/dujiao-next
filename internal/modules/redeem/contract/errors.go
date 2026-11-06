package contract

import "errors"

var (
	ErrDisabled          = errors.New("redeem disabled")
	ErrMaintenance       = errors.New("redeem maintenance")
	ErrInvalidCode       = errors.New("invalid redeem code")
	ErrCodeNotFound      = errors.New("redeem code not found")
	ErrAlreadySucceeded  = errors.New("redeem already succeeded")
	ErrInCooldown        = errors.New("redeem in cooldown")
	ErrBusy              = errors.New("redeem busy")
	ErrPendingConfirm    = errors.New("redeem pending confirm")
	ErrInvalidContact    = errors.New("invalid contact")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrNotAuthorized     = errors.New("not authorized")
	ErrMapMissing        = errors.New("product map missing")
	ErrNotFound          = errors.New("redeem order not found")
)
