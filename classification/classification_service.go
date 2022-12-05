package classification

import (
	"context"
	"encoding/json"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type Service interface {
	EnqueueClassify(ctx context.Context, address string, folders []string, classifiers []SerializableClassifier) error
	Classify(ctx context.Context, address string, folders []string, classifiers []Classifier) error
	ClassificationSummary(ctx context.Context, address string) (root *ClassificationDocument, err error)
	ClassificationLeaf(ctx context.Context, address string, path []string) (leaf *Classification, err error)
}

type DefaultService struct {
	credentialService inboxcredential.Service
	imapServerService imapserver.Service
	repo              ClassificationRepository
	providerService   *inbox.ProviderService
	bridge            *eventbridge.Client
	eventBusName      string
}

func NewService(
	credentialService inboxcredential.Service,
	imapServerService imapserver.Service,
	repo ClassificationRepository,
	providerService *inbox.ProviderService,
	bridge *eventbridge.Client,
	eventBusName string,
) Service {
	return &DefaultService{
		credentialService,
		imapServerService,
		repo,
		providerService,
		bridge,
		eventBusName,
	}
}

func (s DefaultService) Classify(ctx context.Context, address string, folders []string, classifiers []Classifier) error {
	provider, err := s.providerService.GetProvider(ctx, address)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.FindCredential(ctx, address)
	if err != nil {
		return err
	}

	password, err := s.credentialService.DecryptCredential(ctx, cred)
	if err != nil {
		return err
	}

	mailbox, err := provider.Login(address, password)
	if err != nil {
		return err
	}

	organizer := NewOrganizer(mailbox, s.repo, folders, classifiers)

	return organizer.Organize(ctx)
}

type ClassifyQueueItem struct {
	Address     string                   `json:"address"`
	Folders     []string                 `json:"folders"`
	Classifiers []SerializableClassifier `json:"classifiers"`
}

var ClassifyQueueDetailType = "mailboxsweep.classify"

func (s DefaultService) EnqueueClassify(ctx context.Context, address string, folders []string, classifiers []SerializableClassifier) error {
	req := ClassifyQueueItem{
		Address:     address,
		Folders:     folders,
		Classifiers: classifiers,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = s.bridge.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				Detail:       aws.String(string(reqJSON)),
				DetailType:   &ClassifyQueueDetailType,
				EventBusName: &s.eventBusName,
				Source:       aws.String("classification_default_service"),
				TraceHeader:  aws.String("@TODO-request-id"),
			},
		},
	})
	return err
}

func (s DefaultService) ClassificationSummary(ctx context.Context, address string) (root *ClassificationDocument, err error) {
	return s.repo.FetchSummary(ctx, address)
}

func (s DefaultService) ClassificationLeaf(ctx context.Context, address string, path []string) (leaf *Classification, err error) {
	return s.repo.FetchLeaf(ctx, address, path)
}
