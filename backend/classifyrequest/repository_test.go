package classifyrequest

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
)

func TestClassifyRequestRepository(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoRequestRepository(cl, table)
	testAddress := "turkey@bagone.net"
	id := uuid.NewString()

	t.Run("store", func(t *testing.T) {
		req := ClassifyRequest{
			Inbox:       testAddress,
			UserUUID:    id,
			RequestedAt: time.Now().Unix(),
		}
		err := repo.Store(ctx, req)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("list", func(t *testing.T) {
		t.Run("find one", func(t *testing.T) {
			list, err := repo.List(ctx, testAddress, false, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list) == 1).Equals(true)
			assert.That(t, list[0].Inbox).Equals(testAddress)
			assert.That(t, list[0].Error).Equals(nil)
			assert.That(t, list[0].CompletedAtUnix).Equals(nil)
			assert.That(t, list[0].UserUUID).Equals(id)
			assert.ThatOrdered(t, list[0].RequestedAt).GreaterThan(0)
		})

		n := 100
		completed := time.Now().Unix()
		for i := 0; i < n; i++ {
			req := ClassifyRequest{
				Inbox:           testAddress,
				UserUUID:        id,
				RequestedAt:     int64(i) + 10000000,
				CompletedAtUnix: aws.Int64(completed),
				Error:           aws.String("an error"),
			}
			err := repo.Store(ctx, req)
			assert.ThatError(t, err).IsNil()
		}

		t.Run("find many", func(t *testing.T) {
			list, err := repo.List(ctx, testAddress, false, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(n + 1)
			incomplete := 0
			nonError := 0
			for _, r := range list {
				assert.That(t, r.Inbox).Equals(testAddress)
				if r.Error != nil {
					assert.That(t, *r.Error).Equals("an error")
				} else {
					nonError++
				}
				assert.ThatOrdered(t, r.RequestedAt).GreaterThan(0)
				assert.That(t, r.UserUUID).Equals(id)
				if r.CompletedAtUnix != nil {
					assert.That(t, *r.CompletedAtUnix).Equals(completed)
				} else {
					incomplete++
				}
			}
			assert.That(t, incomplete).Equals(1)
			assert.That(t, nonError).Equals(1)
		})

		t.Run("mark completed", func(t *testing.T) {
			var requestedAt int64 = 10000000

			t.Run("successful outcome", func(t *testing.T) {
				updated, err := repo.MarkCompleted(ctx, testAddress, requestedAt, nil)
				assert.ThatError(t, err).IsNil()
				assert.That(t, updated.Inbox).Equals(testAddress)
				assert.ThatOrdered(t, *updated.CompletedAtUnix).GreaterThan(0)
				assert.That(t, updated.UserUUID).Equals(id)
				assert.That(t, updated.Error).Equals(nil)

				var retrieved ClassifyRequest
				list, err := repo.List(ctx, testAddress, false, false)
				assert.ThatError(t, err).IsNil()
				for _, r := range list {
					if r.RequestedAt == requestedAt {
						retrieved = r
					}
				}
				assert.That(t, retrieved.Inbox).SoThat("found updated record").Equals(testAddress)
				assert.ThatError(t, err).IsNil()
				assert.That(t, updated.Inbox).Equals(testAddress)
				assert.ThatOrdered(t, *updated.CompletedAtUnix).GreaterThan(0)
				assert.That(t, updated.UserUUID).Equals(id)
				assert.That(t, updated.Error).Equals(nil)
			})

			t.Run("failed outcome", func(t *testing.T) {
				errMsg := "a big error"
				updated, err := repo.MarkCompleted(ctx, testAddress, requestedAt, &errMsg)
				assert.ThatError(t, err).IsNil()
				assert.That(t, updated.Inbox).Equals(testAddress)
				assert.ThatOrdered(t, *updated.CompletedAtUnix).GreaterThan(0)
				assert.That(t, updated.UserUUID).Equals(id)
				assert.That(t, *updated.Error).Equals(errMsg)

				var retrieved ClassifyRequest
				list, err := repo.List(ctx, testAddress, false, false)
				assert.ThatError(t, err).IsNil()
				for _, r := range list {
					if r.RequestedAt == requestedAt {
						retrieved = r
					}
				}
				assert.That(t, retrieved.Inbox).SoThat("found updated record").Equals(testAddress)
				assert.ThatError(t, err).IsNil()
				assert.That(t, updated.Inbox).Equals(testAddress)
				assert.ThatOrdered(t, *updated.CompletedAtUnix).GreaterThan(0)
				assert.That(t, updated.UserUUID).Equals(id)
				assert.That(t, *updated.Error).Equals(errMsg)
			})
		})

		t.Run("mark executed", func(t *testing.T) {
			requestedAt := 10000001

			var (
				successCount = 10
				failureCount = 20
			)

			t.Run("success", func(t *testing.T) {
				req, err := repo.MarkExecuted(ctx, testAddress, int64(requestedAt), successCount, nil)
				assert.ThatError(t, err).IsNil()
				assert.That(t, req.ExecutedAtUnix).NotEqual(nil)
				assert.That(t, *req.ExecutedAtUnix).NotEqual(0)
				assert.That(t, req.RequestedAt).Equals(int64(requestedAt))
				assert.That(t, req.Inbox).Equals(testAddress)
				assert.That(t, req.Error).Equals(nil)
				assert.That(t, *req.Count).Equals(successCount)
			})

			t.Run("error", func(t *testing.T) {
				errDetails := "something horrible happened"
				req, err := repo.MarkExecuted(ctx, testAddress, int64(requestedAt), failureCount, &errDetails)
				assert.ThatError(t, err).IsNil()
				assert.That(t, req.ExecutedAtUnix).NotEqual(nil)
				assert.That(t, *req.ExecutedAtUnix).NotEqual(0)
				assert.That(t, req.RequestedAt).Equals(int64(requestedAt))
				assert.That(t, req.Inbox).Equals(testAddress)
				assert.That(t, *req.Error).Equals(errDetails)
				assert.That(t, *req.Count).Equals(failureCount)
			})
		})
	})
}
