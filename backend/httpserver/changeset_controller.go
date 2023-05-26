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
		UpdatedPaths: [][]string{},
	}

	var updateFunc func(ctx context.Context, path classification.Path) (*inboxchangeset.ChangeResult, error)

	var err error
	switch reqBody.Operation {
	case Delete:
		switch reqBody.AddOrRemove {
		case Remove:
			updateFunc = c.inboxChangeSetService.RemovePathForDelete
		case Add:
			updateFunc = c.inboxChangeSetService.AddPathForDelete
		default:
			c.respondError(w, fmt.Errorf("addOrRemove must be one of 'add' or 'delete'"), withOverrideStatusCode(http.StatusBadRequest))
			return
		}
	case MarkRead:
		switch reqBody.AddOrRemove {
		case Remove:
			updateFunc = c.inboxChangeSetService.RemovePathForMarkRead
		case Add:
			updateFunc = c.inboxChangeSetService.AddPathForMarkRead
		default:
			c.respondError(w, fmt.Errorf("addOrRemove must be one of 'add' or 'delete'"), withOverrideStatusCode(http.StatusBadRequest))
			return
		}
	default:
		err := fmt.Errorf("unknown update operation '%s'", reqBody.Operation)
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	path := classification.Path(reqBody.Path)
	result, err := updateFunc(ctx, path)
	if err != nil {
		c.respondError(w, err)
		return
	}
	resBody.MessageCount = result.MsgCountDiff
	for _, p := range result.Paths {
		resBody.UpdatedPaths = append(resBody.UpdatedPaths, p)
	}

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

	cs, size, err := c.inboxChangeSetService.FindByOperation(ctx, operation)
	if err != nil {
		c.respondError(w, err)
		return
	}

	paths := [][]string{}
	for _, path := range cs.Paths {
		paths = append(paths, classification.ParsePath(path))
	}
	apiOperation, err := operationToAPI(cs.Operation)
	if err != nil {
		c.respondError(w, err)
		return
	}
	res := ChangeSetResponse{
		Address: addr,
		ChangeSet: ChangeSet{
			Operation: apiOperation,
			Paths:     paths,
		},
		Size: size,
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

func (c ChangeSetController) ClearChangeset(w http.ResponseWriter, req *http.Request, params ClearChangesetParams) {
	ctx := req.Context()

	operation, err := apiToOperation(params.Operation)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	err = c.changeSetService.ClearSelection(ctx, operation)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := StatusResponse{
		Status: "SUCCESS",
	}
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
