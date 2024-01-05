package session

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"
)

func TestSession(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoSessionRepository(db, table)

	token, err := DefaultService{
		sessionKeyLen: 16,
	}.makeToken()
	assert.ThatError(t, err).IsNil()
	expires := time.Now().Add(time.Minute)

	sess := Session{
		UserAddress: "admin130@example.com",
		Token:       token,
		Expires:     expires.Unix(),
	}

	t.Run("store", func(t *testing.T) {
		err := repo.Store(ctx, sess)
		assert.ThatError(t, err).IsNil()
	})

	t.Run("get", func(t *testing.T) {
		ret, err := repo.Get(ctx, sess.Token)
		assert.ThatError(t, err).IsNil()
		assert.That(t, ret.Expires).Equals(sess.Expires)
		assert.That(t, ret.Token).Equals(sess.Token)
		assert.That(t, ret.UserAddress).Equals(sess.UserAddress)
	})

	t.Run("invalidate", func(t *testing.T) {
		err := repo.Invalidate(ctx, sess.Token)
		assert.ThatError(t, err).IsNil()

		ret, err := repo.Get(ctx, sess.Token)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
		assert.That(t, ret).Equals(nil)
	})

	t.Run("list by user", func(t *testing.T) {
		t.Run("no active sessions", func(t *testing.T) {
			sessions, err := repo.ListByUser(ctx, sess.UserAddress)
			assert.That(t, len(sessions)).Equals(0)
			assert.ThatError(t, err).IsNil()
		})

		t.Run("multiple active sessions", func(t *testing.T) {
			nSessions := 3
			tokens := []string{}
			for i := 0; i < nSessions; i++ {
				token, err := DefaultService{
					sessionKeyLen: 16,
				}.makeToken()
				assert.ThatError(t, err).IsNil()
				s := Session{
					UserAddress: sess.UserAddress,
					Token:       token,
					Expires:     expires.Unix(),
				}
				err = repo.Store(ctx, s)
				assert.ThatError(t, err)
				tokens = append(tokens, token)
			}

			list, err := repo.ListByUser(ctx, sess.UserAddress)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(list)).Equals(nSessions)
			for _, s := range list {
				assert.ThatSlice(t, tokens).Contains(s)
			}
		})
	})
}
