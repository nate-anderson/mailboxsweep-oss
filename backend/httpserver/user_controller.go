package httpserver

import (
	"encoding/json"
	"mailboxsweep"
	"mailboxsweep/inboxuser"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type UserController struct {
	baseController
	service inboxuser.Service
}

func NewUserController(service inboxuser.Service, log *logrus.Logger) *UserController {
	return &UserController{
		baseController: baseController{log},
		service:        service,
	}
}

func (c UserController) SetConsentedAgreements(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := SetUserAgreementsRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	agreements := make([]inboxuser.Agreement, len(reqBody.Agreements))
	for i, agrStr := range reqBody.Agreements {
		agr, err := inboxuser.ParseAgreement(agrStr)
		if err != nil {
			c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
			return
		}
		agreements[i] = agr
	}

	if err := c.service.Consent(ctx, agreements); err != nil {
		c.respondError(w, err)
		return
	}

	resBody := SetUserAgreementsResponse(reqBody)

	c.respond(w, http.StatusOK, resBody)
}

func (c UserController) UpdateUserName(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := SetUserNameRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	if err := c.service.SetName(ctx, reqBody.NameFirst, reqBody.NameLast); err != nil {
		c.respondError(w, err)
		return
	}

	msg := "OK"
	resBody := StatusResponse{
		Message: &msg,
		Status:  msg,
	}

	c.respond(w, http.StatusOK, resBody)
}

func (c UserController) ListUserAgreements(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	res := ListUserAgreementsResponse{}

	agreements, err := c.service.ListUserAgreements(ctx)
	if err != nil {
		c.respondError(w, err)
		return
	}

	for _, agr := range agreements {
		ua := UserAgreement{
			Key:      string(agr.Agreement),
			Required: agr.Required,
			Name:     agr.Name,
			Body:     &agr.Body,
		}
		agreedAtStr := agr.AgreedAt.Format(time.RFC3339)
		if agr.Agreed {
			ua.AgreedAt = &agreedAtStr
		}
		res.Agreements = append(res.Agreements, ua)
	}

	c.respond(w, http.StatusOK, res)
}

func (c UserController) GetUserData(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	user, err := c.service.FindByAddress(ctx, addr)
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := UserData{
		Address: user.Address,
	}
	if user.NameFirst != "" {
		res.NameFirst = &user.NameFirst
	}
	if user.NameLast != "" {
		res.NameLast = &user.NameLast
	}

	c.respond(w, http.StatusOK, res)
}

func (c UserController) GetUserAgreementsAgreementSlug(w http.ResponseWriter, req *http.Request, agreementSlug string) {
	ctx := req.Context()

	ua, err := c.service.GetAgreement(ctx, inboxuser.Agreement(agreementSlug))
	if err != nil {
		c.respondError(w, err)
		return
	}

	res := UserAgreement{
		Key:                string(ua.Agreement),
		Required:           ua.Required,
		Name:               ua.Name,
		Body:               &ua.Body,
		Description:        &ua.Description,
		AgreementUpdatedAt: &ua.AgreementUpdated,
	}
	agreedAtStr := ua.AgreedAt.Format(time.RFC3339)
	if ua.Agreed {
		res.AgreedAt = &agreedAtStr
	}

	c.respond(w, http.StatusOK, res)
}
