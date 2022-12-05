package httpserver

import (
	"mailboxsweep/classification"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	*CredentialController
	*ClassificationController
	*FolderController
}

func NewController(
	credentialService inboxcredential.Service,
	classificationService classification.Service,
	folderService inbox.FolderService,
	log *logrus.Logger,
) *Controller {
	credentialController := NewCredentialController(credentialService, log)
	classificationController := NewClassificationController(classificationService, log)
	folderController := NewFolderController(folderService, log)

	return &Controller{
		CredentialController:     credentialController,
		ClassificationController: classificationController,
		FolderController:         folderController,
	}
}
