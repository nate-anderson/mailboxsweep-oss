package httpserver

import (
	"mailboxsweep/inbox"
	"mailboxsweep/session"
	"net/http"

	"github.com/sirupsen/logrus"
)

type FolderController struct {
	folderService inbox.FolderService
	log           *logrus.Logger
	baseController
}

func NewFolderController(folderService inbox.FolderService, log *logrus.Logger) *FolderController {
	return &FolderController{
		folderService,
		log,
		baseController{log},
	}
}

func (c FolderController) GetInboxFolders(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	folders, err := c.folderService.ListInboxFolders(ctx)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadGateway))
		return
	}

	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		c.log.Errorf("no address associated with context inside handler - THIS IS VERY BAD")
		c.respondAuthError(w, err)
		return
	}

	resBody := GetFoldersResponse{
		Address: addr,
		Folders: folders,
	}
	c.respond(w, http.StatusOK, resBody)
}
