package httpserver

import (
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxchangeset"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/login"
	"mailboxsweep/payment"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	*ChangeSetController
	*CredentialController
	*ClassificationController
	*FolderController
	*LoginController
	*UserController
	*PaymentController
	*GmailReauthController
}

func NewController(
	inboxUserService inboxuser.Service,
	credentialService inboxcredential.Service,
	classifyRequestService classifyrequest.Service,
	classificationService classification.Service,
	metadataService inbox.MetadataService,
	providerService inbox.ProviderService,
	gmailLoginService *login.GmailLoginService,
	loginService *login.Service,
	changesetService changeset.Service,
	inboxChangesetService inboxchangeset.Service,
	imapServerService imapserver.Service,
	paymentService payment.Service,
	cookieDomain string,
	cookiePath string,
	isDev bool,
	log *logrus.Logger,
) *Controller {
	credentialController := NewCredentialController(credentialService, providerService, imapServerService, log)
	classificationController := NewClassificationController(classificationService, classifyRequestService, log)
	folderController := NewFolderController(metadataService, log)
	loginController := NewLoginController(loginService, cookieDomain, cookiePath, isDev, log)
	changeSetController := NewChangeSetController(changesetService, inboxChangesetService, log)
	userController := NewUserController(inboxUserService, log)
	paymentController := NewPaymentController(paymentService, log)
	gmailReauthController := NewGmailReauthController(gmailLoginService, cookieDomain, cookiePath, log)

	return &Controller{
		CredentialController:     credentialController,
		ClassificationController: classificationController,
		FolderController:         folderController,
		LoginController:          loginController,
		ChangeSetController:      changeSetController,
		UserController:           userController,
		PaymentController:        paymentController,
		GmailReauthController:    gmailReauthController,
	}
}

func (c Controller) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
