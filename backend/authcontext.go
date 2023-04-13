package mailboxsweep

import (
	"context"
	"errors"
	"testing"
)

// We use context to authenticate actions at the service layer

type ContextKeyT int

const (
	UserKey ContextKeyT = iota + 1
	EnvironmentKey
)

const (
	EnvironmentAPI string = "api"
	EnvironmentJob string = "job"
)

func UserAddressFromContext(ctx context.Context) (string, error) {
	userAddress := ctx.Value(UserKey)
	if str, ok := userAddress.(string); ok {
		return str, nil
	}
	return "", UnauthenticatedError(errors.New("no session associated with request context"))
}

func AuthorizeAPIContext(ctx context.Context, addr string) context.Context {
	authed := context.WithValue(ctx, UserKey, addr)
	labeled := context.WithValue(authed, EnvironmentKey, EnvironmentAPI)
	return labeled
}

func AuthorizeJobContext(ctx context.Context, addr string) context.Context {
	authed := context.WithValue(ctx, UserKey, addr)
	labeled := context.WithValue(authed, EnvironmentKey, EnvironmentJob)
	return labeled
}

func AuthorizeTestContext(ctx context.Context, addr string, t *testing.T) context.Context {
	if t.Name() == "" {
		panic("AuthorizeTestContext called on unnamed test")
	}
	authed := context.WithValue(ctx, UserKey, addr)
	return authed
}
