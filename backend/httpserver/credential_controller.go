package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/imapserver"
	"mailboxsweep/inbox"
	"mailboxsweep/inboxcredential"
	"net/http"
	"time"

	"github.com/mcnijman/go-emailaddress"
	"github.com/sirupsen/logrus"
)

type CredentialController struct {
	service           inboxcredential.Service
	imapServerService imapserver.Service
	providerService   inbox.ProviderService
	log               *logrus.Logger
	baseController
}

func NewCredentialController(credentialService inboxcredential.Service, providerService inbox.ProviderService, imapServerService imapserver.Service, log *logrus.Logger) *CredentialController {
	return &CredentialController{
		service:           credentialService,
		providerService:   providerService,
		imapServerService: imapServerService,
		log:               log,
		baseController:    baseController{log},
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
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		// this should not happen
		c.log.Errorf("no address associated with context inside handler - THIS IS VERY BAD")
		c.respondAuthError(w, err)
		return
	}

	provider, err := c.providerService.GetProvider(ctx, address)
	if err != nil {
		// this will be a 404 if this provider isn't known, and the frontend should present a form
		// to collect the server connection info
		c.respondError(w, err)
		return
	}

	// This endpoint is only called for IMAP providers, because we get the Gmail credentials
	// when the user links their Google account, so it is safe to treat this as an IMAP credential
	credential, err := inboxcredential.NewIMAPInboxCredential(address, reqBody.Password)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusInternalServerError))
		return
	}

	_, err = provider.Login(address, credential)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadGateway))
		return
	}

	// end validation
	err = c.service.StoreCredential(ctx, credential)
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := CreateCredentialResponse{
		Address: address,
	}
	c.respond(w, http.StatusOK, res)
}

const testHeaderCredentialAddress = "X-Test-Address"

func (c CredentialController) GetInboxCredential(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	cred, err := c.service.FindCredential(ctx)
	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
		c.respondError(w, err)
		return
	}

	res := GetCredentialResponse{
		Address: cred.Address(),
		Created: cred.Created().Format(time.RFC3339),
		Expires: cred.Expires().Format(time.RFC3339),
	}
	c.respond(w, http.StatusOK, res)
}

func (c CredentialController) StoreImapServerAndCredentials(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	reqBody := StoreImapServerAndCredentialsRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	parsedEmail, err := emailaddress.Parse(addr)
	if err != nil {
		c.respondError(w, err)
		return
	}

	server := imapserver.Server{
		Domain:    parsedEmail.Domain,
		Host:      reqBody.ImapHost,
		Port:      reqBody.ImapPort,
		CreatedBy: &addr,
		IsGlobal:  false,
	}
	err = c.imapServerService.Store(ctx, server)
	if err != nil {
		if errors.Is(err, imapserver.ErrInvalidDomain) || errors.Is(err, imapserver.ErrInvalidHost) || errors.Is(err, imapserver.ErrInvalidPort) {
			c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
			return
		}
	}

	provider, err := c.providerService.GetProvider(ctx, addr)
	if err != nil {
		// this will be a 404 if this provider isn't known, and the frontend should present a form
		// to collect the server connection info
		c.respondError(w, err)
		return
	}

	// This endpoint is only called for IMAP providers, because we get the Gmail credentials
	// when the user links their Google account, so it is safe to treat this as an IMAP credential
	credential, err := inboxcredential.NewIMAPInboxCredential(addr, reqBody.Password)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusInternalServerError))
		return
	}

	_, err = provider.Login(addr, credential)
	if err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadGateway))
		return
	}

	// end validation

	err = c.service.StoreCredential(ctx, credential)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := CreateCredentialResponse{
		Address: addr,
	}
	c.respond(w, http.StatusOK, resBody)
}
