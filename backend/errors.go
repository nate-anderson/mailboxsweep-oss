package mailboxsweep

import (
	"fmt"
	"net/http"
)

// Sentinel errors
// The error constructors below wrap these
// These should be used for comparisons or wrapped, not returned directly

type StatusError interface {
	StatusCode() int
}

type ApplicationError struct {
	status int
	msg    string
}

func (e ApplicationError) Error() string {
	return e.msg
}

func (e ApplicationError) StatusCode() int {
	return e.status
}

var (
	ErrNotFound = ApplicationError{
		status: http.StatusNotFound,
		msg:    "not found",
	}

	ErrUnauthenticated = ApplicationError{
		status: http.StatusUnauthorized,
		msg:    "unauthenticated",
	}

	ErrNoStoredCredential = ApplicationError{
		status: http.StatusNotFound,
		msg:    "no credential stored for inbox",
	}

	ErrBadInboxCredential = ApplicationError{
		status: http.StatusNotFound,
		msg:    "could not login to inbox with stored credential",
	}

	ErrNeedsPayment = ApplicationError{
		status: http.StatusPaymentRequired,
		msg:    "payment required",
	}

	ErrAlreadyPaid = ApplicationError{
		status: http.StatusUnprocessableEntity,
		msg:    "account is already paid",
	}

	ErrPlanLimitMet = ApplicationError{
		status: http.StatusTooManyRequests,
		msg:    "plan limits met",
	}

	ErrBadGateway = ApplicationError{
		status: http.StatusBadGateway,
		msg:    "bad gateway",
	}

	ErrInsufficientPermissions = ApplicationError{
		status: http.StatusForbidden,
		msg:    "insufficient permissions",
	}
)

/*
The application should use the wrapping functions in this file to inform the upper
layers (service and API) about the nature of the returned error.

These wrapping functions wrap the sentinel errors above so that `errors.Is` can be used
in the service layer

The `StatusError` interface is implemented by all error types here so the API
layer can determine the correct status code. The default is 500 for errors
not implementing StatusError
*/

func NotFoundError(parent error) error {
	return fmt.Errorf("%w: %w", ErrNotFound, parent)
}

func UnauthenticatedError(parent error) error {
	return fmt.Errorf("%w: %w", ErrUnauthenticated, parent)
}

func NoStoredCredentialError(parent error) error {
	return fmt.Errorf("%w: %w", ErrNoStoredCredential, parent)
}

func BadInboxCredentialError(parent error) error {
	return fmt.Errorf("%w: %w", ErrBadInboxCredential, parent)
}

func NeedsPaymentError(parent error) error {
	return fmt.Errorf("%w: %w", ErrNeedsPayment, parent)
}

func PlanLimitMetError(parent error) error {
	return fmt.Errorf("%w: %w", ErrPlanLimitMet, parent)
}

func BadGatewayError(parent error) error {
	return fmt.Errorf("%w: %w", ErrBadGateway, parent)
}

func InsufficientPermissionsError(parent error) error {
	return fmt.Errorf("%w: %w", ErrInsufficientPermissions, parent)
}
