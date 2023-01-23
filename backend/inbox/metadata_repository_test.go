package inbox

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
)

func TestMetadataRepository(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewMetadataRepository(db, table)

	m := Metadata{
		Inbox: "fromage@reuters.beans",
		Folders: map[string]MessageCount{
			"INBOX": {
				All:    498014,
				Unread: 400,
			},
			"Trash": {
				All:    6000,
				Unread: 600,
			},
		},
		Expires: time.Now().Unix() + 30,
	}

	t.Run("store metadata", func(t *testing.T) {
		err := repo.Store(context.Background(), m)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("get metadata", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			r, err := repo.Get(ctx, m.Inbox)
			assert.ThatError(t, err).IsNil()
			assert.That(t, r).NotEqual(nil)
			assert.That(t, r.Inbox).Equals(m.Inbox)
			assert.That(t, r.Expires).Equals(m.Expires)
			assert.That(t, len(r.Folders)).Equals(len(m.Folders))
			for folder, count := range m.Folders {
				assert.ThatMap(t, r.Folders).HasValueAt(folder, count)
			}
		})

		t.Run("not found", func(t *testing.T) {
			r, err := repo.Get(ctx, "nobody@fallows.weep")
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, r).Equals(nil)
		})
	})
}
