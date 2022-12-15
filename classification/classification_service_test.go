package classification

import (
	"context"
	"encoding/json"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxtesting"
	"mailboxsweep/session"
	"testing"
	"time"
)

type testCredentialReadService struct {
	address  string
	password string
}

func (t testCredentialReadService) FindCredential(ctx context.Context, address string) (*inboxcredential.InboxCredential, error) {
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
	credentialReadService := testCredentialReadService{
		address:  testAddress,
		password: testPass,
	}
	publisher := &mailboxsweep.TestEventPublisher{}
	eventBusName := "mailboxsweeptestevents"

	service := NewService(
		credentialReadService,
		imapServerService,
		repo,
		providerService,
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
			authedCtx := session.AuthorizeTestContext(ctx, testAddress, t)
			err := service.EnqueueClassify(authedCtx, folders, []SerializableClassifier{
				{
					TypeKey: typeKeySenderDomain,
				},
			})
			assert.ThatError(t, err).IsNil()
			assert.That(t, len(publisher.Entries)).Equals(1)
			e := publisher.Entries[0]
			item := ClassifyQueueItem{}
			err = json.Unmarshal([]byte(*e.Detail), &item)
			assert.ThatError(t, err).IsNil()

			assert.That(t, *e.DetailType).Equals(ClassifyQueueDetailType)
			assert.That(t, *e.EventBusName).Equals(eventBusName)
			assert.That(t, *e.Source).Equals("classification_default_service")

			serializedClassifier := item.Classifiers[0]
			assert.That(t, item.Address).Equals(testAddress)
			assert.ThatSlice(t, item.Folders).HasLength(len(folders))
			assert.ThatSlice(t, item.Folders).ContainsAll(folders)
			assert.That(t, len(item.Classifiers)).Equals(1)
			assert.That(t, serializedClassifier.TypeKey).Equals(typeKeySenderDomain)
			assert.That(t, serializedClassifier.InitValue).Equals("")

			cl, err := serializedClassifier.Classifier()
			assert.ThatError(t, err).IsNil()
			senderDomain, ok := cl.(SenderDomainClassifier)
			assert.That(t, ok).Equals(true)
			assert.That(t, senderDomain.Dimension()).Equals("sender_domain")
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
			err := service.Classify(ctx, folders, classifiers)
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
		})

		t.Run("authenticated, success", func(t *testing.T) {
			authedCtx := session.AuthorizeTestContext(ctx, testAddress, t)
			providerService.Messages["INBOX"] = messages
			providerService.Messages["trash"] = []inbox.Message{}
			err := service.Classify(authedCtx, folders, classifiers)
			assert.ThatError(t, err).IsNil()

			summary, err := repo.FetchSummary(ctx, testAddress)
			assert.ThatError(t, err).IsNil()
			assert.That(t, summary.Meta.Created.IsZero()).Equals(false)
			assert.That(t, summary.Classification).NotEqual(nil)

			cl := summary.Classification
			assert.That(t, len(cl.Children)).Equals(len(userNames))
			fmt.Printf("%+v", summary.Classification)
		})

	})

}
