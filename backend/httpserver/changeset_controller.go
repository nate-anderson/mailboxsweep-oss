package httpserver

import (
	"context"
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
		MessageCount: 0,
	}

	var updateFunc func(ctx context.Context, path classification.Path) (int, error)

	var err error
	switch reqBody.Operation {
	case Delete:
		switch reqBody.AddOrRemove {
		case Remove:
			updateFunc = c.inboxChangeSetService.RemoveLeafForDelete
		case Add:
			updateFunc = c.inboxChangeSetService.AddLeafForDelete
		default:
			c.respondError(w, fmt.Errorf("addOrDelete must be one of 'add' or 'delete'"), withOverrideStatusCode(http.StatusBadRequest))
			return
		}
	case MarkRead:
		switch reqBody.AddOrRemove {
		case Remove:
			updateFunc = c.inboxChangeSetService.RemoveLeafForMarkRead
		case Add:
			updateFunc = c.inboxChangeSetService.AddLeafForMarkRead
		default:
			c.respondError(w, fmt.Errorf("addOrDelete must be one of 'add' or 'delete'"), withOverrideStatusCode(http.StatusBadRequest))
			return
		}
	default:
		err := fmt.Errorf("unknown update operation '%s'", reqBody.Operation)
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	if updateFunc == nil {
		c.respondError(w, fmt.Errorf("no update function selected"))
		return
	}

	path := classification.Path(reqBody.Path)
	n, err := updateFunc(ctx, path)
	if err != nil {
		c.respondError(w, err)
		return
	}
	resBody.MessageCount = n

	c.respond(w, http.StatusOK, resBody)
}

func (c ChangeSetController) GetChangeset(w http.ResponseWriter, req *http.Request, params GetChangesetParams) {
	ctx := req.Context()

	// yes it's sloppy - I want the address in the API response
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	operation, err := apiToOperation(params.Operation)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	cs, err := c.changeSetService.FindByInboxAndOperation(ctx, operation, false)
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := ChangeSetResponse{
		Address: addr,
		ChangeSet: ChangeSet{
			Operation: params.Operation,
			Paths:     cs.Paths,
		},
	}
	c.respond(w, http.StatusOK, res)
}

func (c ChangeSetController) CommitChangeset(w http.ResponseWriter, req *http.Request) {
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

	operation, err := apiToOperation(reqBody.Operation)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	if err = c.inboxChangeSetService.CommitChanges(ctx, operation); err != nil {
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

func apiToOperation(o UpdateOperation) (changeset.UpdateOperation, error) {
	switch o {
	case MarkRead:
		return changeset.MarkRead, nil
	case Delete:
		return changeset.Delete, nil
	default:
		return "", fmt.Errorf("unknown update operation (API) '%s'", o)
	}
}
