package mailboxsweep

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthenticated    = errors.New("context not authenticated")
	ErrNoStoredCredential = errors.New("no credential stored for inbox")
	ErrBadInboxCredential = errors.New("could not login to inbox with stored credential")
)
