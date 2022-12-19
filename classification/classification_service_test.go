package classification

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxtesting"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sirupsen/logrus"
)

type testCredentialReadService struct {
	address  string
	password string

	findError error
}

func (t testCredentialReadService) FindCredential(ctx context.Context, address string) (*inboxcredential.InboxCredential, error) {
	if t.findError != nil {
		return nil, t.findError
	}

	if address != t.address {
		return nil, mailboxsweep.ErrNotFound
	}

	return &inboxcredential.InboxCredential{
		Address:  address,
		Password: t.password,
		Created:  time.Now(),
		Expires:  time.Now().Add(time.Hour).Unix(),
	}, nil
}

func (t testCredentialReadService) DecryptCredential(ctx context.Context, cred *inboxcredential.InboxCredential) (password string, err error) {
	return cred.Password, nil
}

type testImapServerService struct {
	domain string
	host   string
	port   int
}

func (t testImapServerService) FindByDomain(ctx context.Context, domain string) (*imapserver.Server, error) {
	if domain != t.domain {
		return nil, mailboxsweep.ErrNotFound
	}

	return &imapserver.Server{
		Domain: t.domain,
		Host:   t.host,
		Port:   t.port,
	}, nil
}

type testRequest struct {
	started   int64
	completed *int64
	err       error
}

type testRequestRecorder struct {
	requests map[string][]testRequest
}

func (r *testRequestRecorder) RecordRequest(ctx context.Context) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	r.requests[addr] = append(r.requests[addr], testRequest{
		started: time.Now().Unix(),
	})
	return nil
}

func (r *testRequestRecorder) FindOpenRequests(ctx context.Context) ([]classifyrequest.ClassifyRequest, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, ok := r.requests[addr]
	if !ok {
		return nil, mailboxsweep.ErrNotFound
	}
	var reqs []classifyrequest.ClassifyRequest
	for _, l := range list {
		if l.completed == nil {
			reqs = append(reqs, classifyrequest.ClassifyRequest{
				Inbox:       addr,
				UserUUID:    "UNIMPLEMENTED_FOR_TEST",
				RequestedAt: l.started,
			})
		}
	}
	return reqs, nil
}

func (r *testRequestRecorder) CompleteSuccess(ctx context.Context) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	list, ok := r.requests[addr]
	if !ok {
		return mailboxsweep.ErrNotFound
	}
	for i, l := range list {
		if l.completed == nil {
			updated := l
			updated.completed = aws.Int64(time.Now().Unix())
			list[i] = updated
			r.requests[addr] = list
			return nil
		}
	}
	return mailboxsweep.ErrNotFound
}

func (r *testRequestRecorder) CompleteError(ctx context.Context, classifyErr error) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	list, ok := r.requests[addr]
	if !ok {
		return mailboxsweep.ErrNotFound
	}
	for i, l := range list {
		if l.completed == nil {
			updated := l
			updated.completed = aws.Int64(time.Now().Unix())
			updated.err = classifyErr
			list[i] = updated
			r.requests[addr] = list
			return nil
		}
	}
	return mailboxsweep.ErrNotFound
}

/*
This is a big ugly test,
but at least it's effectively a true unit test!
the only unmocked dependency is the classification repo
*/

func TestClassificationService(t *testing.T) {
	testAddress := "bigservicetest@service.org"
	testPass := "123boogie567"
	testHost := "imap.mail.org"
	testDomain := "mail.org"
	testPort := 474

	ctx := context.Background()

	s3Client, bucket, teardown := mailboxsweep.SetupTestBucket(ctx, t)
	defer teardown()

	repo := NewS3ClassificationRepository(s3Client, bucket)

	imapServerService := testImapServerService{
		domain: testDomain,
		host:   testHost,
		port:   testPort,
	}
	providerService := inboxtesting.NewTestProviderService()
	credentialReadService := &testCredentialReadService{
		address:  testAddress,
		password: testPass,
	}
	publisher := &mailboxsweep.TestEventPublisher[ClassifyQueueItem]{}
	eventBusName := "mailboxsweeptestevents"

	classifyRequestService := testRequestRecorder{
		requests: make(map[string][]testRequest),
	}

	service := NewService(
		credentialReadService,
		imapServerService,
		repo,
		providerService,
		&classifyRequestService,
		logrus.New(),
		publisher,
		eventBusName,
	)

	t.Run("enqueue classification", func(t *testing.T) {
		folders := []string{"INBOX", "spam"}
		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.EnqueueClassify(ctx, folders, []SerializableClassifier{
				{
					TypeKey: typeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(publisher.Entries)).Equals(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddress, t)
			err := service.EnqueueClassify(authedCtx, folders, []SerializableClassifier{
				{
					TypeKey: typeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(publisher.Entries)).Equals(1)
			e := publisher.Entries[0]

			assert.That(t, e.Body.Address).Equals(testAddress)
			assert.That(t, len(e.Body.Classifiers)).Equals(1)
			assert.ThatSlice(t, e.Body.Folders).ContainsAll(folders)

			serializedClassifier := e.Body.Classifiers[0]
			assert.That(t, e.Body.Address).Equals(testAddress)
			assert.ThatSlice(t, e.Body.Folders).HasLength(len(folders))
			assert.ThatSlice(t, e.Body.Folders).ContainsAll(folders)
			assert.That(t, len(e.Body.Classifiers)).Equals(1)
			assert.That(t, serializedClassifier.TypeKey).Equals(typeKeySenderDomain)
			assert.That(t, serializedClassifier.InitValue).Equals("")

			cl, err := serializedClassifier.Classifier()
			assert.ThatError(t, err).IsNil()
			senderDomain, ok := cl.(SenderDomainClassifier)
			assert.That(t, ok).Equals(true)
			assert.That(t, senderDomain.Dimension()).Equals("sender_domain")

			_, ok = classifyRequestService.requests[testAddress]
			assert.That(t, ok).Equals(true)
		})
	})

	t.Run("classify", func(t *testing.T) {
		folders := []string{"INBOX", "trash"}
		classifiers := []Classifier{
			NewSenderUsernameClassifier(),
		}

		n := 30
		userNames := []string{
			"me",
			"you",
			"them",
		}

		messages := make([]inbox.Message, n)
		for i := 0; i < n; i++ {
			un := userNames[i%3]
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

		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.Classify(ctx, testAddress, folders, classifiers)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddress, t)
		providerService.Messages["INBOX"] = messages
		providerService.Messages["trash"] = []inbox.Message{}

		t.Run("upstream error, failure stored in request", func(t *testing.T) {
			/*
				Test that, if Classify fails, the classifyRequest record is updated
				with a completed timestamp and the details of the failure

			*/

			// make a dependency fail
			credentialReadService.findError = errors.New("intentional failure")

			err := service.Classify(authedCtx, testAddress, folders, classifiers)
			assert.ThatError(t, err).IsNotNil()
			credentialReadService.findError = nil

			reqList, ok := classifyRequestService.requests[testAddress]
			assert.That(t, ok).Equals(true)
			assert.That(t, len(reqList)).Equals(1)
			r := reqList[0]
			assert.ThatError(t, r.err).IsNotNil()
			assert.That(t, r.err.Error()).Equals("intentional failure")
			assert.ThatOrdered(t, *r.completed).GreaterThan(0)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			// store new request, last one has been marked a failure
			err := classifyRequestService.RecordRequest(authedCtx)
			assert.ThatError(t, err).IsNil()

			err = service.Classify(authedCtx, testAddress, folders, classifiers)
			assert.ThatError(t, err).IsNil()

			summary, err := repo.FetchSummary(ctx, testAddress)
			assert.ThatError(t, err).IsNil()
			assert.That(t, summary.Meta.Created.IsZero()).Equals(false)
			assert.That(t, summary.Classification).NotEqual(nil)

			cl := summary.Classification
			assert.That(t, len(cl.Children)).Equals(len(userNames))

			for _, ch := range cl.Children {
				assert.That(t, ch.Count).Equals(10)
				assert.That(t, len(ch.Children)).Equals(0)
			}

			// request is marked completed
			reqs, ok := classifyRequestService.requests[testAddress]
			assert.That(t, ok).Equals(true)
			assert.That(t, len(reqs)).Equals(2)
			var r testRequest
			numErrs := 0
			for _, req := range reqs {
				if req.err == nil {
					r = req
					numErrs++
				}
			}
			assert.That(t, numErrs).Equals(1)
			assert.ThatOrdered(t, *r.completed).GreaterThan(0)
			assert.ThatError(t, r.err).IsNil()
		})

		leafPaths := []string{
			"sender=me@gmail.com",
			"sender=you@gmail.com",
			"sender=them@gmail.com",
		}

		for _, path := range leafPaths {
			leaf, err := repo.FetchLeaf(ctx, testAddress, []string{path})
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(leaf.Messages)).Equals(10)
			assert.That(t, leaf.Count).Equals(10)
			assert.That(t, len(leaf.Children)).Equals(0)

			addr := strings.TrimPrefix(path, "sender=")
			for _, m := range leaf.Messages {
				assert.That(t, m.Sender.Address).Equals(addr)
				assert.That(t, m.Sender.Domain).Equals("gmail.com")
			}
		}
	})

}
