package httpserver

import (
	"encoding/json"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"mailboxsweep/session"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type CredentialController struct {
	service         inboxcredential.Service
	providerService inbox.ProviderService
	log             *logrus.Logger
	baseController
}

func NewCredentialController(credentialService inboxcredential.Service, providerService inbox.ProviderService, log *logrus.Logger) *CredentialController {
	return &CredentialController{
		service:         credentialService,
		providerService: providerService,
		log:             log,
		baseController:  baseController{log},
	}
}

func (c CredentialController) StoreInboxCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody CreateCredentialRequest
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	// validate - yes, this is sloppy
	// hard to do this inside the credential service without an import cycle, @TODO revisit
	address, err := session.UserAddressFromContext(ctx)
	if err != nil {
		// this should not happen
		c.log.Errorf("no address associated with context inside handler - THIS IS VERY BAD")
		c.respondAuthError(w, err)
		return
	}
	provider, err := c.providerService.GetProvider(ctx, address)
	if err != nil {
		c.respondError(w, err)
		return
	}

	_, err = provider.Login(address, reqBody.Password)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadGateway))
		return
	}

	// end validation

	addr, err := c.service.StoreCredential(ctx, reqBody.Password)
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
func (c CredentialController) GetInboxCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr, err := session.UserAddressFromContext(ctx)
	if err != nil {
		c.log.Errorf("no address associated with context inside handler - THIS IS VERY BAD")
		c.respondAuthError(w, err)
		return
	}
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
