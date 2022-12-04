package httpserver

import (
	"encoding/json"
	"mailboxsweep/inboxcredential"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type CredentialController struct {
	service inboxcredential.Service
	log     *logrus.Logger
}

func NewCredentialController(credentialService inboxcredential.Service, log *logrus.Logger) *CredentialController {
	return &CredentialController{
		service: credentialService,
		log:     log,
	}
}

func (c CredentialController) PostCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody CreateCredentialRequest
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		respondError(w, c.log, err, withStatusCode(http.StatusBadRequest))
		return
	}

	credInput := inboxcredential.CreateCredentialInput{
		Address:  reqBody.Address,
		Password: reqBody.Password,
	}
	addr, err := c.service.StoreCredential(ctx, credInput)
	if err != nil {
		respondError(w, c.log, err)
		return
	}

	res := CreateCredentialResponse{
		Address: addr,
	}
	respond(w, http.StatusOK, res)
}

const testHeaderCredentialAddress = "X-Test-Address"

// /credential
func (c CredentialController) GetCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr := req.Header.Get(testHeaderCredentialAddress)
	cred, err := c.service.FindCredential(ctx, addr)
	if err != nil {
		respondError(w, c.log, err)
		return
	}

	res := GetCredentialResponse{
		Address: cred.Address,
		Created: cred.Created.Format(time.RFC3339),
		Expires: time.Unix(cred.Expires, 0).Format(time.RFC3339),
	}
	respond(w, http.StatusOK, res)
}
