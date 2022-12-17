package httpserver

import (
	"encoding/json"
	"mailboxsweep/inboxuser"
	"net/http"
)

type UserController struct {
	baseController
	service inboxuser.Service
}

func (c UserController) SetConsentedAgreements(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := SetUserAgreementsRequest{}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	agreements := make([]inboxuser.Agreement, len(reqBody.Agreements))
	for i, agrStr := range reqBody.Agreements {
		agr, err := inboxuser.ParseAgreement(agrStr)
		if err != nil {
			c.respondError(w, err, withStatusCode(http.StatusBadRequest))
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
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
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
