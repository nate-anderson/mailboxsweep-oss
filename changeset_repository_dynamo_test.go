package mailboxsweep

import (
	"context"
	"fmt"
	"mailboxsweep/assert"
	"testing"
)

func TestDynamoChangesetRepository(t *testing.T) {
	cl := setupTestDynamoClient()
	ctx := context.Background()
	table, teardown := setupDynamoTestTable(ctx, t, cl)
	defer teardown()

	repo := NewDynamoChangesetRepository(cl, table)

	upsertUIDs := []string{"1", "109", "10983", "83749871"}

	// hacky way to keep tests from colliding
	inbox := fmt.Sprintf("%T@example.org", repo)
	folder := "INBOX32"
	t.Run("upsert change set", func(t *testing.T) {
		t.Run("insert new", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:  inbox,
				Folder: folder,
				UIDs:   upsertUIDs,
			}
			n, err := repo.Upsert(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(len(upsertUIDs))
		})

		t.Run("update existing", func(t *testing.T) {
			cs := ChangeSet{
				Inbox:  inbox,
				Folder: folder,
				UIDs:   append(upsertUIDs, "8"),
			}
			n, err := repo.Upsert(ctx, cs)
			assert.
				ThatError(t, err).
				IsNil()
			assert.
				That(t, n).
				Equals(1)
		})
	})

	t.Run("list for mailbox", func(t *testing.T) {
		sets, err := repo.ListByInbox(ctx, inbox)
		assert.
			ThatError(t, err).
			IsNil()
		assert.
			That(t, len(sets)).
			Equals(1)
	})

}
