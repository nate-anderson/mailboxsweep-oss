package classifyrequest

import (
	"context"
	"errors"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestClassifyRequestService(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoRequestRepository(cl, table)
	userRepo := inboxuser.NewDynamoInboxUserRepository(cl, table)
	userService, err := inboxuser.NewService(userRepo, logrus.New())
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
			list, err := service.findOpenRequests(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(list)).Equals(0)
		})

		t.Run("authorized, success", func(t *testing.T) {
			list, err := service.findOpenRequests(authedCtx)
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
			assert.ThatError(t, err).Is(ErrNoClassifyingRequests)

			list, err := service.findOpenRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.MarkCompletedSuccess(authedCtx)
			assert.ThatError(t, err).IsNil()

			list, err := service.findOpenRequests(ctx)
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

			list, err := service.findOpenRequests(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(1)
		})

		t.Run("authorized, success", func(t *testing.T) {
			err := service.MarkCompletedError(authedCtx2, testErr)
			assert.ThatError(t, err).IsNil()

			list, err := service.findOpenRequests(authedCtx2)
			assert.ThatError(t, err).Is(ErrNoClassifyingRequests)
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
			assert.ThatError(t, err).Is(ErrNoExecutingRequests)
		})

		t.Run("success", func(t *testing.T) {
			err = service.MarkExecutedSuccess(authedCtx3, count)
			assert.ThatError(t, err).IsNil()

			retrieved, err := service.FindClassifiedRequests(authedCtx3)
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
			assert.ThatError(t, err).Is(ErrNoExecutingRequests)
		})

		// mark it executing
		err = service.MarkExecuting(authedCtx2, operation)
		assert.ThatError(t, err).IsNil()

		t.Run("success", func(t *testing.T) {
			err = service.MarkExecutedError(authedCtx2, count, testErr)
			assert.ThatError(t, err).IsNil()

			retrieved, err := service.FindClassifiedRequests(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(retrieved)).Equals(1)
			assert.That(t, retrieved[0].ExecutedAtUnix).NotEqual(nil)
			assert.That(t, *retrieved[0].ExecutedAtUnix).NotEqual(0)
			assert.ThatString(t, *retrieved[0].Error).Contains(testErr.Error())
			assert.That(t, *retrieved[0].Count).Equals(count)
			assert.That(t, *retrieved[0].Operation).Equals(operation)
		})
	})

	type filterClassifyRequestTestCase struct {
		collection []ClassifyRequest
		expected   ClassifyRequest
		expStatus  ClassifyRequestStatus
	}

	t.Run("FindLatestIncompleteRequest", func(t *testing.T) {
		baseTs := int64(1695793829)

		testCases := map[string]filterClassifyRequestTestCase{
			"a single classifying request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:       "testuser001@mailboxsweep.test",
						RequestedAt: baseTs,
					},
				},
				expected: ClassifyRequest{
					Inbox:       "testuser001@mailboxsweep.test",
					RequestedAt: baseTs,
				},
				expStatus: StatusClassifying,
			},

			"a single classified request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:           "testuser002@mailboxsweep.test",
						RequestedAt:     baseTs,
						CompletedAtUnix: ptr(baseTs + 10),
					},
				},
				expected: ClassifyRequest{
					Inbox:           "testuser002@mailboxsweep.test",
					RequestedAt:     baseTs,
					CompletedAtUnix: ptr(baseTs + 10),
				},
				expStatus: StatusClassified,
			},

			"a single executing request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:                "testuser003@mailboxsweep.test",
						RequestedAt:          baseTs,
						CompletedAtUnix:      ptr(baseTs + 10),
						ExecuteStartedAtUnix: ptr(baseTs + 20),
					},
				},
				expected: ClassifyRequest{
					Inbox:                "testuser003@mailboxsweep.test",
					RequestedAt:          baseTs,
					CompletedAtUnix:      ptr(baseTs + 10),
					ExecuteStartedAtUnix: ptr(baseTs + 20),
				},
				expStatus: StatusExecuting,
			},

			"the most progressed non-error request is returned, regardless of timestamp": {
				collection: []ClassifyRequest{
					{
						Inbox:       "testuser004@mailboxsweep.test",
						RequestedAt: baseTs + 100,
					},
					{
						Inbox:           "testuser004@mailboxsweep.test",
						RequestedAt:     baseTs + 10,
						CompletedAtUnix: ptr(baseTs + 20),
					},
					{
						Inbox:                "testuser004@mailboxsweep.test",
						RequestedAt:          baseTs + 20,
						CompletedAtUnix:      ptr(baseTs + 30),
						ExecuteStartedAtUnix: ptr(baseTs + 40),
					},
					{
						Inbox:                "testuser004@mailboxsweep.test",
						RequestedAt:          baseTs + 30,
						CompletedAtUnix:      ptr(baseTs + 40),
						ExecuteStartedAtUnix: ptr(baseTs + 50),
						ExecutedAtUnix:       ptr(baseTs + 60),
					},
					{
						Inbox:                "testuser004@mailboxsweep.test",
						RequestedAt:          baseTs + 40,
						CompletedAtUnix:      ptr(baseTs + 50),
						ExecuteStartedAtUnix: ptr(baseTs + 60),
						ExecutedAtUnix:       ptr(baseTs + 70),
						Error:                ptr("something went wrong!"),
					},
				},
				expected: ClassifyRequest{
					Inbox:                "testuser004@mailboxsweep.test",
					RequestedAt:          baseTs + 20,
					CompletedAtUnix:      ptr(baseTs + 30),
					ExecuteStartedAtUnix: ptr(baseTs + 40),
				},
				expStatus: StatusExecuting,
			},
		}

		for desc, tc := range testCases {
			t.Run(desc, func(t *testing.T) {
				// create user
				_, err := userService.Store(ctx, tc.expected.Inbox)
				assert.ThatError(t, err).IsNil()

				// store request(s)
				for _, req := range tc.collection {
					err = repo.Store(ctx, req)
					assert.ThatError(t, err).IsNil()
				}

				authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, tc.expected.Inbox)
				current, err := service.FindLatestIncompleteRequest(authedCtx)
				assert.ThatError(t, err).IsNil()
				assert.That(t, current.Equals(tc.expected)).Equals(true)
				status, _ := current.Status()
				assert.That(t, status).Equals(tc.expStatus)
			})
		}
	})

	t.Run("FindLatestRequest", func(t *testing.T) {
		baseTs := int64(1695793829)

		testCases := map[string]filterClassifyRequestTestCase{
			"a single classifying request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:       "testuser011@mailboxsweep.test",
						RequestedAt: baseTs,
					},
				},
				expected: ClassifyRequest{
					Inbox:       "testuser011@mailboxsweep.test",
					RequestedAt: baseTs,
				},
				expStatus: StatusClassifying,
			},

			"a single classified request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:           "testuser012@mailboxsweep.test",
						RequestedAt:     baseTs,
						CompletedAtUnix: ptr(baseTs + 10),
					},
				},
				expected: ClassifyRequest{
					Inbox:           "testuser012@mailboxsweep.test",
					RequestedAt:     baseTs,
					CompletedAtUnix: ptr(baseTs + 10),
				},
				expStatus: StatusClassified,
			},

			"a single executing request is returned": {
				collection: []ClassifyRequest{
					{
						Inbox:                "testuser013@mailboxsweep.test",
						RequestedAt:          baseTs,
						CompletedAtUnix:      ptr(baseTs + 10),
						ExecuteStartedAtUnix: ptr(baseTs + 20),
					},
				},
				expected: ClassifyRequest{
					Inbox:                "testuser013@mailboxsweep.test",
					RequestedAt:          baseTs,
					CompletedAtUnix:      ptr(baseTs + 10),
					ExecuteStartedAtUnix: ptr(baseTs + 20),
				},
				expStatus: StatusExecuting,
			},

			"the most recent request is returned, regardless of status": {
				collection: []ClassifyRequest{
					{
						Inbox:       "testuser014@mailboxsweep.test",
						RequestedAt: baseTs + 100,
					},
					{
						Inbox:           "testuser014@mailboxsweep.test",
						RequestedAt:     baseTs + 10,
						CompletedAtUnix: ptr(baseTs + 20),
					},
					{
						Inbox:                "testuser014@mailboxsweep.test",
						RequestedAt:          baseTs + 20,
						CompletedAtUnix:      ptr(baseTs + 30),
						ExecuteStartedAtUnix: ptr(baseTs + 40),
					},
					{
						Inbox:                "testuser014@mailboxsweep.test",
						RequestedAt:          baseTs + 30,
						CompletedAtUnix:      ptr(baseTs + 40),
						ExecuteStartedAtUnix: ptr(baseTs + 50),
						ExecutedAtUnix:       ptr(baseTs + 60),
					},
					{
						Inbox:                "testuser014@mailboxsweep.test",
						RequestedAt:          baseTs + 40,
						CompletedAtUnix:      ptr(baseTs + 50),
						ExecuteStartedAtUnix: ptr(baseTs + 60),
						ExecutedAtUnix:       ptr(baseTs + 70),
						Error:                ptr("something went wrong!"),
					},
				},
				expected: ClassifyRequest{
					Inbox:       "testuser014@mailboxsweep.test",
					RequestedAt: baseTs + 100,
				},
				expStatus: StatusClassifying,
			},
		}

		for desc, tc := range testCases {
			t.Run(desc, func(t *testing.T) {
				// create user
				_, err := userService.Store(ctx, tc.expected.Inbox)
				assert.ThatError(t, err).IsNil()

				// store request(s)
				for _, req := range tc.collection {
					err = repo.Store(ctx, req)
					assert.ThatError(t, err).IsNil()
				}

				authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, tc.expected.Inbox)
				current, err := service.FindLatestRequest(authedCtx)
				assert.ThatError(t, err).IsNil()
				eq := current.Equals(tc.expected)
				assert.That(t, eq).Equals(true)
				status, _ := current.Status()
				assert.That(t, status).Equals(tc.expStatus)
			})
		}
	})

	t.Run("GetChangesetSums", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			summary, err := service.SummarizeRequests(authedCtx3)
			assert.ThatError(t, err).IsNil()

			assert.That(t, summary.DeletedCount).Equals(30)
			assert.That(t, summary.MarkedReadCount).Equals(0)
			assert.That(t, summary.ExecutedRequestCount).Equals(1)
		})

		t.Run("failure, unauthenticated", func(t *testing.T) {
			summary, err := service.SummarizeRequests(ctx)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, summary).Equals(nil)
		})

		t.Run("no diff", func(t *testing.T) {
			summary, err := service.SummarizeRequests(authedCtx2)
			assert.ThatError(t, err).IsNil()
			assert.That(t, summary.DeletedCount).Equals(0)
			assert.That(t, summary.MarkedReadCount).Equals(0)
			assert.That(t, summary.ExecutedRequestCount).Equals(0)
		})
	})
}

func ptr[T any](val T) *T {
	return &val
}
