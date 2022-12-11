package httpserver

import (
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/login"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	*CredentialController
	*ClassificationController
	*FolderController
	*LoginController
	*UserController
}

func NewController(
	credentialService inboxcredential.Service,
	classificationService classification.Service,
	folderService inbox.FolderService,
	providerService *inbox.ProviderService,
	loginService login.Service,
	log *logrus.Logger,
) *Controller {
	credentialController := NewCredentialController(credentialService, providerService, log)
	classificationController := NewClassificationController(classificationService, log)
	folderController := NewFolderController(folderService, log)
	loginController := NewLoginController(loginService, log)

	return &Controller{
		CredentialController:     credentialController,
		ClassificationController: classificationController,
		FolderController:         folderController,
		LoginController:          loginController,
	}
}
