package changeset

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/classification"
	"mailboxsweep/pkg/set"
	"mailboxsweep/session"
	"testing"
)

type testInternalChangeSet struct {
	deletePaths   set.Set[string]
	markReadPaths set.Set[string]
}

type testRepo struct {
	inbox   string
	folders map[string]*testInternalChangeSet
	t       *testing.T
}

func (r testRepo) Append(ctx context.Context, changeSet ChangeSet) (int, error) {
	r.inbox = changeSet.Inbox

	var cs *testInternalChangeSet
	if c, ok := r.folders[changeSet.Folder]; ok {
		cs = c
	} else {
		cs = &testInternalChangeSet{
			deletePaths:   set.NewSet[string](),
			markReadPaths: set.NewSet[string](),
		}
		r.folders[changeSet.Folder] = cs
	}

	diff := 0
	if changeSet.Operation == Delete {
		for _, path := range changeSet.Paths {
			if cs.deletePaths.Add(path) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, path := range changeSet.Paths {
			if cs.markReadPaths.Add(path) {
				diff++
			}
		}
	}
	return diff, nil
}

func (r testRepo) Remove(ctx context.Context, changeSet ChangeSet) (int, error) {
	r.inbox = changeSet.Inbox
	var cs *testInternalChangeSet
	if c, ok := r.folders[changeSet.Folder]; ok {
		cs = c
	} else {
		cs = &testInternalChangeSet{}
		r.folders[changeSet.Folder] = cs
	}
	diff := 0
	if changeSet.Operation == Delete {
		for _, path := range changeSet.Paths {
			if cs.deletePaths.Remove(path) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, path := range changeSet.Paths {
			if cs.markReadPaths.Remove(path) {
				diff++
			}
		}
	}
	return diff, nil
}

func (r testRepo) ListByInbox(ctx context.Context, inbox string, consistentRead bool) ([]ChangeSet, error) {
	changeSets := make([]ChangeSet, 0, len(r.folders))
	for folder, tcs := range r.folders {
		if tcs.deletePaths.Len() > 0 {
			changeSets = append(changeSets, ChangeSet{
				Inbox:     r.inbox,
				Folder:    folder,
				Operation: Delete,
				Paths:     tcs.deletePaths.Slice(),
			})
		}
		if tcs.markReadPaths.Len() > 0 {
			changeSets = append(changeSets, ChangeSet{
				Inbox:     r.inbox,
				Folder:    folder,
				Operation: MarkRead,
				Paths:     tcs.markReadPaths.Slice(),
			})
		}
	}
	return changeSets, nil
}

func (r testRepo) FindByInboxFolder(ctx context.Context, inbox, folder string, consistentRead bool) (map[UpdateOperation]ChangeSet, error) {
	tcs := r.folders[folder]
	deleteChangeSet := ChangeSet{
		Inbox:     inbox,
		Folder:    folder,
		Operation: Delete,
		Paths:     tcs.deletePaths.Slice(),
	}
	markRedChangeSet := ChangeSet{
		Inbox:     inbox,
		Folder:    folder,
		Operation: MarkRead,
		Paths:     tcs.markReadPaths.Slice(),
	}

	return map[UpdateOperation]ChangeSet{
		Delete:   deleteChangeSet,
		MarkRead: markRedChangeSet,
	}, nil
}

func (r *testRepo) Reset() {
	r.inbox = ""
	r.folders = make(map[string]*testInternalChangeSet)
}

func TestService(t *testing.T) {
	ctx := context.Background()
	inbox := "me@example.com"
	folder := "testfolder"

	authedCtx := session.AuthorizeTestContext(ctx, inbox, t)

	repo := testRepo{
		inbox:   inbox,
		folders: make(map[string]*testInternalChangeSet),
		t:       t,
	}
	service := NewDefaultService(&repo)

	t.Run("mark for delete", func(t *testing.T) {
		deletePaths := []classification.Path{{"domain=mail.google.com", "sender=admin@mail.google.com", "before_2020_01_01=true"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForDelete(ctx, folder, deletePaths)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForDelete(authedCtx, folder, deletePaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(deletePaths))

			assert.ThatSlice(t, repo.folders[folder].deletePaths.Slice()).HasLength(n)
		})
	})

	t.Run("mark for mark read", func(t *testing.T) {
		repo.Reset()
		markReadPaths := []classification.Path{{"domain=mail.google.com", "sender=welcome@mail.google.com", "before_2019_01_01=false"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForMarkRead(ctx, folder, markReadPaths)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForMarkRead(authedCtx, folder, markReadPaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(markReadPaths))

			assert.ThatSlice(t, repo.folders[folder].markReadPaths.Slice()).HasLength(n)
		})
	})

	t.Run("list by inbox", func(t *testing.T) {
		repo.Reset()

		path := "domain=yahoo.com,sender=buddy@yahoo.com,sent_before_2020=false"
		repo.folders[folder] = &testInternalChangeSet{
			deletePaths: set.NewSet(path),
		}

		t.Run("unauthenticated. failure", func(t *testing.T) {
			changeSets, err := service.ListByInbox(ctx, false)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(changeSets)).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			changeSets, err := service.ListByInbox(authedCtx, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(changeSets)).Equals(1)
			cs := changeSets[0]
			assert.That(t, cs.Operation).Equals(Delete)
			assert.That(t, len(cs.Paths)).Equals(1)
			p := cs.Paths[0]
			assert.That(t, p).Equals(path)
		})
	})

	t.Run("list by inbox folder", func(t *testing.T) {
		repo.Reset()

		deletePaths := []string{"12,34", "89041,13413r"}
		markReadPaths := []string{"10,15", "809,11"}
		repo.folders[folder] = &testInternalChangeSet{
			deletePaths:   set.NewSet(deletePaths...),
			markReadPaths: set.NewSet(markReadPaths...),
		}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			changeSetMap, err := service.ListByInboxFolder(ctx, folder, false)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(changeSetMap)).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			changeSetMap, err := service.ListByInboxFolder(authedCtx, folder, false)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(changeSetMap)).Equals(2)
			assert.ThatSlice(t, changeSetMap[Delete].Paths).ContainsAll(deletePaths)
			assert.ThatSlice(t, changeSetMap[MarkRead].Paths).ContainsAll(markReadPaths)
		})
	})

	t.Run("unmark for delete", func(t *testing.T) {
		deletePaths := []classification.Path{{"12", "34"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForDelete(ctx, folder, deletePaths)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.UnmarkForDelete(authedCtx, folder, deletePaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(deletePaths))

			assert.ThatSlice(t, repo.folders[folder].deletePaths.Slice()).HasLength(1)
		})
	})

	t.Run("unmark for mark read", func(t *testing.T) {
		markReadPaths := []classification.Path{{"809", "11"}}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.UnmarkForMarkRead(ctx, folder, markReadPaths)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.UnmarkForMarkRead(authedCtx, folder, markReadPaths)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(markReadPaths))

			assert.ThatSlice(t, repo.folders[folder].markReadPaths.Slice()).HasLength(1)
		})
	})
}
