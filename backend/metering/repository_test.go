package metering

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"testing"
)

func TestUsageRepository(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := NewUsageRepository(db, table)

	var (
		n = 50
		k = 100
	)

	testAddr := "admin@example2.com"

	t.Run("update usage", func(t *testing.T) {
		usage := 0
		for i := 0; i < n; i++ {
			usage += k
			err := repo.Update(ctx, testAddr, usage)
			assert.ThatError(t, err).IsNil()
		}
	})

	t.Run("get usage", func(t *testing.T) {
		rec, err := repo.CheckUsage(ctx, testAddr)
		assert.ThatError(t, err).IsNil()
		assert.That(t, rec.Address).Equals(testAddr)
		assert.That(t, rec.BytesUsed).Equals(k * n)
	})
}
