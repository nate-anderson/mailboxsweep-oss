package httpserver

import (
	"encoding/json"
	"fmt"
	"mailboxsweep/changeset"
	"net/http"
)

type ChangeSetController struct {
	service changeset.Service
	baseController
}

func (c ChangeSetController) UpdateChangeSet(w http.ResponseWriter, req *http.Request) {
	// ctx := req.Context()
	reqBody := ChangeSetRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	switch reqBody.Operation {
	case Delete:
		// @TODO: need to look up folder leaf from repo and add stored UIDs to the deletion set
		// this should be done in the service layer, but there is no service to currently handle this
		// what should it be called? Should the changesetservice definition be updated to accept
		// a leaf path instead of a set of UUIDs?
		// c.service.MarkForDelete(ctx, reqBody.Folder)
	case MarkRead:

	default:
		err := fmt.Errorf("unknown update operation '%s'", reqBody.Operation)
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

}
