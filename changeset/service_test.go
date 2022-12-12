package changeset

import (
	"context"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/pkg/set"
	"mailboxsweep/session"
	"testing"
)

type testInternalChangeSet struct {
	deleteUIDs   set.Set[string]
	markReadUIDs set.Set[string]
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
			deleteUIDs:   set.NewSet[string](),
			markReadUIDs: set.NewSet[string](),
		}
		r.folders[changeSet.Folder] = cs
	}

	diff := 0
	if changeSet.Operation == Delete {
		for _, uid := range changeSet.UIDs {
			if cs.deleteUIDs.Add(uid) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, uid := range changeSet.UIDs {
			if cs.markReadUIDs.Add(uid) {
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
		for _, uid := range changeSet.UIDs {
			if cs.deleteUIDs.Remove(uid) {
				diff++
			}
		}
	} else if changeSet.Operation == MarkRead {
		for _, uid := range changeSet.UIDs {
			if cs.markReadUIDs.Remove(uid) {
				diff++
			}
		}
	}
	return diff, nil
}

func (r testRepo) ListByInbox(ctx context.Context, inbox string, consistentRead bool) ([]ChangeSet, error) {
	changeSets := make([]ChangeSet, 0, len(r.folders))
	for folder, tcs := range r.folders {
		if tcs.deleteUIDs.Len() > 0 {
			changeSets = append(changeSets, ChangeSet{
				Inbox:     r.inbox,
				Folder:    folder,
				Operation: Delete,
				UIDs:      tcs.deleteUIDs.Slice(),
			})
		}
		if tcs.markReadUIDs.Len() > 0 {
			changeSets = append(changeSets, ChangeSet{
				Inbox:     r.inbox,
				Folder:    folder,
				Operation: MarkRead,
				UIDs:      tcs.markReadUIDs.Slice(),
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
		UIDs:      tcs.deleteUIDs.Slice(),
	}
	markRedChangeSet := ChangeSet{
		Inbox:     inbox,
		Folder:    folder,
		Operation: MarkRead,
		UIDs:      tcs.markReadUIDs.Slice(),
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
		deleteUIDs := []string{
			"10", "4908", "194", "9081", "12984", "78", "1",
		}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForDelete(ctx, folder, deleteUIDs)
			assert.ThatError(t, err).IsNotNil()
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForDelete(authedCtx, folder, deleteUIDs)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(deleteUIDs))

			assert.ThatSlice(t, repo.folders[folder].deleteUIDs.Slice()).HasLength(n)
		})
	})

	t.Run("mark for mark read", func(t *testing.T) {
		repo.Reset()
		markReadUIDs := []string{
			"10", "19", "340", "15", "9081",
		}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			n, err := service.MarkForMarkRead(ctx, folder, markReadUIDs)
			assert.ThatError(t, err).IsNotNil()
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, n).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			n, err := service.MarkForMarkRead(authedCtx, folder, markReadUIDs)
			assert.ThatError(t, err).IsNil()
			assert.That(t, n).Equals(len(markReadUIDs))

			assert.ThatSlice(t, repo.folders[folder].markReadUIDs.Slice()).HasLength(n)
		})
	})

	t.Run("list by inbox", func(t *testing.T) {
		repo.Reset()

		ids := []string{"1", "15", "315"}
		repo.folders[folder] = &testInternalChangeSet{
			deleteUIDs: set.NewSet(ids...),
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
			assert.That(t, changeSets[0].Operation).Equals(Delete)
			assert.ThatSlice(t, changeSets[0].UIDs).HasLength(len(ids))
			for _, uid := range ids {
				assert.ThatSlice(t, changeSets[0].UIDs).Contains(uid)
			}
		})
	})

	t.Run("list by inbox folder", func(t *testing.T) {
		repo.Reset()

		deleteIDs := []string{"1", "15", "315"}
		markReadIDs := []string{"10", "809"}
		repo.folders[folder] = &testInternalChangeSet{
			deleteUIDs:   set.NewSet(deleteIDs...),
			markReadUIDs: set.NewSet(markReadIDs...),
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
			assert.ThatSlice(t, changeSetMap[Delete].UIDs).Equals(deleteIDs)
			assert.ThatSlice(t, changeSetMap[MarkRead].UIDs).Equals(markReadIDs)
		})
	})
}
