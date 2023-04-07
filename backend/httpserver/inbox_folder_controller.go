package httpserver

import (
	"mailboxsweep/inbox"
	"net/http"

	"github.com/sirupsen/logrus"
)

type FolderController struct {
	metadataService inbox.MetadataService
	log             *logrus.Logger
	baseController
}

func NewFolderController(metadataService inbox.MetadataService, log *logrus.Logger) *FolderController {
	return &FolderController{
		metadataService,
		log,
		baseController{log},
	}
}

func (c FolderController) GetInboxMetadata(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	meta, err := c.metadataService.Get(ctx)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := InboxMetadataResponse{
		Address:      meta.Inbox,
		MessageCount: meta.MessageCount,
		UnreadCount:  meta.UnreadCount,
		Folders:      meta.Folders,
	}
	c.respond(w, http.StatusOK, resBody)
}
