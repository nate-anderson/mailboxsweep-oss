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
	userService, err := inboxuser.NewService(userRepo)
	assert.ThatError(t, err).IsNil()

	timeBetweenOpenRequests := time.Minute * 60
	timeBetweenClosedRequests := time.Minute * 60

	service := NewDefaultService(userService, repo, timeBetweenOpenRequests, timeBetweenClosedRequests)

	testAddr := "blamo@whamo.namo"
	_, err = userService.Store(ctx, testAddr)
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
			err := service.MarkCompletedSuccess(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authorized, no open requests, falure", func(t *testing.T) {
			localCtx := mailboxsweep.AuthorizeTestContext(ctx, "me@example.com", t)
			err := service.MarkCompletedSuccess(localCtx)
			assert.ThatError(t, err).Is(ErrNoOpenRequests)

			list, err := service.FindOpenRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.MarkCompletedSuccess(authedCtx)
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
			assert.That(t, l.Error).Equals(nil)
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
			err := service.MarkCompletedError(ctx, testErr)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)

			list, err := service.FindOpenRequests(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.MarkCompletedError(authedCtx2, testErr)
			assert.ThatError(t, err).IsNil()

			list, err := service.FindOpenRequests(authedCtx2)
			assert.ThatError(t, err).Is(ErrNoOpenRequests)
			assert.That(t, len(list)).Equals(0)
		})
	})

	testAddr3 := "nobody@nowhere.com"
	count := 30
	operation := "DELETE"
	authedCtx3 := mailboxsweep.AuthorizeAPIContext(ctx, testAddr3)

	t.Run("mark executing", func(t *testing.T) {
		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.MarkExecuting(ctx, operation)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		// store one complete
		_, err = userService.Store(ctx, testAddr3)
		assert.ThatError(t, err).IsNil()
		err := service.RecordRequest(authedCtx3)
		assert.ThatError(t, err).IsNil()
		err = service.MarkCompletedSuccess(authedCtx3)
		assert.ThatError(t, err).IsNil()

		t.Run("success", func(t *testing.T) {
			err := service.MarkExecuting(authedCtx3, operation)
			assert.ThatError(t, err).IsNil()
		})

	})

	t.Run("mark executed - success", func(t *testing.T) {

		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.MarkExecutedSuccess(ctx, 20)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authorized, no executing requests, failure", func(t *testing.T) {
			err := service.MarkExecutedSuccess(authedCtx, count)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatString(t, err.Error()).Contains("no completed requests found")
		})

		// mark it complete
		// err = service.MarkCompletedSuccess(authedCtx3)
		// assert.ThatError(t, err).IsNil()

		t.Run("success", func(t *testing.T) {
			err = service.MarkExecutedSuccess(authedCtx3, count)
			assert.ThatError(t, err).IsNil()

			retrieved, err := service.FindCompletedRequests(authedCtx3)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(retrieved)).Equals(1)
			assert.That(t, retrieved[0].ExecutedAtUnix).NotEqual(nil)
			assert.That(t, *retrieved[0].ExecutedAtUnix).NotEqual(0)
			assert.That(t, *retrieved[0].Count).Equals(count)
			assert.That(t, *retrieved[0].Operation).Equals(operation)
		})
	})

	t.Run("mark executed - error", func(t *testing.T) {
		testErr := errors.New("bungled")
		count := 35
		operation := "MARK_READ"

		t.Run("unauthorized, failure", func(t *testing.T) {
			err := service.MarkExecutedError(ctx, count, testErr)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		// store one incomplete and mark executing
		testAddr2 := "nobody2@nowhere.com"
		_, err = userService.Store(ctx, testAddr2)
		assert.ThatError(t, err).IsNil()

		authedCtx2 := mailboxsweep.AuthorizeAPIContext(ctx, testAddr2)
		err := service.RecordRequest(authedCtx2)
		assert.ThatError(t, err).IsNil()
		err = service.MarkCompletedSuccess(authedCtx2)
		assert.ThatError(t, err).IsNil()

		t.Run("authorized, no completed requests, failure", func(t *testing.T) {
			err := service.MarkExecutedError(authedCtx2, count, testErr)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatString(t, err.Error()).Contains("no completed requests found")
		})

		// mark it executing
		err = service.MarkExecuting(authedCtx2, operation)
		assert.ThatError(t, err).IsNil()

		t.Run("success", func(t *testing.T) {
			err = service.MarkExecutedError(authedCtx2, count, testErr)
			assert.ThatError(t, err).IsNil()

			retrieved, err := service.FindCompletedRequests(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(retrieved)).Equals(1)
			assert.That(t, retrieved[0].ExecutedAtUnix).NotEqual(nil)
			assert.That(t, *retrieved[0].ExecutedAtUnix).NotEqual(0)
			assert.ThatString(t, *retrieved[0].Error).Contains(testErr.Error())
			assert.That(t, *retrieved[0].Count).Equals(count)
			assert.That(t, *retrieved[0].Operation).Equals(operation)
		})
	})
}
