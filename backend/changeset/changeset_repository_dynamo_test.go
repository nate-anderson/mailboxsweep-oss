package changeset

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"strconv"
	"testing"
)

func TestDynamoChangesetRepository(t *testing.T) {
	cl := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo, err := NewDynamoChangesetRepository(cl, table, 64)
	assert.ThatError(t, err).IsNil()

	n := 300
	upsertPaths := []string{}
	appendUID := "1"
	for i := 2; i <= n; i++ {
		istr := strconv.Itoa(i)
		upsertPaths = append(upsertPaths, istr)
	}

	// hacky way to keep tests from colliding
	inbox := fmt.Sprintf("%T@example.org", *repo)
	t.Run("append to change set", func(t *testing.T) {
		t.Run("insert new", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:     inbox,
				Paths:     upsertPaths,
				Operation: Delete,
			}
			n, err := repo.Append(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(len(upsertPaths))
		})

		t.Run("append to existing page", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:     inbox,
				Paths:     []string{appendUID},
				Operation: Delete,
			}
			n, err := repo.Append(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(1)

			retrieved, err := repo.FindByInboxAndOperation(ctx, inbox, Delete, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, retrieved != nil)
			assert.ThatSlice(t, retrieved.Paths).Contains(appendUID)
			assert.That(t, retrieved.Operation).Equals(Delete)
		})
	})

	// seed
	_, err = repo.Append(ctx, ChangeSet{
		Inbox:     inbox,
		Paths:     []string{"markread"},
		Operation: MarkRead,
	})
	assert.ThatError(t, err).IsNil()

	t.Run("list for mailbox", func(t *testing.T) {
		set, err := repo.FindByInboxAndOperation(ctx, inbox, Delete, false)
		assert.
			ThatError(t, err).
			IsNil()

		assert.
			That(t, set).
			NotEqual(nil)

		assert.That(t, set.Inbox).Equals(inbox)
		assert.ThatSlice(t, set.Paths).Contains("8")
		// check that mark-read messages are not included with delete messages
		assert.ThatSlice(t, set.Paths).NotContains("markread")
		for _, uid := range upsertPaths {
			assert.ThatSlice(t, set.Paths).Contains(uid)
		}
	})

	t.Run("remove from change set", func(t *testing.T) {
		// remove all even numbers from set
		evenUIDs := []string{}
		oddUIDs := []string{}
		for i, uid := range upsertPaths {
			if (i+1)%2 == 0 {
				evenUIDs = append(evenUIDs, uid)
			} else {
				oddUIDs = append(oddUIDs, uid)
			}
		}
		delSet := ChangeSet{
			Inbox:     inbox,
			Paths:     evenUIDs,
			Operation: Delete,
		}
		numDeleted, err := repo.Remove(ctx, delSet)
		assert.ThatError(t, err).IsNil()
		assert.That(t, numDeleted).Equals(len(evenUIDs) * -1)

		p, err := repo.FindByInboxAndOperation(ctx, inbox, Delete, false)
		assert.ThatError(t, err).IsNil()
		assert.That(t, p).NotEqual(nil)

		assert.That(t, p.Inbox).Equals(inbox)
		assert.ThatSlice(t, p.Paths).HasLength(n - len(delSet.Paths))
		for _, odd := range oddUIDs {
			assert.ThatSlice(t, p.Paths).Contains(odd)
		}
		for _, even := range evenUIDs {
			assert.ThatSlice(t, p.Paths).NotContains(even)
		}
	})

	t.Run("archive", func(t *testing.T) {
		err := repo.Archive(ctx, inbox, Delete)
		assert.ThatError(t, err).IsNil()

		cs, err := repo.FindByInboxAndOperation(ctx, inbox, Delete, true)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
		assert.That(t, cs == nil).Equals(true)
	})

}
