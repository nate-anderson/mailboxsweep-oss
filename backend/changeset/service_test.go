package changeset

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/classification"
	"mailboxsweep/pkg/set"
	"testing"
	"time"
)

type testRepo struct {
	inbox            string
	deletePaths      set.Set[string]
	markReadPaths    set.Set[string]
	deleteExecuted   time.Time
	markReadExecuted time.Time
	t                *testing.T
}

func (r testRepo) Append(ctx context.Context, changeSet ChangeSet) (int, error) {
	r.inbox = changeSet.Inbox

	diff := 0
	if changeSet.Operation == Delete {
		for _, path := range changeSet.Paths {
			if r.deletePaths.Add(path) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, path := range changeSet.Paths {
			if r.markReadPaths.Add(path) {
				diff++
			}
		}
	}
	return diff, nil
}

func (r testRepo) Remove(ctx context.Context, changeSet ChangeSet) (int, error) {
	r.inbox = changeSet.Inbox

	diff := 0
	if changeSet.Operation == Delete {
		for _, path := range changeSet.Paths {
			if r.deletePaths.Remove(path) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, path := range changeSet.Paths {
			if r.markReadPaths.Remove(path) {
				diff++
			}
		}
	}
	return diff, nil
}

func (r testRepo) FindByInboxAndOperation(ctx context.Context, inbox string, operation UpdateOperation, consistentRead bool) (*ChangeSet, error) {
	if r.inbox != inbox {
		return nil, mailboxsweep.ErrNotFound
	}
	if operation == Delete {
		return &ChangeSet{
			Inbox:     inbox,
			Operation: operation,
			Paths:     r.deletePaths.Slice(),
		}, nil
	}

	if operation == MarkRead {
		return &ChangeSet{
			Inbox:     inbox,
			Operation: MarkRead,
			Paths:     r.markReadPaths.Slice(),
		}, nil
	}

	return nil, ErrInvalidOperation
}

func (r *testRepo) MarkExecuted(ctx context.Context, inbox string, operation UpdateOperation) error {
	if r.inbox != inbox {
		return mailboxsweep.ErrNotFound
	}

	if operation == MarkRead {
		r.markReadExecuted = time.Now()
	} else if operation == Delete {
		r.deleteExecuted = time.Now()
	}

	return nil
}

func (r *testRepo) Reset() {
	r.inbox = ""
	r.deletePaths = set.NewSet[string]()
	r.markReadPaths = set.NewSet[string]()
}

func TestChangeSetService(t *testing.T) {
	ctx := context.Background()
	inbox := "me@example.com"

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, inbox, t)

	repo := testRepo{
		inbox:         inbox,
		deletePaths:   set.NewSet[string](),
		markReadPaths: set.NewSet[string](),
		t:             t,
	}
	service := NewDefaultService(&repo)

	t.Run("mark for delete", func(t *testing.T) {
		deletePaths := []classification.Path{{"domain=mail.google.com", "sender=admin@mail.google.com", "before_2020_01_01=true"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForDelete(ctx, deletePaths)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForDelete(authedCtx, deletePaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(deletePaths))

			assert.ThatSlice(t, repo.deletePaths.Slice()).HasLength(n)
		})
	})

	t.Run("mark for mark read", func(t *testing.T) {
		repo.Reset()
		markReadPaths := []classification.Path{{"domain=mail.google.com", "sender=welcome@mail.google.com", "before_2019_01_01=false"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForMarkRead(ctx, markReadPaths)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForMarkRead(authedCtx, markReadPaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(markReadPaths))

			assert.ThatSlice(t, repo.markReadPaths.Slice()).HasLength(n)
		})
	})

	t.Run("list by inbox", func(t *testing.T) {
		repo.Reset()

		path := "domain=yahoo.com,sender=buddy@yahoo.com,sent_before_2020=false"
		repo.deletePaths = set.NewSet(path)
		repo.inbox = inbox

		t.Run("unauthenticated. failure", func(t *testing.T) {
			changeSet, err := service.FindByOperation(ctx, Delete, false)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, changeSet).Equals(nil)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			changeSet, err := service.FindByOperation(authedCtx, Delete, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, changeSet).NotEqual(nil)
			assert.That(t, changeSet.Operation).Equals(Delete)
			assert.That(t, len(changeSet.Paths)).Equals(1)
			p := changeSet.Paths[0]
			assert.That(t, p).Equals(path)
		})
	})

	t.Run("unmark for delete", func(t *testing.T) {
		deletePaths := []classification.Path{{"12", "34"}}
		repo.deletePaths.Add("12,34")
		repo.markReadPaths.Add("12,34")

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForDelete(ctx, deletePaths)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.UnmarkForDelete(authedCtx, deletePaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(deletePaths))

			assert.ThatSlice(t, repo.deletePaths.Slice()).HasLength(1)
			assert.ThatSlice(t, repo.markReadPaths.Slice()).HasLength(1)
			assert.That(t, repo.markReadPaths.Has("12,34")).Equals(true)
		})
	})

	t.Run("unmark for mark read", func(t *testing.T) {
		markReadPaths := []classification.Path{{"809", "11"}}
		repo.markReadPaths.Add("809,11")
		repo.deletePaths.Add("809,11")

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.UnmarkForMarkRead(ctx, markReadPaths)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.UnmarkForMarkRead(authedCtx, markReadPaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(markReadPaths))

			assert.ThatSlice(t, repo.markReadPaths.Slice()).HasLength(1)
			assert.That(t, repo.deletePaths.Has("809,11")).Equals(true)
		})
	})
}
