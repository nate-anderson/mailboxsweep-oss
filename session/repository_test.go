package session

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSession(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoSessionRepository(db, table)

	userUUID := uuid.NewString()
	token, err := DefaultService.makeToken(DefaultService{})
	assert.ThatError(t, err).IsNil()
	expires := time.Now().Add(time.Minute)

	sess := Session{
		UserUUID: userUUID,
		Token:    token,
		Expires:  expires.Unix(),
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
		assert.That(t, ret.UserUUID).Equals(sess.UserUUID)
	})

	t.Run("invalidate", func(t *testing.T) {
		err := repo.Invalidate(ctx, sess.Token)
		assert.ThatError(t, err).IsNil()

		ret, err := repo.Get(ctx, sess.Token)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
		assert.That(t, ret).Equals(nil)
	})
}
