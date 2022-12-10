package httpserver

import (
	"encoding/json"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type CredentialController struct {
	service         inboxcredential.Service
	providerService *inbox.ProviderService
	log             *logrus.Logger
	baseController
}

func NewCredentialController(credentialService inboxcredential.Service, providerService *inbox.ProviderService, log *logrus.Logger) *CredentialController {
	return &CredentialController{
		service:         credentialService,
		providerService: providerService,
		log:             log,
		baseController:  baseController{log},
	}
}

func (c CredentialController) PostCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody CreateCredentialRequest
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	// @TODO: fix this with auth
	address := req.Header.Get(testAddressHeader)
	credInput := inboxcredential.CreateCredentialInput{
		Address:  address,
		Password: reqBody.Password,
	}

	// validate - yes, this is sloppy
	provider, err := c.providerService.GetProvider(ctx, address)
	if err != nil {
		c.respondError(w, err)
		return
	}

	_, err = provider.Login(address, reqBody.Password)
	if err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadGateway))
		return
	}

	addr, err := c.service.StoreCredential(ctx, credInput)
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := CreateCredentialResponse{
		Address: addr,
	}
	c.respond(w, http.StatusOK, res)
}

const testHeaderCredentialAddress = "X-Test-Address"

// /credential
func (c CredentialController) GetCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr := req.Header.Get(testHeaderCredentialAddress)
	cred, err := c.service.FindCredential(ctx, addr)
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := GetCredentialResponse{
		Address: cred.Address,
		Created: cred.Created.Format(time.RFC3339),
		Expires: time.Unix(cred.Expires, 0).Format(time.RFC3339),
	}
	c.respond(w, http.StatusOK, res)
}
