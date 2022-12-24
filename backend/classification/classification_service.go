package classification

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/payment"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/sirupsen/logrus"
)

type Service interface {
	EnqueueClassify(ctx context.Context, folders []string, classifiers []SerializableClassifier) error
	Classify(ctx context.Context, folders []string, classifiers []Classifier) error
	ClassificationSummary(ctx context.Context) (root *ClassificationDocument, err error)
	ClassificationLeaf(ctx context.Context, path Path) (leaf *Classification, err error)
	ListClassifiers() []ClassifierInfo
}

type DefaultService struct {
	credentialService      inboxcredential.CredentialReadService
	imapServerService      imapserver.Service
	repo                   ClassificationRepository
	providerService        inbox.ProviderService
	classifyRequestService classifyrequest.Service
	paymentService         payment.Service
	logger                 *logrus.Logger
	eventPublisher         mailboxsweep.EventPublisher[ClassifyQueueItem]
	eventBusName           string
}

func NewService(
	credentialService inboxcredential.CredentialReadService,
	imapServerService imapserver.Service,
	repo ClassificationRepository,
	providerService inbox.ProviderService,
	classifyRequestService classifyrequest.Service,
	paymentService payment.Service,
	logger *logrus.Logger,
	bridge mailboxsweep.EventPublisher[ClassifyQueueItem],
	eventBusName string,
) *DefaultService {
	return &DefaultService{
		credentialService,
		imapServerService,
		repo,
		providerService,
		classifyRequestService,
		paymentService,
		logger,
		bridge,
		eventBusName,
	}
}

// Runs from the context of a job worker
func (s DefaultService) Classify(ctx context.Context, folders []string, classifiers []Classifier) error {
	var err error
	var addr string
	defer func() {
		var recordErr error
		if err != nil {
			recordErr = s.classifyRequestService.CompleteError(ctx, err)
		} else {
			recordErr = s.classifyRequestService.CompleteSuccess(ctx)
		}
		if recordErr != nil {
			s.logger.
				WithError(recordErr).
				WithField("address", addr).
				Errorf(
					"failed to store result of Classify action: record error: '%s' (original error %v)",
					recordErr,
					err,
				)
		}
	}()

	addr, err = mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	_, err = s.paymentService.VerifyPayment(ctx)
	if err != nil {
		return err
	}

	// confirm there is an outstanding request from the
	_, err = s.classifyRequestService.FindOpenRequests(ctx)
	if err != nil {
		return err
	}

	provider, err := s.providerService.GetProvider(ctx, addr)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.FindCredential(ctx, addr)
	if err != nil {
		return err
	}

	password, err := s.credentialService.DecryptCredential(ctx, cred)
	if err != nil {
		return err
	}

	mailbox, err := provider.Login(addr, password)
	if err != nil {
		return err
	}

	organizer := NewOrganizer(mailbox, s.repo, folders, classifiers)

	err = organizer.Organize(ctx)
	return err
}

type ClassifyQueueItem struct {
	Address     string                   `json:"address"`
	Folders     []string                 `json:"folders"`
	Classifiers []SerializableClassifier `json:"classifiers"`
}

var ClassifyQueueDetailType = "mailboxsweep.classify"

func (s DefaultService) EnqueueClassify(ctx context.Context, folders []string, classifiers []SerializableClassifier) error {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	// make sure user has paid
	pmt, err := s.paymentService.VerifyPayment(ctx)
	if err != nil {
		return err
	}

	// make sure user has not already met their inbox scan quota
	// only consider successful scans since last payment was received
	plan, err := s.paymentService.PlanDetails(pmt.Plan)
	if err != nil {
		return err
	}
	completedScans, err := s.classifyRequestService.FindCompletedRequests(ctx)
	if err != nil {
		return err
	}

	planStartedTimestamp := pmt.StartedAt.Unix()
	successfulScans := 0

	for _, scan := range completedScans {
		if scan.CompletedAtUnix == nil {
			continue
		}
		if scan.Error == nil && *scan.CompletedAtUnix > planStartedTimestamp {
			successfulScans++
		}
	}

	if successfulScans >= plan.AllowedScans {
		err = fmt.Errorf("plan limit of %d scans has been met", plan.AllowedScans)
		return mailboxsweep.PlanLimitMetError(err)
	}

	req := ClassifyQueueItem{
		Address:     address,
		Folders:     folders,
		Classifiers: classifiers,
	}

	// store classify request
	err = s.classifyRequestService.RecordRequest(ctx)
	if err != nil {
		return err
	}

	err = s.eventPublisher.PublishEvent(ctx, mailboxsweep.EventBody[ClassifyQueueItem]{
		RequestID: aws.String("@TODO"),
		Body:      req,
	})
	return err
}

func (s DefaultService) ClassificationSummary(ctx context.Context) (root *ClassificationDocument, err error) {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.FetchSummary(ctx, address)
}

func (s DefaultService) ClassificationLeaf(ctx context.Context, path Path) (leaf *Classification, err error) {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.FetchLeaf(ctx, address, path)
}

func (s DefaultService) ListClassifiers() []ClassifierInfo {
	info := make([]ClassifierInfo, len(allClassifiers))
	for i, cl := range allClassifiers {
		info[i] = cl.Info()
	}
	return info
}
