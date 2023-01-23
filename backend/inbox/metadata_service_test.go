package inbox_test

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxtesting"
	"testing"
	"time"
)

func TestMetadataService(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	testAddress := "test@example.com"

	providerService := inboxtesting.NewTestProviderService()
	n := 10

	messages := make([]inbox.Message, n)
	for i := 0; i < n; i++ {
		un := "me"
		sender := fmt.Sprintf("%s@gmail.com", un)
		msg := inbox.Message{
			UID:     uint32(i),
			Folder:  "INBOX",
			Subject: fmt.Sprintf("msg %d", i),
			Sender: inbox.Sender{
				Address:  sender,
				Name:     un,
				Username: un,
				Domain:   "gmail.com",
			},
			SendDate: time.Now().Add(time.Hour * 24 * 365 * -1 * time.Duration(i)),
		}
		messages[i] = msg
	}
	providerService.Messages["INBOX"] = messages
	providerService.Messages["trash"] = []inbox.Message{}

	credentialService := inboxtesting.TestCredentialReadService{
		Address:  testAddress,
		Password: "password",
	}

	repo := inbox.NewMetadataRepository(db, table)
	service := inbox.NewMetadataService(
		repo,
		providerService,
		credentialService,
		time.Minute,
	)

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddress, t)

	t.Run("Get", func(t *testing.T) {
		t.Run("unauthenticated, failure", func(t *testing.T) {
			m, err := service.Get(context.Background())
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, m).Equals(nil)
		})

		t.Run("cache miss", func(t *testing.T) {
			m, err := service.Get(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, m.Expires).NotEqual(0)
			assert.That(t, m.Inbox).Equals(testAddress)
			assert.That(t, len(m.Folders)).Equals(2)
			assert.That(t, m.Folders["INBOX"].All).Equals(uint32(n))
			assert.That(t, m.Folders["trash"].All).Equals(0)

			// check that cache is populated
			mr, err := repo.Get(ctx, testAddress)
			assert.ThatError(t, err).IsNil()
			assert.That(t, mr.Expires).NotEqual(0)
			assert.That(t, mr.Inbox).Equals(testAddress)
			assert.That(t, len(mr.Folders)).Equals(2)
			assert.That(t, mr.Folders["INBOX"].All).Equals(uint32(n))
			assert.That(t, mr.Folders["trash"].All).Equals(0)
		})

		t.Run("cache hit", func(t *testing.T) {
			m, err := service.Get(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, m.Expires).NotEqual(0)
			assert.That(t, m.Inbox).Equals(testAddress)
			assert.That(t, len(m.Folders)).Equals(2)
			assert.That(t, m.Folders["INBOX"].All).Equals(uint32(n))
			assert.That(t, m.Folders["trash"].All).Equals(0)
		})
	})

	t.Run("Set", func(t *testing.T) {
		m := inbox.Metadata{
			Inbox: testAddress,
			Folders: map[string]inbox.MessageCount{"INBOX2": {
				All:    20,
				Unread: 10,
			}},
			Expires: time.Now().Add(time.Minute).Unix(),
		}
		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.Set(context.Background(), m)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("success", func(t *testing.T) {
			err := service.Set(authedCtx, m)
			assert.ThatError(t, err).IsNil()

			r, err := service.Get(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, r.Expires).Equals(m.Expires)
			assert.That(t, r.Inbox).Equals(testAddress)
			assert.ThatMap(t, r.Folders).HasLength(1)
			assert.ThatMap(t, r.Folders).HasValueAt("INBOX2", inbox.MessageCount{
				All:    20,
				Unread: 10,
			})
		})
	})
}
