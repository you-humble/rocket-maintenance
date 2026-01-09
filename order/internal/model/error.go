package model

import "errors"

var (
	ErrValidation         = errors.New("validation error")    // 400
	ErrOrderNotFound      = errors.New("order not found")     // 404
	ErrOrderConflict      = errors.New("order conflict")      // 409
	ErrRateLimited        = errors.New("rate limited")        // 429
	ErrBadGateway         = errors.New("bad gateway")         // 502
	ErrServiceUnavailable = errors.New("service unavailable") // 503
	ErrUnauthorized       = errors.New("unauthorized user")
	ErrForbidden          = errors.New("forbidden")
	ErrPartsOutOfStock    = errors.New("parts out of stock")
	ErrUnknownStatus      = errors.New("unknown status")
	ErrPartNotFound       = errors.New("part not found")
)
