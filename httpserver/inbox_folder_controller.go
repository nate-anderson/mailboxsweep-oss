package httpserver

import (
	"mailboxsweep/inbox"
	"net/http"

	"github.com/sirupsen/logrus"
)

type FolderController struct {
	folderService inbox.FolderService
	log           *logrus.Logger
}

func NewFolderController(folderService inbox.FolderService, log *logrus.Logger) *FolderController {
	return &FolderController{
		folderService,
		log,
	}
}

// @TODO: remove this once auth is working, this has no place in an authed system
var testAddressHeader = "X-Test-Address"

func (c FolderController) GetInboxFolders(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// @TODO: get this from auth context
	addr := req.Header.Get(testAddressHeader)

	folders, err := c.folderService.ListInboxFolders(ctx, addr)
	if err != nil {
		respondError(w, c.log, err, withStatusCode(http.StatusBadGateway))
		return
	}

	resBody := GetFoldersResponse{
		Address: addr,
		Folders: folders,
	}
	respond(w, http.StatusOK, resBody)
}
