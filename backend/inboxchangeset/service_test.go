package inboxchangeset

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxtesting"
	"mailboxsweep/inboxuser"
	"mailboxsweep/payment"
	"mailboxsweep/pkg/slice"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type testImapServerService struct {
	t              *testing.T
	expectedDomain string
	wasCalled      bool
}

func (s *testImapServerService) Find(ctx context.Context) (*imapserver.Server, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domain, err := mailboxsweep.DomainFromEmail(addr)
	if err != nil {
		return nil, err
	}

	assert.That(s.t, domain).Equals(domain)
	s.wasCalled = true
	return &imapserver.Server{
		Domain: s.expectedDomain,
		Host:   "imap." + s.expectedDomain,
		Port:   993,
	}, nil
}

func (s testImapServerService) Store(ctx context.Context, server imapserver.Server) error {
	s.t.Fatalf("don't call Store on testImapServerService")
	return nil
}

func (s testImapServerService) FindByUserAddress(ctx context.Context) (*imapserver.Server, error) {
	return nil, errors.New("testImapServerService does not implement FindByUserAddress")
}

func TestInboxChangesetService(t *testing.T) {
	db := mailboxsweep.SetupTestDynamoClient()
	ctx := context.Background()
	table, teardownTable := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardownTable()

	changeSetRepo, err := changeset.NewDynamoChangesetRepository(db, table, 13)
	assert.ThatError(t, err).IsNil()

	changeSetService := changeset.NewDefaultService(changeSetRepo)

	s3Client, bucketName, teardownBucket := mailboxsweep.SetupTestBucket(ctx, t)
	defer teardownBucket()

	var (
		testAddress = "inboxchangeset@mailboxsweep.com"
		testPass    = "gobblygook"
	)

	credentialReadService := &inboxtesting.TestCredentialReadService{
		Address:  testAddress,
		Password: testPass,
	}

	imapServerService := &testImapServerService{
		expectedDomain: "mailboxsweep.com",
		t:              t,
	}

	providerService := inboxtesting.NewTestProviderService()

	classificationRepo := classification.NewS3ClassificationRepository(s3Client, bucketName, 10)

	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService, err := inboxuser.NewService(userRepo)
	assert.ThatError(t, err).IsNil()

	classifyRequestRepo := classifyrequest.NewDynamoRequestRepository(db, table)
	classifyRequestService := classifyrequest.NewDefaultService(userService, classifyRequestRepo, time.Minute, time.Minute*2)

	paymentProcessor := payment.NewTestProcessor()
	paymentProcessor.SessionID = "test_session_123"
	cartRepo := payment.NewCartRepository(db, table)
	paymentService, err := payment.NewService(userService, paymentProcessor, cartRepo, time.Hour)
	assert.ThatError(t, err).IsNil()

	metadataRepo := inbox.NewMetadataRepository(db, table)
	metadataService := inbox.NewMetadataService(metadataRepo, providerService, credentialReadService, time.Hour)

	publisher := &mailboxsweep.TestEventPublisher[classification.ClassifyQueueItem]{}

	classificationService := classification.NewService(
		credentialReadService,
		imapServerService,
		classificationRepo,
		providerService,
		classifyRequestService,
		paymentService,
		metadataService,
		logrus.New(),
		publisher,
		5,
	)

	service := NewService(
		changeSetService,
		classificationService,
		providerService,
		credentialReadService,
		classifyRequestService,
		metadataService,
		logrus.New(),
	)

	// seed messages
	folder := "INBOX"
	messages := []inbox.Message{}
	numMessages := 10

	senders := []string{
		"sender1@mailboxsweep.com",
		"sender2@example.org",
		"sender3@example.org",
	}

	for i := 0; i < numMessages; i++ {
		senderIndex := i % 3
		m := inbox.Message{
			UID:     strconv.Itoa(i),
			Folder:  folder,
			Subject: fmt.Sprintf("Msg #%d", i),
			Sender: inbox.Sender{
				Address:  senders[senderIndex],
				Name:     senders[senderIndex],
				Username: strings.Split(senders[senderIndex], "@")[0],
				Domain:   strings.Split(senders[senderIndex], "@")[1],
			},
			SendDate: time.Now().Add(time.Duration(-1*i) * time.Hour),
			Read:     false,
		}
		messages = append(messages, m)
	}

	providerService.Messages[folder] = messages

	authedCtx := mailboxsweep.AuthorizeAPIContext(ctx, testAddress)
	dateClassifier, err := classification.NewSendDateClassifier("01-01-2020")
	assert.ThatError(t, err).IsNil()

	// seed the user
	_, err = userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	// seed a classification request
	err = classifyRequestService.RecordRequest(authedCtx)
	assert.ThatError(t, err).IsNil()

	// seed payment
	_, err = paymentService.InitiatePayment(authedCtx, inboxuser.Basic)
	assert.ThatError(t, err).IsNil()

	err = paymentService.CompletePayment(authedCtx, inboxuser.PaymentMethod("Test"), paymentProcessor.SessionID)
	assert.ThatError(t, err).IsNil()

	classifiers := []classification.Classifier{
		classification.NewSenderDomainClassifier(),
		classification.NewSenderUsernameClassifier(),
		dateClassifier,
	}
	err = classificationService.Classify(authedCtx, []string{"INBOX"}, classifiers)
	assert.ThatError(t, err).IsNil()

	// seed changesets

	t.Run("add for mark read", func(t *testing.T) {
		paths := []classification.Path{
			{
				"domain=example.org",
				"sender=sender3@example.org",
				"on_or_after_01-01-2020",
			},
			{
				"domain=mailboxsweep.com",
				"sender=sender1@mailboxsweep.com",
				"on_or_after_01-01-2020",
			},
		}

		t.Run("add", func(t *testing.T) {
			for _, p := range paths {
				markReadRes, err := service.AddPathForMarkRead(authedCtx, p)
				assert.ThatError(t, err).IsNil()
				assert.That(t, len(markReadRes.Paths)).Equals(1)
				assert.ThatSlice(t, markReadRes.Paths[0]).Equals(p)
			}
		})

		t.Run("result is stored", func(t *testing.T) {
			markReadSet, markReadCount, err := service.FindByOperation(authedCtx, changeset.MarkRead)
			assert.ThatError(t, err).IsNil()
			assert.That(t, markReadCount).Equals(7)
			assert.ThatSlice(t, markReadSet.Paths).HasLength(2)
			for _, pathStr := range markReadSet.Paths {
				p := classification.ParsePath(pathStr)
				assert.ThatSlice(t, p).HasLength(3)
				found := false
				for _, seedPath := range paths {
					if slicesAreEqual(seedPath, p) {
						found = true
					}
				}
				assert.That(t, found).Equals(true)
			}
		})

		t.Run("adding a path twice is a no-op", func(t *testing.T) {
			markReadRes, err := service.AddPathForMarkRead(authedCtx, paths[0])
			assert.ThatError(t, err).IsNil()
			assert.That(t, markReadRes.MsgCountDiff).Equals(0)
			assert.That(t, len(markReadRes.Paths)).Equals(0)
		})
	})

	t.Run("remove for mark read", func(t *testing.T) {
		// add then remove
		path := classification.Path{
			"domain=example.org",
			"sender=sender2@example.org",
			"on_or_after_01-01-2020",
		}

		_, err := service.AddPathForMarkRead(authedCtx, path)
		assert.ThatError(t, err).IsNil()

		t.Run("unauthenticated, failure", func(t *testing.T) {
			res, err := service.RemovePathForMarkRead(ctx, path)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, res == nil).Equals(true)
		})

		t.Run("success", func(t *testing.T) {
			res, err := service.RemovePathForMarkRead(authedCtx, path)
			assert.ThatError(t, err).IsNil()
			assert.That(t, res.MsgCountDiff).Equals(-3)
			assert.ThatSlice(t, res.Paths[0]).Equals(path)

			expectedPaths := []classification.Path{
				{
					"domain=example.org",
					"sender=sender3@example.org",
					"on_or_after_01-01-2020",
				},
				{
					"domain=mailboxsweep.com",
					"sender=sender1@mailboxsweep.com",
					"on_or_after_01-01-2020",
				},
			}

			markReadSet, markReadCount, err := service.FindByOperation(authedCtx, changeset.MarkRead)
			assert.ThatError(t, err).IsNil()
			assert.That(t, markReadCount).Equals(7)
			assert.ThatSlice(t, markReadSet.Paths).HasLength(2)

			for _, pathStr := range markReadSet.Paths {
				p := classification.ParsePath(pathStr)
				assert.ThatSlice(t, p).HasLength(3)
				found := false
				for _, seedPath := range expectedPaths {
					if slicesAreEqual(seedPath, p) {
						found = true
					}
				}
				assert.That(t, found).Equals(true)
			}
		})

		t.Run("removing a path twice is a no-op", func(t *testing.T) {
			res, err := service.RemovePathForMarkRead(authedCtx, path)
			assert.ThatError(t, err).IsNil()
			assert.That(t, res.MsgCountDiff).Equals(0)
			assert.That(t, len(res.Paths)).Equals(0)
		})
	})

	t.Run("add for delete", func(t *testing.T) {
		paths := []classification.Path{
			{
				"domain=example.org",
				"sender=sender2@example.org",
				"on_or_after_01-01-2020",
			},
			{
				"domain=mailboxsweep.com",
				"sender=sender1@mailboxsweep.com",
				"on_or_after_01-01-2020",
			},
		}

		t.Run("add", func(t *testing.T) {
			for _, p := range paths {
				deletePaths, err := service.AddPathForDelete(authedCtx, p)
				assert.ThatError(t, err).IsNil()
				assert.That(t, len(deletePaths.Paths)).Equals(1)
				assert.ThatSlice(t, deletePaths.Paths[0]).Equals(p)
			}
		})

		t.Run("result is stored", func(t *testing.T) {
			deleteSet, deleteCount, err := service.FindByOperation(authedCtx, changeset.Delete)
			assert.ThatError(t, err).IsNil()
			assert.That(t, deleteCount).Equals(7)
			assert.ThatSlice(t, deleteSet.Paths).HasLength(2)
			for _, pathStr := range deleteSet.Paths {
				p := classification.ParsePath(pathStr)
				assert.ThatSlice(t, p).HasLength(3)
				found := false
				for _, seedPath := range paths {
					if slicesAreEqual(seedPath, p) {
						found = true
					}
				}
				assert.That(t, found).Equals(true)
			}
		})

		t.Run("adding twice is a no-op", func(t *testing.T) {
			deleteRes, err := service.AddPathForDelete(authedCtx, paths[0])
			assert.ThatError(t, err).IsNil()
			assert.That(t, deleteRes.MsgCountDiff).Equals(0)
			assert.That(t, len(deleteRes.Paths)).Equals(0)
		})
	})

	t.Run("remove for delete", func(t *testing.T) {
		// add then remove
		path := classification.Path{
			"domain=example.org",
			"sender=sender3@example.org",
			"on_or_after_01-01-2020",
		}

		_, err := service.AddPathForDelete(authedCtx, path)
		assert.ThatError(t, err).IsNil()

		t.Run("unauthenticated, failure", func(t *testing.T) {
			res, err := service.RemovePathForDelete(ctx, path)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, res == nil).Equals(true)
		})

		t.Run("success", func(t *testing.T) {
			res, err := service.RemovePathForDelete(authedCtx, path)
			assert.ThatError(t, err).IsNil()
			assert.That(t, res.MsgCountDiff).Equals(-3)
			assert.ThatSlice(t, res.Paths[0]).Equals(path)

			expectedPaths := []classification.Path{
				{
					"domain=example.org",
					"sender=sender2@example.org",
					"on_or_after_01-01-2020",
				},
				{
					"domain=mailboxsweep.com",
					"sender=sender1@mailboxsweep.com",
					"on_or_after_01-01-2020",
				},
			}

			deleteSet, deleteCount, err := service.FindByOperation(authedCtx, changeset.Delete)
			assert.ThatError(t, err).IsNil()
			assert.That(t, deleteCount).Equals(7)
			assert.ThatSlice(t, deleteSet.Paths).HasLength(2)

			for _, pathStr := range deleteSet.Paths {
				p := classification.ParsePath(pathStr)
				assert.ThatSlice(t, p).HasLength(3)
				found := false
				for _, seedPath := range expectedPaths {
					if slicesAreEqual(seedPath, p) {
						found = true
					}
				}
				assert.That(t, found).Equals(true)
			}
		})

		t.Run("removing a path twice is a no-op", func(t *testing.T) {
			res, err := service.RemovePathForDelete(authedCtx, path)
			assert.ThatError(t, err).IsNil()
			assert.That(t, res.MsgCountDiff).Equals(0)
			assert.That(t, len(res.Paths)).Equals(0)
		})
	})

	t.Run("Execute: mark read", func(t *testing.T) {
		err := service.CommitChanges(authedCtx, changeset.MarkRead)
		assert.ThatError(t, err).IsNil()

		provider := providerService.Providers[testAddress]
		assert.That(t, provider).NotEqual(nil)

		inbox := provider.Inboxes[testAddress]
		assert.That(t, inbox).NotEqual(nil)

		inboxFolder := inbox.InboxFolders[folder]
		assert.That(t, inboxFolder).NotEqual(nil)

		expectedMarkReadUIDs := []string{
			"2", "5", "8", "0", "3", "6", "9",
		}

		t.Run("no messages have been deleted", func(t *testing.T) {
			count := inboxFolder.Count()
			assert.That(t, count).Equals(uint32(numMessages))
		})

		t.Run("messages have been marked read", func(t *testing.T) {
			assert.ThatSlice(t, inboxFolder.MarkReads).HasLength(7)
			assert.ThatSlice(t, inboxFolder.MarkReads).ContainsAll(expectedMarkReadUIDs)
		})

		t.Run("other messages have not been marked read", func(t *testing.T) {
			for i := '0'; i <= '9'; i++ {
				if !slice.Contains(expectedMarkReadUIDs, string(i)) {
					assert.ThatSlice(t, inboxFolder.MarkReads).NotContains(string(i))
				}
			}
		})

		t.Run("classify request is marked 'executed'", func(t *testing.T) {
			requests, err := classifyRequestService.FindCompletedRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(requests)).Equals(1)
			assert.That(t, requests[0].ExecutedAtUnix).NotEqual(nil)
			assert.ThatOrdered(t, *requests[0].ExecutedAtUnix).GreaterThan(0)
		})

		t.Run("inbox metadata is updated", func(t *testing.T) {
			meta, err := metadataService.Get(authedCtx)
			assert.ThatError(t, err).IsNil()

			t.Run("total count is unchanged", func(t *testing.T) {
				assert.That(t, meta.MessageCount).Equals(numMessages)
			})

			t.Run("unread count is updated", func(t *testing.T) {
				assert.That(t, meta.UnreadCount).Equals(numMessages - len(expectedMarkReadUIDs))
			})
		})

		t.Run("changeset is no longer visible to user (archived)", func(t *testing.T) {
			cs, err := changeSetService.FindByOperation(authedCtx, changeset.MarkRead, true)
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, cs == nil).Equals(true)
		})
	})

	t.Run("Execute: delete", func(t *testing.T) {
		err := service.CommitChanges(authedCtx, changeset.Delete)
		assert.ThatError(t, err).IsNil()

		provider := providerService.Providers[testAddress]
		assert.That(t, provider).NotEqual(nil)

		inbox := provider.Inboxes[testAddress]
		assert.That(t, inbox).NotEqual(nil)

		inboxFolder := inbox.InboxFolders[folder]
		assert.That(t, inboxFolder).NotEqual(nil)

		expectedDeleteUIDs := []string{
			"1", "4", "7", "0", "3", "6", "9",
		}

		t.Run("messages have been deleted", func(t *testing.T) {
			assert.ThatSlice(t, inboxFolder.Deletes).HasLength(len(expectedDeleteUIDs))
			assert.ThatSlice(t, inboxFolder.Deletes).ContainsAll(expectedDeleteUIDs)
		})

		t.Run("other messages have not been deleted", func(t *testing.T) {
			for i := '0'; i <= '9'; i++ {
				if !slice.Contains(expectedDeleteUIDs, string(i)) {
					assert.ThatSlice(t, inboxFolder.Deletes).NotContains(string(i))
				}
			}
		})

		t.Run("classify request is marked 'executed'", func(t *testing.T) {
			requests, err := classifyRequestService.FindCompletedRequests(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(requests)).Equals(1)
			assert.That(t, requests[0].ExecutedAtUnix).NotEqual(nil)
			assert.ThatOrdered(t, *requests[0].ExecutedAtUnix).GreaterThan(0)
		})

		t.Run("inbox metadata is updated", func(t *testing.T) {
			meta, err := metadataService.Get(authedCtx)
			assert.ThatError(t, err).IsNil()

			t.Run("total count is updated", func(t *testing.T) {
				assert.That(t, meta.MessageCount).Equals(numMessages - len(expectedDeleteUIDs))
			})

			t.Run("unread count is updated", func(t *testing.T) {
				assert.That(t, meta.UnreadCount).Equals(0)
			})
		})

		t.Run("changeset is no longer visible to user (archived)", func(t *testing.T) {
			cs, err := changeSetService.FindByOperation(authedCtx, changeset.Delete, true)
			assert.ThatError(t, err).Is(mailboxsweep.ErrNotFound)
			assert.That(t, cs == nil).Equals(true)
		})
	})

}

func slicesAreEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i, amem := range a {
		if amem != b[i] {
			return false
		}
	}

	return true
}
