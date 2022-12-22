package httpserver

import (
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/inboxuser"
	"mailboxsweep/login"
	"mailboxsweep/payment"

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
}

func NewController(
	inboxUserService inboxuser.Service,
	credentialService inboxcredential.Service,
	classificationService classification.Service,
	folderService inbox.FolderService,
	providerService inbox.ProviderService,
	loginService login.Service,
	changesetService changeset.Service,
	paymentService payment.Service,
	cookieDomain string,
	cookiePath string,
	isDev bool,
	log *logrus.Logger,
) *Controller {
	credentialController := NewCredentialController(credentialService, providerService, log)
	classificationController := NewClassificationController(classificationService, log)
	folderController := NewFolderController(folderService, log)
	loginController := NewLoginController(loginService, cookieDomain, cookiePath, isDev, log)
	changeSetController := NewChangeSetController(changesetService, log)
	userController := NewUserController(inboxUserService, log)
	paymentController := NewPaymentController(paymentService, log)

	return &Controller{
		CredentialController:     credentialController,
		ClassificationController: classificationController,
		FolderController:         folderController,
		LoginController:          loginController,
		ChangeSetController:      changeSetController,
		UserController:           userController,
		PaymentController:        paymentController,
	}
}
