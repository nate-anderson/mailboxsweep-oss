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
	"mailboxsweep/inboxtesting"
	"mailboxsweep/inboxuser"
	"mailboxsweep/payment"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sirupsen/logrus"
)

type testImapServerService struct {
	domain string
	host   string
	port   int
}

func (t testImapServerService) Find(ctx context.Context) (*imapserver.Server, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	domain, err := mailboxsweep.DomainFromEmail(addr)
	if err != nil {
		return nil, err
	}

	if domain != t.domain {
		return nil, mailboxsweep.ErrNotFound
	}

	return &imapserver.Server{
		Domain: t.domain,
		Host:   t.host,
		Port:   t.port,
	}, nil
}

func (t testImapServerService) Store(ctx context.Context, server imapserver.Server) error {
	return errors.New("don't call this method")
}

type testRequest struct {
	started        int64
	completed      *int64
	executeStarted *int64
	executed       *int64
	err            error
	count          int
	operation      *string
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

func (r *testRequestRecorder) FindCompletedRequests(ctx context.Context) ([]classifyrequest.ClassifyRequest, error) {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, ok := r.requests[addr]
	if !ok {
		return []classifyrequest.ClassifyRequest{}, nil
	}
	var reqs []classifyrequest.ClassifyRequest
	for _, l := range list {
		if l.completed != nil {
			reqs = append(reqs, classifyrequest.ClassifyRequest{
				Inbox:           addr,
				UserUUID:        "UNIMPLEMENTED_FOR_TEST",
				RequestedAt:     l.started,
				CompletedAtUnix: l.completed,
			})
		}
	}
	return reqs, nil
}

func (r *testRequestRecorder) MarkCompletedSuccess(ctx context.Context) error {
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

func (r *testRequestRecorder) MarkCompletedError(ctx context.Context, classifyErr error) error {
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

func (r *testRequestRecorder) MarkExecuting(ctx context.Context, operation string) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	list, ok := r.requests[addr]
	if !ok {
		return mailboxsweep.ErrNotFound
	}
	for i, l := range list {
		if l.completed != nil {
			updated := l
			updated.executeStarted = aws.Int64(time.Now().Unix())
			updated.operation = &operation
			list[i] = updated
			r.requests[addr] = list
			return nil
		}
	}
	return mailboxsweep.ErrNotFound
}

func (r *testRequestRecorder) MarkExecutedSuccess(ctx context.Context, count int) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	list, ok := r.requests[addr]
	if !ok {
		return mailboxsweep.ErrNotFound
	}
	for i, l := range list {
		if l.completed != nil {
			updated := l
			updated.executed = aws.Int64(time.Now().Unix())
			updated.count = count

			list[i] = updated
			r.requests[addr] = list
			return nil
		}
	}
	return mailboxsweep.ErrNotFound
}

func (r *testRequestRecorder) MarkExecutedError(ctx context.Context, count int, classifyErr error) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	list, ok := r.requests[addr]
	if !ok {
		return mailboxsweep.ErrNotFound
	}
	for i, l := range list {
		if l.completed != nil {
			updated := l
			updated.executed = aws.Int64(time.Now().Unix())
			updated.err = classifyErr
			updated.count = count
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

	ctx := context.Background()

	s3Client, bucket, teardown := mailboxsweep.SetupTestBucket(ctx, t)
	defer teardown()

	db := mailboxsweep.SetupTestDynamoClient()
	table, dynamoTeardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer dynamoTeardown()

	repo := NewS3ClassificationRepository(s3Client, bucket, 2)

	providerService := inboxtesting.NewTestProviderService()
	credentialReadService := &inboxtesting.TestCredentialReadService{
		Address:  testAddress,
		Password: testPass,
	}
	publisher := &mailboxsweep.TestEventPublisher[ClassifyQueueItem]{}

	classifyRequestService := testRequestRecorder{
		requests: make(map[string][]testRequest),
	}
	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	userService, err := inboxuser.NewService(userRepo)
	assert.ThatError(t, err).IsNil()

	_, err = userService.Store(ctx, testAddress)
	assert.ThatError(t, err).IsNil()

	cartRepo := payment.NewCartRepository(db, table)
	proc := payment.NewTestProcessor()
	paymentService, err := payment.NewService(
		userService,
		proc,
		cartRepo,
		time.Hour*30*24,
	)
	assert.ThatError(t, err).IsNil()

	inboxMetadataRepo := inbox.NewMetadataRepository(db, table)
	inboxMetadataService := inbox.NewMetadataService(inboxMetadataRepo, providerService, credentialReadService, time.Minute)

	service := NewService(
		credentialReadService,
		repo,
		providerService,
		&classifyRequestService,
		paymentService,
		inboxMetadataService,
		userService,
		logrus.New(),
		publisher,
		5,
	)

	authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddress, t)

	assert.ThatError(t, err).IsNil()

	t.Run("enqueue classification", func(t *testing.T) {
		folders := []string{"INBOX", "spam"}
		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.EnqueueClassify(ctx, folders, []SerializableClassifier{
				{
					TypeKey: TypeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, len(publisher.Entries)).Equals(0)
		})

		t.Run("authenticated, terms and conditions not agreed, failure", func(t *testing.T) {
			err := service.EnqueueClassify(authedCtx, folders, []SerializableClassifier{
				{
					TypeKey: TypeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).Is(mailboxsweep.ErrTermsAndConditionsNotAgreed)
			err = userService.Consent(authedCtx, []inboxuser.Agreement{inboxuser.TermsAndConditions})
			assert.ThatError(t, err).IsNil()
		})

		t.Run("authenticated, not paid, failure", func(t *testing.T) {
			err := service.EnqueueClassify(authedCtx, folders, []SerializableClassifier{
				{
					TypeKey: TypeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).Is(mailboxsweep.ErrNeedsPayment)
		})

		// t.Run("authenticated, terms not accepted, failure", func(t *testing.T) {})

		t.Run("authenticated, paid, success", func(t *testing.T) {
			err = userService.StorePayment(authedCtx, inboxuser.Payment{
				Plan: inboxuser.Basic,

				AmountInCents: 1000,
				Method:        inboxuser.Stripe,
				ExternalID:    "12345",
				Timestamp:     time.Now(),
			})
			assert.ThatError(t, err).IsNil()
			err = service.EnqueueClassify(authedCtx, folders, []SerializableClassifier{
				{
					TypeKey: TypeKeySenderDomain,
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
			assert.That(t, serializedClassifier.TypeKey).Equals(TypeKeySenderDomain)
			assert.That(t, serializedClassifier.InitValue).Equals("")

			cl, err := serializedClassifier.Classifier()
			assert.ThatError(t, err).IsNil()
			senderDomain, ok := cl.(SenderDomainClassifier)
			assert.That(t, ok).Equals(true)
			assert.That(t, senderDomain.Dimension()).Equals("sender_domain")

			_, ok = classifyRequestService.requests[testAddress]
			assert.That(t, ok).Equals(true)
		})

		t.Run("authenticated, paid, limits met", func(t *testing.T) {
			tooManyAddr := "bumg@tots.biz"
			_, err := userService.Store(ctx, tooManyAddr)
			assert.ThatError(t, err).IsNil()
			exceededCtx := mailboxsweep.AuthorizeTestContext(ctx, tooManyAddr, t)
			err = userService.Consent(exceededCtx, []inboxuser.Agreement{inboxuser.TermsAndConditions})
			assert.ThatError(t, err).IsNil()

			// store successful payment
			err = userService.StorePayment(exceededCtx, inboxuser.Payment{
				Plan: inboxuser.Basic,

				AmountInCents: 1000,
				Method:        inboxuser.Stripe,
				ExternalID:    "12345",
				Timestamp:     time.Now().Add(time.Hour * -24),
			})
			assert.ThatError(t, err).IsNil()

			// store some extra completed requests
			plan, err := paymentService.PlanDetails(inboxuser.Basic)
			assert.ThatError(t, err).IsNil()
			for i := 0; i <= plan.AllowedScans; i++ {
				completed := time.Now().Add(time.Hour*-1).Unix() + int64(i)
				r := testRequest{
					started:   time.Now().Add(time.Minute * -1 * time.Duration(i)).Unix(),
					completed: &completed,
					err:       nil,
				}
				classifyRequestService.requests[tooManyAddr] = append(classifyRequestService.requests[tooManyAddr], r)
			}

			err = service.EnqueueClassify(exceededCtx, folders, []SerializableClassifier{
				{
					TypeKey: TypeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).Is(mailboxsweep.ErrPlanLimitMet)
		})
	})

	t.Run("classify", func(t *testing.T) {
		folders := []string{"INBOX", "trash"}
		n := 32

		// store inbox metadata
		err = inboxMetadataService.Set(authedCtx, inbox.Metadata{
			Inbox:        testAddress,
			Folders:      folders,
			MessageCount: n,
			Expires:      time.Now().Add(time.Hour).Unix(),
		})

		classifiers := []Classifier{
			NewSenderUsernameClassifier(),
		}

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
				UID:     strconv.Itoa(i),
				Folder:  "INBOX",
				Subject: fmt.Sprintf("msg %d", i),
				Sender: inbox.Sender{
					Address:  sender,
					Name:     un,
					Username: un,
					Domain:   "gmail.com",
				},
				SendDate: time.Now().Add(time.Hour * 24 * 365 * -1 * time.Duration(i)),
				Read:     i%2 == 0,
			}
			messages[i] = msg
		}

		t.Run("unauthenticated, failure", func(t *testing.T) {
			err := service.Classify(ctx, folders, classifiers)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		authedCtx := mailboxsweep.AuthorizeTestContext(ctx, testAddress, t)
		providerService.Messages["INBOX"] = messages
		providerService.Messages["trash"] = []inbox.Message{}

		t.Run("authenticated, terms and conditions not agreed, failure", func(t *testing.T) {
			unagreedAddr := "me0@me.com"
			_, err := userService.Store(ctx, unagreedAddr)
			assert.ThatError(t, err).IsNil()
			unagreedCtx := mailboxsweep.AuthorizeJobContext(ctx, unagreedAddr)
			err = service.Classify(unagreedCtx, folders, classifiers)
			assert.ThatError(t, err).Is(mailboxsweep.ErrTermsAndConditionsNotAgreed)
		})

		t.Run("not paid, failure", func(t *testing.T) {
			unpaidAddr := "me1@me.com"
			_, err := userService.Store(ctx, unpaidAddr)
			assert.ThatError(t, err).IsNil()
			unpaidCtx := mailboxsweep.AuthorizeJobContext(ctx, unpaidAddr)
			err = userService.Consent(unpaidCtx, []inboxuser.Agreement{inboxuser.TermsAndConditions})
			assert.ThatError(t, err).IsNil()
			err = service.Classify(unpaidCtx, folders, classifiers)
			assert.ThatError(t, err).Is(mailboxsweep.ErrNeedsPayment)
		})

		err = userService.StorePayment(authedCtx, inboxuser.Payment{})
		assert.ThatError(t, err).IsNil()

		t.Run("upstream error, failure stored in request", func(t *testing.T) {
			/*
				Test that, if Classify fails, the classifyRequest record is updated
				with a completed timestamp and the details of the failure

			*/

			// make a dependency fail
			credentialReadService.FindError = errors.New("intentional failure")

			err := service.Classify(authedCtx, folders, classifiers)
			assert.ThatError(t, err).IsNotNil()
			credentialReadService.FindError = nil

			reqList, ok := classifyRequestService.requests[testAddress]
			assert.That(t, ok).Equals(true)
			assert.That(t, len(reqList)).Equals(1)
			r := reqList[0]
			assert.ThatError(t, r.err).IsNotNil()
			assert.That(t, r.err.Error()).Equals("intentional failure")
			assert.ThatOrdered(t, *r.completed).GreaterThan(0)
		})

		senderCounts := map[string]int{
			"sender=me@gmail.com":   11,
			"sender=you@gmail.com":  11,
			"sender=them@gmail.com": 10,
		}

		userCounts := map[string]int{
			"me (me@gmail.com)":     11,
			"you (you@gmail.com)":   11,
			"them (them@gmail.com)": 10,
		}

		t.Run("authenticated, success", func(t *testing.T) {
			// store new request, last one has been marked a failure
			err := classifyRequestService.RecordRequest(authedCtx)
			assert.ThatError(t, err).IsNil()

			err = service.Classify(authedCtx, folders, classifiers)
			assert.ThatError(t, err).IsNil()

			summary, err := service.ClassificationSummary(authedCtx, nil)
			assert.ThatError(t, err).IsNil()
			assert.That(t, summary.Meta.Created.IsZero()).Equals(false)
			assert.That(t, summary.Classification).NotEqual(nil)

			// summary children are sorted
			assert.
				ThatOrdered(t, *summary.Classification.Children["sender=me@gmail.com"].RankIndex).
				LessThan(*summary.Classification.Children["sender=them@gmail.com"].RankIndex)

			assert.
				ThatOrdered(t, *summary.Classification.Children["sender=you@gmail.com"].RankIndex).
				LessThan(*summary.Classification.Children["sender=them@gmail.com"].RankIndex)

			assert.
				That(t, *summary.Classification.Children["sender=me@gmail.com"].RankIndex).
				NotEqual(*summary.Classification.Children["sender=you@gmail.com"].RankIndex)

			cl := summary.Classification
			assert.That(t, len(cl.Children)).Equals(len(userNames))

			for _, ch := range cl.Children {
				assert.That(t, ch.Count).Equals(uint(userCounts[ch.Label]))
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
			assert.That(t, r.operation).Equals(nil)

			// metadata is updated with unread count
			meta, err := inboxMetadataService.Get(authedCtx)
			assert.ThatError(t, err).IsNil()

			assert.That(t, meta.UnreadCount).Equals(n / 2)
			assert.That(t, meta.MessageCount).Equals(n)
		})

		for path, count := range senderCounts {
			leaf, err := repo.FetchLeaf(ctx, testAddress, []string{path})
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(leaf.Messages)).Equals(count)
			assert.That(t, leaf.Count).Equals(uint(count))
			assert.That(t, len(leaf.Children)).Equals(0)

			addr := strings.TrimPrefix(path, "sender=")
			for _, m := range leaf.Messages {
				assert.That(t, m.Sender.Address).Equals(addr)
				assert.That(t, m.Sender.Domain).Equals("gmail.com")
			}
		}
	})

}
