package inboxuser

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDynamoInboxUserRepository(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewDynamoInboxUserRepository(db, table)
	var created time.Time

	testUser := InboxUser{
		UUID:       uuid.NewString(),
		Address:    "admin15@example.com",
		Created:    time.Now(),
		LastSeen:   time.Now(),
		Agreements: make(map[Agreement]time.Time),
		Payments:   []Payment{},
	}

	t.Run("store inbox user", func(t *testing.T) {
		uuid, err := repo.Store(ctx, testUser)
		assert.ThatError(t, err).IsNil()
		assert.That(t, uuid).Equals(testUser.UUID)
	})

	t.Run("find inbox user by address", func(t *testing.T) {
		retrieved, err := repo.FindByAddress(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.Address).Equals(testUser.Address)
		assert.That(t, retrieved.Created.Equal(testUser.Created)).Equals(true)
		assert.That(t, retrieved.LastSeen.Equal(testUser.LastSeen)).Equals(true)
		assert.That(t, retrieved.UUID).Equals(testUser.UUID)

		created = retrieved.Created
	})

	t.Run("find inbox user by uuid", func(t *testing.T) {
		retrieved, err := repo.FindByUUID(ctx, testUser.UUID)
		assert.ThatError(t, err).IsNil()
		assert.That(t, retrieved.Address).Equals(testUser.Address)
		assert.That(t, retrieved.Created.Equal(testUser.Created)).Equals(true)
		assert.That(t, retrieved.LastSeen.Equal(testUser.LastSeen)).Equals(true)
		assert.That(t, retrieved.UUID).Equals(testUser.UUID)
	})

	t.Run("consent to agreement", func(t *testing.T) {
		err := repo.Consent(ctx, testUser.Address, PersonalInbox)
		assert.ThatError(t, err).IsNil()
		agreedAt := time.Now()

		user, err := repo.FindByAddress(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()
		assert.ThatMap(t, user.Agreements).ContainsKey(PersonalInbox)
		assert.That(t, user.Agreements[PersonalInbox].Year()).Equals(agreedAt.Year())
		assert.That(t, user.Agreements[PersonalInbox].Month()).Equals(agreedAt.Month())
		assert.That(t, user.Agreements[PersonalInbox].Day()).Equals(agreedAt.Day())
		assert.That(t, user.Agreements[PersonalInbox].Hour()).Equals(agreedAt.Hour())
		assert.That(t, user.Agreements[PersonalInbox].Minute()).Equals(agreedAt.Minute())
	})

	t.Run("store payment", func(t *testing.T) {
		p := Payment{
			Plan:          Deluxe,
			AmountInCents: 2000,
			Method:        Stripe,
			Timestamp:     time.Now(),
		}
		err := repo.StorePayment(ctx, testUser.Address, p)
		assert.ThatError(t, err).IsNil()

		user, err := repo.FindByAddress(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()
		assert.ThatSlice(t, user.Payments).HasLength(1)

		retrieved := user.Payments[0]
		assert.That(t, retrieved.AmountInCents).Equals(p.AmountInCents)
		assert.That(t, retrieved.Method).Equals(Stripe)
		assert.That(t, retrieved.Plan).Equals(Deluxe)
		assert.That(t, retrieved.Timestamp.Year()).Equals(p.Timestamp.Year())
		assert.That(t, retrieved.Timestamp.Month()).Equals(p.Timestamp.Month())
		assert.That(t, retrieved.Timestamp.Day()).Equals(p.Timestamp.Day())
		assert.That(t, retrieved.Timestamp.Hour()).Equals(p.Timestamp.Hour())
		assert.That(t, retrieved.Timestamp.Minute()).Equals(p.Timestamp.Minute())
	})

	t.Run("touch last seen", func(t *testing.T) {
		user, err := repo.FindByAddress(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()

		assert.That(t, user.LastSeen.After(created)).Equals(true)
	})

	t.Run("set user name", func(t *testing.T) {
		nameFirst := "Buddy"
		nameLast := "Holly"
		err := repo.SetName(ctx, testUser.Address, nameFirst, nameLast)
		assert.ThatError(t, err).IsNil()

		user, err := repo.FindByAddress(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()
		assert.That(t, user.NameFirst).Equals(nameFirst)
		assert.That(t, user.NameLast).Equals(nameLast)
	})

	t.Run("lookup unknown user", func(t *testing.T) {
		addr := "bambino@gambino.wampino"
		user, err := repo.FindByAddress(ctx, addr)
		assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
		assert.That(t, user).Equals(nil)
	})
}
