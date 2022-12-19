package classifyrequest

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"testing"
	"time"
)

func TestClassifyRequestService(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoRequestRepository(cl, table)
	userRepo := inboxuser.NewDynamoInboxUserRepository(cl, table)
	userService := inboxuser.NewService(userRepo)

	timeBetweenOpenRequests := time.Minute * 60
	timeBetweenClosedRequests := time.Minute * 60

	service := NewDefaultService(userService, repo, timeBetweenOpenRequests, timeBetweenClosedRequests)

	testAddr := "blamo@whamo.namo"
	_, err := userService.Store(ctx, testAddr)
	assert.ThatError(t, err).IsNil()

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddr, t)

	t.Run("record request", func(t *testing.T) {
		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.RecordRequest(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.RecordRequest(authedCtx)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("second attempt (too many open requests)", func(t *testing.T) {
			err := service.RecordRequest(authedCtx)
			assert.ThatError(t, err).IsNotNil()
		})
	})

	t.Run("list open requests", func(t *testing.T) {
		t.Run("unauthorized, failure", func(t *testing.T) {
			list, err := service.FindOpenRequests(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(list)).Equals(0)
		})

		t.Run("authorized, success", func(t *testing.T) {
			list, err := service.FindOpenRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
			l := list[0]
			assert.That(t, l.Inbox).Equals(testAddr)
			assert.That(t, l.CompletedAtUnix).Equals(nil)
			assert.That(t, l.Error).Equals(nil)
			assert.ThatOrdered(t, l.RequestedAt).GreaterThan(0)
		})
	})

	t.Run("mark complete - success", func(t *testing.T) {
		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.CompleteSuccess(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authorized, no open requests, falire", func(t *testing.T) {
			localCtx := mailboxsweep.AuthorizeTestContext(ctx, "me@example.com", t)
			err := service.CompleteSuccess(localCtx)
			assert.ThatError(t, err).Is(ErrNoOpenRequests)

			list, err := service.FindOpenRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.CompleteSuccess(authedCtx)
			assert.ThatError(t, err).IsNil()

			list, err := service.FindOpenRequests(ctx)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, len(list)).Equals(0)

			all, err := repo.List(ctx, testAddr, false, true)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(all)).Equals(1)
			l := all[0]
			assert.That(t, l.CompletedAtUnix).NotEqual(nil)
			assert.ThatOrdered(t, *l.CompletedAtUnix).GreaterThan(0)
			assert.That(t, l.Inbox).Equals(testAddr)
			assert.That(t, *l.Error).Equals("")
			assert.ThatOrdered(t, l.RequestedAt).GreaterThan(0)
		})
	})

	// store a new request
	testAddr2 := "ham@bone.com"
	_, err = userService.Store(ctx, testAddr2)
	assert.ThatError(t, err).IsNil()
	authedCtx2 := mailboxsweep.AuthorizeTestContext(ctx, testAddr2, t)
	err = service.RecordRequest(authedCtx2)
	assert.ThatError(t, err).IsNil()

	t.Run("mark complete - failure", func(t *testing.T) {
		testErr := errors.New("something went wrong")
		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.CompleteError(ctx, testErr)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.CompleteError(authedCtx2, testErr)
			assert.ThatError(t, err).IsNil()

		})
	})
}
