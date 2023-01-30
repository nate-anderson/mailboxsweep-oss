package httpserver

import (
	"mailboxsweep"
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

func (c FolderController) GetInboxFolders(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	meta, err := c.metadataService.Get(ctx)
	if err != nil {
		c.respondError(w, err)
		return
	}

	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.log.Errorf("no address associated with context inside handler - THIS IS VERY BAD")
		c.respondAuthError(w, err)
		return
	}

	folders := make([]FolderMessageCount, len(meta.Folders))
	for folder, counts := range meta.Folders {
		folders = append(folders, FolderMessageCount{
			Folder:         folder,
			AllMessages:    int(counts.All),
			UnreadMessages: int(counts.Unread),
		})
	}

	resBody := GetFoldersResponse{
		Address: addr,
		Folders: folders,
	}
	c.respond(w, http.StatusOK, resBody)
}
