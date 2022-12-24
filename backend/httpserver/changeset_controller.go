package httpserver

import (
	"encoding/json"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ChangeSetController struct {
	service changeset.Service
	log     *logrus.Logger
	baseController
}

func NewChangeSetController(service changeset.Service, log *logrus.Logger) *ChangeSetController {
	return &ChangeSetController{
		service:        service,
		log:            log,
		baseController: baseController{log},
	}
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

func (c ChangeSetController) GetChangesets(w http.ResponseWriter, req *http.Request, params GetChangesetsParams) {
	ctx := req.Context()
	var changeSets []changeset.ChangeSet
	// yes it's sloppy - I want the address in the API response
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}
	if params.Folder != nil {
		changeSetMap, err := c.service.ListByInboxFolder(ctx, *params.Folder, false)
		if err != nil {
			c.respondError(w, err)
			return
		}
		for _, cs := range changeSetMap {
			changeSets = append(changeSets, cs)
		}

	} else {
		var err error
		changeSets, err = c.service.ListByInbox(ctx, false)
		if err != nil {
			c.respondError(w, err)
			return
		}

	}

	resultChangeSets := make([]ChangeSet, len(changeSets))
	for i, cs := range changeSets {
		op, err := operationToAPI(cs.Operation)
		if err != nil {
			c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
			return
		}
		resultChangeSets[i] = ChangeSet{
			Folder:    cs.Folder,
			Paths:     cs.Paths,
			Operation: op,
		}
	}

	res := ChangeSetResponse{
		Address:    addr,
		ChangeSets: resultChangeSets,
	}
	c.respond(w, http.StatusOK, res)
}

func operationToAPI(operation changeset.UpdateOperation) (UpdateOperation, error) {
	switch operation {
	case changeset.Delete:
		return Delete, nil
	case changeset.MarkRead:
		return MarkRead, nil
	default:
		return "", fmt.Errorf("unknown update operation '%s'", operation)
	}
}
