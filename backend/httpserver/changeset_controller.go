package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/inboxchangeset"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ChangeSetController struct {
	changeSetService      changeset.Service
	inboxChangeSetService inboxchangeset.Service
	log                   *logrus.Logger
	baseController
}

func NewChangeSetController(service changeset.Service, inboxChangeSetService inboxchangeset.Service, log *logrus.Logger) *ChangeSetController {
	return &ChangeSetController{
		changeSetService:      service,
		log:                   log,
		inboxChangeSetService: inboxChangeSetService,
		baseController:        baseController{log},
	}
}

func (c ChangeSetController) UpdateChangeSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := ChangeSetRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	resBody := UpdateChangeSetResponse{
		Folder: reqBody.Folder,
	}

	path := classification.Path(reqBody.Path)
	var err error
	switch reqBody.Operation {
	case Delete:
		resBody.MessageCount, err = c.inboxChangeSetService.AddLeafForDelete(ctx, reqBody.Folder, path)
		if err != nil {
			c.respondError(w, err)
			return
		}
	case MarkRead:
		resBody.MessageCount, err = c.inboxChangeSetService.AddLeafForMarkRead(ctx, reqBody.Folder, path)
		if err != nil {
			c.respondError(w, err)
			return
		}

	default:
		err := fmt.Errorf("unknown update operation '%s'", reqBody.Operation)
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	c.respond(w, http.StatusOK, resBody)
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
		changeSetMap, err := c.changeSetService.ListByInboxFolder(ctx, *params.Folder, false)
		if err != nil {
			c.respondError(w, err)
			return
		}
		for _, cs := range changeSetMap {
			changeSets = append(changeSets, cs)
		}

	} else {
		var err error
		changeSets, err = c.changeSetService.ListByInbox(ctx, false)
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

func (c ChangeSetController) CommitChangesets(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := CommitRequest{}
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	if !reqBody.Confirm {
		c.respondError(w, errors.New("request must set confirm=true"), withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	if err = c.inboxChangeSetService.CommitChanges(ctx); err != nil {
		c.respondError(w, err)
		return
	}

	resBody := StatusResponse{}
	resBody.Message = ptr("OK!")
	resBody.Status = "SUCCESS"

	c.respond(w, http.StatusOK, resBody)
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
