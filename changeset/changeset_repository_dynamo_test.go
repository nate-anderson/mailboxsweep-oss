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
	folder := "INBOX32"
	t.Run("append to change set", func(t *testing.T) {
		t.Run("insert new", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:     inbox,
				Folder:    folder,
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
				Folder:    folder,
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

			retrieved, err := repo.ListByInbox(ctx, inbox, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(retrieved) == 1)
			assert.ThatSlice(t, retrieved[0].Paths).Contains(appendUID)
			assert.That(t, retrieved[0].Operation).Equals(Delete)
		})
	})

	t.Run("list for mailbox", func(t *testing.T) {
		sets, err := repo.ListByInbox(ctx, inbox, false)
		assert.
			ThatError(t, err).
			IsNil()

		assert.
			That(t, len(sets)).
			Equals(1)

		set := sets[0]
		assert.That(t, set.Inbox).Equals(inbox)
		assert.That(t, set.Folder).Equals(folder)
		assert.ThatSlice(t, set.Paths).Contains("8")
		for _, uid := range upsertPaths {
			assert.ThatSlice(t, set.Paths).Contains(uid)
		}
	})

	t.Run("remove from change set", func(t *testing.T) {
		// remove all even numbers from set
		evenPaths := []string{}
		oddPaths := []string{}
		for i, uid := range upsertPaths {
			if (i+1)%2 == 0 {
				evenPaths = append(evenPaths, uid)
			} else {
				oddPaths = append(oddPaths, uid)
			}
		}
		delSet := ChangeSet{
			Inbox:     inbox,
			Folder:    folder,
			Paths:     evenPaths,
			Operation: Delete,
		}
		numDeleted, err := repo.Remove(ctx, delSet)
		assert.ThatError(t, err).IsNil()
		assert.That(t, numDeleted).Equals(len(evenPaths))

		post, err := repo.ListByInbox(ctx, inbox, false)
		assert.ThatError(t, err).IsNil()

		p := post[0]
		assert.That(t, p.Folder).Equals(folder)
		assert.That(t, p.Inbox).Equals(inbox)
		assert.ThatSlice(t, p.Paths).HasLength(n - len(delSet.Paths))
		for _, odd := range oddPaths {
			assert.ThatSlice(t, p.Paths).Contains(odd)
		}
		for _, even := range evenPaths {
			assert.ThatSlice(t, p.Paths).NotContains(even)
		}
	})

	t.Run("find by inbox and folder", func(t *testing.T) {
		// store some
		randFolder := "bs_folder_123"
		cs := ChangeSet{
			Inbox:     inbox,
			Folder:    randFolder,
			Paths:     []string{"19", "66", "1056"},
			Operation: MarkRead,
		}
		n, err := repo.Append(ctx, cs)
		assert.ThatError(t, err).IsNil()
		assert.That(t, n).Equals(len(cs.Paths))

		sets, err := repo.FindByInboxFolder(ctx, inbox, randFolder, false)
		assert.ThatError(t, err).IsNil()
		assert.ThatSlice(t, sets[MarkRead].Paths).HasLength(3)
		for _, uid := range cs.Paths {
			assert.ThatSlice(t, sets[MarkRead].Paths).Contains(uid)
		}
		assert.That(t, sets[MarkRead].Operation).Equals(MarkRead)
	})

}
