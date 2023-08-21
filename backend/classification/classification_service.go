package classification

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/payment"
	"time"

	"github.com/sirupsen/logrus"
)

type Service interface {
	EnqueueClassify(ctx context.Context, folders []string, classifiers []SerializableClassifier) error
	Classify(ctx context.Context, folders []string, classifiers []Classifier) error
	ClassificationSummary(ctx context.Context, page *int) (root *ClassificationDocument, err error)
	ClassificationLeaf(ctx context.Context, path Path) (leaf *Classification, err error)
	GetLeavesOfPrefix(ctx context.Context, prefix []string) ([]Path, error)
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
	inboxMetadataService   inbox.MetadataService
	inboxUserService       inboxuser.Service
	numSubjectPreviews     int
}

func NewService(
	credentialService inboxcredential.CredentialReadService,
	imapServerService imapserver.Service,
	repo ClassificationRepository,
	providerService inbox.ProviderService,
	classifyRequestService classifyrequest.Service,
	paymentService payment.Service,
	inboxMetadataService inbox.MetadataService,
	inboxUserService inboxuser.Service,
	logger *logrus.Logger,
	pub mailboxsweep.EventPublisher[ClassifyQueueItem],
	numSubjectPreviews int,
) *DefaultService {
	return &DefaultService{
		credentialService,
		imapServerService,
		repo,
		providerService,
		classifyRequestService,
		paymentService,
		logger,
		pub,
		inboxMetadataService,
		inboxUserService,
		numSubjectPreviews,
	}
}

// Runs from the context of a job worker
// also updates the unread count because the IMAP response doesn't contain it
func (s DefaultService) Classify(ctx context.Context, folders []string, classifiers []Classifier) error {
	log := s.logger.
		WithField("service", "classification.Service").
		WithField("method", "classify")

	var err error
	var addr string
	defer func() {
		var recordErr error
		if err != nil {
			recordErr = s.classifyRequestService.MarkCompletedError(ctx, err)
		} else {
			recordErr = s.classifyRequestService.MarkCompletedSuccess(ctx)
		}
		if recordErr != nil {
			log.
				WithError(recordErr).
				Errorf(
					"failed to store result of Classify action: record error: '%s' (original error %v)",
					recordErr,
					err,
				)
		} else {
			log.Info("completed classification")
		}
	}()

	addr, err = mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}
	log = log.WithField("address", addr)
	log.Info("beginning classification")

	// verify user has agreed to terms and conditions
	terms, err := s.inboxUserService.GetAgreement(ctx, inboxuser.TermsAndConditions)
	if err != nil {
		return err
	}
	if !terms.Agreed {
		return mailboxsweep.TermsAndConditionsNotAgreedError()
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

	encryptedCred, err := s.credentialService.FindCredential(ctx)
	if err != nil {
		return err
	}

	// #DEBUG
	log.Infof("encrypted credential of type %T: %+v", encryptedCred, encryptedCred)
	decryptedCred, err := s.credentialService.DecryptCredential(ctx, encryptedCred)
	if err != nil {
		return err
	}
	log.Infof("got decrypted credential %+v", decryptedCred)

	mailbox, err := provider.Login(ctx, addr, decryptedCred)
	if err != nil {
		return fmt.Errorf("failed to login with provider of type %T: %w", provider, err)
	}

	organizer := NewOrganizer(mailbox, s.repo, folders, classifiers, s.numSubjectPreviews)

	classificationRoot, err := organizer.Organize(ctx)
	if err != nil {
		return err
	}

	meta, err := s.inboxMetadataService.Get(ctx)
	if err != nil {
		return err
	}

	meta.UnreadCount = int(classificationRoot.UnreadCount)
	err = s.inboxMetadataService.Set(ctx, *meta)
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

	log := s.logger.WithField("address", address).
		WithField("method", "EnqueueClassify")

	// validate classifiers and input values
	for _, scl := range classifiers {
		_, err := scl.Classifier()
		if err != nil {
			return err
		}
	}
	log = log.WithField("num_classifiers", len(classifiers))
	defer func() {
		if err != nil {
			log.
				WithError(err).
				Errorf("EnqueueClassify failed")
		}
	}()

	// verify user has agreed to terms and conditions
	terms, err := s.inboxUserService.GetAgreement(ctx, inboxuser.TermsAndConditions)
	if err != nil {
		return err
	}
	if !terms.Agreed {
		return mailboxsweep.TermsAndConditionsNotAgreedError()
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
	log = log.
		WithField("completed_scans", len(completedScans)).
		WithField("plan", plan.DisplayName).
		WithField("payment_exp", pmt.ValidUntil)

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

	if pmt.ValidUntil.Before(time.Now()) {
		return mailboxsweep.NeedsPaymentError(fmt.Errorf("plan expired at %s", pmt.ValidUntil.String()))
	}

	// make sure we can log in
	provider, err := s.providerService.GetProvider(ctx, address)
	if err != nil {
		return err
	}

	cred, err := s.credentialService.FindCredential(ctx)
	if err != nil {
		return err
	}

	password, err := s.credentialService.DecryptCredential(ctx, cred)
	if err != nil {
		return err
	}

	_, err = provider.Login(ctx, address, password)
	if err != nil {
		return err
	}

	// store classify request
	err = s.classifyRequestService.RecordRequest(ctx)
	if err != nil {
		return err
	}

	// enqueue work
	req := ClassifyQueueItem{
		Address:     address,
		Folders:     folders,
		Classifiers: classifiers,
	}

	err = s.eventPublisher.PublishEvent(ctx, mailboxsweep.EventBody[ClassifyQueueItem]{
		Body: req,
	})
	return err
}

func (s DefaultService) ClassificationSummary(ctx context.Context, page *int) (root *ClassificationDocument, err error) {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if page != nil {
		return s.repo.FetchSummary(ctx, address, *page)
	}

	pageCount := 1
	pages := []ClassificationDocument{}
	for i := 0; i < pageCount; i++ {
		page, err := s.repo.FetchSummary(ctx, address, i)
		if err != nil {
			return nil, err
		}
		pages = append(pages, *page)
		pageCount = page.TotalPages
	}

	merged := mergePaginatedDocs(pages)
	return &merged, nil
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

func (s DefaultService) GetLeavesOfPrefix(ctx context.Context, prefix []string) ([]Path, error) {
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return s.repo.GetLeavesOfPrefix(ctx, address, prefix)
}
