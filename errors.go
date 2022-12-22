package mailboxsweep

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors
// The error constructors below wrap these
// These should be used for comparisons, not returned directly
var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthenticated    = errors.New("context not authenticated")
	ErrNoStoredCredential = errors.New("no credential stored for inbox")
	ErrBadInboxCredential = errors.New("could not login to inbox with stored credential")
	ErrNeedsPayment       = errors.New("payment required")
	ErrAlreadyPaid        = errors.New("account is already paid")
)

type StatusError interface {
	error
	StatusCode() int
	Unwrap() error
}

/*
The application should use the wrapping functions in this file to inform the upper
layers (service and API) about the nature of the returned error.

These wrapping functions wrap the sentinel errors above so that `errors.Is` can be used
in the service layer

The `StatusError` interface is implemented by all error types here so the API
layer can determine the correct status code. The default is 500 for errors
not implementing StatusError
*/

func join(parent error, sentinel error) error {
	if parent == nil {
		return sentinel
	}
	return fmt.Errorf("%s: %w", parent.Error(), sentinel)
}

func NotFoundError(parent error) error {
	return notFoundError{
		parent: join(parent, ErrNotFound),
	}
}

func UnauthenticatedError(parent error) error {
	return unauthenticatedError{
		parent: join(parent, ErrUnauthenticated),
	}
}

func NoStoredCredentialError(parent error) error {
	return credentialError{
		parent: join(parent, ErrNoStoredCredential),
	}
}

func BadInboxCredentialError(parent error) error {
	return credentialError{
		parent: join(parent, ErrBadInboxCredential),
	}
}

func NeedsPaymentError(parent error) error {
	return paymentError{
		parent: join(parent, ErrNeedsPayment),
	}
}

// Something wasn't found
type notFoundError struct {
	parent error
}

func (n notFoundError) Error() string {
	return fmt.Sprintf("not found: %s", n.parent)
}

func (n notFoundError) StatusCode() int {
	return http.StatusNotFound
}

func (n notFoundError) Unwrap() error {
	return n.parent
}

type unauthenticatedError struct {
	parent error
}

func (u unauthenticatedError) Error() string {
	return "unauthenticated"
}

func (u unauthenticatedError) StatusCode() int {
	return http.StatusUnauthorized
}

func (u unauthenticatedError) Unwrap() error {
	return u.parent
}

type credentialError struct {
	parent error
}

func (c credentialError) Error() string {
	return fmt.Sprintf("credential error: %s", c.parent.Error())
}

func (c credentialError) StatusCode() int {
	return http.StatusNotFound
}

func (c credentialError) Unwrap() error {
	return c.parent
}

type paymentError struct {
	parent error
}

func (p paymentError) Error() string {
	return ErrNeedsPayment.Error()
}

func (p paymentError) StatusCode() int {
	return http.StatusPaymentRequired
}

func (p paymentError) Unwrap() error {
	return p.parent
}
