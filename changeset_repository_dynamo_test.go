package mailboxsweep

import (
	"context"
	"fmt"
	"mailboxsweep/assert"
	"strconv"
	"testing"
)

func TestDynamoChangesetRepository(t *testing.T) {
	cl := setupTestDynamoClient()
	ctx := context.Background()
	table, _ := setupDynamoTestTable(ctx, t, cl)
	// defer teardown()
	fmt.Println("TABLE: ", table)
	repo, err := NewDynamoChangesetRepository(cl, table, 10, imapUIDHash)
	assert.ThatError(t, err).IsNil()

	n := 300
	upsertUIDs := []string{}
	appendUID := "1"
	for i := 2; i <= n; i++ {
		istr := strconv.Itoa(i)
		upsertUIDs = append(upsertUIDs, istr)
	}

	// hacky way to keep tests from colliding
	inbox := fmt.Sprintf("%T@example.org", *repo)
	folder := "INBOX32"
	t.Run("append to change set", func(t *testing.T) {
		t.Run("insert new", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:     inbox,
				Folder:    folder,
				UIDs:      upsertUIDs,
				Operation: Delete,
			}
			n, err := repo.Append(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(len(upsertUIDs))
		})

		t.Run("append to existing page", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:     inbox,
				Folder:    folder,
				UIDs:      []string{appendUID},
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
			assert.ThatSlice(t, retrieved[0].UIDs).Contains(appendUID)
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
		assert.ThatSlice(t, set.UIDs).Contains("8")
		for _, uid := range upsertUIDs {
			assert.ThatSlice(t, set.UIDs).Contains(uid)
		}
	})

	t.Run("remove from change set", func(t *testing.T) {
		// remove all even numbers from set
		evenUIDs := []string{}
		oddUIDs := []string{}
		for i, uid := range upsertUIDs {
			if (i+1)%2 == 0 {
				evenUIDs = append(evenUIDs, uid)
			} else {
				oddUIDs = append(oddUIDs, uid)
			}
		}
		delSet := ChangeSet{
			Inbox:     inbox,
			Folder:    folder,
			UIDs:      evenUIDs,
			Operation: Delete,
		}
		numDeleted, err := repo.Remove(ctx, delSet)
		assert.ThatError(t, err).IsNil()
		assert.That(t, numDeleted).Equals(len(evenUIDs))

		post, err := repo.ListByInbox(ctx, inbox, false)
		assert.ThatError(t, err).IsNil()

		p := post[0]
		assert.That(t, p.Folder).Equals(folder)
		assert.That(t, p.Inbox).Equals(inbox)
		assert.ThatSlice(t, p.UIDs).HasLength(n - len(delSet.UIDs))
		for _, odd := range oddUIDs {
			assert.ThatSlice(t, p.UIDs).Contains(odd)
		}
		for _, even := range evenUIDs {
			assert.ThatSlice(t, p.UIDs).NotContains(even)
		}
	})

	t.Run("find by inbox and folder", func(t *testing.T) {
		// store some
		randFolder := "bs_folder_123"
		cs := ChangeSet{
			Inbox:     inbox,
			Folder:    randFolder,
			UIDs:      []string{"19", "66", "1056"},
			Operation: MarkRead,
		}
		n, err := repo.Append(ctx, cs)
		assert.ThatError(t, err).IsNil()
		assert.That(t, n).Equals(len(cs.UIDs))

		sets, err := repo.FindByInboxFolder(ctx, inbox, randFolder, false)
		assert.ThatError(t, err).IsNil()
		assert.ThatSlice(t, sets[MarkRead].UIDs).HasLength(3)
		for _, uid := range cs.UIDs {
			assert.ThatSlice(t, sets[MarkRead].UIDs).Contains(uid)
		}
		assert.That(t, sets[MarkRead].Operation).Equals(MarkRead)
	})

}
