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
	table, teardown := setupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo, err := NewDynamoChangesetRepository(cl, table, 10, imapUIDHash)
	assert.ThatError(t, err).IsNil()

	n := 300
	upsertUIDs := make([]string, 0, n-1)
	appendUID := "1"
	for i := 2; i < n; i++ {
		istr := strconv.Itoa(i)
		upsertUIDs = append(upsertUIDs, istr)
	}

	// hacky way to keep tests from colliding
	inbox := fmt.Sprintf("%T@example.org", *repo)
	folder := "INBOX32"
	t.Run("upsert change set", func(t *testing.T) {
		t.Run("insert new", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:  inbox,
				Folder: folder,
				UIDs:   upsertUIDs,
			}
			n, err := repo.Append(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(len(upsertUIDs))
		})

		t.Run("update existing (add, single value)", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:  inbox,
				Folder: folder,
				UIDs:   append(upsertUIDs, appendUID),
			}
			n, err := repo.Append(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(1)

			retrieved, err := repo.ListByInbox(ctx, inbox)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(retrieved) == 1)
			assert.ThatSlice(t, retrieved[0].UIDs).Contains(appendUID)
		})

		t.Run("update existing (remove, single page)", func(t *testing.T) {

		})

		t.Run("update existing (add, multiple pages)", func(t *testing.T) {

		})

		t.Run("update existing (remove, multiple pages)", func(t *testing.T) {})
		/*
			This will be the tricky one - if the new changeset is much smaller than the
			stored one, later pages won't be updated
		*/
	})

	t.Run("list for mailbox", func(t *testing.T) {
		sets, err := repo.ListByInbox(ctx, inbox)
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

}
