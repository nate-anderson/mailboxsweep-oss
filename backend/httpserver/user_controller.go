package httpserver

import (
	"encoding/json"
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

	resBody := SetUserAgreementsResponse{
		Agreements: reqBody.Agreements,
	}

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
			Name:     agr.Description,
		}
		agreedAtStr := agr.AgreedAt.Format(time.RFC3339)
		if agr.Agreed {
			ua.AgreedAt = &agreedAtStr
		}
		res.Agreements = append(res.Agreements, ua)
	}

	c.respond(w, http.StatusOK, res)
}
