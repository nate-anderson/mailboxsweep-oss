package batch

import (
	"mailboxsweep/assert"
	"testing"
)

func TestBatchWithLimit(t *testing.T) {
	n := 10241
	limit := 1000

	population := make([]int, n)
	for i := 0; i < n; i++ {
		population[i] = i
	}

	batches := WithLimit(population, limit)
	assert.That(t, len(batches)).Equals(11)
	unBatched := make([]int, 0, n)
	for i := 0; i < 10; i++ {
		assert.ThatSlice(t, batches[i]).HasLength(limit)
		unBatched = append(unBatched, batches[i]...)
	}
	assert.ThatSlice(t, batches[10]).HasLength(n % limit)
	unBatched = append(unBatched, batches[10]...)
	assert.ThatSlice(t, unBatched).HasLength(n)

	assert.ThatSlice(t, unBatched).Equals(population)
}
